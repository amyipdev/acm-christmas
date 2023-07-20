package main

import (
	"fmt"
	"image/color"
	"strings"
)

type colorFlag color.RGBA

func (c *colorFlag) Set(s string) error {
	if !strings.HasPrefix(s, "#") {
		return fmt.Errorf("invalid color format: %q", s)
	}

	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return fmt.Errorf("invalid color format: %q", s)
	}

	var r, g, b uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return fmt.Errorf("invalid color format: %q", s)
	}

	*c = colorFlag(color.RGBA{R: r, G: g, B: b, A: 0xFF})
	return nil
}

func (c colorFlag) String() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func (c colorFlag) AsColor() color.RGBA {
	return color.RGBA(c)
}

func (c colorFlag) Type() string {
	return "color"
}
