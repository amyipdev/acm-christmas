package leddraw

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"libdb.so/acm-christmas/internal/intmath"
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

// LEDStrip is a strip of LEDs. It is represented as a slice of colors, where
// each color represents the color of an LED.
type LEDStrip []RGB

// SetRGBA sets the RGBA color of the LED at index i. Alpha is ignored.
func (s LEDStrip) SetRGBA(i int, c color.RGBA) {
	s[i] = RGB{c.R, c.G, c.B}
}

// Set sets the color of the LED at index i.
func (s LEDStrip) Set(i int, c color.Color) {
	s.SetRGBA(i, color.RGBAModel.Convert(c).(color.RGBA))
}

// Clear clears the LED strip.
func (s LEDStrip) Clear() {
	// This should be replaced with a memclr by the compiler.
	// On ARM, it does 32 bytes (~10 LEDs) at a time.
	for i := range s {
		s[i] = RGB{}
	}
}

const maxLEDs = math.MaxInt32

// LEDCanvas is a canvas of LED points.
type LEDCanvas struct {
	leds    LEDStrip
	ledData []ledData
	ledRect image.Rectangle

	// pixelData is a precalculated array of pixels. A pixel is deemed relevant
	// if it is within the radius of any LED. The radius is a custom parameter.
	pixelMap   map[image.Point]pixelData
	canvasRect image.Rectangle

	opts LEDCanvasOpts
}

// ledData is the data of an LED. It is specifically crafted to be as small as
// possible.
type ledData struct {
	// neighborPixels is a list of pixels that are within the radius of the LED.
	neighborPixels []pointIntensity
	// neighborColors is a list of colors of the pixels that are within the
	// radius of the LED. It has the threshold already applied to it.
	neighborColors []RGB
}

type pointIntensity struct {
	image.Point
	Intensity float64
}

type pixelData struct {
	neighborLEDs []neighborLED
}

type neighborLED struct {
	// index is the index of the nearest LED to the pixel.
	index int32
	// intensity is multiplied by the color of the pixel to get the color of the
	// LED. LEDs that have multiple pixels within their radius will have their
	// color averaged.
	//
	// TODO: instead of averaging, use the pixel that is closest to the LED.
	// Perhaps consider more complex algorithms, like Lanczos resampling.
	intensity float32
}

func applyIntensity(c RGB, intensity float32) RGB {
	return RGB{
		R: uint8(float32(c.R) * intensity),
		G: uint8(float32(c.G) * intensity),
		B: uint8(float32(c.B) * intensity),
	}
}

// IntensityFunc is a function that calculates the intensity of a pixel based on
// the distance between the pixel and the nearest LED. The intensity is a value
// between 0 and 1, where 0 is the lowest intensity and 1 is the highest
// intensity.
//
// For examples of intensity functions visualized, see
// https://www.desmos.com/calculator/thw9ho0ivd.
type IntensityFunc func(distance float64) float64

// NewLinearIntensity creates a new IntensityFunc that calculates the intensity
// of a pixel based on the distance between the pixel and the nearest LED. The
// intensity is calculated using a linear function.
func NewLinearIntensity(maxDistance float64) IntensityFunc {
	return func(distance float64) float64 {
		return 1 - distance/maxDistance
	}
}

// NewCubicIntensity creates a new IntensityFunc that calculates the intensity
// of a pixel based on the distance between the pixel and the nearest LED. The
// intensity is calculated using a cubic function.
func NewCubicIntensity(maxDistance float64) IntensityFunc {
	return func(distance float64) float64 {
		return 1 - cubicEaseInOut(distance/maxDistance)
	}
}

func cubicEaseInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}

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

// LEDCanvasOpts is a set of options for creating a new LEDCanvas.
type LEDCanvasOpts struct {
	// Intensity is the intensity function used to calculate the intensity of a
	// pixel based on the distance between the pixel and the nearest LED.
	Intensity IntensityFunc
	// Average is the averaging function used to average the colors of the
	// pixels that are within the radius of an LED.
	Average AveragingFunc
	// PPI is the number of pixels per inch of the final LED canvas. The higher
	// the PPI, the higher the resolution of the final LED canvas.
	PPI float64
}

// NewLEDCanvas creates a new LEDCanvas from the given LED positions.
//
// ledPositions is a slice of points, where each point represents the position
// of an LED.
func NewLEDCanvas(ledPositions []image.Point, opts LEDCanvasOpts) (*LEDCanvas, error) {
	if len(ledPositions) > maxLEDs {
		return nil, fmt.Errorf("too many LEDs (%d), max %d", len(ledPositions), maxLEDs)
	}

	if opts.PPI == 0 {
		opts.PPI = 128
	}
	if opts.Intensity == nil {
		opts.Intensity = NewCubicIntensity(2)
	}
	if opts.Average == nil {
		opts.Average = NewSquaredAveraging()
	}

	var ledRect image.Rectangle
	for _, ledPos := range ledPositions {
		ledRect.Min.X = intmath.Min(ledRect.Min.X, ledPos.X)
		ledRect.Min.Y = intmath.Min(ledRect.Min.Y, ledPos.Y)
		ledRect.Max.X = intmath.Max(ledRect.Max.X, ledPos.X)
		ledRect.Max.Y = intmath.Max(ledRect.Max.Y, ledPos.Y)
	}

	aspectRatio := float64(ledRect.Dx()) / float64(ledRect.Dy())
	canvasRect := image.Rect(0, 0, int(opts.PPI*aspectRatio), int(opts.PPI))

	pixelMap := make(map[image.Point]pixelData, int(opts.PPI)*len(ledPositions))
	leds := make([]ledData, len(ledPositions))

	for i, led := range ledPositions {
		nearestPixels := allPixelsWithIntensity(canvasRect, ledRect, led, opts.Intensity, 0.01)

		leds[i] = ledData{
			neighborPixels: nearestPixels,
			neighborColors: make([]RGB, 0, len(nearestPixels)),
		}

		for _, pixel := range nearestPixels {
			data := pixelMap[pixel.Point]
			data.neighborLEDs = append(data.neighborLEDs, neighborLED{
				index:     int32(i),
				intensity: float32(pixel.Intensity),
			})
			pixelMap[pixel.Point] = data
		}
	}

	return &LEDCanvas{
		leds:       make(LEDStrip, len(ledPositions)),
		ledData:    leds,
		ledRect:    ledRect,
		pixelMap:   pixelMap,
		canvasRect: canvasRect,
		opts:       opts,
	}, nil
}

