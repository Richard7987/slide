make:
	go run main.go examples/basics/slides.md

test:
	go test ./... -short

build:
	go build -o slides
