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
	pdf := flag.Bool("pdf", true, "produce PDFs (-pdf=false to skip)")
	keepPNG := flag.Bool("keep-png", false, "keep the PNGs alongside the PDFs")
	seed := flag.Int64("seed", 0, "RNG seed (0 = random)")
	flag.Parse()

	if err := run(*n, *d, *out, *pdf, *keepPNG, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run is the core logic, separated from main so it can be called from tests.
func run(n int, difficulty, outDir string, makePDF, keepPNG bool, seed int64) error {
	// 1. Validate inputs.
	if n < 1 {
		return fmt.Errorf("n must be >= 1, got %d", n)
	}
	switch difficulty {
	case "easy", "medium", "hard":
		// valid
	default:
		return fmt.Errorf("unknown difficulty %q: must be easy, medium, or hard", difficulty)
	}

	// 2. Seed RNG.
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))
	fmt.Printf("seed: %d\n", seed)

	// 3. Create output directory.
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	var puzzleImgs []image.Image
	var solutionImgs []image.Image
	var pngPaths []string // track for cleanup when !keepPNG

	// 4. Generation loop.
	m := manifest{Seed: seed, Difficulty: difficulty}
	filesWritten := 0
	for page := 1; page <= n; page++ {
		left, err := puzzle.Generate(page*2-1, difficulty, rng)
		if err != nil {
			return fmt.Errorf("generating puzzle %d: %w", page*2-1, err)
		}
		right, err := puzzle.Generate(page*2, difficulty, rng)
		if err != nil {
			return fmt.Errorf("generating puzzle %d: %w", page*2, err)
		}

		// Render puzzle page.
		puzzleImg, err := render.RenderPage(left, right, false)
		if err != nil {
			return fmt.Errorf("rendering puzzle page %d: %w", page, err)
		}
		puzzlePath := filepath.Join(outDir, fmt.Sprintf("puzzle_%03d.png", page))
		if err := savePNG(puzzleImg, puzzlePath); err != nil {
			return fmt.Errorf("saving puzzle page %d: %w", page, err)
		}
		filesWritten++

		// Render solution page.
		solutionImg, err := render.RenderPage(left, right, true)
		if err != nil {
			return fmt.Errorf("rendering solution page %d: %w", page, err)
		}
		solutionPath := filepath.Join(outDir, fmt.Sprintf("solution_%03d.png", page))
		if err := savePNG(solutionImg, solutionPath); err != nil {
			return fmt.Errorf("saving solution page %d: %w", page, err)
		}
		filesWritten++

		// Print per-page summary with actual clue counts.
		fmt.Printf("page %d: #%d (clues %d), #%d (clues %d)\n",
			page, left.ID, left.ClueCount, right.ID, right.ClueCount)

		m.Pages = append(m.Pages, pageInfo{
			Page: page,
			Puzzles: []puzzleInfo{
				{ID: left.ID, ClueCount: left.ClueCount},
				{ID: right.ID, ClueCount: right.ClueCount},
			},
		})

		if makePDF {
			puzzleImgs = append(puzzleImgs, puzzleImg)
			solutionImgs = append(solutionImgs, solutionImg)
			pngPaths = append(pngPaths, puzzlePath, solutionPath)
		}
	}

	// 5. PDF generation.
	if makePDF {
		puzzlesPDF := filepath.Join(outDir, "puzzles.pdf")
		if err := render.BundlePDF(puzzleImgs, puzzlesPDF); err != nil {
			return fmt.Errorf("writing puzzles.pdf: %w", err)
		}
		filesWritten++

		solutionsPDF := filepath.Join(outDir, "solutions.pdf")
		if err := render.BundlePDF(solutionImgs, solutionsPDF); err != nil {
			return fmt.Errorf("writing solutions.pdf: %w", err)
		}
		filesWritten++

		if !keepPNG {
			for _, p := range pngPaths {
				if err := os.Remove(p); err != nil {
					return fmt.Errorf("removing PNG %s: %w", p, err)
				}
			}
		}
	}

	// Write manifest.
	manifestPath := filepath.Join(outDir, "manifest.json")
	if err := writeManifest(m, manifestPath); err != nil {
		return err
	}
	filesWritten++

	// 6. Summary.
	fmt.Printf("done: seed=%d, %d file(s) written to %s\n", seed, filesWritten, outDir)
	return nil
}

// savePNG encodes img as PNG and writes it to path, truncating any existing file.
func savePNG(img image.Image, path string) error {
	f, err := os.Create(path) // overwrites existing file — documented behavior
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
