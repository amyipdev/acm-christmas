package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/spf13/pflag"
	"libdb.so/acm-christmas/internal/xdraw"
	"libdb.so/acm-christmas/lib/leddraw"

	_ "golang.org/x/image/bmp"
)

// TODO: make this take in an image and draws this image onto a bunch of LED
// points.

var (
	ledPointsFile = "led-points.csv"
	outFile       = "led-image.png"
	maxPtDistance = 0.0 // auto
	ppi           = 72.0
)

func init() {
	pflag.StringVarP(&ledPointsFile, "led-points", "i", ledPointsFile, "path to the CSV file containing the LED points")
	pflag.StringVarP(&outFile, "out", "o", outFile, "path to the output PNG file")
	pflag.Float64Var(&maxPtDistance, "max-distance", maxPtDistance, "maximum distance between a point and an LED")
	pflag.Float64Var(&ppi, "ppi", ppi, "pixels per inch")
}

func main() {
	pflag.Parse()

	if !strings.HasSuffix(outFile, ".png") {
		log.Fatalln("output file must be a PNG")
	}

	img, err := decodeImageFile(pflag.Arg(0))
	if err != nil {
		log.Fatalln("failed to decode source image:", err)
	}

	ledPoints, err := readCSVPoints(ledPointsFile)
	if err != nil {
		log.Fatalln("failed to read LED points:", err)
	}

	var ledCanvasOpts leddraw.LEDCanvasOpts
	ledCanvasOpts.PPI = ppi
	if maxPtDistance > 0 {
		ledCanvasOpts.Intensity = leddraw.NewCubicIntensity(maxPtDistance)
	}

	ledCanvas, err := leddraw.NewLEDCanvas(ledPoints, ledCanvasOpts)
	if err != nil {
		log.Fatalln("failed to create LED canvas:", err)
	}

	// TODO: scale while preserving aspect ratio, and center the image.
	canvasBounds := ledCanvas.CanvasBounds()

	var imagedCanvas *image.NRGBA
	imagedCanvas = imaging.Fit(img, canvasBounds.Dx(), canvasBounds.Dy(), imaging.NearestNeighbor)
	imagedCanvas = imaging.PasteCenter(image.NewNRGBA(canvasBounds), imagedCanvas)

	start := time.Now()
	if err := ledCanvas.Render((*image.RGBA)(imagedCanvas)); err != nil {
		log.Fatalln("failed to render image:", err)
	}
	log.Println("rendered in", time.Since(start))

	ledBounds := ledCanvas.LEDBounds()

	outImage := image.NewRGBA(ledCanvas.LEDBounds())
	draw.Draw(outImage, outImage.Bounds(), image.NewUniform(color.Black), image.ZP, draw.Src)

	for i, led := range ledCanvas.LEDs() {
		ledPt := ledPoints[i].Sub(ledBounds.Min)
		xdraw.DrawCircle(outImage, ledPt, 3, led)
	}

	out, err := os.Create(outFile)
	if err != nil {
		log.Fatalln("failed to open output file:", err)
	}
	defer out.Close()

	if err := png.Encode(out, outImage); err != nil {
		log.Fatalln("failed to encode PNG:", err)
	}

	if err := out.Close(); err != nil {
		log.Fatalln("failed to close output file:", err)
	}

	log.Println("wrote output to", outFile)
}

func readCSVPoints(csvPath string) ([]image.Point, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}

	csvr := csv.NewReader(bufio.NewReader(f))
	var points []image.Point

	for {
		record, err := csvr.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(record) != 2 {
			return nil, fmt.Errorf("expected x,y point, got %v", record)
		}

		var p image.Point

		p.X, err = strconv.Atoi(record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse x point %q: %v", record[0], err)
		}

		p.Y, err = strconv.Atoi(record[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse y point %q: %v", record[1], err)
		}

		points = append(points, p)
	}

	return points, nil
}

func decodeImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}
