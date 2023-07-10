package vision

import (
	"image"
	"image/color"

	"github.com/pierrre/imageutil"
)

// BoundaryImage is an image that marks some boundary by filling the boundary
// with a color and leaving the rest of the image in a different color.
type BoundaryImage struct {
	img image.Image
	at  imageutil.AtFunc
	bc  [4]uint32
}

// NewBoundaryImage creates a new BoundaryImage from the given image and
// boundary color.
func NewBoundaryImage(img image.Image, bc color.Color) *BoundaryImage {
	cr, cg, cb, ca := bc.RGBA()
	return &BoundaryImage{
		img: img,
		at:  imageutil.NewAtFunc(img),
		bc:  [4]uint32{cr, cg, cb, ca},
	}
}

// PtIn returns true if the given point is in the boundary.
func (bi *BoundaryImage) PtIn(pt image.Point) bool {
	ar, ag, ab, aa := bi.at(pt.X, pt.Y)
	return ar == bi.bc[0] && ag == bi.bc[1] && ab == bi.bc[2] && aa == bi.bc[3]
}
