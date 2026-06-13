# Plan 005: Bundle rendered images into A4-landscape PDFs

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in the "STOP conditions" section occurs, stop and report — do not
> improvise. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: This plan was written against commit `e90880a`.
> Run `git diff --stat e90880a..HEAD -- render/`. Plan 004 should have added
> `render/image.go`, `render/image_test.go`, `render/font.go`, and
> `render/fonts/`. Confirm `RenderPage` matches "Current state". If
> `render/pdf.go` already exists, read it and compare against the Target; on a
> mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/004-image-rendering.md
- **Category**: correctness + tests
- **Planned at**: commit `e90880a`, 2026-06-13

## Why this matters

The PDF is the deliverable people actually send to a printer. The trap here is
units: `gopdf` pages are measured in **points** (A4 landscape = 841.89 × 595.28 pt),
but the images are 3508 × 2480 px at 300 DPI. The image rectangle must fill the
full page rect, or the PDF comes out wrong-sized or low-resolution. There is also
an API-version trap: older `gopdf` only accepted image *file paths*, while newer
versions expose `ImageFrom(image.Image, ...)`. This plan verifies which is
available and falls back safely.

## Current state

`render` package (plan 004) provides `RenderPage(...) (image.Image, error)`.
`github.com/signintech/gopdf` is in `go.mod` (plan 001). No `render/pdf.go` yet.

## Commands you will need

| Purpose   | Command                  | Expected on success    |
|-----------|--------------------------|------------------------|
| Build     | `go build ./...`         | exit 0                 |
| Vet       | `go vet ./...`           | exit 0                 |
| Test pkg  | `go test ./render/`      | `ok  	sudoprint/render` |
| Inspect API | `go doc github.com/signintech/gopdf GoPdf` | lists methods incl. `ImageFrom` or `Image` |

## Scope

**In scope**:
- `render/pdf.go` (create)
- `render/pdf_test.go` (create)

**Out of scope**: `render/image.go`, `puzzle/*`, `main.go`.

## Git workflow

Not a git repository. Do not init/commit/push.

## Target

`render/pdf.go`:

```go
package render

import "image"

// BundlePDF writes images as a single PDF at outputPath, one image per page,
// each page A4 landscape (841.89 x 595.28 pt) with the image filling the page.
// Returns an error if images is empty or the file cannot be written.
func BundlePDF(images []image.Image, outputPath string) error
```

### Implementation

```go
import (
	"fmt"
	"image"

	"github.com/signintech/gopdf"
)

func BundlePDF(images []image.Image, outputPath string) error {
	if len(images) == 0 {
		return fmt.Errorf("BundlePDF: no images")
	}
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4Landscape})
	pageRect := gopdf.PageSizeA4Landscape // *gopdf.Rect{W:841.89, H:595.28}
	for _, img := range images {
		pdf.AddPage()
		if err := pdf.ImageFrom(img, 0, 0, pageRect); err != nil {
			return fmt.Errorf("BundlePDF: add image: %w", err)
		}
	}
	return pdf.WritePdf(outputPath)
}
```

**API verification (do this in Step 1 before writing the above):** run
`go doc github.com/signintech/gopdf GoPdf`.
- If `ImageFrom(img image.Image, x, y float64, rect *gopdf.Rect) error` is
  listed, use the code above.
- If it is **not** listed (older version), there are two options: (a) `go get -u
  github.com/signintech/gopdf@latest && go mod tidy` to get a version with
  `ImageFrom`, then re-check; or (b) fall back to encoding each image to a temp
  PNG file and using `pdf.Image(path, 0, 0, pageRect)`. Prefer (a). If neither
  `ImageFrom` nor `Image` exists, STOP and report the available methods.

Also confirm `gopdf.PageSizeA4Landscape` exists (`go doc github.com/signintech/gopdf
PageSizeA4Landscape`). If the symbol name differs, define the rect explicitly:
`&gopdf.Rect{W: 841.89, H: 595.28}`.

