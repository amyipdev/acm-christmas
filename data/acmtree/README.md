# Generating the ACM tree

## Requirements

- Any video camera. I used my Pixel 6a recording at 60fps.
- The setup with the `worm` program loaded in. Prefer 200ms for the delay.

## Steps

### Recording the video

Record the video of the ACM tree while the `worm` program is running.

It is recommended that you record with a stabilizer or tripod. If you don't
have one, you can use a stack of books or something. The less shaky the video,
the better the results will be.

It is also recommended that you record at 60fps. This will give you more frames
to work with and more lee-way when you're trying to find the timestamp of the
first LED turning on.

If you use Android, it is recommended that you use the [Open
Camera][open-camera] app. It has a lot of useful features, including the
ability to tune down the ISO and exposure time, which will reduce the amount of
noise in the video.

[open-camera]: https://opencamera.org.uk/

### Finding the timestamp

Find the timestamp of the frame where the first LED turns on.

You might want to use <kbd>,</kbd> and <kbd>.</kbd> to go frame by frame. Some
video players like `mpv` support this.

It's OK if you're off by a few frames. You can always run the process again.

### Extracting the frames

Run `extract-frames`.

Examples:

```sh
extract-frames --start-time 1.9 path/to/video.mp4
extract-frames --start-time 1:25 path/to/video.mp4 path/to/frames/output
```

Keep `--num-leds` and `--worm-speed` the same as the `worm` program.

### Resizing the frames

Run `ffmpeg-bulk` to resize the images to a reasonable size, e.g. 512px.

Examples:

```sh
ffmpeg-bulk.sh -y -vf scale=512:512:force_original_aspect_ratio=decrease \
    path/to/frames/output \
    path/to/frames/output-small/?.jpg
```

This step is optional, but it will make using the data points much easier.

### Threshold-masking the frames

Run `ffmpeg-bulk` again with `ffutil` to apply a threshold mask to the images.

Examples:

```sh
ffmpeg-bulk.sh -y $(ffutil threshold 288x512 0.8) \
    path/to/frames/output-small \
    path/to/frames/output-threshold/?.jpg
```

If any of the LEDs are not visible in the thresholded images, the next step
will fail. You can try adjusting the threshold value (`0.8` in the example
above).

### Finding the LED positions

Run `big-spot` to find the positions of the LEDs in the thresholded images.
The output will be in a `led-points.csv` file containing `(x, y, area)`.

Examples:

```sh
big-spot -o path/to/frames/output-threshold/data path/to/frames/output-threshold
```

If any of the coordinates are off, you can try running `big-spot` again with
the `--output-png` flag, which will output all chosen spots for each frame in
a PNG file. You can then open the PNG file and see if the spots are correct.

The PNG files are not required for the next step. Only the `led-points.csv`
file is required.
