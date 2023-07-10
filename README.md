# acm-christmas

Needs Nix. To enter, run `nix-shell`.

## live-capture

Starts an FFmpeg daemon that keeps an up-to-date BMP image that is the
current frame of the given webcam.

To start capturing:

```sh
live-capture start /dev/video1 /run/user/1000/camera.bmp
```

Using `/run/user/1000` is recommended, since on most systems it is a
tmpfs mount, so the image will be stored in RAM.

To view a snapshot of the current frame:

```sh
live-capture view /run/user/1000/camera.bmp
```

To take a snapshot of the current frame onto a PNG file:

```sh
live-capture snapshot /run/user/1000/camera.bmp /tmp/snapshot.png
```

Note that if the snapshot is also a BMP file, no conversion is
performed, so the snapshot will be very fast.
