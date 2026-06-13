# Plan 004: Render a two-puzzle A4-landscape page to a PNG with embedded font

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: This plan was written against commit `e90880a`.
> Run `git diff --stat e90880a..HEAD -- puzzle/ render/`. Confirm the
> `puzzle.Puzzle` type in "Current state" matches the live `puzzle` package.
> The `render/` directory may not exist yet — that is expected; create it. If
> `render/image.go` already exists, read it and compare against the Target; on a
> mismatch, STOP.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/003-generator.md (needs the `puzzle.Puzzle` type)
- **Category**: correctness + tests
- **Planned at**: commit `e90880a`, 2026-06-13

## Why this matters

This produces the actual printable artifact. Two rendering traps are fixed here:
a non-unmaintained text stack (`golang.org/x/image/font/opentype`, **not**
`github.com/golang/freetype`), and integer rounding of gridlines so the last line
aligns with the outer border instead of drifting (cell size 1228/9 ≈ 136.44 px is
not an integer). Getting the geometry and draw order right here is what makes the
output look clean on paper.

## Current state

`puzzle` package (plan 003) provides:

```go
type Grid [9][9]int
type Puzzle struct {
	ID         int
	Clues      Grid
	Solution   Grid
	Difficulty string
	ClueCount  int
}
```

The font file `fonts/JetBrainsMono-Regular.ttf` exists (plan 001). No `render/`
package exists yet.

Geometry from `PLAN.md` §115–164 (authoritative):
- Page: A4 landscape, 300 DPI = **3508 × 2480 px**, white background.
- Split into left/right halves at x = 1754. Each half is 1754 px wide.
- Per half: outer margin ≈ 15% of half-width (≈ 263 px) each side → grid width
  ≈ 1228 px, square. Grid vertically centered in 2480 px.
- Cell borders 1 px `#CCCCCC`; box (3×3) borders 3 px `#000000`; outer border
  4 px `#000000`.
- Digits: ~55% of cell height ≈ 75 px; given clues `#000000`, solution-page
  filled cells `#888888`; centered in each cell.
- Center divider: 1 px vertical line at x = 1754, full height, `#DDDDDD`.
- Label below each grid: e.g. `#1 · MEDIUM`, uppercase ~28 px, `#AAAAAA`,
  centered ~40 px below the grid.

## Commands you will need

| Purpose   | Command                  | Expected on success     |
|-----------|--------------------------|-------------------------|
| Build     | `go build ./...`         | exit 0                  |
| Vet       | `go vet ./...`           | exit 0                  |
| Test pkg  | `go test ./render/`      | `ok  	sudoprint/render`  |

## Scope

**In scope**:
- `fonts/fonts.go` (create) — turns the existing `fonts/` directory into a Go
  package that embeds and exports the font bytes (single source of truth; see
  "Font embedding" and Step 1)
- `render/font.go` (create) — the `newFace` helper; imports `sudoprint/fonts`
- `render/image.go` (create)
- `render/image_test.go` (create)

**Out of scope** (do NOT touch):
- `fonts/JetBrainsMono-Regular.ttf` — already committed; do NOT move, copy, or
  modify it. There must remain exactly ONE copy of the `.ttf`.
- `render/pdf.go` — plan 005.
- `puzzle/*`, `main.go`.

## Git workflow

Not a git repository. Do not init/commit/push.

## Target

`render/image.go` exposes:

```go
package render

import (
	"image"
	"sudoprint/puzzle"
)

// RenderPage draws two puzzles side by side on one A4-landscape (3508x2480)
// canvas. If solution is true, it renders each puzzle's full Solution grid in
// grey (#888888); otherwise it renders the Clues grid in black (#000000).
func RenderPage(left, right puzzle.Puzzle, solution bool) (image.Image, error)
```

Note this signature differs from `PLAN.md` §178 (`RenderPage(left, right Puzzle)`)
by adding a `solution bool`, so one function renders both the puzzle page and the
solution page. This is intentional — it avoids duplicating layout code.

### Font embedding (single copy — no duplication)

`go:embed` can only reach files in the **same directory** as the `.go` file or a
subdirectory — it cannot reference a parent (`../fonts/...` is illegal). The font
lives at the repo-root `fonts/JetBrainsMono-Regular.ttf`, but the `render`
package needs its bytes. Resolve this **without copying the font** by making
`fonts/` its own tiny package that embeds and exports the bytes, then importing
it from `render`.

Create `fonts/fonts.go` (the `.ttf` already sits in this directory, so the embed
path is just the bare filename):

```go
// Package fonts embeds the bundled TTF assets.
package fonts

import _ "embed"

//go:embed JetBrainsMono-Regular.ttf
var Regular []byte
```

Then `render` imports `sudoprint/fonts` and uses `fonts.Regular`. There is
exactly ONE copy of the `.ttf`, at its original path — do NOT copy the font into
`render/`.

Create `render/font.go` with the face helper:

```go
package render

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"sudoprint/fonts"
)

func newFace(px float64) (font.Face, error) {
	f, err := opentype.Parse(fonts.Regular)
	if err != nil {
		return nil, err
	}
	// DPI 72 makes Size in points equal pixels, so px maps 1:1.
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size: px, DPI: 72, Hinting: font.HintingFull,
	})
}
```

### Drawing helpers

Work on `*image.RGBA` sized `image.Rect(0,0,3508,2480)`, filled white.

- **fillRect(dst *image.RGBA, x0,y0,x1,y1 int, c color.Color)** — draw a solid
  rectangle via `draw.Draw(dst, rect, image.NewUniform(c), image.Point{}, draw.Src)`.
  Use this for every line (a line is a thin rectangle), drawing thin light lines
  first, then thick black lines on top, so overlaps look correct.
