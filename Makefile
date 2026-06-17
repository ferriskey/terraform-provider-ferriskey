BINARY  := terraform-provider-ferriskey
VERSION ?= dev

.PHONY: build install test testacc fmt vet lint docs tidy clean

# Build the provider binary.
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

# Install the binary into the Go bin directory for local dev overrides.
install:
	go install -ldflags "-X main.version=$(VERSION)" .

# Unit tests (fast, no network).
test:
	go test ./... -count=1 -timeout 120s

# Acceptance tests: run real CRUD against a live FerrisKey instance. Requires
# TF_ACC=1 and the FERRISKEY_* environment variables (URL, REALM, CLIENT_ID and
# either PASSWORD or CLIENT_SECRET).
testacc:
	TF_ACC=1 go test ./internal/provider/... -count=1 -timeout 30m -v

fmt:
	gofmt -s -w .

vet:
	go vet ./...

# golangci-lint must be installed: https://golangci-lint.run
lint:
	golangci-lint run

# Regenerate documentation from schema + examples.
docs:
	go generate ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
