// Package puzzle generates sudoku grids and validates their uniqueness.
package puzzle

// Grid is a 9x9 sudoku grid. 0 means an empty cell.
type Grid [9][9]int

// valid reports whether placing val at (row,col) keeps the grid legal.
// It ignores the current contents of (row,col) itself.
func valid(g *Grid, row, col, val int) bool {
	// Check row.
	for c := 0; c < 9; c++ {
		if c != col && g[row][c] == val {
			return false
		}
	}
	// Check column.
	for r := 0; r < 9; r++ {
		if r != row && g[r][col] == val {
			return false
		}
	}
	// Check 3x3 box.
	boxRow := row / 3 * 3
	boxCol := col / 3 * 3
	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			if (r != row || c != col) && g[r][c] == val {
				return false
			}
		}
	}
	return true
}

// gridIsValid reports whether all pre-filled (non-zero) cells in g are
// mutually consistent — i.e., no row, column, or 3x3 box contains a
// duplicate non-zero value. This is used to short-circuit on an
// already-contradictory input grid.
func gridIsValid(g *Grid) bool {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			v := g[r][c]
			if v == 0 {
				continue
			}
			// Temporarily clear the cell so valid() can check it without
			// excluding itself.
			g[r][c] = 0
			ok := valid(g, r, c, v)
			g[r][c] = v
			if !ok {
				return false
			}
		}
	}
	return true
}

// countSolutions is the internal recursive helper that counts solutions up to
// limit. It mutates g in place and backtracks.
func countSolutions(g *Grid, limit int) int {
	// Find the first empty cell in row-major order.
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if g[r][c] != 0 {
				continue
			}
			// Empty cell found — try each candidate.
			count := 0
			for val := 1; val <= 9; val++ {
				if valid(g, r, c, val) {
					g[r][c] = val
					count += countSolutions(g, limit-count)
					g[r][c] = 0
					if count >= limit {
						return count
					}
				}
			}
			return count
		}
	}
	// No empty cell found — this is a complete solution.
	return 1
}

// CountSolutions returns the number of distinct solutions of g, counting up to
// at most limit (it stops early once limit is reached). Use limit=2 for
// uniqueness checks: a return of 1 means a unique solution, >=2 means
// ambiguous, 0 means unsolvable. g is not mutated.
func CountSolutions(g Grid, limit int) int {
	// g is already a copy (passed by value); operate on a pointer to it.
	// Reject grids that are already contradictory.
	if !gridIsValid(&g) {
		return 0
	}
	return countSolutions(&g, limit)
}
