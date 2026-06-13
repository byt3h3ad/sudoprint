package render

import (
	"image"
	"image/color"
	"os"
	"testing"
)

func whiteRGBA(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, white)
		}
	}
	return img
}

func TestBundlePDFEmpty(t *testing.T) {
	tmp := t.TempDir()
	err := BundlePDF(nil, tmp+"/out.pdf")
	if err == nil {
		t.Fatal("expected non-nil error for empty images, got nil")
	}
}

func TestBundlePDFWritesValidFile(t *testing.T) {
	imgs := []image.Image{
		whiteRGBA(100, 100),
		whiteRGBA(100, 100),
	}
	out := t.TempDir() + "/output.pdf"
	if err := BundlePDF(imgs, out); err != nil {
		t.Fatalf("BundlePDF returned error: %v", err)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("could not open output file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		t.Fatalf("could not read header: %v", err)
	}
	if string(header) != "%PDF" {
		t.Fatalf("expected file to start with %%PDF, got %q", string(header))
	}
}

func TestBundlePDFPageCount(t *testing.T) {
	tmp := t.TempDir()

	img1 := whiteRGBA(100, 100)
	path1 := tmp + "/one.pdf"
	if err := BundlePDF([]image.Image{img1}, path1); err != nil {
		t.Fatalf("BundlePDF(1 image) error: %v", err)
	}

	imgs3 := []image.Image{whiteRGBA(100, 100), whiteRGBA(100, 100), whiteRGBA(100, 100)}
	path3 := tmp + "/three.pdf"
	if err := BundlePDF(imgs3, path3); err != nil {
		t.Fatalf("BundlePDF(3 images) error: %v", err)
	}

	info1, _ := os.Stat(path1)
	info3, _ := os.Stat(path3)
	if info3.Size() <= info1.Size() {
		t.Fatalf("expected 3-page PDF (%d bytes) to be larger than 1-page PDF (%d bytes)", info3.Size(), info1.Size())
	}
}
