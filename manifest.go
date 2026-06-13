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
