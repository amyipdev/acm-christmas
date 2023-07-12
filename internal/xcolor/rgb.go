package xcolor

import (
	"image"
	"image/color"
)

// RGBAImage is an image that can be converted to RGBA.
type RGBAImage interface {
	image.Image
	RGBAAt(x, y int) color.RGBA
}

var _ RGBAImage = (*image.RGBA)(nil)

// RGB is a color in the RGB color space. It is represented as 3 8-bit values
// for red, green, and blue.
type RGB struct {
	R, G, B uint8
}

// RGBFromRGBA converts a color.RGBA to RGB.
func RGBFromRGBA(c color.RGBA) RGB {
	return RGB{c.R, c.G, c.B}
}

// RGBFromColor converts any color.Color to RGB.
func RGBFromColor(c color.Color) RGB {
	if c, ok := c.(color.RGBA); ok {
		return RGBFromRGBA(c)
	}
	r, g, b, _ := c.RGBA()
	return RGB{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
	}
}

// RGBA implements the color.Color interface.
func (c RGB) RGBA() (r, g, b, a uint32) {
	a = 0xFF << 8

	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8

	return
}
