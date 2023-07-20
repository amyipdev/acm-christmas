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

bin/live-capture.sh: cmd/live-capture.sh
	cp $< $@

bin/ffmpeg-bulk.sh: cmd/ffmpeg-bulk.sh
	cp $< $@
