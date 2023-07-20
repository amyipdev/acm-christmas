package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const numLEDs = 50

var colors [numLEDs]color.RGBA

var cycles = []color.RGBA{
	{10, 150, 204, 255},
	{255, 255, 255, 255},
	{236, 127, 168, 255},
}

func main() {
	machine.GPIO1.Configure(machine.PinConfig{Mode: machine.PinOutput})

	led := ws2812.New(machine.GPIO1)
	var cycle int

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		current := cycles[cycle]
		cycle = (cycle + 1) % len(cycles)

		for i := range colors {
			colors[i] = current
		}
		led.WriteColors(colors[:])
	}
}
