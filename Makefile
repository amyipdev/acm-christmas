bin: bin/brightest-spot bin/live-capture

bin/brightest-spot: $(shell find cmd/brightest-spot -type f)
	go build -o $@ ./cmd/brightest-spot

bin/live-capture: $(shell find cmd/live-capture -type f)
	go build -o $@ ./cmd/live-capture

bin/live-capture.sh: cmd/live-capture.sh
	cp $< $@
