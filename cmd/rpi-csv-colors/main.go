package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/Jon-Bright/ledctl/pixarray"
	"libdb.so/acm-christmas/internal/csvutil"
	"libdb.so/acm-christmas/internal/xcolor"
)

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		log.Println("rpi-csv-colors renders a CSV file of colors to LED lights.")
		log.Println("Usage: rpi-csv-colors <csv-colors-file>")
	}
	flag.Parse()

	csvColorsFile := flag.Arg(0)
	if csvColorsFile == "" {
		flag.Usage()
		os.Exit(2)
	}

	colors, err := csvutil.UnmarshalFile[xcolor.RGB](csvColorsFile)
	if err != nil {
		log.Fatalln("failed to unmarshal CSV colors:", err)
	}

	log.Println("got", len(colors), "LED lights")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	strip, err := pixarray.NewWS281x(
		len(colors),  // LEDs
		3,            // 3 bytes per pixel
		pixarray.RGB, // RGB channel order
		800000,       // 800 KHz
		10,           // DMA 10
		[]int{12},    // GPIO 18
	)
	if err != nil {
		log.Fatalln("failed to create pixarray:", err)
	}

	for i, color := range colors {
		strip.SetPixel(i, pixarray.Pixel{
			R: int(color.R),
			G: int(color.G),
			B: int(color.B),
		})
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := strip.Write(); err != nil {
				log.Fatalln("failed to write pixels:", err)
			}
		}
	}
}
