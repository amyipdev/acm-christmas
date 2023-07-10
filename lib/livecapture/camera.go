// Package livecapture provides a way to capture images from a live camera.
// It is a port of the live-capture script in cmd/.
package livecapture

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/image/bmp"
)

// ImageFormat describes the format of an image.
type ImageFormat string

const (
	FormatBMP ImageFormat = "bmp"
	FormatJPG ImageFormat = "jpg"
	FormatPNG ImageFormat = "png"
)

// SupportedImageFormats is the list of supported image formats.
var SupportedImageFormats = []ImageFormat{
	FormatBMP,
	FormatJPG,
	FormatPNG,
}

// Camera describes the options for a camera.
type Camera struct {
	// Path is the path to the camera device.
	Path string
	// Size is the size of the camera image.
	Size image.Point
	// Format is the format of the camera image.
	Format CameraFormat
	// FrameRate is the frame rate of the camera.
	FrameRate int
}

// CameraFormat describes the format of a camera image.
type CameraFormat string

const (
	FormatYUYV422 CameraFormat = "yuyv422"
	FormatMJPG    CameraFormat = "mjpg"
)

// SupportedCameraFormats is the list of supported camera formats.
var SupportedCameraFormats = []CameraFormat{
	FormatYUYV422,
	FormatMJPG,
}

// IsSupported returns whether the given string is a supported camera format.
func IsSupported[T ~string](s T, supported []T) bool {
	for _, v := range supported {
		if s == v {
			return true
		}
	}
	return false
}

// CaptureOpts describes the options for a live-captured camera.
type CaptureOpts struct {
	// Camera is the camera to capture from.
	Camera
	// ImagePath is the path to the image captured by the camera.
	// It will be continuously updated as the camera captures images.
	// It is recommended to use a BMP file in /run/user/1000 so that there's
	// minimum overhead in writing and reading the image.
	ImagePath string
	// FilterArgs are the arguments to pass to FFmpeg to filter the camera
	// image.
	FilterArgs []string
	// Image2Args are the arguments to pass to FFmpeg to add arguments to the
	// written image.
	Image2Args []string
}

// Capture describes a live-captured camera.
type Capture struct {
	opts        CaptureOpts
	imageFormat ImageFormat

	snapshotBufsz   atomic.Int64
	snapshotBufPool sync.Pool
}

// NewCapture creates a new stopped camera.
func NewCapture(opts CaptureOpts) (*Capture, error) {
	var imageFormat ImageFormat
	switch ext := filepath.Ext(opts.ImagePath); ext {
	case ".bmp":
		imageFormat = FormatBMP
	case ".jpg":
		imageFormat = FormatJPG
	case ".png":
		imageFormat = FormatPNG
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	if !IsSupported(opts.Format, SupportedCameraFormats) {
		return nil, fmt.Errorf("unsupported camera format: %s", opts.Format)
	}

	c := Capture{
		opts:        opts,
		imageFormat: imageFormat,
	}

	c.snapshotBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, c.snapshotBufsz.Load())
		},
	}

	return &c, nil
}

// Start starts the camera. It will block until the given context is canceled.
func (c *Capture) Start(ctx context.Context) error {
	ffmpegArgs := []string{
		"ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "video4linux2",
		"-framerate", strconv.Itoa(c.opts.FrameRate),
		"-video_size", strconv.Itoa(c.opts.Size.X) + "x" + strconv.Itoa(c.opts.Size.Y),
		"-pixel_format", string(c.opts.Format),
		"-i", c.opts.Path,
	}
	ffmpegArgs = append(ffmpegArgs, c.opts.FilterArgs...)
	ffmpegArgs = append(ffmpegArgs,
		"-f", "image2",
		"-update", "1",
	)
	ffmpegArgs = append(ffmpegArgs, c.opts.Image2Args...)
	ffmpegArgs = append(ffmpegArgs, c.opts.ImagePath)

	cmd := exec.CommandContext(ctx, ffmpegArgs[0], ffmpegArgs[1:]...)
	if err := cmd.Run(); err != nil {
		return wrapProcessError(cmd, err)
	}

	if err := os.Remove(c.opts.ImagePath); err != nil {
		return errors.Wrap(err, "failed to remove image file")
	}

	return nil
}

// WaitForFile waits for the image file to be created.
func (c *Capture) WaitForFile(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, err := os.Stat(c.opts.ImagePath); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// SnapshotToFile takes a snapshot from the camera and writes it to the given
// file.
func (c *Capture) SnapshotToFile(ctx context.Context, dst string) error {
	if filepath.Ext(dst) == filepath.Ext(c.opts.ImagePath) {
		return copyFile(c.opts.ImagePath, dst)
	}

	cmd := exec.CommandContext(ctx,
		"ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-i", c.opts.ImagePath,
		dst)
	if err := cmd.Run(); err != nil {
		return wrapProcessError(cmd, err)
	}

	return nil
}

// Snapshot takes a snapshot from the camera and returns it.
// It is safe to call this method concurrently.
func (c *Capture) Snapshot(ctx context.Context) (image.Image, error) {
	snapshotBufsz := c.snapshotBufsz.Load()
	if snapshotBufsz == 0 {
		s, err := os.Stat(c.opts.ImagePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to stat image file")
		}
		snapshotBufsz = s.Size()
		c.snapshotBufsz.Store(snapshotBufsz)
	}

	f, err := os.Open(c.opts.ImagePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open image file")
	}
	defer f.Close()

	buf := c.snapshotBufPool.Get().([]byte)
	defer c.snapshotBufPool.Put(buf)

	if len(buf) < int(snapshotBufsz) {
		return nil, fmt.Errorf(
			"BUG: snapshot buffer size is too small: %d < %d",
			len(buf), snapshotBufsz)
	}

	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, errors.Wrap(err, "failed to read image file")
	}

	reader := bytes.NewReader(buf)

	var img image.Image
	switch c.imageFormat {
	case FormatBMP:
		img, err = bmp.Decode(reader)
	case FormatJPG:
		img, err = jpeg.Decode(reader)
	case FormatPNG:
		img, err = png.Decode(reader)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", c.imageFormat)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to decode image file")
	}

	return img, nil
}

// View opens the camera image in an image viewer.
func (c *Capture) View(ctx context.Context) error {
	imageName, _, _ := strings.Cut(filepath.Base(c.opts.ImagePath), ".")
	imageExt := filepath.Ext(c.opts.ImagePath)

	tmpdst := filepath.Join(os.TempDir(), imageName+".snapshot"+imageExt)
	if err := c.SnapshotToFile(ctx, tmpdst); err != nil {
		return errors.Wrap(err, "failed to take snapshot")
	}

	cmd := exec.CommandContext(ctx, "xdg-open", tmpdst)
	if err := cmd.Run(); err != nil {
		return wrapProcessError(cmd, err)
	}

	return nil
}

func wrapProcessError(cmd *exec.Cmd, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Stderr != nil {
			return fmt.Errorf(
				"%s exited with status %d: %s",
				cmd.Args[0], exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return fmt.Errorf(
			"%s exited with status %d",
			cmd.Args[0], exitErr.ExitCode())
	}
	return err
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "failed to open source file")
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "failed to create destination file")
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrap(err, "failed to copy file")
	}

	return nil
}
