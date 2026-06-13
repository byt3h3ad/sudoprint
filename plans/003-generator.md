# Plan 003: Implement the puzzle generator with varied grids and reported clue counts

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: This plan was written against commit `e90880a`.
> Run `git diff --stat e90880a..HEAD -- puzzle/`. Plan 002 should have added
> `puzzle/solve.go` and `puzzle/solve_test.go`; confirm the `Grid` /
> `CountSolutions` / `Solve` API in "Current state" matches the live code. If
> `puzzle/generate.go` already exists, read it and compare against the "Target";
> on a mismatch, STOP and report.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/002-solver.md
- **Category**: correctness + tests
- **Planned at**: commit `e90880a`, 2026-06-13

## Why this matters

This is the heart of the tool. Two correctness traps from the spec are fixed here:

1. **Difficulty targets are not guaranteed.** Random cell removal with a
   uniqueness gate frequently *cannot* reach the lowest clue counts (especially
   `hard`/25), so a naive "stop when target reached or stuck" loop silently emits
   an easier puzzle. **Decision: treat the clue count as a target with an
   accepted range, and report the actual clue count achieved** so the output is
   honest. (Selected approach: "range + report actual.")
2. **Limited grid variety.** Building the solved grid from a single shifted
   pattern produces structurally similar grids. We apply random
   band/stack/row/column swaps, transpose, and digit relabeling so puzzles look
   distinct — including the two on the same page.

## Current state

`puzzle/solve.go` (from plan 002) defines:

```go
type Grid [9][9]int                      // 0 = empty
func CountSolutions(g Grid, limit int) int  // unique => 1, ambiguous => >=2, unsolvable => 0
func Solve(g *Grid) bool
```

No generator or `Puzzle` type exists yet.

## Commands you will need

| Purpose   | Command                      | Expected on success    |
|-----------|------------------------------|------------------------|
| Build     | `go build ./...`             | exit 0                 |
| Vet       | `go vet ./...`               | exit 0                 |
| Test pkg  | `go test ./puzzle/`          | `ok  	sudoprint/puzzle` |
| Verbose   | `go test -v -run Gen ./puzzle/` | `--- PASS` per case |

## Scope

**In scope**:
- `puzzle/generate.go` (create)
- `puzzle/generate_test.go` (create)

**Out of scope** (do NOT touch):
- `puzzle/solve.go`, `puzzle/solve_test.go` — plan 002, frozen.
- `render/`, `main.go`.

## Git workflow

Not a git repository. Do not init/commit/push.

## Target

`puzzle/generate.go`:

```go
package puzzle

import "math/rand"

// Difficulty target clue counts and the accepted range. Generation aims for
// Target but accepts any result within [Min, Max]; the actual count is reported
// on the Puzzle.
type difficultySpec struct {
	target, min, max int
}

var difficulties = map[string]difficultySpec{
	"easy":   {target: 36, min: 34, max: 40},
	"medium": {target: 30, min: 28, max: 34},
	"hard":   {target: 25, min: 23, max: 30},
}

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
func Generate(id int, difficulty string, rng *rand.Rand) (Puzzle, error)
```

### Algorithm

**A. Build a full solved grid (`generateSolved(rng) Grid`):**
1. Seed a base pattern. A valid completed grid is given by
   `base[r][c] = ((r*3 + r/3 + c) % 9) + 1`.
2. Shuffle it to add variety, each step using `rng`:
   - relabel digits: pick a random permutation of 1..9 and map every cell;
   - for each of the 3 bands (row groups 0-2, 3-5, 6-8): shuffle the 3 rows
     within the band;
   - for each of the 3 stacks (col groups): shuffle the 3 columns within the stack;
   - shuffle the order of the 3 bands; shuffle the order of the 3 stacks;
   - optionally transpose the whole grid with 50% probability.
   All of these transformations preserve sudoku validity.

**B. Carve clues (`carve`):**
1. Copy the solved grid into `clues`.
2. Build a list of all 81 cell positions and shuffle it with `rng`.
3. `removed := 0`. Walk the shuffled positions. For each position still filled:
   tentatively set it to 0, call `CountSolutions(clues, 2)`. If the result is
   exactly 1, the removal is safe — keep it and `removed++`. Otherwise restore
   the cell.
   - Stop early once `81 - removed == spec.target` (target clue count reached).
4. After the pass, `clueCount = 81 - removed`. This is **best-effort toward
   target, bounded below by what uniqueness allows.**

**C. Accept-or-retry to honor the range:**
- If `clueCount` is within `[spec.min, spec.max]`, accept.
- Otherwise (almost always because too *many* clues remain, i.e. carving got
  stuck above `max`), retry B from a fresh full grid, up to a cap of **20
  attempts**, keeping the attempt with the **fewest clues** seen.
