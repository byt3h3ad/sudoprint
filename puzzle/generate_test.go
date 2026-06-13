package puzzle

import (
	"math/rand"
	"testing"
)

// isValidSolution checks that g has no zeros and that every row, column, and
// 3x3 box contains exactly the digits 1..9. Named differently from isComplete
// in solve_test.go to avoid redeclaration.
func isValidSolution(g Grid) bool {
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

func TestGenerateUnique(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		n := CountSolutions(p.Clues, 2)
		if n != 1 {
			t.Errorf("Generate(%q): CountSolutions(Clues,2) = %d; want 1", diff, n)
		}
	}
}

func TestGenerateSolutionValid(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		if !isValidSolution(p.Solution) {
			t.Errorf("Generate(%q): Solution is not a valid complete grid", diff)
		}
	}
}

func TestGenerateCluesSubsetOfSolution(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if p.Clues[r][c] != 0 && p.Clues[r][c] != p.Solution[r][c] {
					t.Errorf("Generate(%q): Clues[%d][%d]=%d but Solution[%d][%d]=%d",
						diff, r, c, p.Clues[r][c], r, c, p.Solution[r][c])
				}
			}
		}
	}
}

func TestClueCountAccurate(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		actual := 0
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if p.Clues[r][c] != 0 {
					actual++
				}
			}
		}
		if p.ClueCount != actual {
			t.Errorf("Generate(%q): ClueCount=%d but actual non-zero cells=%d", diff, p.ClueCount, actual)
		}
	}
}

func TestClueCountInRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		spec := difficulties[diff]
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		if p.ClueCount < spec.min || p.ClueCount > spec.max {
			t.Errorf("Generate(%q): ClueCount=%d not in [%d, %d]", diff, p.ClueCount, spec.min, spec.max)
		}
	}
}

func TestDeterministic(t *testing.T) {
	rng1 := rand.New(rand.NewSource(42))
	p1, err := Generate(1, "medium", rng1)
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	rng2 := rand.New(rand.NewSource(42))
	p2, err := Generate(1, "medium", rng2)
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}

	if p1.Clues != p2.Clues {
		t.Error("Deterministic: Clues differ between two calls with the same seed")
	}
	if p1.Solution != p2.Solution {
		t.Error("Deterministic: Solution differs between two calls with the same seed")
	}
}

func TestUnknownDifficulty(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	_, err := Generate(1, "extreme", rng)
	if err == nil {
		t.Error("Generate with unknown difficulty should return non-nil error")
	}
}

// TestCluesAreSymmetric verifies that the clue pattern produced by Generate
// has 180-degree rotational symmetry: cell (r,c) is empty iff cell (8-r,8-c)
// is also empty.
func TestCluesAreSymmetric(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, diff := range []string{"easy", "medium", "hard"} {
		p, err := Generate(1, diff, rng)
		if err != nil {
			t.Fatalf("Generate(%q): unexpected error: %v", diff, err)
		}
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if (p.Clues[r][c] == 0) != (p.Clues[8-r][8-c] == 0) {
					t.Errorf("Generate(%q): asymmetric clue pattern at (%d,%d): Clues[%d][%d]=%d, Clues[%d][%d]=%d",
						diff, r, c, r, c, p.Clues[r][c], 8-r, 8-c, p.Clues[8-r][8-c])
				}
			}
		}
	}
}

// TestClueCountInRangeMultiSeed checks that across seeds 0..7 and every
// difficulty, the generated puzzle has a unique solution and its clue count
// falls within the difficulty's configured [min, max].
func TestClueCountInRangeMultiSeed(t *testing.T) {
	for seed := 0; seed < 8; seed++ {
		for _, diff := range []string{"easy", "medium", "hard"} {
			spec := difficulties[diff]
			rng := rand.New(rand.NewSource(int64(seed)))
			p, err := Generate(1, diff, rng)
			if err != nil {
				t.Fatalf("seed %d Generate(%q): unexpected error: %v", seed, diff, err)
			}
			if CountSolutions(p.Clues, 2) != 1 {
				t.Errorf("seed %d Generate(%q): puzzle is not uniquely solvable", seed, diff)
			}
			if p.ClueCount < spec.min || p.ClueCount > spec.max {
				t.Errorf("seed %d Generate(%q): ClueCount=%d not in [%d, %d]",
					seed, diff, p.ClueCount, spec.min, spec.max)
			}
		}
	}
}
