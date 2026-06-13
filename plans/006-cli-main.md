# Plan 006: Wire the CLI in main.go — flags, generation loop, output, and summary

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: Run `ls`. You should see `go.mod`, `puzzle/`,
> `render/`, `fonts/`, `PLAN.md`, `plans/` and NO `main.go`. If `main.go`
> already exists, read it and compare against the Target; on a mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: plans/003-generator.md, plans/004-image-rendering.md, plans/005-pdf-bundling.md
- **Category**: dx + correctness
- **Planned at**: no git repo (greenfield), 2026-06-13

## Why this matters

This is the entry point that ties generation and rendering into a usable command.
It also implements the user-facing honesty decisions: print the actual seed (so
random runs are reproducible) and print the actual clue count per puzzle (so
"hard" that landed at 28 clues is visible, not hidden). It also pins down the
previously-undefined output behavior (directory creation, overwrite).

## Current state

Available APIs from earlier plans:

```go
// package puzzle
func Generate(id int, difficulty string, rng *rand.Rand) (puzzle.Puzzle, error)
type Puzzle struct { ID int; Clues, Solution Grid; Difficulty string; ClueCount int }

// package render
func RenderPage(left, right puzzle.Puzzle, solution bool) (image.Image, error)
func BundlePDF(images []image.Image, outputPath string) error
```

No `main.go` exists. CLI spec is `PLAN.md` §31–67 and the flow is §211–233.

CLI flags (from `PLAN.md` §37–44):

| Flag | Type | Default | Meaning |
|------|------|---------|---------|
| `-n` | int | 1 | pages to generate (2 puzzles/page) |
| `-d` | string | `medium` | difficulty: easy/medium/hard |
| `-o` | string | `.` | output dir (created if absent) |
| `-pdf` | bool | false | also produce PDFs |
| `-keep-png` | bool | true | keep PNGs when `-pdf` set; `-keep-png=false` to discard |
| `-seed` | int64 | random | RNG seed |

Output files (`PLAN.md` §46–54): `puzzle_NNN.png`, `solution_NNN.png`
(3-digit, 1-based, one per page), and when `-pdf`: `puzzles.pdf`, `solutions.pdf`.

## Commands you will need

| Purpose   | Command                                   | Expected on success |
|-----------|-------------------------------------------|---------------------|
| Build     | `go build -o sudoprint .`                 | exit 0, binary made |
| Build all | `go build ./...`                          | exit 0              |
| Vet       | `go vet ./...`                            | exit 0              |
| Test      | `go test ./...`                           | exit 0              |
| Smoke run | `./sudoprint -n 1 -d easy -seed 42 -o ./_smoke` | prints summary, writes 2 PNGs |

(On Windows the binary is `sudoprint.exe`; run `.\sudoprint.exe ...`.)

## Scope

**In scope**:
- `main.go` (create)
- `main_test.go` (create — integration test)

**Out of scope**: everything under `puzzle/` and `render/` (frozen by their plans).

## Git workflow

Not a git repository. Do not init/commit/push. After Step's smoke run, delete the
`_smoke` output directory you created so you don't leave artifacts.

## Target

`main.go` flow (matches `PLAN.md` §211–233):

```go
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"sudoprint/puzzle"
	"sudoprint/render"
)

func main() {
	n := flag.Int("n", 1, "number of pages (2 puzzles per page)")
	d := flag.String("d", "medium", "difficulty: easy, medium, hard")
	out := flag.String("o", ".", "output directory")
	pdf := flag.Bool("pdf", false, "also produce PDFs")
	keepPNG := flag.Bool("keep-png", true, "keep PNGs when -pdf is set")
	seed := flag.Int64("seed", 0, "RNG seed (0 = random)")
	flag.Parse()

	if err := run(*n, *d, *out, *pdf, *keepPNG, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Put the real logic in a testable `run(...)` function (so `main_test.go` can call
it without spawning a process):

```go
func run(n int, difficulty, outDir string, makePDF, keepPNG bool, seed int64) error
```

`run` must:
1. **Validate**: `n >= 1`; `difficulty` ∈ {easy,medium,hard} (reject otherwise
   with a clear error — do not silently default).
2. **Seed**: if `seed == 0`, set `seed = time.Now().UnixNano()`. Create
   `rng := rand.New(rand.NewSource(seed))`. Print `seed: <seed>` to stdout so
   random runs are reproducible.
3. **Output dir**: `os.MkdirAll(outDir, 0o755)`.
4. **Loop** `page := 1..n`:
   - `left, err := puzzle.Generate(page*2-1, difficulty, rng)`
   - `right, err := puzzle.Generate(page*2, difficulty, rng)`
   - render puzzle page: `img, _ := render.RenderPage(left, right, false)`,
     save to `filepath.Join(outDir, fmt.Sprintf("puzzle_%03d.png", page))`
   - render solution page: `render.RenderPage(left, right, true)`, save to
     `solution_%03d.png`
   - print one line per page including the **actual clue counts**, e.g.
     `page 1: #1 (clues 35), #2 (clues 36)`.
