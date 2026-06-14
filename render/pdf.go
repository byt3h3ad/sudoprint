package render

import (
	"fmt"
	"image"

	"github.com/signintech/gopdf"
)

// BundlePDF writes images as a single PDF at outputPath, one image per page,
// each page A4 landscape (842 x 595 pt) with the image filling the page.
// Returns an error if images is empty or the file cannot be written.
func BundlePDF(images []image.Image, outputPath string) error {
	if len(images) == 0 {
		return fmt.Errorf("BundlePDF: no images")
	}
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4Landscape})
	pageRect := gopdf.PageSizeA4Landscape
	for _, img := range images {
		pdf.AddPage()
		if err := pdf.ImageFrom(img, 0, 0, pageRect); err != nil {
			return fmt.Errorf("BundlePDF: add image: %w", err)
		}
	}
	return pdf.WritePdf(outputPath)
}
