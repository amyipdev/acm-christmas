package ffutil

import (
	"fmt"
	"image"
)

// MakeThreshold makes a threshold filter.
func MakeThreshold(size image.Point, threshold float64) FFmpegArgs {
	// Convert our [0, 1] threshold to a grey color.
	color := int(threshold * 255)
	hex := fmt.Sprintf("#%02x%02x%02x", color, color, color)

	sizeArg := fmt.Sprintf("%dx%d", size.X, size.Y)
	return FFmpegArgs{
		"-f", "lavfi", "-i", fmt.Sprintf("color=%s:s=%s", hex, sizeArg),
		"-f", "lavfi", "-i", fmt.Sprintf("color=black:s=%s", sizeArg),
		"-f", "lavfi", "-i", fmt.Sprintf("color=white:s=%s", sizeArg),
		"-filter_complex", "threshold",
	}
}
