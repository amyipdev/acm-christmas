package main

import (
	"image/color"
	"machine"
	"runtime/interrupt"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const numLEDs = 50

const wormLength = 1
const wormSpeed = 200 * time.Millisecond

var colorOn = color.RGBA{255, 255, 255, 0}

func main() {
	machine.GPIO1.Configure(machine.PinConfig{Mode: machine.PinOutput})

	led := ws2812.New(machine.GPIO1)
	colors := make([]color.RGBA, numLEDs)

	var tail int // worm tail position

	ticker := time.NewTicker(wormSpeed)
	defer ticker.Stop()

	writeTicker := time.NewTicker(10 * time.Millisecond)
	defer writeTicker.Stop()

	for {
		select {
		case <-ticker.C:
			// Turn off the tail LED bulb.
			colors[tail] = color.RGBA{0, 0, 0, 0}

			// Move the worm tail forward.
			tail = (tail + 1) % numLEDs

			// Turn on the head LED bulb.
			head := (tail + wormLength - 1) % numLEDs
			colors[head] = colorOn

		case <-writeTicker.C:
		}

		// Update the LED strip.
		state := interrupt.Disable()
		led.WriteColors(colors)
		interrupt.Restore(state)
	}
}
