package xdraw

import (
	"image"
	"image/draw"
)

type subImager interface {
	draw.Image
	SubImage(r image.Rectangle) image.Image
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value may or may not share pixels with the original
// image.
func SubImage(img image.Image, bounds image.Rectangle) image.Image {
	subImage, ok := img.(subImager)
	if ok {
		return subImage.SubImage(bounds)
	}
	subImage = image.NewRGBA(img.Bounds())
	draw.Draw(subImage, subImage.Bounds(), img, image.Point{}, draw.Src)
	return subImage
}
