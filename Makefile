bin: bin/*

.PHONY: bin/live-capture
bin/live-capture:
	go build -o $@ ./cmd/live-capture

.PHONY: bin/ffutil
bin/ffutil:
	go build -o $@ ./cmd/ffutil

.PHONY: bin/random-tree
bin/random-tree:
	go build -o $@ ./cmd/random-tree

bin/live-capture.sh: cmd/live-capture.sh
	cp $< $@
