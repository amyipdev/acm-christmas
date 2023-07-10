# acm-christmas

Needs Nix. To enter, run `nix-shell`.

## Shopping List

- [ ] [Christmas Tree](https://www.amazon.com/Best-Choice-Products-Artificial-Christmas/dp/B018FDYGVM)
    - Price: $49.99
    - Size: 6ft (~1.8m)
- [x] Raspberry Pi
    - ~~Stolen~~ Borrowed from ACM
- [ ] [LED strips](https://www.amazon.com/ALITOVE-LED-Individually-Addressable-Waterproof/dp/B01AG923GI)
    - Actual LED bulbs rather than strips, so better for 2D use.
    - Price:
        - $15.50 (50x, 13ft, 4m)
        - $31.00 (100x, 26ft, 8m)
        - $46.50 (150x, 39ft, 12m)

## Tools

**You must run `make`** before running any of the tools.

### live-capture

Starts an FFmpeg daemon that keeps an up-to-date BMP image that is the
current frame of the given webcam.

Before running `live-capture`, you must first edit `camerarc` to specify the
webcam you want to use. You can find the name of your webcam by running
`v4l2-ctl --list-devices`.

Then, to start capturing, run:

```sh
live-capture start
```

To start capturing with a black-and-white threshold filter:

```sh
live-capture start --filter-args "$(ffutil threshold 640x480 0.5)"
```

To view a snapshot of the current frame:

```sh
live-capture view
```

To take a snapshot of the current frame onto a PNG file:

```sh
live-capture snapshot /tmp/snapshot.png
```

> **Note**: If the snapshot is also a BMP file, no conversion is performed, so
> the snapshot will be very fast.

### ledd

The actual daemon that controls the LEDs. It exposes an HTTP server that can be
invoked to set the color of a given LED as well as perform various other
higher-level tasks.

See [proto/ledd.proto](proto/ledd.proto) for the full API.

TODO: implement
