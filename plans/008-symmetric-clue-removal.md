# Plan 008: Carve clues with 180° rotational symmetry

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> (unless a reviewer dispatched you and said they maintain the index).
>
> **Drift check (run first)**: This plan was written against commit `fe3b33a`.
> Run `git diff --stat fe3b33a..HEAD -- puzzle/generate.go`. If `carve` or the
> `difficulties` map differ from the "Current state" excerpts below, reconcile
> before proceeding; on a mismatch you cannot resolve, STOP and report.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/003-generator.md (DONE) — reopens `puzzle/generate.go`
- **Category**: correctness (aesthetic quality)
- **Planned at**: commit `fe3b33a`, 2026-06-14

## Why this matters

Today `carve` removes cells in fully random order, producing blotchy,
asymmetric clue patterns. Published/newspaper sudoku almost always have 180°
rotational symmetry — it is the single biggest thing that makes a grid look
*designed* rather than machine-generated, and it shows on a printed page. This
plan changes carving to remove cells in rotationally-symmetric pairs.

The trade-off, which this plan explicitly measures and handles: symmetric
carving is coarser (it removes two cells at a time) and cannot carve as sparse
while preserving a unique solution, so the achievable clue counts shift upward —
especially for `hard`. The existing **clue-count range design absorbs this**,
but the ranges in the `difficulties` map likely need re-tuning, which Step 2
does empirically.

## Current state

`puzzle/generate.go` (at `fe3b33a`). The difficulty ranges (lines 8–18):

```go
type difficultySpec struct {
	target, min, max int
}

var difficulties = map[string]difficultySpec{
	"easy":   {target: 36, min: 34, max: 40},
	"medium": {target: 30, min: 28, max: 34},
	"hard":   {target: 25, min: 23, max: 30},
}
```

The current `carve` (lines 164–198), which this plan REPLACES:

```go
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
```

`carve` is called from `Generate` (line 46) inside a retry loop that keeps the
attempt with the fewest clues and accepts the first attempt whose count is
within `[spec.min, spec.max]`. `CountSolutions(g Grid, limit int) int` (from
`puzzle/solve.go`) returns 1 for a unique grid. All randomness already flows
through the injected `*rand.Rand` — keep it that way (`rng.Perm`, `rng.Shuffle`;
no global `rand.*`, no `time.Now`).

Repo conventions: see `puzzle/generate_test.go` for test style (fixed
`rand.NewSource` seeds, `t.Errorf`/`t.Fatalf`). The existing test
`TestClueCountInRange` reads ranges from the `difficulties` map, so it adapts
automatically when you re-tune.

## Commands you will need

| Purpose   | Command                          | Expected on success    |
|-----------|----------------------------------|------------------------|
| Build     | `go build ./...`                 | exit 0                 |
| Vet       | `go vet ./...`                   | exit 0                 |
| Test pkg  | `go test ./puzzle/`              | `ok  	sudoprint/puzzle` |
| Verbose   | `go test -v -run Symmetric ./puzzle/` | `--- PASS`        |

(If `go test` reports an Application Control / "blocked this file" error running
a test binary from the temp dir — an environment policy, not a code failure —
compile and run locally: `go test -c -o ./_t.exe ./puzzle && ./_t.exe` then
delete `_t.exe`.)

## Scope

**In scope**:
- `puzzle/generate.go` (edit `carve`; re-tune the `difficulties` ranges in Step 2)
- `puzzle/generate_test.go` (add symmetry + multi-seed range tests)

**Out of scope** (do NOT touch):
- `puzzle/solve.go`, `puzzle/solve_test.go` — frozen.
- `render/*`, `main.go`, `fonts/*`, `go.mod`, `go.sum`, `plans/*`, `PLAN.md`.

## Git workflow

The repo IS a git repository, Conventional Commits style. If dispatched into a
worktree, commit there with a message like
`feat: carve clues with 180-degree rotational symmetry (plan 008)` and the
trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Do NOT push.

