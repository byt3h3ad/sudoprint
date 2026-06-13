package render

import (
	"image"
	"image/color"
	"testing"

	"sudoprint/puzzle"
)

// makePuzzles returns two trivial puzzle.Puzzle literals with a few clue cells.
func makePuzzles() (puzzle.Puzzle, puzzle.Puzzle) {
	var clues puzzle.Grid
	clues[0][0] = 5
	clues[0][1] = 3
	clues[1][0] = 6

	var solution puzzle.Grid
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			solution[r][c] = (r+c)%9 + 1
		}
	}

	left := puzzle.Puzzle{
		ID:         1,
		Clues:      clues,
		Solution:   solution,
		Difficulty: "medium",
		ClueCount:  3,
	}
	right := puzzle.Puzzle{
		ID:         2,
		Clues:      clues,
		Solution:   solution,
		Difficulty: "hard",
		ClueCount:  3,
	}
	return left, right
}

func TestRenderPageDimensions(t *testing.T) {
	l, r := makePuzzles()
	img, err := RenderPage(l, r, false)
	if err != nil {
		t.Fatalf("RenderPage returned error: %v", err)
	}
	want := image.Rect(0, 0, 3508, 2480)
	if img.Bounds() != want {
		t.Errorf("bounds = %v, want %v", img.Bounds(), want)
	}
}

func TestRenderPageBackgroundWhite(t *testing.T) {
	l, r := makePuzzles()
	img, err := RenderPage(l, r, false)
	if err != nil {
		t.Fatalf("RenderPage returned error: %v", err)
	}
	// color.Color.RGBA() returns 16-bit values; white is R=G=B=A=0xffff.
	got := img.At(0, 0)
	rr, gg, bb, aa := got.RGBA()
	if rr != 0xffff || gg != 0xffff || bb != 0xffff || aa != 0xffff {
		t.Errorf("pixel (0,0) = %v, want white (0xffff,0xffff,0xffff,0xffff)", color.RGBAModel.Convert(got))
	}
}

func TestRenderDrawsInk(t *testing.T) {
	l, r := makePuzzles()
	img, err := RenderPage(l, r, false)
	if err != nil {
		t.Fatalf("RenderPage returned error: %v", err)
	}
	bounds := img.Bounds()
	nonWhite := 0
	// Sample every 17th pixel to keep it cheap.
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 17 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 17 {
			rr, gg, bb, _ := img.At(x, y).RGBA()
			if rr != 0xffff || gg != 0xffff || bb != 0xffff {
				nonWhite++
			}
		}
	}
	if nonWhite == 0 {
		t.Error("expected non-white pixels (ink) but found none")
	}
}

func TestRenderSolutionVariant(t *testing.T) {
	l, r := makePuzzles()
	img, err := RenderPage(l, r, true)
	if err != nil {
		t.Fatalf("RenderPage(solution=true) returned error: %v", err)
	}
	want := image.Rect(0, 0, 3508, 2480)
	if img.Bounds() != want {
		t.Errorf("bounds = %v, want %v", img.Bounds(), want)
	}
}
