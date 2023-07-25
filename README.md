# acm-christmas

Needs Nix. To enter, run `nix-shell`.

## Shopping List

- [ ] [Christmas Tree](https://www.amazon.com/Best-Choice-Products-Artificial-Christmas/dp/B018FDYGVM)
    - Price: $49.99
    - Size: 6ft (~1.8m)
- [x] Raspberry Pi
    - ~~Stolen~~ Borrowed from ACM
- [ ] [LED Strips (12V)](https://www.amazon.com/ALITOVE-LED-Individually-Addressable-Waterproof/dp/B01AG923EU)
    - Actual LED bulbs rather than strips, so better for 2D use.
    - 0.3W per bulb, 50 bulbs per strip, 15W per strip
        - For 5V, 3A per strip
        - For 12V, 1.25A per strip
    - Price:
        - $18.50 (50x, 13ft, 4m)
        - $37.00 (100x, 26ft, 8m)
        - $55.50 (150x, 39ft, 12m)
- [ ] [LED Power Supply (12V)](https://www.amazon.com/ALITOVE-Adapter-Converter-100-240V-5-5x2-1mm/dp/B01GEA8PQA)
    - Price: $11.99
    - 12V 5A, so can power 4 strips or 200 bulbs

## Hardware

### Raspberry Pi

The Raspberry Pi will run the main daemon, which is in charge of:

- Accepting Protobuf commands and handling them,
- Running interpolation algorithms to convert images into LED data
  (if required), and
- Controlling the LEDs.

The Raspberry Pi should run a very minimal Linux distribution, such as [Alpine
Linux](https://alpinelinux.org/) or, even better, some kind of embedded Linux
distribution that supports real-time scheduling.

Ideally, the Go daemon should also be optimized:

- No allocations should be performed during the main loop, and
- It should run with real-time scheduling.

We could leverage Go's pprof to profile the daemon and see where it's spending
most of its time. Some real-world benchmarking will be required.

The LEDs are driven using the Raspberry Pi's DMA capabilities and a PWM pin.
This allows the Pi to achieve the hardware-level timing required to drive the
WS2811 LEDs.

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
