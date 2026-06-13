# Plan 002: Implement the backtracking solver with a solution-counter

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: This plan was written against commit `e90880a`.
> Run `git diff --stat e90880a..HEAD -- puzzle/`. Plan 001 should have added
> `puzzle/doc.go` and `puzzle/doc_test.go` and nothing else under `puzzle/`. If
> `puzzle/solve.go` already exists, read it and compare against the "Target"
> below before proceeding; on a mismatch, STOP and report.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/001-scaffold-and-baseline.md
- **Category**: tests + correctness
- **Planned at**: commit `e90880a`, 2026-06-13

## Why this matters

The solver is the correctness foundation of the whole tool. Generation
(plan 003) proves a puzzle has a *unique* solution by asking the solver to count
solutions and short-circuit at 2. If the counter is wrong, every generated
puzzle could be ambiguous or unsolvable, and nothing downstream would catch it.
This plan builds the solver test-first so its behavior is pinned before anything
depends on it (addresses the spec's complete absence of tests).

## Current state

`puzzle/` contains only the placeholder `doc.go` (`package puzzle`) and
`doc_test.go` from plan 001. The `Grid` type does not exist yet — define it here.

Design intent from `PLAN.md` §105–111: a standard backtracking solver used for
(1) uniqueness checking during generation (count solutions, short-circuit at 2)
and (2) verifying a completed grid.

## Commands you will need

| Purpose   | Command                  | Expected on success        |
|-----------|--------------------------|----------------------------|
| Build     | `go build ./...`         | exit 0                     |
| Vet       | `go vet ./...`           | exit 0                     |
| Test pkg  | `go test ./puzzle/`      | `ok  	sudoprint/puzzle`     |
| Verbose   | `go test -v ./puzzle/`   | each test prints `--- PASS`|

## Scope

**In scope**:
- `puzzle/solve.go` (create)
- `puzzle/solve_test.go` (create)
- `puzzle/doc.go` (you may delete it once `solve.go` declares the package and the
  `Grid` type; or keep it — both are fine)
- `puzzle/doc_test.go` (delete once real tests exist)

**Out of scope** (do NOT touch):
- `puzzle/generate.go` — plan 003 owns generation.
- `render/`, `main.go` — not created yet.

## Git workflow

Not a git repository. Do not init/commit/push. Create files on disk.

## Target

Define the shared grid type and the solver API in `puzzle/solve.go`:

```go
package puzzle

// Grid is a 9x9 sudoku grid. 0 means an empty cell.
type Grid [9][9]int

// valid reports whether placing val at (row,col) keeps the grid legal.
// It ignores the current contents of (row,col) itself.
func valid(g *Grid, row, col, val int) bool

// CountSolutions returns the number of distinct solutions of g, counting up to
// at most `limit` (it stops early once `limit` is reached). Use limit=2 for
// uniqueness checks: a return of 1 means a unique solution, >=2 means ambiguous,
// 0 means unsolvable. g is not mutated.
func CountSolutions(g Grid, limit int) int

// Solve fills g in place with the first solution found and returns true, or
// returns false (leaving g unchanged) if no solution exists.
func Solve(g *Grid) bool
```

Implementation notes for the executor:
- Find the first empty cell (row-major). If none, the grid is complete →
  one solution.
- For each candidate 1..9 that `valid` allows, place it, recurse, then undo
  (backtrack). For counting, accumulate the recursive count and **return early
  once the running count reaches `limit`** so uniqueness checks stay fast.
- `CountSolutions` must not mutate the caller's grid: operate on a local copy
  (arrays are value types in Go, so `g Grid` passed by value is already a copy —
  recurse on `&g`).
- `valid` checks the row, the column, and the 3x3 box (box origin =
  `row/3*3`, `col/3*3`).

## Steps

### Step 1: Write the solver tests first (red)

Create `puzzle/solve_test.go`. Include these cases:

1. **`TestSolveKnownPuzzle`** — a known solvable puzzle (clues below); assert
   `Solve` returns true and the result is a fully-filled legal grid (no zeros;
   every row, column, and box is a permutation of 1..9). Write a small
   `isComplete(g Grid) bool` test helper for the legality check.
2. **`TestCountUnique`** — the same known puzzle has exactly one solution:
   `CountSolutions(p, 2) == 1`.
3. **`TestCountAmbiguous`** — an empty grid (all zeros) has many solutions:
   `CountSolutions(Grid{}, 2) == 2` (short-circuits at the limit).
4. **`TestCountUnsolvable`** — a grid with two identical values in one row is
   unsolvable: `CountSolutions(bad, 2) == 0`.
5. **`TestCountDoesNotMutate`** — capture a copy of the input, call
   `CountSolutions(p, 2)`, assert the input grid is byte-for-byte unchanged.

Use this known-unique puzzle and its solution as fixtures (the classic "world's
hardest"-style is unnecessary; this standard one is fine):

```go
// clues
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
```

**Verify**: `go test ./puzzle/` → FAILS to compile (symbols not defined yet).
This confirms the tests reference the real API. This expected failure is fine.

### Step 2: Implement `solve.go` (green)

Implement `Grid`, `valid`, `CountSolutions`, and `Solve` per the Target above.
If you kept `puzzle/doc.go`, remove its `package puzzle` line duplication is not
an issue (multiple files, one `package puzzle` each is correct). Delete
`puzzle/doc_test.go` now that real tests exist.

**Verify**: `go test ./puzzle/` → `ok`, all five tests pass
(`go test -v ./puzzle/` shows 5 `--- PASS`).

### Step 3: Confirm the gate

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0

## Test plan

- New file `puzzle/solve_test.go` with the five cases above: happy-path solve,
  unique count, ambiguous count (early stop), unsolvable, and no-mutation.
- No existing test to model after (this is the first real test); follow standard
  Go table/`t.Fatalf` style.
- Verification: `go test ./puzzle/` → all pass, 5 tests.

## Done criteria

ALL must hold:

- [ ] `puzzle/solve.go` defines `Grid`, `CountSolutions(Grid, int) int`, `Solve(*Grid) bool`
- [ ] `CountSolutions` short-circuits at `limit` and does not mutate its input
- [ ] `go test ./puzzle/` exits 0 with 5 passing tests
- [ ] `puzzle/doc_test.go` removed
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 002 updated to DONE

## STOP conditions

Stop and report back if:

- `solve.go` already exists with a different API than the Target.
- `TestCountAmbiguous` returns something other than 2, or `TestCountUnsolvable`
  returns non-zero after a reasonable fix attempt — this indicates a logic bug
  in `valid` or the counter; report the failing values.

## Maintenance notes

- Plan 003 (generator) calls `CountSolutions(p, 2)` once per cell-removal
  attempt. The early-exit at `limit` is what keeps generation fast — do not
  remove it.
- If a future feature needs the count of *all* solutions, call with a large
  limit; do not change the default behavior of stopping at the limit.
- Reviewer should scrutinize the `valid` box-origin math and that
  `CountSolutions` recurses on a copy, not the caller's grid.
