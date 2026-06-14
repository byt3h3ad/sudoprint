# sudoprint

> Print-ready sudoku, straight from your terminal to your printer.

```
┌───────┬───────┬───────┐
│ 9 5 · │ · · · │ 2 1 · │   sudoprint generates batches of
│ · · 5 │ 7 2 3 │ 8 · · │   uniquely-solvable sudoku puzzles as
│ · 7 · │ · 8 · │ 4 5 · │   clean, minimal A4-landscape pages —
├───────┼───────┼───────┤   two puzzles per sheet, no clutter,
│ 8 · · │ · · · │ · · 9 │   no decoration. Just the puzzle.
│ · 4 6 │ · 3 · │ · 8 · │
│ · · 3 │ 8 4 · │ 6 7 · │   Print-ready PDFs by default.
├───────┼───────┼───────┤   Reproducible. Symmetric. Honest.
│ · · · │ · · · │ · · · │
│ 4 2 · │ 5 7 · │ · · · │
└───────┴───────┴───────┘
```

`sudoprint` is a single, dependency-light Go binary. Point it at an output
directory, tell it how many pages and how hard, and it hands you printer-ready
artwork plus a machine-readable record of exactly what it made.

---

## Why it exists

Most sudoku you find online come wrapped in ads, watermarks, fixed difficulty,
or a layout that wastes half a page. `sudoprint` does one thing: produce **clean,
print-ready puzzles you actually want to send to a printer**. Black grid on white
paper, two puzzles per A4 sheet, a matching solution page, and nothing else.

Three principles shaped it:

- **Correct by construction.** Every puzzle is verified to have *exactly one*
  solution before it's ever drawn.
- **Honest.** Difficulty is a target range, not a promise — and the tool tells
  you the *actual* clue count of every puzzle it made.
- **Reproducible.** A seed fully determines a batch. Same seed, same puzzles,
  byte-for-byte, forever.

---

## Quick start

```bash
# Build
go build -o sudoprint .

# 5 pages (10 puzzles), medium difficulty — PDFs (PNGs discarded once bundled)
./sudoprint -n 5 -d medium -o ./out

# Same, but also keep the per-page PNGs
./sudoprint -n 10 -d hard -o ./out -keep-png

# PNGs only, no PDF
./sudoprint -n 5 -d medium -o ./out -pdf=false

# Reproducible batch — same seed always yields the same puzzles
./sudoprint -n 3 -d easy -seed 42 -o ./out
```

Requires **Go 1.25+** (a transitive requirement of `golang.org/x/image`).

---

## Usage

```
sudoprint [flags]
```

