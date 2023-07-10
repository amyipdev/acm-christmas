// Package ffutil provides utilities for ffmpeg.
package ffutil

import "strings"

// FFmpegArgs is a type describing the list of arguments to pass to ffmpeg.
type FFmpegArgs []string

func (a FFmpegArgs) String() string {
	return strings.Join(a, " ")
}
