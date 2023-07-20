package vision

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	_ "embed"

	"github.com/alecthomas/assert/v2"
)

//go:embed dots.png
var dotsPNG []byte

func loadDots(t testing.TB) image.Image {
	t.Helper()

	dots, err := png.Decode(bytes.NewReader(dotsPNG))
	assert.NoError(t, err, "cannot decode dots.png")

	return dots
}

func TestFindBiggestSpot(t *testing.T) {
	t.Run("dots", func(t *testing.T) {
		dots := loadDots(t)

		blob, err := FindBiggestSpot(dots, color.White)
		assert.NoError(t, err, "cannot find biggest blob")

		if testing.Verbose() {
			outFile, err := os.CreateTemp("", "biggest-blob-*.png")
			assert.NoError(t, err, "cannot create temp file")

			defer outFile.Close()

			err = png.Encode(outFile, blob.Filled)
			assert.NoError(t, err, "cannot encode image")

			t.Logf("biggest slot output written to %s", outFile.Name())
			t.Logf("biggest slot: area=%d center=%v", blob.Area, blob.Center)
		}

		blob.Filled = nil // don't compare the image
		assert.Equal(t, BigSpot{
			Center: image.Point{X: 303, Y: 255},
			Area:   53,
		}, blob)
	})
}

func BenchmarkFindBiggestSpot(b *testing.B) {
	dots := loadDots(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		FindBiggestSpot(dots, color.White)
	}
}
