package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	_ "embed"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	_ "image/png"

	"github.com/fogleman/poissondisc"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"libdb.so/acm-christmas/lib/vision"

	_ "golang.org/x/image/bmp"
)

//go:embed description.txt
var description string

var (
	seed    = int64(0)
	numLED  = 50
	ledSize = 5
	outDir  = "."
	csvName = "led-points.csv"
	pngName = "led-points.png"
)

func init() {
	pflag.IntVarP(&numLED, "num-led", "n", numLED, "number of LEDs")
	pflag.IntVarP(&ledSize, "led-size", "l", ledSize, "LED size/radius in px")
	pflag.Int64VarP(&seed, "seed", "s", seed, "random seed")
	pflag.StringVarP(&outDir, "out-dir", "o", outDir, "output directory")
	pflag.StringVar(&csvName, "csv-name", csvName, "LED points CSV file name")
	pflag.StringVar(&pngName, "png-name", pngName, "LED points PNG image name")
}

func main() {
	log.SetFlags(0)

	pflag.Usage = func() {
		log.Println(description)
		log.Println("Usage:")
		log.Println("  random-tree [flags...] <mask-image>")
		log.Println()
		log.Println("Flags:")
		pflag.PrintDefaults()
	}

	pflag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	maskImage, err := decodeImageFile(pflag.Arg(0))
	if err != nil {
		return errors.Wrap(err, "failed to decode mask image")
	}

	boundaryImage := vision.NewBoundaryImage(maskImage, color.White)

	boundarySize := boundaryImage.Count()
	if boundarySize < numLED {
		return fmt.Errorf(
			"boundary size (%d) is less than number of LEDs (%d)",
			boundarySize, numLED)
	}

	// Find the boundary box of the mask image, which is the smallest rectangle
	// that contains all the white pixels in the mask image.
	boundaryBox := boundaryImage.BoundaryBox()

	rand := rand.New(rand.NewSource(seed))

	// Create an image for the Poisson Disk sampling. This image will have the
	// same size as the mask image.
	poissonImage := &image.Paletted{
		Pix:    make([]uint8, maskImage.Bounds().Dx()*maskImage.Bounds().Dy()),
		Stride: maskImage.Bounds().Dx(),
		Rect:   maskImage.Bounds(),
		Palette: []color.Color{
			color.Black,
			color.White,
		},
	}

	// Binary search for the right Poisson Disk parameter that would yield us
	// the exact number of points we need.
	//
	// We'll start by setting the maximum distance between points to the size of
	// the image. This will guarantee that we'll get at least one point. We will
	// then binary search until we get the exact number of points we need. The
	// smallest the distance should be is the size of the LED.
	//
	// The larger the distance, the less points we'll get. The smaller the
	// distance, the more points we'll get. This effectively means the searching
	// array is sorted in descending order.
	poissonRadius := searchFloat(
		float64(ledSize),
		float64(max(maskImage.Bounds().Dx(), maskImage.Bounds().Dy())),
		0.01, // search the radius to the nearest 0.01
		func(r float64) bool {
			k := 10000 // max attempts to add neighbor points

			// Sample the Poisson Disk and draw the poissonPoints on the image.
			poissonPoints := poissondisc.Sample(
				0, 0,
				float64(poissonImage.Bounds().Dx()),
				float64(poissonImage.Bounds().Dy()),
				r, k, rand)

			for _, ppt := range poissonPoints {
				pt := image.Pt(int(math.Round(ppt.X)), int(math.Round(ppt.Y)))
				if !pt.In(boundaryBox) {
					continue
				}
				poissonImage.SetColorIndex(pt.X, pt.Y, 1)
			}

			// Count the number of points on the boundary image.
			var count int
			boundaryImage.EachPt(func(pt image.Point) bool {
				if poissonImage.ColorIndexAt(pt.X, pt.Y) == 1 {
					count++
				}
				return false
			})

			// Zero out the image. This will be replaced with memclr by the
			// compiler which is way faster.
			for i := range poissonImage.Pix {
				poissonImage.Pix[i] = 0
			}

			log.Printf("for radius %f, got %d points", r, count)
			return count <= numLED
		},
	)

	poissonPoints := poissondisc.Sample(
		0, 0,
		float64(poissonImage.Bounds().Dx()),
		float64(poissonImage.Bounds().Dy()),
		float64(poissonRadius), 100, rand)

	poissonMap := make(map[image.Point]struct{}, len(poissonPoints))
	for _, ppt := range poissonPoints {
		pt := image.Pt(int(math.Round(ppt.X)), int(math.Round(ppt.Y)))
		if !pt.In(boundaryBox) {
			continue
		}
		poissonMap[pt] = struct{}{}
	}

	ledPoints := make([]image.Point, 0, numLED)

	boundaryImage.EachPt(func(pt image.Point) bool {
		if _, ok := poissonMap[pt]; ok {
			ledPoints = append(ledPoints, pt)
			eachCirclePx(pt, ledSize, func(pt image.Point) bool {
				poissonImage.SetColorIndex(pt.X, pt.Y, 1)
				return false
			})
		}
		return false
	})

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create output directory")
	}

	if csvName != "" {
		csvPath := filepath.Join(outDir, csvName)
		log.Println("writing CSV file to", csvPath)

		csvFile, err := os.Create(csvPath)
		if err != nil {
			return errors.Wrap(err, "failed to create CSV file")
		}
		defer csvFile.Close()

		csv := csv.NewWriter(csvFile)
		for _, pt := range ledPoints {
			if err := csv.Write([]string{
				strconv.Itoa(pt.X),
				strconv.Itoa(pt.Y),
			}); err != nil {
				return errors.Wrap(err, "failed to write CSV record")
			}
		}

		csv.Flush()
		if err := csv.Error(); err != nil {
			return errors.Wrap(err, "failed to flush CSV writer")
		}

		if err := csvFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close CSV file")
		}
	}

	if pngName != "" {
		pngPath := filepath.Join(outDir, pngName)
		log.Println("writing PNG image to", pngPath)

		pngFile, err := os.Create(pngPath)
		if err != nil {
			return errors.Wrap(err, "failed to create PNG file")
		}
		defer pngFile.Close()

		if err := png.Encode(pngFile, poissonImage); err != nil {
			return errors.Wrap(err, "failed to encode PNG file")
		}

		if err := pngFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close PNG file")
		}
	}

	return nil
}

