package vision

import (
	"bytes"
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

	fill, err := b.findBiggestSpot(matchingColor)
	if err != nil {
		return BigSpot{}, err
	}

	spot := b.toBigSpot(fill)
	return spot, nil
}

type fillMark = uint8 // make bytes.F happy

const (
	markNotFilled fillMark = 0
	markFilling            = 'f'
	markFilled             = 'F'
	markBiggest            = 'b'
)

type fillMarks []fillMark

func replaceFillMarks(marks []fillMark, old, new fillMark) {
	i := 0
	for i < len(marks) {
		// Do a regular check. This works fine if we have a run of old pixels.
		if marks[i] == old {
			marks[i] = new
			i++
			continue
		}

		// We might have a run of non-old pixels. Skip them.
		next := bytes.IndexByte(marks[i:], old)
		if next == -1 {
			break
		}

		i += next
	}
}

type spotFinder struct {
	src        image.Image
	srcAt      imageutil.AtFunc
	size       image.Point
	filled     []fillMark
	floodQueue []image.Point
}

func newSpotFinder(src image.Image) *spotFinder {
	w := src.Bounds().Max.X
	h := src.Bounds().Max.Y
	return &spotFinder{
		src:        src,
		srcAt:      imageutil.NewAtFunc(src),
		size:       image.Point{w, h},
		filled:     make([]fillMark, w*h),
		floodQueue: make([]image.Point, 0, 16),
	}
}

func (b *spotFinder) Reset(img image.Image) {
	if b.src.Bounds() != img.Bounds() {
		*b = *newSpotFinder(img)
		return
	}

	b.src = img
	b.srcAt = imageutil.NewAtFunc(img)
	for k := range b.filled {
		b.filled[k] = markNotFilled
	}
}

func (b *spotFinder) findBiggestSpot(matchingColor color.Color) (fillResult, error) {
	var biggest fillResult

	for y := 0; y < b.size.Y; y++ {
		for x := 0; x < b.size.X; x++ {
			ptIx := b.filledIx(x, y)
			if b.filled[ptIx] != markNotFilled {
				continue
			}

			if !atPixelEq(b.srcAt, x, y, matchingColor) {
				continue
			}

			fill := b.fill(x, y, matchingColor)
			if fill.Area > biggest.Area {
				biggest = fill
				for i, v := range b.filled {
					switch v {
					case markBiggest:
						// Demote the previous biggest to filled.
						b.filled[i] = markFilled
					case markFilling:
						// Upgrade the filling marks to biggest.
						b.filled[i] = markBiggest
					}
				}
			}
		}
	}

	if biggest.Area == 0 {
		return fillResult{}, ErrNoSpots
	}

	return biggest, nil
}

type fillResult struct {
	Area   int
	Bounds image.Rectangle
}

func (b *spotFinder) toBigSpot(r fillResult) BigSpot {
	center := image.Point{
		X: r.Bounds.Min.X + r.Bounds.Dx()/2,
		Y: r.Bounds.Min.Y + r.Bounds.Dy()/2,
	}

	// Clear everything except for the biggest marks. Set the biggest marks
	// to 1 and everything else to 0.
	for i, v := range b.filled {
		if v == markBiggest {
			b.filled[i] = 1
		} else {
			b.filled[i] = 0
		}
	}

	filledImage := &image.Paletted{
		Pix:    b.filled,
		Rect:   b.src.Bounds(),
		Stride: b.src.Bounds().Dx(),
		Palette: []color.Color{
			color.Transparent,
			b.src.At(center.X, center.Y),
		},
	}

	return BigSpot{
		Filled: filledImage,
		Center: center,
		Area:   r.Area,
	}

}

// fill flood-fills the buffer with 1s where the color matches. It returns
// the number of pixels filled. The filled buffer for this fill is marked
// with the given id. The id must not be 0.
func (b *spotFinder) fill(x, y int, color color.Color) fillResult {
	b.floodQueue = b.floodQueue[:0]
	queue := append(b.floodQueue, image.Point{x, y})

	replaceFillMarks(b.filled, markFilling, markNotFilled)

	var area int
	bounds := image.Rectangle{
		Min: image.Point{x, y},
		Max: image.Point{x + 1, y + 1},
	}

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

		// Skip if we've already filled this pixel.
		ptIx := b.filledIx(p.X, p.Y)
		if b.filled[ptIx] == markFilling {
			continue
		}

		// Mark this pixel as being filled.
		b.filled[ptIx] = markFilling

		bounds = bounds.Union(image.Rectangle{
			Min: p,
			Max: p.Add(image.Point{1, 1}),
		})

		area++
		queue = append(queue, image.Point{p.X - 1, p.Y})
		queue = append(queue, image.Point{p.X + 1, p.Y})
		queue = append(queue, image.Point{p.X, p.Y - 1})
		queue = append(queue, image.Point{p.X, p.Y + 1})
	}

	return fillResult{
		Area:   area,
		Bounds: bounds,
	}
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
