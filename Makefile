.PHONY: fmt vet test lint build clean

fmt:
	cd GoServer && go fmt ./...

vet:
	cd GoServer && go vet ./...

test:
	cd GoServer && go test ./...

lint:
	npm --prefix Web run lint

build:
	npm run build

clean:
	npm run build:clean
