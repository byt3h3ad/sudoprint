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
