.PHONY: fmt vet test lint build clean

fmt:
	cd GoServer && go fmt ./...

vet:
	cd GoServer && go vet ./...

test:
	cd GoServer && go test ./...

lint:
	cd GoServer && golangci-lint run ./...

build:
	python build.py

clean:
	python build.py --clean
