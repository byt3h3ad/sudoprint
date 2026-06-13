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
	"easy":   {target: 35, min: 33, max: 38},
	"medium": {target: 29, min: 27, max: 32},
	"hard":   {target: 24, min: 22, max: 32},
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

// carve removes cells from a copy of solution in 180-degree rotationally
// symmetric pairs, so the resulting clue pattern is symmetric (the look of a
// published puzzle). The center cell (4,4) is its own mirror and is removed
// singly. A group is removed only if doing so keeps the solution unique;
// otherwise the whole group is restored. Carving stops once the clue count
// reaches or drops just below spec.target. Returns the clue grid and the
// actual clue count. All randomness uses rng.
func carve(solution Grid, spec difficultySpec, rng *rand.Rand) (Grid, int) {
	clues := solution

	// Build the 41 symmetric groups: pairs {pos, 80-pos} for pos in 0..39,
	// plus the center {40}. Shuffle the group order with rng.
	type cell struct{ r, c int }
	var groups [][]cell
	for pos := 0; pos < 40; pos++ {
		p := 80 - pos
		groups = append(groups, []cell{
			{pos / 9, pos % 9},
			{p / 9, p % 9},
		})
	}
	groups = append(groups, []cell{{4, 4}}) // center, self-symmetric
	rng.Shuffle(len(groups), func(i, j int) { groups[i], groups[j] = groups[j], groups[i] })

	removed := 0
	for _, g := range groups {
		// Skip if every cell in the group is already empty.
		allEmpty := true
		for _, cl := range g {
			if clues[cl.r][cl.c] != 0 {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		// Tentatively remove the whole group.
		saved := make([]int, len(g))
		for i, cl := range g {
			saved[i] = clues[cl.r][cl.c]
			clues[cl.r][cl.c] = 0
		}

		if CountSolutions(clues, 2) == 1 {
			removed += len(g)
		} else {
			// Restore the whole group.
			for i, cl := range g {
				clues[cl.r][cl.c] = saved[i]
			}
		}

		// Stop once at or just below the target (steps of 2 mean exact target
		// may be unreachable; <= keeps us near it, and Generate's range check
		// accepts the result).
		if 81-removed <= spec.target {
			break
		}
	}

	return clues, 81 - removed
}
