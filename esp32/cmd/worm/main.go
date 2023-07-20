package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const numLEDs = 50

const wormLength = 3
const wormSpeed = 100 * time.Millisecond

func main() {
	machine.GPIO1.Configure(machine.PinConfig{Mode: machine.PinOutput})

	led := ws2812.New(machine.GPIO1)
	colors := make([]color.RGBA, numLEDs)

	var tail int // worm tail position

	ticker := time.NewTicker(wormSpeed)
	defer ticker.Stop()

	for range ticker.C {
		// Turn off the tail LED bulb.
		colors[tail] = color.RGBA{0, 0, 0, 0}

		// Move the worm tail forward.
		tail = (tail + 1) % numLEDs

		// Turn on the head LED bulb.
		head := (tail + wormLength - 1) % numLEDs
		colors[head] = color.RGBA{255, 0, 0, 0}

		// Update the LED strip.
		led.WriteColors(colors)
	}
}
