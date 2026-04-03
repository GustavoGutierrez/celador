.PHONY: fmt test vet verify

fmt:
	gofmt -w cmd internal test

test:
	go test ./...

vet:
	go vet ./...

verify: fmt vet test
