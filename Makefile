build:
	@go build -o bin/ggpoker

run: build 
	@./bin/ggpoker

test:
	go test -v ./...
