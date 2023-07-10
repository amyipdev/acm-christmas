package vision

import (
	"errors"
	"image"
	"image/color"

	"github.com/pierrre/imageutil"
)

// BigSpot is a monochromic spot with the largest area in the image.
type BigSpot struct {
	Filled *image.Paletted
	Center image.Point
	Area   int
}

// maxSpots is the maximum number of spots to detect.
const maxSpots = 0xFF - 1

// ErrTooManySpots is returned when there are too many spots in the image. This
// limitation exists because we use a single byte to represent the spot in the
// fill buffer, with each byte representing a spot and 0 reserved for none. This
// means we can only detect 255 spots.
var ErrTooManySpots = errors.New("too many spots (max 255)")

// ErrNoSpots is returned when there are no spots in the image.
var ErrNoSpots = errors.New("no spots found in image")

// FindBiggestSpot finds the spot with the largest area in the image.
//
// The color parameter is the color of the spot. Colors are compared
// exactly, so the color of the spot must match exactly the color
// parameter.
func FindBiggestSpot(img image.Image, matchingColor color.Color) (BigSpot, error) {
	b := newSpotFinder(img)

	spots, err := b.findSpots(matchingColor)
	if err != nil {
		return BigSpot{}, err
	}

	biggest, ok := b.findBiggest(spots)
	if !ok {
		return BigSpot{}, ErrNoSpots
	}

	return biggest, nil
}

type spotFinder struct {
	src        image.Image
	srcAt      imageutil.AtFunc
	size       image.Point
	filled     []uint8
	floodQueue []image.Point
}

func newSpotFinder(src image.Image) *spotFinder {
	w := src.Bounds().Max.X
	h := src.Bounds().Max.Y
	return &spotFinder{
		src:        src,
		srcAt:      imageutil.NewAtFunc(src),
		size:       image.Point{w, h},
		filled:     make([]byte, w*h),
		floodQueue: make([]image.Point, 0, 16),
	}
}

type spotData struct {
	Area int
	ID   byte
}

func (b *spotFinder) findSpots(matchingColor color.Color) ([]spotData, error) {
	spots := make([]spotData, 0, 8)

	for y := 0; y < b.size.Y; y++ {
		stride := y * b.size.X
		for x := 0; x < b.size.X; x++ {
			if b.filled[stride+x] != 0 {
				continue
			}

			if !atPixelEq(b.srcAt, x, y, matchingColor) {
				continue
			}

			if len(spots) >= maxSpots {
				return nil, ErrTooManySpots
			}

			id := byte(len(spots) + 1)

			area := b.fill(x, y, matchingColor, id)
			if area == 0 {
				panic("area is 0")
			}

			spots = append(spots, spotData{
				ID:   id,
				Area: area,
			})
		}
	}

	return spots, nil
}

func (b *spotFinder) findBiggest(spots []spotData) (BigSpot, bool) {
	var biggest spotData
	for _, b := range spots {
		if b.Area > biggest.Area {
			biggest = b
		}
	}

	if biggest.ID == 0 {
		return BigSpot{}, false
	}

	// Find the minimum and maximum x and y values for the spot.
	minX := b.size.X
	minY := b.size.Y
	maxX := 0
	maxY := 0

	w := b.size.X
	h := b.size.Y

	for y := 0; y < h; y++ {
		stride := y * w
		for x := 0; x < w; x++ {
			i := stride + x
			if b.filled[i] != biggest.ID {
				// Set this to 0 for black.
				b.filled[i] = 0
				continue
			}

			// Set this to 1 for the given color.
			// We'll use this with the palette to set the color.
			b.filled[i] = 1

			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	center := image.Point{
		X: (minX + maxX) / 2,
		Y: (minY + maxY) / 2,
	}

	filledImage := &image.Paletted{
		Pix:    b.filled,
		Rect:   b.src.Bounds(),
		Stride: w,
		Palette: []color.Color{
			color.Transparent,
			b.src.At(center.X, center.Y),
		},
	}

	return BigSpot{
		Filled: filledImage,
		Center: center,
		Area:   biggest.Area,
	}, true
}

// fill flood-fills the buffer with 1s where the color matches. It returns
// the number of pixels filled. The filled buffer for this fill is marked
// with the given id. The id must not be 0.
func (b *spotFinder) fill(x, y int, color color.Color, id byte) int {
	queue := b.floodQueue[:0]
	queue = append(queue, image.Point{x, y})

	var count int
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		// Skip if this pixel is out of bounds.
		if !ptInSize(p, b.size) {
			continue
		}

		// Skip if this pixel doesn't match the color.
		if !atPixelEq(b.srcAt, p.X, p.Y, color) {
			continue
		}

		i := b.filledIx(p.X, p.Y)

		// Skip if we've already filled this pixel.
		if b.filled[i] != 0 {
			continue
		}

		count++
		b.filled[i] = id

		queue = append(queue, image.Point{p.X - 1, p.Y})
		queue = append(queue, image.Point{p.X + 1, p.Y})
		queue = append(queue, image.Point{p.X, p.Y - 1})
		queue = append(queue, image.Point{p.X, p.Y + 1})
	}

	return count
}

func (b *spotFinder) filledIx(x, y int) int {
	return y*b.size.X + x
}

func ptInSize(p, size image.Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < size.X && p.Y < size.Y
}

func colorEq(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// atPixelEq is a higher-level version of colorEq that uses an AtFunc.
// This performs much better than using colorEq, since it bypasses Go's
// color.Color conversion which causes heap allocations.
func atPixelEq(at imageutil.AtFunc, x, y int, c color.Color) bool {
	ar, ag, ab, aa := at(x, y)
	br, bg, bb, ba := c.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
