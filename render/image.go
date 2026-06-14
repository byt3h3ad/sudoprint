package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"sudoprint/puzzle"
)

const (
	pageW    = 3508
	pageH    = 2480
	halfW    = 1754
	gridSize = 1228
)

// fillRect draws a solid rectangle on dst from (x0,y0) to (x1,y1) (exclusive).
func fillRect(dst *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	r := image.Rect(x0, y0, x1, y1)
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// linePositions returns the 10 pixel positions (inclusive) for grid lines
// across a span of total pixels, for 9 cells, using rounding to avoid drift.
func linePositions(origin, total int) [10]int {
	var pos [10]int
	for i := 0; i <= 9; i++ {
		pos[i] = origin + int(math.Round(float64(i)*float64(total)/9))
	}
	return pos
}

// drawGrid renders a single puzzle half onto dst.
// halfX0 is the left edge of this half (0 for left, 1754 for right).
func drawGrid(dst *image.RGBA, p puzzle.Puzzle, solution bool, halfX0 int) error {
	gridX := halfX0 + (halfW-gridSize)/2 // centers horizontally in the half
	gridY := (pageH - gridSize) / 2      // centers vertically

	xs := linePositions(gridX, gridSize)
	ys := linePositions(gridY, gridSize)

	// --- Draw cell lines (1px #CCCCCC) ---
	cellColor := color.RGBA{0xCC, 0xCC, 0xCC, 0xFF}
	for i := 0; i <= 9; i++ {
		// Vertical line at xs[i]
		fillRect(dst, xs[i], gridY, xs[i]+1, gridY+gridSize, cellColor)
		// Horizontal line at ys[i]
		fillRect(dst, gridX, ys[i], gridX+gridSize, ys[i]+1, cellColor)
	}

	// --- Draw box lines (3px #000000) at i in {0,3,6,9} ---
	boxColor := color.RGBA{0x00, 0x00, 0x00, 0xFF}
	for _, i := range []int{0, 3, 6, 9} {
		// Vertical box line
		fillRect(dst, xs[i]-1, gridY, xs[i]+2, gridY+gridSize, boxColor)
		// Horizontal box line
		fillRect(dst, gridX, ys[i]-1, gridX+gridSize, ys[i]+2, boxColor)
	}

	// --- Draw outer border (4px #000000) as four thick rects on perimeter ---
	outerColor := color.RGBA{0x00, 0x00, 0x00, 0xFF}
	// Top
	fillRect(dst, gridX-2, gridY-2, gridX+gridSize+2, gridY+2, outerColor)
	// Bottom
	fillRect(dst, gridX-2, gridY+gridSize-2, gridX+gridSize+2, gridY+gridSize+2, outerColor)
	// Left
	fillRect(dst, gridX-2, gridY-2, gridX+2, gridY+gridSize+2, outerColor)
	// Right
	fillRect(dst, gridX+gridSize-2, gridY-2, gridX+gridSize+2, gridY+gridSize+2, outerColor)

	// --- Draw digits ---
	// Determine which grid and color to use.
	var g puzzle.Grid
	var digitColor color.Color
	if solution {
		g = p.Solution
		digitColor = color.RGBA{0x88, 0x88, 0x88, 0xFF}
	} else {
		g = p.Clues
		digitColor = color.RGBA{0x00, 0x00, 0x00, 0xFF}
	}

	// Cell height in pixels: roughly gridSize/9, use ~55% for font size.
	cellHeightApprox := float64(gridSize) / 9.0
	digitPx := cellHeightApprox * 0.55

	face, err := newFace(digitPx)
	if err != nil {
		return fmt.Errorf("newFace(digit): %w", err)
	}
	defer face.Close()

	m := face.Metrics()

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(digitColor),
		Face: face,
	}

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			v := g[row][col]
			if v == 0 {
				continue
			}
			s := fmt.Sprintf("%d", v)

			// Cell pixel bounds from consecutive line positions.
			cx0 := xs[col]
			cx1 := xs[col+1]
			cy0 := ys[row]
			cy1 := ys[row+1]

			cellCenterX := (cx0 + cx1) / 2
			cellCenterY := (cy0 + cy1) / 2

			// Measure string width for horizontal centering.
			adv := d.MeasureString(s)
			// Vertical centering: baseline = cellCenterY + (ascent - descent)/2
			// All in fixed.Int26_6.
			cellCenterYFixed := fixed.I(cellCenterY)
			baseline := cellCenterYFixed + (m.Ascent-m.Descent)/2

			dotX := fixed.I(cellCenterX) - adv/2

			d.Dot = fixed.Point26_6{X: dotX, Y: baseline}
			d.DrawString(s)
		}
	}

	// --- Draw label below grid ---
	labelText := fmt.Sprintf("#%d · %s", p.ID, strings.ToUpper(p.Difficulty))
	labelPx := 28.0
	labelFace, err := newFace(labelPx)
	if err != nil {
		return fmt.Errorf("newFace(label): %w", err)
	}
	defer labelFace.Close()

	labelColor := color.RGBA{0xAA, 0xAA, 0xAA, 0xFF}
	labelY := gridY + gridSize + 40

	ld := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(labelColor),
		Face: labelFace,
	}
	labelAdv := ld.MeasureString(labelText)
	labelCenterX := halfX0 + halfW/2
	ld.Dot = fixed.Point26_6{
		X: fixed.I(labelCenterX) - labelAdv/2,
		Y: fixed.I(labelY),
	}
	ld.DrawString(labelText)

	return nil
}

// RenderPage draws two puzzles side by side on one A4-landscape (3508x2480)
// canvas. If solution is true, it renders each puzzle's full Solution grid in
// grey (#888888); otherwise it renders the Clues grid in black (#000000).
func RenderPage(left, right puzzle.Puzzle, solution bool) (image.Image, error) {
	dst := image.NewRGBA(image.Rect(0, 0, pageW, pageH))

	// Fill background white.
	fillRect(dst, 0, 0, pageW, pageH, color.White)

	// Draw left puzzle.
	if err := drawGrid(dst, left, solution, 0); err != nil {
		return nil, fmt.Errorf("left grid: %w", err)
	}

	// Draw right puzzle.
	if err := drawGrid(dst, right, solution, halfW); err != nil {
		return nil, fmt.Errorf("right grid: %w", err)
	}

	// Center divider: 1px vertical line at x=1753 (width 1px → x=1753 to 1754).
	fillRect(dst, 1753, 0, 1754, pageH, color.RGBA{0xDD, 0xDD, 0xDD, 0xFF})

	return dst, nil
}
