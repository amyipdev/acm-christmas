package xcolor

import (
	"math"

	"libdb.so/acm-christmas/internal/intmath"
)

// AveragingFunc is a function that averages a slice of colors into a single
// color.
type AveragingFunc func([]RGB) RGB

// NewSimpleAveraging creates a new AveragingFunc that averages a slice of
// colors into a single color. It simply averages the red, green, and blue
// values of the colors.
//
// This function can handle about 2.8 million colors before overflowing.
func NewSimpleAveraging() AveragingFunc {
	const maxColors = math.MaxInt32 / (3 * 0xFF)
	return func(colors []RGB) RGB {
		var r, g, b uint32
		for _, c := range colors {
			r += uint32(c.R)
			g += uint32(c.G)
			b += uint32(c.B)
		}
		n := uint32(len(colors))
		return RGB{
			R: uint8(r / n),
			G: uint8(g / n),
			B: uint8(b / n),
		}
	}
}

// NewSquaredAveraging creates a new AveragingFunc that averages a slice of
// colors into a single color. It squares the red, green, and blue values of the
// colors before averaging them to give more weight to brighter colors.
//
// This function can handle about 33 thousand colors before overflowing.
//
// For more information, see:
// https://sighack.com/post/averaging-rgb-colors-the-right-way
func NewSquaredAveraging() AveragingFunc {
	const maxColors = math.MaxInt32 / (3 * 0xFF * 0xFF)
	return func(colors []RGB) RGB {
		var r, g, b uint32
		for _, c := range colors {
			r += uint32(c.R) * uint32(c.R)
			g += uint32(c.G) * uint32(c.G)
			b += uint32(c.B) * uint32(c.B)
		}
		n := uint32(len(colors))
		return RGB{
			R: uint8(intmath.Sqrt32(int32(r / n))),
			G: uint8(intmath.Sqrt32(int32(g / n))),
			B: uint8(intmath.Sqrt32(int32(b / n))),
		}
	}
}