- **Gridline positions without drift**: for line index `i` in `0..9`, the x of
  that vertical line is `gridX + int(math.Round(float64(i)*gridW/9))`, and
  similarly for y with `gridH`. Compute each cell's pixel bounds from consecutive
  line positions, not from `i*cellSize`.
- **Digit drawing**: use `font.Drawer{Dst, Src: image.NewUniform(c), Face}`.
  Center horizontally with `d.MeasureString(s)` (a `fixed.Int26_6`); center
  vertically using face metrics (`m := face.Metrics(); baseline = cellCenterY +
  (m.Ascent - m.Descent)/2` in fixed-point, converted to pixels). Set
  `d.Dot = fixed.Point26_6{X: ..., Y: ...}` then `d.DrawString(s)`.

### Layout per half

For half index `h` (0 = left, 1 = right), `halfX0 = h*1754`. Within the half:
- `gridW = gridH = 1228`
- `gridX = halfX0 + (1754-1228)/2` (centers horizontally)
- `gridY = (2480-1228)/2` (centers vertically)
- Draw cell lines (1px `#CCCCCC`) at all 10 positions, then box lines (3px
  `#000000`) at i ∈ {0,3,6,9}, then the outer border (4px `#000000`) as four
  thick rects on the perimeter.
- Draw digits: pick `g := p.Clues` if `!solution` else `p.Solution`; color black
  or `#888888` accordingly; skip zero cells.
- Draw the label centered at `y = gridY + gridH + 40` using a ~28px face,
  text `fmt.Sprintf("#%d · %s", p.ID, strings.ToUpper(p.Difficulty))`, color
  `#AAAAAA`.

Finally draw the center divider: `fillRect(dst, 1753, 0, 1754, 2480, #DDDDDD)`.

## Steps

### Step 1: Set up the embedded font and `newFace`

Create `fonts/fonts.go` (the `fonts/` directory already contains
`JetBrainsMono-Regular.ttf`) with the `package fonts` embed shown in "Font
embedding" above. Then create `render/font.go` with the `newFace` helper that
imports `sudoprint/fonts`. Do NOT copy the `.ttf` anywhere.

**Verify**: `go build ./fonts/ ./render/` → exit 0 (proves the embed directive
resolves). If you get "pattern ...: no matching files found", the directive in
`fonts/fonts.go` must name the file by its bare name
(`//go:embed JetBrainsMono-Regular.ttf`), not a path. (The actual TTF parse runs
at runtime inside `newFace`, so a corrupt-font error surfaces in the Step 3
tests, not here — see STOP conditions.)

### Step 2: Write the render test first (red)

Create `render/image_test.go`:

1. **`TestRenderPageDimensions`** — build two trivial puzzles (you can construct
   `puzzle.Puzzle` literals directly; a few clue cells are enough), call
   `RenderPage(l, r, false)`, assert no error and `img.Bounds()` equals
   `image.Rect(0,0,3508,2480)`.
2. **`TestRenderPageBackgroundWhite`** — assert the top-left corner pixel
   `(0,0)` is white (R,G,B = 255).
3. **`TestRenderDrawsInk`** — count non-white pixels in the returned image;
   assert it is > 0 (proves something was drawn). Keep it cheap: sample a stride
   (e.g. every 17th pixel) rather than all 8.7M.
4. **`TestRenderSolutionVariant`** — `RenderPage(l, r, true)` returns no error
   and the correct dimensions.

**Verify**: `go test ./render/` → compile failure (RenderPage undefined). Expected.

### Step 3: Implement `render/image.go` (green)

Implement helpers and `RenderPage` per the Target.

**Verify**: `go test ./render/` → `ok`, all four tests pass.

### Step 4: Confirm the gate

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0

## Test plan

- New file `render/image_test.go`: dimensions (puzzle + solution variants),
  white background, ink-present. Rendering exactness is hard to unit-test;
  these smoke + invariant checks are the pragmatic bar. Pixel-perfect golden
  testing is deliberately out of scope.
- Verification: `go test ./render/` → all pass.

## Done criteria

ALL must hold:

- [ ] `render/image.go` defines `RenderPage(left, right puzzle.Puzzle, solution bool) (image.Image, error)`
- [ ] Output image is exactly 3508×2480
- [ ] Font is embedded via the `fonts` package; exactly ONE copy of the `.ttf`
      exists — `test ! -e render/fonts/JetBrainsMono-Regular.ttf` (no second copy)
- [ ] Uses `golang.org/x/image/font/opentype`; `grep -rn "golang/freetype" fonts/ render/` returns nothing
- [ ] Gridline positions computed via rounding (no `i*cellSize` accumulation) —
      reviewer-checkable in `image.go`
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 004 updated to DONE

## STOP conditions

Stop and report back if:

- The `go:embed` directive in `fonts/fonts.go` cannot find the font (report the
  exact build error — the directive must use the bare filename, no path or `..`).
- `opentype.Parse(fonts.Regular)` returns an error — the font asset is bad.
- Text rendering produces a panic from `font.Drawer` — likely a nil face or
  out-of-bounds `Dot`; report the stack.

## Maintenance notes

- If page DPI or size changes, every geometry constant (3508, 2480, 1754, 1228,
  margins, font sizes) must be revisited together — consider extracting them to
  named constants at the top of `image.go` to make that safe.
- The font lives in exactly one place: `fonts/JetBrainsMono-Regular.ttf`,
  embedded and exported by the `fonts` package and imported by `render`. There is
  no second copy to keep in sync.
- Plan 005 (PDF) consumes the `image.Image` values this returns; keep the return
  type `image.Image`.
- Reviewer should eyeball an actual rendered PNG once (run the tool after plan
  006) — unit tests confirm size/ink but not visual correctness.
