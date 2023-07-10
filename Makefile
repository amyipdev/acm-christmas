bin: bin/brightest-spot bin/live-capture bin/ffutil bin/live-capture.sh

.PHONY: bin/brightest-spot
bin/brightest-spot:
	go build -o $@ ./cmd/brightest-spot

.PHONY: bin/live-capture
bin/live-capture:
	go build -o $@ ./cmd/live-capture

.PHONY: bin/ffutil
bin/ffutil:
	go build -o $@ ./cmd/ffutil

bin/live-capture.sh: cmd/live-capture.sh
	cp $< $@