## Target

### Replacement `carve`

Carve in rotationally-symmetric groups. For position `pos` (0..80), its 180°
partner is `80 - pos`. There are 40 pairs (`pos` 0..39 with `80-pos`) plus the
center cell `pos == 40` (its own partner), for 41 groups covering all 81 cells.

```go
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
			for _, cl := range g {
				if cl.r != 4 || cl.c != 4 {
					// counted below; nothing special
				}
			}
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
```

(Drop the empty `for ... if cl.r != 4` block — it is a leftover; just do
`removed += len(g)`. Keep the function otherwise as shown.)

Key changes vs. the old `carve`:
- removes symmetric groups all-or-nothing (was: single cells),
- stop condition is `81-removed <= spec.target` (was: `== spec.target`),
- still purely `rng`-driven and deterministic.

### Re-tuned `difficulties` ranges

Step 2 measures the real achievable clue counts and sets the ranges. Do NOT
guess — measure first. Symmetric carving typically lands `hard` higher than the
old `[23,30]`. Keep `target` near the low end of each measured range.

## Steps

### Step 1: Replace `carve`

Replace the existing `carve` in `puzzle/generate.go` with the Target version
(omitting the leftover empty block noted above). Do not change `Generate` or the
`difficulties` map yet.

**Verify**: `go build ./...` → exit 0. `go vet ./...` → exit 0.

### Step 2: Measure achievable clue counts and re-tune the ranges

Create a TEMPORARY file `puzzle/zz_tune_test.go` that prints the observed clue
count range per difficulty across many seeds (this file is deleted in Step 4 —
it must NOT be committed):

```go
package puzzle

import (
	"math/rand"
	"testing"
)

func TestZZTune(t *testing.T) {
	const seeds = 40
	for _, diff := range []string{"easy", "medium", "hard"} {
		spec := difficulties[diff]
		minC, maxC := 81, 0
		for s := 0; s < seeds; s++ {
			rng := rand.New(rand.NewSource(int64(s)))
			p, err := Generate(1, diff, rng)
			if err != nil {
				t.Fatal(err)
			}
			if CountSolutions(p.Clues, 2) != 1 {
				t.Fatalf("%s seed %d not unique", diff, s)
			}
			// verify symmetry of the clue pattern
			for r := 0; r < 9; r++ {
				for c := 0; c < 9; c++ {
					if (p.Clues[r][c] == 0) != (p.Clues[8-r][8-c] == 0) {
						t.Fatalf("%s seed %d: asymmetric at %d,%d", diff, s, r, c)
					}
				}
			}
			if p.ClueCount < minC {
				minC = p.ClueCount
			}
			if p.ClueCount > maxC {
				maxC = p.ClueCount
			}
		}
		t.Logf("%s: observed clue counts [%d, %d] (current spec [%d,%d])",
			diff, minC, maxC, spec.min, spec.max)
	}
}
```

Run it: `go test -v -run TestZZTune -timeout 300s ./puzzle/` (or the
local-compile workaround). Read the logged `[min, max]` per difficulty.

Then set the `difficulties` map so that, for each difficulty, `[min, max]`
comfortably contains the observed range (add a small margin of ~±2), and
`target` sits at or just below the observed minimum. Example shape (USE YOUR
MEASURED NUMBERS, not these):

```go
var difficulties = map[string]difficultySpec{
	"easy":   {target: 36, min: 34, max: 42},
	"medium": {target: 30, min: 28, max: 36},
	"hard":   {target: 25, min: 23, max: 34}, // hard runs higher under symmetry
}
```

Re-run `TestZZTune` after tuning and confirm every observed value is within the
new ranges across all 40 seeds.

**Verify**: `TestZZTune` passes (no uniqueness/symmetry failures) and its logged
ranges all fall inside the ranges you set.

