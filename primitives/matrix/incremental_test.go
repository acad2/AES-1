package matrix

import (
	"fmt"
	"testing"

	"crypto/rand"
)

func ExampleIncrementalMatrix() {
	im := NewIncrementalMatrix(128)

	for !im.FullyDefined() {
		im.Add(GenerateRandomRow(rand.Reader, 128))
	}

	fmt.Println(im.Matrix())
}

func TestIncrementalMatrix(t *testing.T) {
	im := NewIncrementalMatrix(128)

	m := GenerateRandom(rand.Reader, 128)
	mInv, _ := m.Invert()

	for _, row := range m[0:126] {
		ok := im.Add(row)
		if !ok {
			t.Fatalf("Failed to add row from invertible matrix.")
		}
	}

	if im.FullyDefined() {
		t.Fatalf("FullyDefined returned true without all rows.")
	}

	ok1 := im.Add(m[3].Add(m[6]).Add(m[100]).Add(m[121]))
	ok2 := im.Add(m[8].Add(m[73]).Add(m[98]).Add(m[100]))

	if ok1 || ok2 {
		t.Fatalf("Add returned true on redundant vector.")
	} else if len(im.raw) != 126 || len(im.simplest) != 126 || len(im.inverse) != 126 {
		t.Fatalf("Add mutated state on redundant vector.")
	} else if im.FullyDefined() {
		t.Fatalf("FullyDefined returned true after being given dependent rows.")
	}

	for _, row := range m[126:] {
		ok := im.Add(row)
		if !ok {
			t.Fatalf("Failed to add row from invertible matrix.")
		}
	}

	if !im.FullyDefined() {
		t.Fatalf("FullDefined returned false on IncrementalMatrix with all rows.")
	}

	mT, mTinv := im.Matrix(), im.Inverse()

	for i := 0; i < 128; i++ {
		for j := 0; j < rowsToColumns(128); j++ {
			if m[i][j] != mT[i][j] {
				t.Fatalf("Raw matrix is different than original!")
			}

			if mInv[i][j] != mTinv[i][j] {
				t.Fatalf("Incrementally derived inverse is different than real inverse!")
			}
		}
	}
}

func TestIncrementalNovel(t *testing.T) {
	im := NewIncrementalMatrix(128)
	for im.Len() < 126 {
		im.Add(GenerateRandomRow(rand.Reader, 128))
	}

	for i := 0; i < 100; i++ {
		cand := im.Novel()
		if im.IsInSpan(cand) {
			t.Fatal("Novel returned row that was in span of matrix.")
		}
	}
}
