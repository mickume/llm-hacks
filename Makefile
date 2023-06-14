
.PHONY: all
all: fetch

.PHONY: fetch
fetch:
	cd cmd/fetch && go build -o fetch main.go && mv fetch ../../bin/fetch

.PHONY: crawler
crawler:
	cd cmd/crawler && go build -o crawler main.go && mv crawler ../../bin/crawler