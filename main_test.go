package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRunWritesPNGs verifies that run produces 4 PNG files for n=2.
func TestRunWritesPNGs(t *testing.T) {
	dir := t.TempDir()
	if err := run(2, "easy", dir, false, true, 42); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	expectedFiles := []string{
		"puzzle_001.png",
		"puzzle_002.png",
		"solution_001.png",
		"solution_002.png",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", name)
		}
	}
}

// TestRunRejectsBadDifficulty verifies that an unknown difficulty returns an error.
func TestRunRejectsBadDifficulty(t *testing.T) {
	err := run(1, "extreme", t.TempDir(), false, true, 1)
	if err == nil {
		t.Fatal("expected error for difficulty 'extreme', got nil")
	}
}

// TestRunRejectsBadN verifies that n=0 returns an error.
func TestRunRejectsBadN(t *testing.T) {
	err := run(0, "easy", t.TempDir(), false, true, 1)
	if err == nil {
		t.Fatal("expected error for n=0, got nil")
	}
}

// TestRunPDF verifies PDF generation and PNG cleanup when keepPNG=false.
func TestRunPDF(t *testing.T) {
	dir := t.TempDir()
	if err := run(1, "easy", dir, true, false, 1); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	// PDFs must exist and start with %PDF.
	pdfHeader := []byte("%PDF")
	for _, name := range []string{"puzzles.pdf", "solutions.pdf"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected PDF %s: %v", name, err)
		}
		if !bytes.HasPrefix(data, pdfHeader) {
			t.Errorf("file %s does not start with %%PDF", name)
		}
	}

	// PNGs must NOT exist (keepPNG=false).
	for _, name := range []string{"puzzle_001.png", "solution_001.png"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected PNG %s to be removed (keepPNG=false), but it exists", name)
		}
	}
}

// TestRunDeterministic verifies that identical seeds produce identical output.
func TestRunDeterministic(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := run(1, "easy", dir1, false, true, 99); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if err := run(1, "easy", dir2, false, true, 99); err != nil {
		t.Fatalf("second run error: %v", err)
	}

	data1, err := os.ReadFile(filepath.Join(dir1, "puzzle_001.png"))
	if err != nil {
		t.Fatalf("reading run1 output: %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(dir2, "puzzle_001.png"))
	if err != nil {
		t.Fatalf("reading run2 output: %v", err)
	}

	if !bytes.Equal(data1, data2) {
		t.Error("puzzle_001.png differs between runs with the same seed — determinism broken")
	}
}

// TestRunWritesManifest verifies that run produces a manifest.json with correct fields.
func TestRunWritesManifest(t *testing.T) {
	dir := t.TempDir()
	if err := run(2, "easy", dir, false, true, 42); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("reading manifest.json: %v", err)
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshaling manifest.json: %v", err)
	}

	if m.Seed != 42 {
		t.Errorf("Seed: got %d, want 42", m.Seed)
	}
	if m.Difficulty != "easy" {
		t.Errorf("Difficulty: got %q, want \"easy\"", m.Difficulty)
	}
	if len(m.Pages) != 2 {
		t.Fatalf("len(Pages): got %d, want 2", len(m.Pages))
	}

	expectedIDs := [][]int{{1, 2}, {3, 4}}
	for i, pg := range m.Pages {
		if len(pg.Puzzles) != 2 {
			t.Errorf("page %d: len(Puzzles): got %d, want 2", pg.Page, len(pg.Puzzles))
			continue
		}
		for j, pz := range pg.Puzzles {
			wantID := expectedIDs[i][j]
			if pz.ID != wantID {
				t.Errorf("page %d puzzle %d: ID: got %d, want %d", pg.Page, j, pz.ID, wantID)
			}
			// easy difficulty: clue count in [34, 40]
			if pz.ClueCount < 34 || pz.ClueCount > 40 {
				t.Errorf("page %d puzzle %d: ClueCount %d out of easy range [34,40]", pg.Page, j, pz.ClueCount)
			}
		}
	}
}

// TestManifestDeterministic verifies that identical seeds produce byte-identical manifest.json.
func TestManifestDeterministic(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := run(2, "easy", dir1, false, true, 77); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if err := run(2, "easy", dir2, false, true, 77); err != nil {
		t.Fatalf("second run error: %v", err)
	}

	data1, err := os.ReadFile(filepath.Join(dir1, "manifest.json"))
	if err != nil {
		t.Fatalf("reading run1 manifest.json: %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(dir2, "manifest.json"))
	if err != nil {
		t.Fatalf("reading run2 manifest.json: %v", err)
	}

	if !bytes.Equal(data1, data2) {
		t.Error("manifest.json differs between runs with the same seed — determinism broken")
	}
}
