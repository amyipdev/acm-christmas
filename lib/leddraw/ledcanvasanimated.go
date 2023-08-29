package leddraw

import (
	"context"
	"image"

	"golang.org/x/sync/errgroup"
	"libdb.so/acm-christmas/internal/animation"
	"libdb.so/acm-christmas/internal/concqueue"
)

// LEDCanvas wraps an LEDCanvas and provides animation capabilities.
// Frames are sent to the C channel.
type LEDCanvasAnimated struct {
	C <-chan animation.Frame[LEDStrip]

	renderer *concqueue.Queue[animation.Frame[*image.RGBA], animation.Frame[LEDStrip]]
	player   *animation.Player[LEDStrip]
	opts     LEDCanvasOpts
}

// NewLEDCanvasAnimated creates a new LEDCanvasAnimated.
func NewLEDCanvasAnimated(ledPositions []image.Point, opts LEDCanvasOpts) (*LEDCanvasAnimated, error) {
	canvas, err := NewLEDCanvas(ledPositions, opts)
	if err != nil {
		return nil, err
	}

	renderFunc := func(ctx context.Context, frame animation.Frame[*image.RGBA]) (animation.Frame[LEDStrip], error) {
		if err := canvas.Render(frame.Image); err != nil {
			return animation.Frame[LEDStrip]{}, err
		}
		return animation.Frame[LEDStrip]{
			Image:          append(LEDStrip(nil), canvas.LEDs()...),
			JumpBackAmount: frame.JumpBackAmount,
			DurationMs:     frame.DurationMs,
		}, nil
	}

	c := &LEDCanvasAnimated{
		renderer: concqueue.NewQueue(renderFunc),
		player:   animation.NewPlayer[LEDStrip](),
		opts:     opts,
	}
	c.C = c.player.C
	return c, nil
}

// Run starts the animated canvas player.
func (c *LEDCanvasAnimated) Run(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error { return c.player.Run(ctx) })
	errg.Go(func() error { return c.renderer.Run(ctx) })
	errg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame := <-c.renderer.Out:
				if err := c.player.AddFrame(ctx, frame.Value()); err != nil {
					return err
				}
			}
		}
	})
	return errg.Wait()
}

// AddFrames adds frames to the animated canvas.
func (c *LEDCanvasAnimated) AddFrames(ctx context.Context, images []animation.Frame[*image.RGBA]) error {
	return c.renderer.EnqueueList(ctx, images)
}

type ledCanvasQueue struct {
	frames chan []*image.RGBA
}
