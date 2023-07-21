package main

import (
	"image/color"
	"machine"
	"runtime/interrupt"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const wormSpeed = 200 * time.Millisecond

var colorOn = color.RGBA{255, 255, 255, 0}
var ledOrder = []int{
	0,
	1,
	49,
	2,
	3,
	4,
	48,
	46,
	43,
	5,
	47,
	44,
	45,
	6,
	42,
	7,
	41,
	8,
	9,
	40,
	39,
	10,
	38,
	11,
	12,
	37,
	36,
	35,
	13,
	14,
	34,
	15,
	33,
	16,
	32,
	31,
	17,
	18,
	30,
	29,
	19,
	20,
	21,
	28,
	27,
	22,
	23,
	26,
	25,
	24,
}

func main() {
	machine.GPIO1.Configure(machine.PinConfig{Mode: machine.PinOutput})

	led := ws2812.New(machine.GPIO1)
	colors := make([]color.RGBA, len(ledOrder))

	ticker := time.NewTicker(wormSpeed)
	defer ticker.Stop()

	var i int
	for range ticker.C {
		colors[ledOrder[i]] = color.RGBA{0, 0, 0, 0}

		i = (i + 1) % len(ledOrder)
		colors[ledOrder[i]] = colorOn

		// Update the LED strip.
		state := interrupt.Disable()
		led.WriteColors(colors)
		interrupt.Restore(state)
	}
}