| Flag        | Type   | Default  | Description                                                |
|-------------|--------|----------|------------------------------------------------------------|
| `-n`        | int    | `1`      | Number of pages to generate (**2 puzzles per page**)       |
| `-d`        | string | `medium` | Difficulty: `easy`, `medium`, or `hard`                    |
| `-o`        | string | `.`      | Output directory (created if it doesn't exist)             |
| `-pdf`      | bool   | `true`   | Bundle the pages into PDFs; `-pdf=false` to skip           |
| `-keep-png` | bool   | `false`  | Keep the PNGs alongside the PDFs instead of discarding them |
| `-seed`     | int64  | random   | RNG seed for reproducible output                           |

> **Tip:** Go's `flag` package needs `-pdf=false` (with the `=`) to disable a
> bool flag; `-keep-png` on its own turns it on.

Every run prints the seed it used (even a random one) and the **actual clue
count** of each puzzle, so nothing about a batch is hidden:

```
seed: 42
page 1: #1 (clues 35), #2 (clues 35)
page 2: #3 (clues 34), #4 (clues 35)
done: seed=42, 5 file(s) written to ./out
```

---

## Output

PDFs are written by default. PNGs are intermediate artifacts, discarded once
the PDFs are bundled unless you pass `-keep-png` (or `-pdf=false`, which skips
the PDF step and leaves the PNGs in place).

| File                 | When            | Contents                                  |
|----------------------|-----------------|-------------------------------------------|
| `puzzles.pdf`        | default         | All puzzle pages, in order                |
| `solutions.pdf`      | default         | All solution pages, in order              |
| `puzzle_NNN.png`     | `-keep-png` / `-pdf=false` | One page = two puzzles (the clues) |
| `solution_NNN.png`   | `-keep-png` / `-pdf=false` | The matching solved grids (in grey) |
| `manifest.json`      | always          | The batch record (see below)              |

Pages are **A4 landscape at 300 DPI (3508 × 2480 px)** — split down the middle,
one puzzle per half, vertically centered, with a small `#N · DIFFICULTY` caption
beneath each grid. Puzzle clues render in black; solution digits in grey.

### The manifest

Every run drops a `manifest.json` next to the images — seed-derived only (no
timestamps), so the same `-seed` produces a byte-identical file. It's the
durable record of a batch: reproduce it, audit it, or feed it to your own tools.

```json
{
  "seed": 42,
  "difficulty": "easy",
  "pages": [
    { "page": 1, "puzzles": [ { "id": 1, "clueCount": 35 },
                              { "id": 2, "clueCount": 35 } ] }
  ]
}
```

The manifest is always written and never cleaned up — it's metadata, not a
render artifact.

---

## How it works

The pipeline is small and each stage does exactly one job.

### 1 · Generate a solved grid — `puzzle/generate.go`

Start from a known-valid pattern grid (`base[r][c] = (r*3 + r/3 + c) % 9 + 1`),
then scramble it with a sequence of **validity-preserving transforms**: relabel
the digits, shuffle rows within bands, columns within stacks, reorder the bands
and stacks, and transpose with 50% probability. The result is a fresh, fully
solved grid that still obeys every sudoku rule — and looks nothing like its
neighbours.

### 2 · Carve out the clues — *symmetrically*

Cells are removed in **180° rotationally-symmetric pairs** (the center cell is
its own mirror). Each candidate removal is kept only if the puzzle still has a
unique solution; otherwise the whole pair is restored. This is what gives the
finished puzzles their *published* look — the clue pattern is balanced, the way
newspaper sudoku are, not a random scatter.

### 3 · Guarantee uniqueness — `puzzle/solve.go`

A small backtracking solver underpins everything. Its key trick is
`CountSolutions(grid, limit)`, which counts solutions but **short-circuits at the
limit** — call it with `limit=2` and it answers the only question that matters
during carving: *"is this still unique?"* (`1` = unique, `≥2` = ambiguous,
`0` = unsolvable).

### 4 · Honest difficulty

Random symmetric carving can't always hit an exact clue count, so difficulty is
an **accepted range** with a target, and the generator keeps the sparsest valid
attempt:

| Difficulty | Target | Accepted range |
|------------|--------|----------------|
| `easy`     | 35     | 33 – 38        |
| `medium`   | 29     | 27 – 32        |
| `hard`     | 24     | 22 – 32        |

The *actual* count achieved is reported on stdout and in the manifest — so a
"hard" puzzle that landed at 26 clues tells you so, instead of quietly pretending.

### 5 · Render — `render/image.go`

Pages are drawn pixel-by-pixel onto an RGBA canvas using
`golang.org/x/image/font/opentype` for text. Grid lines are computed with
rounding (so the 1228/9-px cells never drift away from the outer border), drawn
light-to-dark so the 3 px box borders and 4 px frame sit cleanly on top. Digits
are centered with font metrics. The font — **JetBrains Mono** — is embedded
directly into the binary, so there are no runtime asset dependencies.

### 6 · Bundle — `render/pdf.go`

`BundlePDF` lays each rendered image onto its own A4-landscape page with
`github.com/signintech/gopdf` (pure Go, no CGO). One call for puzzles, one for
solutions.

---

## Project layout

```
sudoprint/
├── main.go              CLI: flags, generation loop, output, summary
├── manifest.go          the deterministic batch manifest
├── puzzle/
│   ├── solve.go         backtracking solver + uniqueness counter
│   ├── generate.go      grid generation + symmetric clue carving
│   └── *_test.go
├── render/
│   ├── image.go         A4-landscape PNG rendering
│   ├── pdf.go           PDF bundling
│   ├── font.go          font face helper (x/image/opentype)
│   └── *_test.go
├── fonts/
│   ├── fonts.go         //go:embed of the TTF (single source of truth)
│   └── JetBrainsMono-Regular.ttf
└── main_test.go         end-to-end CLI tests
```

`go.mod` pulls only two direct dependencies: `golang.org/x/image` (text
rendering) and `github.com/signintech/gopdf` (PDF). No CGO.

---

## Development

```bash
go build ./...     # compile everything
go vet ./...       # static checks
go test ./...      # full suite (solver, generator, rendering, PDF, CLI)
```

The test suite covers the parts that matter: solver correctness and
short-circuiting, generated-grid validity, **uniqueness of every puzzle**, clue
counts staying in range across many seeds, clue-pattern symmetry, deterministic
output for a fixed seed, and the end-to-end CLI behavior (including PDF bundling
and `-keep-png` cleanup).

> **Windows note:** some endpoint-security ("Application Control") policies block
> Go's test binaries from executing out of `%TEMP%`, which surfaces as
> `go test ./...` failing with *"An Application Control policy has blocked this
> file."* — an environment issue, not a code one. Work around it by compiling the
> test binary into the project tree first:
> `go test -c -o t.exe ./puzzle && ./t.exe`.

---

## Design notes & roadmap

A few decisions worth calling out:

- **One font, embedded once.** The TTF lives in exactly one place and is exposed
  through a tiny `fonts` package, so the `render` package can `go:embed` it
  without duplicating the asset or reaching across directories.
- **Symmetry is unconditional.** Every puzzle is symmetric. A future
  `-symmetric=false` escape hatch (for minimal-clue asymmetric grids) is a
  natural addition if anyone wants it.

Deliberately **not** built (yet):

- **Parallel batch generation.** Generation is the only CPU-bound stage, so large
  batches could fan out across cores — but doing it correctly means deriving a
  per-puzzle seed to preserve reproducibility, so it's a contained future change
  rather than a quick `go func`.
- Streaming pages into the PDF instead of buffering them (only matters for very
  large `-n`).

---

## Credits

- Typeface: [**JetBrains Mono**](https://github.com/JetBrains/JetBrainsMono),
  used under the SIL Open Font License.
- PDF generation: [`signintech/gopdf`](https://github.com/signintech/gopdf).
- Text rasterization: [`golang.org/x/image`](https://pkg.go.dev/golang.org/x/image).

---

## License

Released under the [MIT License](LICENSE). The bundled JetBrains Mono font is
licensed separately under the SIL Open Font License.

---

<p align="center"><em>Generate. Print. Solve. Repeat.</em></p>
```
