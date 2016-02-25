package xiao

import (
	"github.com/OpenWhiteBox/primitives/encoding"
	"github.com/OpenWhiteBox/primitives/matrix"
	"github.com/OpenWhiteBox/primitives/random"
	"github.com/OpenWhiteBox/primitives/table"

	"github.com/OpenWhiteBox/AES/constructions/common"
	"github.com/OpenWhiteBox/AES/constructions/saes"
)

// StandardAES form of Xiao-Lai:
//
// func (constr Construction) Encrypt(dst, src []byte) {
// 	roundKeys := constr.StretchedKey()
// 	copy(dst, src)
//
// 	for i := 0; i <= 9; i++ {
// 		constr.ShiftRows(roundKeys[i])
// 	}
//
// 	for i := 0; i < 9; i++ {
// 		constr.ShiftRows(dst)
// 		constr.AddRoundKey(roundKeys[i], dst)
// 		constr.SubBytes(dst)
// 		constr.MixColumns(dst)
// 	}
//
// 	constr.ShiftRows(dst)
// 	constr.AddRoundKey(roundKeys[9], dst)
// 	constr.SubBytes(dst)
// 	constr.AddRoundKey(roundKeys[10], dst)
// }

// generateRoundMaterial creates the TMC (TBox + MixColumns) tables.
func generateRoundMaterial(rs *random.Source, out *Construction, hidden func(int, int) table.DoubleToWord) {
	for round := 0; round < 10; round++ {
		for pos := 0; pos < 16; pos += 2 {
			out.TBoxMixCol[round][pos/2] = encoding.DoubleToWordTable{
				encoding.NewDoubleLinear(common.MixingBijection(rs, 16, round, pos/2)),
				encoding.InverseWord{
					encoding.NewWordLinear(common.MixingBijection(rs, 32, round, pos/4)),
				},
				hidden(round, pos),
			}
		}
	}
}

// generateBarriers creates the encoding barriers between rounds that compute ShiftRows and re-encodes data.
func generateBarriers(rs *random.Source, out *Construction, inputMask, outputMask, sr *matrix.Matrix) {
	// Generate the ShiftRows and re-encoding matrices.
	out.ShiftRows[0] = maskSwap(rs, 16, 0).Compose(*sr).Compose(*inputMask)

	for round := 1; round < 10; round++ {
		out.ShiftRows[round] = maskSwap(rs, 16, round).Compose(*sr).Compose(maskSwap(rs, 32, round-1))
	}

	// We need to apply a final matrix transformation to convert the double-level encoding to a block-level one.
	out.FinalMask = outputMask.Compose(maskSwap(rs, 32, 9))
}

// GenerateEncryptionKeys creates a white-boxed version of the AES key `key` for encryption, with any non-determinism
// generated by `seed`.
func GenerateEncryptionKeys(key, seed []byte, opts common.KeyGenerationOpts) (out Construction, inputMask, outputMask matrix.Matrix) {
	rs := random.NewSource("Xiao Encryption", seed)

	constr := saes.Construction{key}
	roundKeys := constr.StretchedKey()

	// Apply ShiftRows to round keys 0 to 9.
	for k := 0; k < 10; k++ {
		constr.ShiftRows(roundKeys[k])
	}

	hidden := func(round, pos int) table.DoubleToWord {
		if round == 9 {
			return tBox{
				[2]table.Byte{
					common.TBox{constr, roundKeys[9][pos+0], roundKeys[10][pos+0]},
					common.TBox{constr, roundKeys[9][pos+1], roundKeys[10][pos+1]},
				},
				sideFromPos(pos),
			}
		} else {
			return tBoxMixCol{
				[2]table.Byte{
					common.TBox{constr, roundKeys[round][pos+0], 0x00},
					common.TBox{constr, roundKeys[round][pos+1], 0x00},
				},
				mixColumns,
				sideFromPos(pos),
			}
		}
	}

	common.GenerateMasks(&rs, opts, &inputMask, &outputMask)
	generateRoundMaterial(&rs, &out, hidden)
	generateBarriers(&rs, &out, &inputMask, &outputMask, &shiftRows)

	return out, inputMask, outputMask
}

// GenerateDecryptionKeys creates a white-boxed version of the AES key `key` for decryption, with any non-determinism
// generated by `seed`.
func GenerateDecryptionKeys(key, seed []byte, opts common.KeyGenerationOpts) (out Construction, inputMask, outputMask matrix.Matrix) {
	rs := random.NewSource("Xiao Decryption", seed)

	constr := saes.Construction{key}
	roundKeys := constr.StretchedKey()

	// Apply UnShiftRows to round keys 10.
	constr.UnShiftRows(roundKeys[10])

	hidden := func(round, pos int) table.DoubleToWord {
		if round == 0 {
			return tBoxMixCol{
				[2]table.Byte{
					common.InvTBox{constr, roundKeys[10][pos+0], roundKeys[9][pos+0]},
					common.InvTBox{constr, roundKeys[10][pos+1], roundKeys[9][pos+1]},
				},
				unMixColumns,
				sideFromPos(pos),
			}
		} else if 0 < round && round < 9 {
			return tBoxMixCol{
				[2]table.Byte{
					common.InvTBox{constr, 0x00, roundKeys[9-round][pos+0]},
					common.InvTBox{constr, 0x00, roundKeys[9-round][pos+1]},
				},
				unMixColumns,
				sideFromPos(pos),
			}
		} else {
			return tBox{
				[2]table.Byte{
					common.InvTBox{constr, 0x00, roundKeys[0][pos+0]},
					common.InvTBox{constr, 0x00, roundKeys[0][pos+1]},
				},
				sideFromPos(pos),
			}
		}
	}

	common.GenerateMasks(&rs, opts, &inputMask, &outputMask)
	generateRoundMaterial(&rs, &out, hidden)
	generateBarriers(&rs, &out, &inputMask, &outputMask, &unShiftRows)

	return out, inputMask, outputMask
}
