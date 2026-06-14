# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`sudoprint` is a single-binary Go CLI that generates print-ready sudoku puzzles
as A4-landscape PDFs (with intermediate PNGs), two puzzles per page. By default
it bundles PDFs and discards the PNGs; `-pdf=false` skips the PDF and keeps the
PNGs, and `-keep-png` keeps both. Module path is
`sudoprint`; requires **Go 1.25+** (transitive requirement of
`golang.org/x/image`). Two direct deps, no CGO: `golang.org/x/image` (text) and
`github.com/signintech/gopdf` (PDF).

## Commands

```bash
go build -o sudoprint .   # build the CLI binary (sudoprint.exe on Windows)
go build ./...            # compile all packages
go vet ./...              # static checks
go test ./...             # full suite
go test ./puzzle/         # one package
go test -run TestCountUnique ./puzzle/   # one test
```

**Windows gotcha:** an "Application Control" endpoint policy on this machine
blocks Go test binaries from executing out of `%TEMP%`, surfacing as
`go test ./...` failing with *"An Application Control policy has blocked this
file."* This is environmental, **not** a code failure. Work around it by
compiling the test binary into the project tree and running it there:

```bash
go test -c -o t.exe ./puzzle && ./t.exe          # then: rm t.exe
go test -c -o t.exe ./puzzle && ./t.exe -test.run TestCluesAreSymmetric -test.v
```

The built `sudoprint.exe` itself runs fine from the project dir (only `%TEMP%`
execution is blocked).

## Architecture

The pipeline is linear and each package does one job. `main.go`'s `run()`
orchestrates: for each page, generate two puzzles → render a puzzle PNG and a
solution PNG → (optionally) bundle PDFs → write `manifest.json`.

```
main.go (run) ──> puzzle.Generate ──> render.RenderPage ──> savePNG
                                   └─> render.BundlePDF (if -pdf)
                                   └─> writeManifest (manifest.go)
```

- **`puzzle/solve.go`** — a backtracking solver. The linchpin is
  `CountSolutions(grid, limit)`, which counts solutions but **short-circuits once
  it reaches `limit`**. The generator calls it with `limit=2` to ask "is this
  still uniquely solvable?" (`1`=unique, `≥2`=ambiguous, `0`=unsolvable). The
  early-exit is what keeps generation fast — do not remove it.
- **`puzzle/generate.go`** — `Generate` builds a solved grid (`generateSolved`:
  a base pattern then validity-preserving shuffles), then `carve` removes clues.
  Carving is **180° rotationally symmetric** (cells removed in mirror pairs;
  center cell is self-paired), keeping a removal only if the puzzle stays unique.
- **`render/image.go`** — `RenderPage(left, right, solution)` draws one
  3508×2480 (A4 landscape @300 DPI) page with two grids. Uses
  `golang.org/x/image/font/opentype` (deliberately **not** `golang/freetype`,
  which is unmaintained). Gridline positions use rounding to avoid drift.
- **`render/pdf.go`** — `BundlePDF` lays each image on its own A4-landscape page
  via gopdf (pages are in **points**, 842×595pt; images are pixels).
- **`fonts/`** — JetBrains Mono is `go:embed`-ed once in `fonts/fonts.go`
  (package `fonts`, exporting `Regular []byte`) and imported by `render`. There
  is exactly **one** copy of the `.ttf`; do not duplicate it into `render/`
  (go:embed can't reach a parent dir, hence the separate package).

## Invariants — read before changing generation or the CLI

These are load-bearing contracts, several enforced by tests:

- **Determinism.** All randomness MUST flow through the injected `*rand.Rand`
  (`rng.Intn`, `rng.Shuffle`, `rng.Perm`). No global `math/rand` calls, no
  `time.Now()` inside generation. A fixed `-seed` must reproduce a batch
  byte-for-byte (`TestDeterministic`, `TestManifestDeterministic`).
- **Uniqueness.** Every generated puzzle must satisfy
  `CountSolutions(Clues, 2) == 1`. Non-unique output is a hard failure.
- **Difficulty is a reported range, not a promise.** `difficulties` maps each
  level to `{target, min, max}`; `Generate` retries (keeping the sparsest valid
  attempt) and stamps the *actual* `ClueCount`, which is printed to stdout and
  written to the manifest. If you change the carving strategy, re-measure
  achievable clue counts across many seeds and **re-tune the ranges** — symmetric
  carving moves in steps of ~2, so ranges must absorb that. (`hard` must stay
  meaningfully sparser than `medium`.)
- **Manifest is seed-derived only.** `manifest.json` must contain no timestamp or
  other non-deterministic field, or determinism breaks.
- **PDF-before-delete.** When PNGs are discarded (the default: `-pdf` on,
  `-keep-png` off), they are removed only *after* both PDFs are written
  successfully; the manifest is never deleted.

## Conventions

- Commits follow Conventional Commits (`feat:`, `docs:`, `chore:`), with a
  `Co-Authored-By:` trailer where applicable — match `git log`.
- Tests are first-class: solver correctness, grid validity, uniqueness, clue-count
  ranges across seeds, clue-pattern symmetry, determinism, and end-to-end CLI
  behavior all have coverage. Add to them when changing behavior.
- Geometry/layout constants live at the top of `render/image.go`; if page size or
  DPI changes, they must move together.

## plans/

This project was built plan-by-plan via the `improve` skill workflow.
`plans/README.md` is the index (execution order, status, dependency notes, and
deferred items like batch-generation concurrency). `plans/NNN-*.md` are the
self-contained implementation plans. They are historical/aspirational records,
not always current code — verify against the source before relying on a plan's
excerpts.