func decodeImageFile(file string) (image.Image, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode image")
	}

	return img, nil
}

func ptIx(r image.Rectangle, x, y int) int {
	return (y-r.Min.Y)*r.Dx() + (x - r.Min.X)
}

func drawCircle(img draw.Image, center image.Point, radius int, color color.Color) {
	// TODO: generalize this into a circle-filling algorithm so that we can
	// check if circles overlap and try again if they do.

	for x := center.X - radius; x <= center.X+radius; x++ {
		for y := center.Y - radius; y <= center.Y+radius; y++ {
			if (x-center.X)*(x-center.X)+(y-center.Y)*(y-center.Y) <= radius*radius {
				img.Set(x, y, color)
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// eachCirclePx calls fn for each pixel within the circle defined by center and
// radius.
func eachCirclePx(center image.Point, radius int, fn func(image.Point) (stop bool)) {
	// Iterate from the top-left corner of the bounding box of the circle to the
	// bottom-right corner, and check if each point is within the circle.
	//
	// See this ASCII art for an example:
	//
	//    x---o--o---x
	//    |o        o|
	//    o          o
	//    o          o
	//    |o        o|
	//    x---o--o---x
	//

	// subtract 1 from the radius because we want to iterate over the pixels
	// *within* the circle, not the pixels on the edge of the circle.
	radius--

	x0 := center.X - radius
	x1 := center.X + radius
	y0 := center.Y - radius
	y1 := center.Y + radius
	rr := radius * radius

	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			if (x-center.X)*(x-center.X)+(y-center.Y)*(y-center.Y) <= rr {
				if fn(image.Point{X: x, Y: y}) {
					return
				}
			}
		}
	}
}

func searchFloat(min, max, acc float64, fn func(float64) bool) float64 {
	for min < max {
		mid := (min + max) / 2
		if !fn(mid) {
			min = mid + acc
		} else {
			max = mid
		}
	}
	return min
}
