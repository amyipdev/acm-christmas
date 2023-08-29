package leddraw

import "math"

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
		if distance > maxDistance {
			return 0
		}
		return 1 - distance/maxDistance
	}
}

// NewCubicIntensity creates a new IntensityFunc that calculates the intensity
// of a pixel based on the distance between the pixel and the nearest LED. The
// intensity is calculated using a cubic function.
func NewCubicIntensity(maxDistance float64) IntensityFunc {
	return func(distance float64) float64 {
		if distance > maxDistance {
			return 0
		}
		return 1 - cubicEaseInOut(distance/maxDistance)
	}
}

// NewStepIntensity creates a new IntensityFunc that calculates the intensity
// of a pixel based on the distance between the pixel and the nearest LED. The
// intensity is 1 if the distance is less than the max distance, and 0
// otherwise.
func NewStepIntensity(maxDistance float64) IntensityFunc {
	return func(distance float64) float64 {
		if distance > maxDistance {
			return 0
		}
		return 1
	}
}

func cubicEaseInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}
