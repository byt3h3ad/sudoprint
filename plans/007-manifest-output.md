# Plan 007: Emit a deterministic manifest.json describing each batch

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`
> (unless a reviewer dispatched you and said they maintain the index).
>
> **Drift check (run first)**: This plan was written against commit `fe3b33a`.
> Run `git diff --stat fe3b33a..HEAD -- main.go`. If `main.go` differs from the
> "Current state" excerpt below (especially the `run` signature or the
> generation loop), reconcile before proceeding; on a mismatch you cannot
> resolve, STOP and report.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/006-cli-main.md (DONE)
- **Category**: dx
- **Planned at**: commit `fe3b33a`, 2026-06-14

## Why this matters

The "report actual clue count" decision currently writes per-puzzle clue counts
only to **stdout**, which disappears as soon as the terminal scrolls. A
`manifest.json` persists the full record of a batch — the seed, the difficulty,
and per-puzzle clue counts — alongside the output files. That makes batches
reproducible (the seed is recorded), auditable (diff two manifests, or verify a
batch's difficulty), and consumable by downstream tooling (e.g. a puzzle-book
compiler). It is the natural completion of the honesty decision already in the
code.

## Current state

`main.go` (package `main`) defines `run`, which generates puzzles, writes PNGs,
optionally writes PDFs, and prints a stdout summary. The relevant excerpt
(`main.go`, generation loop and tail — lines ~57–132 at `fe3b33a`):

```go
	var puzzleImgs []image.Image
	var solutionImgs []image.Image
	var pngPaths []string // track for cleanup when !keepPNG

	// 4. Generation loop.
	filesWritten := 0
	for page := 1; page <= n; page++ {
		left, err := puzzle.Generate(page*2-1, difficulty, rng)
		// ... error handling ...
		right, err := puzzle.Generate(page*2, difficulty, rng)
		// ... render + savePNG both pages, filesWritten++ each ...

		// Print per-page summary with actual clue counts.
		fmt.Printf("page %d: #%d (clues %d), #%d (clues %d)\n",
			page, left.ID, left.ClueCount, right.ID, right.ClueCount)

		if makePDF {
			puzzleImgs = append(puzzleImgs, puzzleImg)
			solutionImgs = append(solutionImgs, solutionImg)
			pngPaths = append(pngPaths, puzzlePath, solutionPath)
		}
	}

	// 5. PDF generation. (writes puzzles.pdf, solutions.pdf; deletes PNGs if !keepPNG)
	// ...

	// 6. Summary.
	fmt.Printf("done: seed=%d, %d file(s) written to %s\n", seed, filesWritten, outDir)
	return nil
}
```

The `puzzle.Puzzle` type (from `puzzle/generate.go`) provides the fields the
manifest records:

```go
type Puzzle struct {
	ID         int
	Clues      Grid
	Solution   Grid
	Difficulty string
	ClueCount  int
}
```

Repo conventions to match: standard library only where possible (the project
uses `encoding/json` from stdlib — no new dependency); error wrapping with
`fmt.Errorf("...: %w", err)`; table/`t.Fatalf` test style (see
`main_test.go`).

## Commands you will need

| Purpose   | Command                       | Expected on success |
|-----------|-------------------------------|---------------------|
| Build     | `go build ./...`              | exit 0              |
| Build bin | `go build -o sudoprint.exe .` | exit 0              |
| Vet       | `go vet ./...`                | exit 0              |
| Test      | `go test ./...`               | exit 0              |

(On Windows the binary is `sudoprint.exe`. If `go test ./...` reports an
Application Control / "blocked this file" error executing a test binary from the
temp dir — an environment policy, not a code failure — compile and run the
package test locally instead: `go test -c -o ./_t.exe ./<pkg> && ./_t.exe` then
delete `_t.exe`.)

## Scope

**In scope**:
- `manifest.go` (create — package `main`: the manifest types + `writeManifest`)
- `main.go` (edit — collect per-page entries in the loop; write the manifest)
- `main_test.go` (edit — add manifest tests)

**Out of scope** (do NOT touch):
- `puzzle/*`, `render/*`, `fonts/*` — frozen here.
- `go.mod`, `go.sum` — no new dependency (`encoding/json` is stdlib).
- `plans/*`, `PLAN.md`.

## Git workflow

The repo IS a git repository. Branch/commit conventions: this project uses
Conventional Commits (see `git log`, e.g.
`feat: PDF bundling + remove redundant tools.go (plan 005)`). If you were
dispatched by a reviewer into a worktree, commit there with a message like
`feat: emit manifest.json for each batch (plan 007)` and end it with the
trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Do NOT push
or open a PR.

## Target

### `manifest.go` (new file, package `main`)

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// manifest is the persistent record of one generated batch. It contains only
// seed-derived data (NO timestamp) so that two runs with the same seed produce
// a byte-identical manifest.
type manifest struct {
	Seed       int64      `json:"seed"`
	Difficulty string     `json:"difficulty"`
	Pages      []pageInfo `json:"pages"`
}

type pageInfo struct {
	Page    int          `json:"page"`
	Puzzles []puzzleInfo `json:"puzzles"` // exactly two, left then right
}

type puzzleInfo struct {
	ID        int `json:"id"`
	ClueCount int `json:"clueCount"`
}

// writeManifest marshals m as indented JSON to path (trailing newline).
func writeManifest(m manifest, path string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}
```

### Changes to `run` in `main.go`

1. Before the generation loop, initialize the manifest:
   ```go
   m := manifest{Seed: seed, Difficulty: difficulty}
   ```
2. Inside the loop, after the per-page stdout line, append the page's entry:
   ```go
   m.Pages = append(m.Pages, pageInfo{
       Page: page,
       Puzzles: []puzzleInfo{
           {ID: left.ID, ClueCount: left.ClueCount},
           {ID: right.ID, ClueCount: right.ClueCount},
       },
   })
   ```
3. After the PDF block (step 5) and before the final summary (step 6), write the
   manifest and count it:
   ```go
   manifestPath := filepath.Join(outDir, "manifest.json")
   if err := writeManifest(m, manifestPath); err != nil {
       return err
   }
   filesWritten++
   ```

Notes:
- The manifest is written on **every** run (PNG-only too), and is **never**
  deleted by `-keep-png=false` — only PNGs are deleted; the manifest is metadata.
- Do NOT add a timestamp or any non-seed-derived field — it would break the
  determinism guarantee and the test below.
- Do NOT add a new CLI flag; always-emit is intentional and cheap.

## Steps

### Step 1: Create `manifest.go`

Add the file exactly as in the Target.

**Verify**: `go build ./...` → exit 0.

### Step 2: Wire the manifest into `run`

Make the three edits to `main.go` described above (init, append per page, write
after PDF).

**Verify**: `go build -o sudoprint.exe .` → exit 0. Then run
`./sudoprint.exe -n 2 -d easy -seed 42 -o ./_smoke` and confirm
`_smoke/manifest.json` exists and is valid JSON, e.g.:
`go run nothing` is not needed — just inspect: on bash,
`cat _smoke/manifest.json` shows `"seed": 42`, `"difficulty": "easy"`, and a
`pages` array of length 2 with the same clue counts printed on stdout. Delete
`_smoke` afterward (`rm -rf _smoke`).

### Step 3: Add manifest tests to `main_test.go`

Add two tests (model the style on the existing tests in `main_test.go`):

1. **`TestRunWritesManifest`** —
   ```go
   dir := t.TempDir()
   if err := run(2, "easy", dir, false, true, 42); err != nil { t.Fatal(err) }
   data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
   // unmarshal into manifest; assert:
   //   m.Seed == 42, m.Difficulty == "easy", len(m.Pages) == 2,
   //   each page has 2 puzzles, IDs are 1,2,3,4 in order,
   //   every ClueCount is within [34,40] (easy's range).
   ```
   (Use the unexported `manifest`/`pageInfo`/`puzzleInfo` types directly — the
   test is in package `main`.)
2. **`TestManifestDeterministic`** —
   ```go
   // run twice with the same seed into two temp dirs;
   // read both manifest.json files; assert bytes.Equal(a, b).
   ```

**Verify**: `go test ./...` → `ok` for all packages, including the two new tests.
(If the temp-exec policy error appears, use the local-compile workaround from
"Commands you will need".)

### Step 4: Confirm the full gate

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0
- No stray `_smoke` directory left behind

## Test plan

- New tests in `main_test.go`: `TestRunWritesManifest` (structure + values +
  range), `TestManifestDeterministic` (byte-identical for a fixed seed).
- The existing tests (`TestRunWritesPNGs`, `TestRunPDF`, `TestRunDeterministic`,
  the two rejection tests) must continue to pass unchanged — the manifest is an
  additional file and does not affect their assertions.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `manifest.go` defines `manifest`/`pageInfo`/`puzzleInfo` and `writeManifest`
- [ ] `run` writes `manifest.json` to the output dir on every run
- [ ] The manifest contains `seed`, `difficulty`, and a `pages` array whose
      per-puzzle `id`/`clueCount` match what is printed on stdout
- [ ] The manifest contains NO timestamp or other non-seed-derived field
- [ ] Two runs with the same `-seed` produce a byte-identical `manifest.json`
- [ ] `manifest.json` is NOT deleted by `-keep-png=false`
- [ ] `grep -rn "time.Now" manifest.go` returns nothing
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 007 updated to DONE

## STOP conditions

Stop and report back if:

- `main.go`'s `run` signature differs from `run(n int, difficulty, outDir string, makePDF, keepPNG bool, seed int64) error` (the codebase drifted) — reconcile, do not guess.
- A determinism assertion fails — that would indicate a non-seed-derived field
  crept into the manifest (re-check for timestamps / map iteration order).

## Maintenance notes

- If a future change parallelizes generation (the deferred concurrency item),
  the manifest must still list pages/puzzles in ID order — sort before writing.
- If solution grids are ever needed in the manifest (full offline regeneration
  without the RNG), add a field then; deliberately omitted now to keep the file
  small — seed + difficulty already determine the batch.
- A reviewer should confirm the manifest has no timestamp and that
  `-keep-png=false` leaves it in place.
