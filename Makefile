bin: $(patsubst cmd/%,bin/%,$(wildcard cmd/*))

.PHONY: bin/live-capture
bin/live-capture:
	go build -o $@ ./cmd/live-capture

.PHONY: bin/ffutil
bin/ffutil:
	go build -o $@ ./cmd/ffutil

.PHONY: bin/random-tree
bin/random-tree:
	go build -o $@ ./cmd/random-tree

.PHONY: bin/tree-canvas
bin/tree-canvas:
	go build -o $@ ./cmd/tree-canvas

.PHONY: bin/extract-frames
bin/extract-frames:
	go build -o $@ ./cmd/extract-frames

.PHONY: bin/big-spot
bin/big-spot:
	go build -o $@ ./cmd/big-spot

.PHONY: bin/generate-patterns
bin/generate-patterns:
	go build -o $@ ./cmd/generate-patterns

.PHONY: bin/rpi-scanup
bin/rpi-scanup:
	GOOS=linux GOARCH=arm go build -o $@ ./cmd/rpi-scanup

.PHONY: bin/rpi-worm
bin/rpi-worm:
	GOOS=linux GOARCH=arm go build -o $@ ./cmd/rpi-worm

.PHONY: bin/rpi-csv-colors
bin/rpi-csv-colors:
	GOOS=linux GOARCH=arm go build -o $@ ./cmd/rpi-csv-colors

bin/ffmpeg-bulk: cmd/ffmpeg-bulk
	cp $< $@

bin/move-to-pi: cmd/move-to-pi
	cp $< $@

bin/prep-pi: cmd/prep-pi
	cp $< $@