## Steps

### Step 1: Verify the gopdf API

Run `go doc github.com/signintech/gopdf GoPdf` and
`go doc github.com/signintech/gopdf PageSizeA4Landscape`. Decide which code path
(ImageFrom vs Image-with-tempfile) applies per the Target.

**Verify**: you have confirmed which image method exists. Record it in a comment
at the top of `pdf.go`.

### Step 2: Write the PDF test first (red)

Create `render/pdf_test.go`:

1. **`TestBundlePDFEmpty`** — `BundlePDF(nil, <tmp>)` returns a non-nil error.
2. **`TestBundlePDFWritesValidFile`** — generate 2 small images. The page area
   (3508×2480) is large; for the test you may use `RenderPage` with literal
   puzzles, or smaller `image.NewRGBA` images filled white — `BundlePDF` does not
   require a specific size. Call `BundlePDF(imgs, path)` where `path` is in
   `t.TempDir()`. Assert: no error; the file exists and is non-empty; its first
   4 bytes are `%PDF` (read the file header and compare to `[]byte("%PDF")`).
3. **`TestBundlePDFPageCount`** (best-effort) — assert the file size grows with
   more pages: bundle 1 image vs 3 images into two temp files and assert the
   3-page file is larger. (A full PDF page-count parse is overkill; size
   monotonicity is a sufficient smoke check.)

**Verify**: `go test ./render/` → compile failure (BundlePDF undefined). Expected.

### Step 3: Implement `render/pdf.go` (green)

Implement per the Target and the API path chosen in Step 1.

**Verify**: `go test ./render/` → `ok`, all render tests pass (plan 004's plus
these new ones).

### Step 4: Confirm the gate

**Verify**:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → exit 0

## Test plan

- New file `render/pdf_test.go`: empty-input error, valid `%PDF` file written,
  size grows with page count.
- Verification: `go test ./render/` → all pass.

## Done criteria

ALL must hold:

- [ ] `render/pdf.go` defines `BundlePDF([]image.Image, string) error`
- [ ] Empty input returns an error
- [ ] Output file begins with `%PDF` and is non-empty
- [ ] Pages are A4 landscape and the image fills the page (verified by the
      page-rect usage in code)
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` all exit 0
- [ ] `plans/README.md` status row for 005 updated to DONE

## STOP conditions

Stop and report back if:

- Neither `ImageFrom(image.Image,...)` nor `Image(path,...)` exists on
  `gopdf.GoPdf` (report `go doc` output).
- `WritePdf` errors with a permissions/path issue you cannot resolve within the
  temp dir.
- The written file does not start with `%PDF` after a reasonable fix attempt.

## Maintenance notes

- **Follow-up F1 (cleanup):** Plan 001 added a root `tools.go` (`//go:build
  tools`) solely to keep `golang.org/x/image` and `github.com/signintech/gopdf`
  in `go.mod` before any real code imported them. After this plan imports
  `github.com/signintech/gopdf` (and plan 004 imports `golang.org/x/image/...`),
  `tools.go` is redundant. As part of this plan, **delete `tools.go` from the
  repo root, then run `go mod tidy`** and confirm `go.mod` still requires both
  deps and the gate (`go build/vet/test ./...`) stays green. If `go mod tidy`
  removes either dep after deleting `tools.go`, that means the real imports
  aren't in place yet — STOP and report rather than leaving `tools.go` in.
- If `render/image.go` ever changes page dimensions, the PDF page rect
  (841.89×595.28 pt) still corresponds to A4 landscape regardless of image px —
  `ImageFrom` scales the image into the rect. Only revisit if you switch paper
  sizes.
- `main.go` (plan 006) calls `BundlePDF` twice (puzzles, solutions). Keep the
  signature stable.
- Reviewer should open a generated PDF once in a viewer to confirm orientation
  and that images aren't stretched (unit tests can't see that).
