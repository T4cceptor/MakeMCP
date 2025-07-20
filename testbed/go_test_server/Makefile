.PHONY: install run build clean

install:
	go mod tidy
	go mod download

build:
	go build -o bin/server .

run:
	go run .

clean:
	rm -rf bin/
	rm -f test.db