**STOP** and report if `easy` cannot reach its `[min,max]` even after tuning, or
if `hard`'s observed minimum is so high (e.g. > 35) that "hard" stops being
meaningfully harder than "medium" — that needs a human design decision about the
symmetry/difficulty trade-off, not an executor guess.

### Step 3: Add the committed tests

Delete `puzzle/zz_tune_test.go`. Add permanent tests to
`puzzle/generate_test.go`:

1. **`TestCluesAreSymmetric`** — for each difficulty (seed `rand.NewSource(1)`),
   `Generate`, then assert for all `r,c`:
   `(p.Clues[r][c] == 0) == (p.Clues[8-r][8-c] == 0)`.
2. **`TestClueCountInRangeMultiSeed`** — for seeds 0..7, for each difficulty,
   assert `p.ClueCount` is within that difficulty's (re-tuned) `[min,max]` and
   `CountSolutions(p.Clues,2) == 1`. (8 seeds × 3 difficulties is a few seconds;
   acceptable.)

The existing `TestClueCountInRange`, `TestGenerateUnique`,
`TestGenerateSolutionValid`, `TestGenerateCluesSubsetOfSolution`,
`TestClueCountAccurate`, `TestDeterministic`, `TestUnknownDifficulty` must still
pass unchanged (ranges adapt because they read the map).

**Verify**: `go test ./puzzle/` → `ok`, all tests pass.

### Step 4: Confirm the full gate

**Verify**:
- `git status --short` shows NO `puzzle/zz_tune_test.go` (deleted)
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0
- `grep -nE "rand\.(Intn|Shuffle|Perm)" puzzle/generate.go` shows only `rng.`-receiver calls; `grep -n "time.Now" puzzle/generate.go` → nothing

## Test plan

- New committed tests in `puzzle/generate_test.go`: `TestCluesAreSymmetric`
  (pattern symmetry), `TestClueCountInRangeMultiSeed` (range + uniqueness over 8
  seeds).
- Temporary `zz_tune_test.go` used only to measure ranges in Step 2, then
  deleted (must not be committed).
- All pre-existing generator/solver tests stay green.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `carve` removes cells in 180° symmetric groups; clue pattern is symmetric
      (`TestCluesAreSymmetric` passes)
- [ ] Generated puzzles remain unique (`CountSolutions(Clues,2)==1`) for all
      difficulties across the multi-seed test
- [ ] `difficulties` ranges re-tuned so every difficulty reliably lands in range
      (`TestClueCountInRangeMultiSeed` passes for seeds 0–7)
- [ ] Determinism preserved (`TestDeterministic` passes; randomness via `rng` only)
- [ ] `puzzle/zz_tune_test.go` deleted (not committed)
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 008 updated to DONE

## STOP conditions

Stop and report back if:

- After tuning, some difficulty still cannot land in any sensible range across 40
  seeds (uniqueness gate too restrictive under symmetry) — report observed
  ranges.
- `hard`'s achievable minimum is so high it is no longer meaningfully harder than
  `medium` — this is a design decision (symmetry vs. difficulty), not an
  executor call.
- Any generated puzzle is non-unique, or a single `Generate` takes > 5 s
  (symmetric carving does more `CountSolutions` calls per group — if it blows the
  budget, report rather than waiting).

## Maintenance notes

- Symmetry is now unconditional. If someone later wants minimal-clue asymmetric
  puzzles, add a `-symmetric=false` path (deferred — would touch `Generate`'s
  signature and `main.go`).
- The re-tuned ranges are the contract that keeps "report actual clue count"
  honest under symmetry. If carving strategy changes again, re-run the Step-2
  measurement.
- `main.go` and the manifest (plan 007) read `ClueCount` only — no changes needed
  there; symmetric puzzles simply report their (somewhat higher) counts.
- A reviewer should re-render a page (or run the binary) and confirm the clue
  pattern visibly has 180° symmetry, and that `hard` is still harder than `easy`.
