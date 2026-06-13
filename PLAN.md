# sudoprint — Build Plan

A Go CLI tool that generates print-ready sudoku puzzles as A4 landscape PNGs and PDFs.

---

## Goal

Generate batches of sudoku puzzles (two per page, A4 landscape) suitable for sending directly to a printer. Output is minimal and clean — no decoration, just the puzzle.

---

## Project Structure

```
sudoprint/
├── main.go
├── go.mod
├── puzzle/
│   ├── generate.go      # grid generation
│   └── solve.go         # backtracker for uniqueness validation
├── render/
│   ├── image.go         # PNG rendering
│   └── pdf.go           # PDF bundling
└── fonts/
    └── JetBrainsMono-Regular.ttf   # embedded via go:embed
```

---

## CLI Interface

```
sudoprint [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-n` | int | `1` | Number of pages to generate (2 puzzles per page) |
| `-d` | string | `medium` | Difficulty: `easy`, `medium`, `hard` |
| `-o` | string | `.` | Output directory (created if it doesn't exist) |
| `-pdf` | bool | `false` | Also produce PDFs in addition to PNGs |
| `-keep-png` | bool | `true` | Keep PNGs when `-pdf` is set. Set to `false` to discard after PDF bundling |
| `-seed` | int64 | random | RNG seed for reproducible output |

### Output files

PNGs (always):
- `puzzle_001.png`, `puzzle_002.png`, ...
- `solution_001.png`, `solution_002.png`, ...

PDFs (when `-pdf` is set):
- `puzzles.pdf` — all puzzle pages in order
- `solutions.pdf` — all solution pages in order

### Example invocations

```bash
# 5 pages (10 puzzles), medium difficulty, PNGs only
sudoprint -n 5 -d medium -o ./out

# 10 pages, hard, PNGs + PDFs, discard PNGs after
sudoprint -n 10 -d hard -o ./out -pdf -keep-png=false

# Reproducible output
sudoprint -n 3 -d easy -seed 42 -o ./out
```

---

## Puzzle Generation (`puzzle/`)

### `generate.go`

**Approach:** backtracking solver seeded with a shuffled base grid.

1. Start with a valid solved 9×9 grid:
   - Fill using a shuffled latin square approach (shuffle digits 1–9, then shift rows by box)
   - This gives a valid solved grid without backtracking
2. Remove cells according to difficulty, one at a time in random order
3. After each removal, run the uniqueness check — if the puzzle now has multiple solutions, restore the cell and skip it
4. Stop when target clue count is reached or no more cells can be removed

**Clue counts by difficulty:**

| Difficulty | Clues remaining | Cells removed |
|------------|----------------|---------------|
| `easy`     | 36             | 45            |
| `medium`   | 30             | 51            |
| `hard`     | 25             | 56            |

**Types:**

```go
type Grid [9][9]int  // 0 = empty cell

type Puzzle struct {
    ID       int
    Clues    Grid   // puzzle (with empty cells)
    Solution Grid   // fully solved grid
    Difficulty string
}
```

### `solve.go`

Standard backtracking solver used for two purposes:
1. **Uniqueness check** during generation: count solutions up to 2; if count > 1, the puzzle is not unique
2. Can also be used to verify a completed puzzle

Keep the solver simple and fast — it only needs to count to 2, so it short-circuits early.

---

## Rendering (`render/`)

### Canvas

- **Page size:** A4 landscape at 300 DPI = **3508 × 2480 px**
- **Background:** white
- **Two puzzle areas:** left half and right half, split at x = 1754

### `image.go`

Renders one PNG containing two puzzles side by side.

**Layout per half (width = 1754px):**

```
[  outer margin  |  grid  |  outer margin  ]
```

- Outer margin: ~15% of half-width on each side (~263px)
- Grid width: remaining space = ~1228px
- Grid is square, so grid height = grid width = ~1228px
- Grid is vertically centered in the 2480px height
- Label sits ~40px below the bottom of the grid

**Grid drawing:**

- Cell size = grid width / 9 ≈ 136px
- **Cell borders:** 1px, color `#CCCCCC`
- **Box borders (3×3):** 3px, color `#000000`
- **Outer border:** 4px, color `#000000`
- No fill — white background

**Numbers:**

- Font: JetBrains Mono Regular, embedded via `go:embed`
- Font size: ~55% of cell height ≈ 75px
- Color: `#000000` for given clues (puzzle) / `#888888` for filled cells (solution page)
- Each digit centered horizontally and vertically within its cell

**Center divider:**

- Thin vertical line at x = 1754, full page height
- 1px wide, color `#DDDDDD`

**Label (below each grid):**

- Format: `#1 · medium`
- Small caps style: uppercase, font size ~28px
- Color: `#AAAAAA`
- Horizontally centered under the grid

**Font loading:**

```go
//go:embed fonts/JetBrainsMono-Regular.ttf
var fontBytes []byte
```

Use `golang.org/x/image/font` + `github.com/golang/freetype` for text rendering.

**Function signature:**

```go
func RenderPage(left, right Puzzle) (image.Image, error)
```

### `pdf.go`

Bundles a slice of PNG images into a single PDF.

- Use `github.com/signintech/gopdf` — lightweight, no CGO
- Each image becomes one page, A4 landscape
- No metadata, no cover page, no page numbers in the PDF itself (numbers are on the puzzle)
- Two separate calls: one for puzzles, one for solutions

**Function signature:**

```go
func BundlePDF(images []image.Image, outputPath string) error
```

---

## Dependencies

```
golang.org/x/image              # font rendering support
github.com/golang/freetype      # TTF rasterizer
github.com/signintech/gopdf     # PDF generation, no CGO
```

Download JetBrains Mono from:
`https://github.com/JetBrains/JetBrainsMono/releases` (OFL license, free to embed)

---

## `main.go` Flow

```
1. Parse flags, validate inputs
2. Create output directory if needed
3. Init RNG with seed (or random seed, print it to stdout for reference)
4. Loop n times:
   a. Generate puzzle LEFT  (ID = page*2 - 1)
   b. Generate puzzle RIGHT (ID = page*2)
   c. Render puzzle page PNG  → save as puzzle_NNN.png
   d. Render solution page PNG → save as solution_NNN.png
5. If -pdf:
   a. Load all puzzle PNGs → BundlePDF → puzzles.pdf
   b. Load all solution PNGs → BundlePDF → solutions.pdf
   c. If -keep-png=false, delete PNGs
6. Print summary to stdout: seed used, files written, output dir
```

Print seed to stdout even when randomly chosen, so output is always reproducible:
```
seed: 8675309
written: ./out/puzzle_001.png ... (10 files)
```

---

## Non-goals

- No interactive mode
- No color
- No custom fonts via flag (font is embedded)
- No web server or API
- No answer validation or hint system
- No other puzzle types

---

## Build & Run

```bash
go mod tidy
go build -o sudoprint .
./sudoprint -n 5 -d medium -o ./out -pdf
```
