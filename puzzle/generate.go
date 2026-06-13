package puzzle

import (
	"fmt"
	"math/rand"
)

// Difficulty target clue counts and the accepted range. Generation aims for
// target but accepts any result within [min, max]; the actual count is reported.
type difficultySpec struct {
	target, min, max int
}

var difficulties = map[string]difficultySpec{
	"easy":   {target: 36, min: 34, max: 40},
	"medium": {target: 30, min: 28, max: 34},
	"hard":   {target: 25, min: 23, max: 30},
}

// Puzzle holds a generated sudoku puzzle with its solution and metadata.
type Puzzle struct {
	ID         int
	Clues      Grid   // puzzle (0 = empty cell)
	Solution   Grid   // fully solved grid
	Difficulty string // "easy" | "medium" | "hard"
	ClueCount  int    // actual number of non-empty cells in Clues
}

// Generate builds one puzzle of the given difficulty using rng for all
// randomness (so output is reproducible for a fixed seed). id is stamped onto
// the result. It returns an error if difficulty is not one of easy/medium/hard.
func Generate(id int, difficulty string, rng *rand.Rand) (Puzzle, error) {
	spec, ok := difficulties[difficulty]
	if !ok {
		return Puzzle{}, fmt.Errorf("unknown difficulty %q: must be easy, medium, or hard", difficulty)
	}

	const maxAttempts = 20

	var bestClues Grid
	var bestSolution Grid
	bestCount := 81 // start with worst possible (all filled)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		solution := generateSolved(rng)
		clues, count := carve(solution, spec, rng)

		if count < bestCount {
			bestCount = count
			bestClues = clues
			bestSolution = solution
		}

		// Accept if within range.
		if count >= spec.min && count <= spec.max {
			break
		}
	}

	return Puzzle{
		ID:         id,
		Clues:      bestClues,
		Solution:   bestSolution,
		Difficulty: difficulty,
		ClueCount:  bestCount,
	}, nil
}

// generateSolved creates a fully solved sudoku grid using the rng for all
// randomness. It starts from a valid base pattern and applies structure-
// preserving transformations to produce varied grids.
func generateSolved(rng *rand.Rand) Grid {
	// A. Build base pattern: base[r][c] = ((r*3 + r/3 + c) % 9) + 1
	var g Grid
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			g[r][c] = (r*3+r/3+c)%9 + 1
		}
	}

	// B. Apply structure-preserving shuffles.

	// 1. Relabel digits: pick a random permutation of 1..9 and map every cell.
	perm := rng.Perm(9) // perm[i] gives the new digit-1 for old digit (i+1)
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			g[r][c] = perm[g[r][c]-1] + 1
		}
	}

	// 2. Shuffle rows within each band.
	for band := 0; band < 3; band++ {
		// Get 3 row indices within this band: band*3, band*3+1, band*3+2.
		rows := []int{band*3, band*3 + 1, band*3 + 2}
		rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		// Apply the permutation: extract rows, put them back in shuffled order.
		var tmp [3][9]int
		for i := 0; i < 3; i++ {
			tmp[i] = g[rows[i]]
		}
		for i := 0; i < 3; i++ {
			g[band*3+i] = tmp[i]
		}
	}

	// 3. Shuffle columns within each stack.
	for stack := 0; stack < 3; stack++ {
		cols := []int{stack*3, stack*3 + 1, stack*3 + 2}
		rng.Shuffle(len(cols), func(i, j int) { cols[i], cols[j] = cols[j], cols[i] })
		// Extract the 3 columns in the original shuffled order, then put them back.
		var tmp [3][9]int
		for i := 0; i < 3; i++ {
			for r := 0; r < 9; r++ {
				tmp[i][r] = g[r][cols[i]]
			}
		}
		for i := 0; i < 3; i++ {
			for r := 0; r < 9; r++ {
				g[r][stack*3+i] = tmp[i][r]
			}
		}
	}

	// 4. Shuffle the order of the 3 bands.
	bandOrder := rng.Perm(3)
	var tmp3 [3][9][9]int // temp storage for full bands
	for b := 0; b < 3; b++ {
		for i := 0; i < 3; i++ {
			tmp3[b][i] = g[b*3+i]
		}
	}
	for b := 0; b < 3; b++ {
		for i := 0; i < 3; i++ {
			g[b*3+i] = tmp3[bandOrder[b]][i]
		}
	}

	// 5. Shuffle the order of the 3 stacks.
	stackOrder := rng.Perm(3)
	var tmp9 [9][9]int = g
	for s := 0; s < 3; s++ {
		src := stackOrder[s]
		for r := 0; r < 9; r++ {
			g[r][s*3+0] = tmp9[r][src*3+0]
			g[r][s*3+1] = tmp9[r][src*3+1]
			g[r][s*3+2] = tmp9[r][src*3+2]
		}
	}

	// 6. Optionally transpose with 50% probability.
	if rng.Intn(2) == 1 {
		var t Grid
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				t[c][r] = g[r][c]
			}
		}
		g = t
	}

	return g
}

// carve removes cells from a copy of solution to create a puzzle with as few
// clues as possible while maintaining uniqueness. It stops early if it reaches
// the target. Returns the clue grid and the clue count.
func carve(solution Grid, spec difficultySpec, rng *rand.Rand) (Grid, int) {
	clues := solution

	// Build shuffled list of all 81 positions.
	positions := rng.Perm(81)

	removed := 0
	for _, pos := range positions {
		r := pos / 9
		c := pos % 9

		if clues[r][c] == 0 {
			continue // already empty
		}

		saved := clues[r][c]
		clues[r][c] = 0

		if CountSolutions(clues, 2) == 1 {
			removed++
		} else {
			clues[r][c] = saved // restore
		}

		// Stop early if we've reached the target clue count.
		if 81-removed == spec.target {
			break
		}
	}

	return clues, 81 - removed
}
