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

// isComplete checks that g has no zeros and that every row, column, and 3x3
// box contains exactly the digits 1..9.
func isComplete(g Grid) bool {
	for r := 0; r < 9; r++ {
		var rowSeen, colSeen [10]bool
		for c := 0; c < 9; c++ {
			rv := g[r][c]
			cv := g[c][r]
			if rv < 1 || rv > 9 || cv < 1 || cv > 9 {
				return false
			}
			if rowSeen[rv] || colSeen[cv] {
				return false
			}
			rowSeen[rv] = true
			colSeen[cv] = true
		}
	}
	for br := 0; br < 3; br++ {
		for bc := 0; bc < 3; bc++ {
			var boxSeen [10]bool
			for r := br * 3; r < br*3+3; r++ {
				for c := bc * 3; c < bc*3+3; c++ {
					v := g[r][c]
					if v < 1 || v > 9 || boxSeen[v] {
						return false
					}
					boxSeen[v] = true
				}
			}
		}
	}
	return true
}

func TestSolveKnownPuzzle(t *testing.T) {
	g := known
	if !Solve(&g) {
		t.Fatal("Solve returned false for a known solvable puzzle")
	}
	if !isComplete(g) {
		t.Fatal("Solve produced an incomplete or illegal grid")
	}
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