- After the cap, return the best attempt regardless, and set `ClueCount` to its
  actual value. Do **not** loop forever and do **not** fabricate a count.

Stamp `ID`, `Difficulty`, and the real `ClueCount` onto the returned `Puzzle`.

## Steps

### Step 1: Write generator tests first (red)

Create `puzzle/generate_test.go`. Use a fixed seed so results are deterministic:
`rng := rand.New(rand.NewSource(1))`.

Helper to add: `isValidSolution(g Grid) bool` — no zeros; every row, column, box
is a permutation of 1..9. (You may copy the equivalent helper idea from
`solve_test.go`; keep them independent — do not import test code across files in
a way that breaks compilation. Duplicating a small helper is acceptable.)

Cases:
1. **`TestGenerateUnique`** — for each difficulty, `Generate` returns no error,
   `CountSolutions(p.Clues, 2) == 1`.
2. **`TestGenerateSolutionValid`** — `isValidSolution(p.Solution)` is true.
3. **`TestGenerateCluesSubsetOfSolution`** — every non-zero cell in `p.Clues`
   equals the same cell in `p.Solution`.
4. **`TestClueCountAccurate`** — `p.ClueCount` equals the actual number of
   non-zero cells in `p.Clues`.
5. **`TestClueCountInRange`** — for each difficulty, `p.ClueCount` is within that
   difficulty's `[min, max]`. (If `hard` cannot reach its range, see STOP
   conditions — but with the 20-attempt retry it should land in `[23,30]`.)
6. **`TestDeterministic`** — two calls with freshly-seeded `rand.NewSource(42)`
   produce identical `Clues` and `Solution`.
7. **`TestUnknownDifficulty`** — `Generate(1, "extreme", rng)` returns a non-nil
   error.

**Verify**: `go test ./puzzle/` → compile failure (symbols undefined). Expected.

### Step 2: Implement `generate.go` (green)

Implement per the Target and Algorithm.

**Verify**: `go test ./puzzle/` → `ok`, all tests pass (the two from plan 002
plus the seven new ones). `go test -v -run Gen ./puzzle/` shows the new cases
passing.

### Step 3: Sanity-check difficulty separation (informational)

Run `go test -v -run TestClueCountInRange ./puzzle/`. The test already asserts
ranges; this step is just to eyeball that `easy` > `medium` > `hard` in clue
count on average. No new assertion required.

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0

## Test plan

- New file `puzzle/generate_test.go` with the seven cases above: uniqueness,
  valid solution, clue⊂solution, accurate count, count-in-range, determinism,
  bad-difficulty error.
- Model the assertion style after `puzzle/solve_test.go`.
- Verification: `go test ./puzzle/` → all pass.

## Done criteria

ALL must hold:

- [ ] `puzzle/generate.go` defines `Puzzle` (with `ClueCount`) and
      `Generate(int, string, *rand.Rand) (Puzzle, error)`
- [ ] Generated puzzles are unique (`CountSolutions(Clues,2)==1`) for all difficulties
- [ ] `ClueCount` is accurate and within the difficulty range
- [ ] Determinism holds for a fixed seed
- [ ] All randomness uses the passed `*rand.Rand` (no `rand.Intn`/global rand, no
      `time.Now()` inside generation) — verify by grep: `grep -n "rand.Intn\|rand.Shuffle\|time.Now" puzzle/generate.go` shows only calls on the `rng` receiver (e.g. `rng.Intn`, `rng.Shuffle`)
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 003 updated to DONE

## STOP conditions

Stop and report back if:

- `TestClueCountInRange` fails for `hard` even after the 20-attempt retry — this
  may mean the uniqueness gate or the base-pattern shuffling has a bug; report
  the actual clue counts observed across a few seeds.
- Generation for any single puzzle takes more than ~5 seconds — that indicates
  the solver isn't short-circuiting at limit 2 (a plan 002 regression); STOP and
  report rather than waiting.
- A generated puzzle ever fails `CountSolutions(Clues,2)==1` — a non-unique
  puzzle is a hard failure; report the seed and grid.

## Maintenance notes

- The accepted ranges in `difficulties` are the contract that makes "report
  actual" honest. If someone tightens `hard` toward 17 clues (the theoretical
  minimum), the retry cap and runtime will need revisiting — carving to near-17
  is expensive.
- `main.go` (plan 006) prints `ClueCount` per puzzle; keep the field populated.
- Symmetric (rotational) clue removal would make puzzles look more
  "published-quality" — deliberately deferred to keep this plan focused; note it
  as a future enhancement.
- Reviewer should confirm all randomness flows through the injected `*rand.Rand`,
  or reproducibility (`-seed`) silently breaks.