func ptIx(r image.Rectangle, x, y int) int {
	return (y-r.Min.Y)*r.Dx() + (x - r.Min.X)
}

// Bounds returns the bounds of the LED canvas.
func (c *LEDCanvas) Bounds() image.Rectangle {
	return c.canvasRect
}

// Stride returns the stride of the LED canvas.
func (c *LEDCanvas) Stride() int {
	return c.canvasRect.Dx()
}

// Buffer returns the internal buffer of the LED canvas.
func (c *LEDCanvas) Buffer() LEDStrip {
	return c.leds
}

// Clear clears the LED canvas.
func (c *LEDCanvas) Clear() {
	c.leds.Clear()
}

// Render renders the given image to the LED canvas.
func (c *LEDCanvas) Render(src RGBAImage, r image.Rectangle) {
	c.Clear()

	// There are two main ways to render this image:
	//
	// 1. Iterate over all pixels in the image, and for each pixel, check if it
	//    has corresponding LEDs. If it does, calculate the intensity of the
	//    pixel and add it to the LED.
	// 2. Iterate over known pixels that have corresponding LEDs, and for each
	//    pixel, calculate the intensity of the pixel and add it to the LED.

	// TODO: benchmark
	c.render(src, r)
}

func (c *LEDCanvas) render(src RGBAImage, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			data, ok := c.pixelMap[image.Point{X: x, Y: y}]
			if !ok {
				continue
			}

			color := RGBFromRGBA(src.RGBAAt(x, y))

			// Calculate the intensity of the pixel.
			for _, led := range data.neighborLEDs {
				data := &c.ledData[led.index]
				data.neighborColors = append(data.neighborColors, applyIntensity(color, led.intensity))
			}
		}
	}

	// Reset the neighbor colors.
	for i, data := range c.ledData {
		if len(data.neighborColors) == 0 {
			c.leds[i] = RGB{}
		} else {
			c.leds[i] = c.opts.Average(data.neighborColors)
		}
		// clear for next render
		c.ledData[i].neighborColors = c.ledData[i].neighborColors[:0]
	}
}

// allPixelsWithIntensity returns all pixels surrounding the given point that
// has an intensity greater than minIntensity.
func allPixelsWithIntensity(
	canvasRect, ledRect image.Rectangle,
	led image.Point,
	intensityFn IntensityFunc,
	minIntensity float64,
) []pointIntensity {
	// Calculate the pt nearest to the given LED point.
	// This is our "current position".
	pt := image.Point{
		X: int(math.Round(float64(led.X-ledRect.Min.X) / float64(ledRect.Dx()) * float64(canvasRect.Dx()))),
		Y: int(math.Round(float64(led.Y-ledRect.Min.Y) / float64(ledRect.Dy()) * float64(canvasRect.Dy()))),
	}

	// Iterate from the center point and outwards in a spiral.
	// See https://stackoverflow.com/a/3706260/5041327.

	direction := image.Point{X: 1, Y: 0}
	segmentLength := 1
	segmentPassed := 0

	// TODO: guesstimate the capacity
	points := make([]pointIntensity, 0, 0)

	// duds counts how many points have a lower intensity than minIntensity.
	var duds int

	for {
		intensity := intensityFn(distance(pt, led))
		if intensity > minIntensity {
			points = append(points, pointIntensity{
				Point:     pt,
				Intensity: intensity,
			})
		} else {
			duds++
		}

		pt.X += direction.X
		pt.Y += direction.Y
		segmentPassed++

		if segmentPassed == segmentLength {
			if duds == segmentLength {
				// The entire segment is duds, so we're done.
				// Going further out will not yield any more points.
				break
			}

			duds = 0
			segmentPassed = 0

			direction.X, direction.Y = -direction.Y, direction.X // rotate
			if direction.Y == 0 {
				segmentLength++
			}
		}
	}

	return points
}

func distance(pt1, pt2 image.Point) float64 {
	pt1x := float64(pt1.X)
	pt1y := float64(pt1.Y)
	pt2x := float64(pt2.X)
	pt2y := float64(pt2.Y)
	return math.Sqrt(math.Pow(pt1x-pt2x, 2) + math.Pow(pt1y-pt2y, 2))
}

// PtDistance is a pair of points and the distance between them.
type PtDistance struct {
	pt1, pt2 image.Point
	distance float64
}

// FindMinDistance returns the pair of points with the smallest distance
// between them. It runs in O(n^2) time.
func FindMinDistance(points []image.Point) PtDistance {
	if len(points) < 2 {
		return PtDistance{}
	}

	minDistance := math.MaxFloat64
	var minPt1, minPt2 image.Point

	for i, pt1 := range points {
		for _, pt2 := range points[i+1:] {
			if d := distance(pt1, pt2); d < minDistance {
				minDistance = d
				minPt1 = pt1
				minPt2 = pt2
			}
		}
	}

	return PtDistance{
		pt1:      minPt1,
		pt2:      minPt2,
		distance: minDistance,
	}
}
