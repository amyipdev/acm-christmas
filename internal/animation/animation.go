package animation

import (
	"context"
	"errors"
	"expvar"
	"time"

	"gopkg.in/typ.v4/lists"
)

// Milliseconds is a duration in milliseconds.
type Milliseconds uint32

// DurationToMs converts a time.Duration to Milliseconds. The duration is
// rounded down to the nearest millisecond.
func DurationToMs(d time.Duration) Milliseconds {
	return Milliseconds(d / time.Millisecond)
}

// Frame is a single frame of an animation.
type Frame[Image any] struct {
	Image          Image
	JumpBackAmount int32
	DurationMs     Milliseconds
}

// Duration returns the duration of the frame as a time.Duration.
func (f Frame[Image]) Duration() time.Duration {
	return time.Duration(f.DurationMs) * time.Millisecond
}

// ErrFramebufferOverflow is returned when the framebuffer is full.
var ErrFramebufferOverflow = errors.New("framebuffer overflow")

const (
	metricDroppedFrames = "dropped_frames"
	metricTotalFrames   = "total_frames"
	metricFrameJitter   = "frame_jitter"
)

var metrics = expvar.NewMap("animation")

// Player is an animation player. It is safe to use from multiple goroutines.
type Player[Image any] struct {
	C <-chan *Frame[Image]

	ch    chan *Frame[Image]
	addCh chan []Frame[Image] // nil clears

	insert   *lists.Ring[Frame[Image]] // points to empty slot next to last frame
	playback *lists.Ring[Frame[Image]] // points to current frame
}

// NewPlayer creates a new animation player that can hold up to maxFrames
// frames.
func NewPlayer[Image any](maxFrames int) *Player[Image] {
	ch := make(chan *Frame[Image])
	frames := lists.NewRing[Frame[Image]](maxFrames)

	return &Player[Image]{
		C:        ch,
		ch:       ch,
		addCh:    make(chan []Frame[Image]),
		insert:   frames,
		playback: frames.Prev(),
	}
}

// AddFrames adds frames to the animation. If the player is full, the oldest
// frames will be overwritten.
func (p *Player[Image]) AddFrames(frames []Frame[Image]) {
	p.addCh <- frames
}

// ClearFrames clears all frames from the animation.
func (p *Player[Image]) ClearFrames() {
	p.addCh <- nil
}

// Play starts playing the animation. Play returns when the animation is
// finished or when the context is canceled.
func (p *Player[Image]) Play(ctx context.Context) error {
	var frameCh chan *Frame[Image]

	var currentFrame *Frame[Image]
	var nextFrame *Frame[Image]

	nextFrameTimer := time.NewTimer(0)
	nextFrameTimer.Stop()
	defer nextFrameTimer.Stop()

	scheduleNextFrame := func() {
		f, ok := p.nextFrame()
		if ok {
			nextFrameTimer.Reset(f.Duration())
			nextFrame = f
		} else {
			nextFrameTimer.Stop()
			nextFrame = nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case frames := <-p.addCh:
			if frames == nil {
				p.clearFrames()
				break
			}

			if !p.addFrames(frames) {
				return ErrFramebufferOverflow
			}

			if currentFrame == nil {
				// No frame is currently being played, so start playing the
				// first frame.
				scheduleNextFrame()
			}

		case <-nextFrameTimer.C:
			if currentFrame != nil {
				// Timer for next frame fired, but the previous frame hasn't
				// been sent yet. This means that the receiver is too slow.
				metrics.Add(metricDroppedFrames, 1)
			}

			currentFrame = nextFrame
			// A nil frame means that we ran out of frames, so don't send any
			// more.
			if currentFrame != nil {
				frameCh = p.ch
				// Advancing the frame here instead of waiting for the receiver
				// to pick up the frame. This ensures that the animation is
				// played at the correct speed even if the receiver is slow.
				scheduleNextFrame()
			}

		case frameCh <- currentFrame:
			frameCh = nil
			currentFrame = nil

			metrics.Add(metricTotalFrames, 1)
		}
	}
}

// addFrame adds a frame to the player. If the player is already full, false is
// returned and the player halts.
func (p *Player[Image]) addFrame(f Frame[Image]) bool {
	p.insert.Value = f
	p.insert = p.insert.Next()
	return p.insert != p.playback
}

func (p *Player[Image]) addFrames(frames []Frame[Image]) bool {
	for _, f := range frames {
		if !p.addFrame(f) {
			return false
		}
	}
	return true
}

func (p *Player[Image]) clearFrames() {
	p.playback = p.insert.Prev()
}

// nextFrame returns the next frame in the animation. False is returned if the
// animation is finished.
func (p *Player[Image]) nextFrame() (*Frame[Image], bool) {
	current := p.playback

	if current.Value.JumpBackAmount > 0 {
		jumpBackAmount := current.Value.JumpBackAmount
		for i := int32(0); i < jumpBackAmount; i++ {
			current = current.Prev()
			if current == p.insert {
				return nil, false
			}
		}
	} else {
		current = current.Next()
		if current == p.insert {
			return nil, false
		}
	}

	p.playback = current
	return &current.Value, true
}
