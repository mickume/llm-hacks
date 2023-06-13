
.PHONY: cli
cli:
	cd cmd/ao3 && go build -o aoc main.go && mv aoc ${GOPATH}/bin/aoc
