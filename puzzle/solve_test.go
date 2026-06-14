package puzzle

import "testing"

var known = Grid{
	{5, 3, 0, 0, 7, 0, 0, 0, 0},
	{6, 0, 0, 1, 9, 5, 0, 0, 0},
	{0, 9, 8, 0, 0, 0, 0, 6, 0},
	{8, 0, 0, 0, 6, 0, 0, 0, 3},
	{4, 0, 0, 8, 0, 3, 0, 0, 1},
	{7, 0, 0, 0, 2, 0, 0, 0, 6},
	{0, 6, 0, 0, 0, 0, 2, 8, 0},
	{0, 0, 0, 4, 1, 9, 0, 0, 5},
	{0, 0, 0, 0, 8, 0, 0, 7, 9},
}

func TestCountUnique(t *testing.T) {
	n := CountSolutions(known, 2)
	if n != 1 {
		t.Fatalf("CountSolutions(known, 2) = %d; want 1", n)
	}
}

func TestCountAmbiguous(t *testing.T) {
	n := CountSolutions(Grid{}, 2)
	if n != 2 {
		t.Fatalf("CountSolutions(empty, 2) = %d; want 2", n)
	}
}

func TestCountUnsolvable(t *testing.T) {
	// Two 5s in the first row — no solution possible.
	bad := Grid{}
	bad[0][0] = 5
	bad[0][1] = 5
	n := CountSolutions(bad, 2)
	if n != 0 {
		t.Fatalf("CountSolutions(bad, 2) = %d; want 0", n)
	}
}

func TestCountDoesNotMutate(t *testing.T) {
	original := known
	CountSolutions(known, 2)
	if known != original {
		t.Fatal("CountSolutions mutated the caller's grid")
	}
}