5. **PDF** (if `makePDF`): collect the puzzle images and solution images while
   looping (keep them in slices), then `render.BundlePDF(puzzleImgs,
   filepath.Join(outDir,"puzzles.pdf"))` and likewise `solutions.pdf`. If
   `!keepPNG`, delete the PNG files after the PDFs are written successfully.
6. **Summary** to stdout: seed, number of files written, output dir (§229–233).

PNG saving helper:

```go
func savePNG(img image.Image, path string) error {
	f, err := os.Create(path) // overwrites existing file — documented behavior
	if err != nil { return err }
	defer f.Close()
	return png.Encode(f, img)
}
```

**Overwrite behavior (resolves the spec gap):** `os.Create` truncates/overwrites
any existing `puzzle_NNN.png` from a previous run. This is intentional and simple;
document it in `-h` help text or a comment. Do not invent numbered subdirectories.

## Steps

### Step 1: Implement `main.go`

Write `main.go` with `main` + `run` + `savePNG` per the Target. Keep image slices
for the PDF path only when `makePDF` is true (avoid holding all images in memory
otherwise — for large `-n` this matters).

**Verify**: `go build -o sudoprint .` → exit 0.

### Step 2: Smoke-run the binary

```
./sudoprint -n 2 -d easy -seed 42 -o ./_smoke
```

(Windows: `.\sudoprint.exe -n 2 -d easy -seed 42 -o ./_smoke`.)

**Verify**:
- stdout contains `seed: 42`
- stdout contains per-page clue lines
- `ls _smoke` shows `puzzle_001.png`, `puzzle_002.png`, `solution_001.png`,
  `solution_002.png` (4 files for 2 pages)
- each PNG is non-empty
Then run `./sudoprint -n 1 -d hard -seed 7 -o ./_smoke -pdf -keep-png=false`:
- `_smoke` now also contains `puzzles.pdf` and `solutions.pdf`
- the `puzzle_*.png`/`solution_*.png` from THIS run are gone (keep-png=false),
  PDFs begin with `%PDF`.

Delete `_smoke` afterward.

### Step 3: Write the integration test

Create `main_test.go`:

1. **`TestRunWritesPNGs`** — `dir := t.TempDir(); err := run(2,"easy",dir,false,true,42)`;
   assert no error and the 4 expected PNG files exist and are non-empty.
2. **`TestRunRejectsBadDifficulty`** — `run(1,"extreme",t.TempDir(),false,true,1)`
   returns a non-nil error.
3. **`TestRunRejectsBadN`** — `run(0,"easy",t.TempDir(),false,true,1)` returns an
   error.
4. **`TestRunPDF`** — `run(1,"easy",dir,true,false,1)`; assert `puzzles.pdf` and
   `solutions.pdf` exist and begin with `%PDF`, and the per-run PNGs do NOT exist
   (keepPNG=false).
5. **`TestRunDeterministic`** — run twice into two temp dirs with the same seed;
   assert `puzzle_001.png` byte content is identical across the two runs.

**Verify**: `go test ./...` → `ok` for all three packages.

### Step 4: Confirm the full gate

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0
- No stray `_smoke` directory left behind (`ls` shows only the repo files)

## Test plan

- New file `main_test.go` exercising `run(...)` directly: PNG output, bad
  difficulty, bad n, PDF + keep-png=false, determinism.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `go build -o sudoprint .` produces a working binary
- [ ] All six flags parse and behave per the table; bad `-d`/`-n` produce a
      non-zero exit with a clear stderr message
- [ ] Random runs print `seed: <value>`; every run prints actual per-puzzle clue counts
- [ ] `-pdf` writes `puzzles.pdf` + `solutions.pdf`; `-keep-png=false` removes PNGs
- [ ] Output identical across runs with the same `-seed`
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `_smoke` cleaned up; `plans/README.md` status row for 006 updated to DONE

## STOP conditions

Stop and report back if:

- Generation hangs or a single run of `-n 1 -d hard` takes more than ~10s
  (points to a solver/generator perf regression in plans 002/003).
- `RenderPage` or `BundlePDF` return errors during the smoke run — capture and
  report; do not work around by skipping pages.
- Determinism test fails — randomness is leaking outside the injected `*rand.Rand`
  (re-check plan 003's done criteria).

## Maintenance notes

- For very large `-n` with `-pdf`, all page images are held in memory to build
  the PDF. If this becomes a problem, stream pages to the PDF as they're rendered
  instead of collecting slices — note this as a known scaling limit.
- A `manifest.json` (seed + per-puzzle id/difficulty/clue count) would make
  batches auditable and is a natural follow-up to the "report actual clue count"
  decision — deliberately deferred to keep this plan focused.
- Reviewer should confirm `-keep-png=false` only deletes files this run created,
  and that a failed PDF write does NOT delete the PNGs (write PDFs first, delete
  PNGs only on success).
