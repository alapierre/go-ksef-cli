.PHONY: build windows install release-check release-snapshot

build:
	cd cmd/ksef-cli && go build

install:
	cp cmd/ksef-cli/ksef-cli ~/.local/bin/ksef-cli

release-check:
	go run github.com/goreleaser/goreleaser/v2@latest check --config .goreleaser.yaml

release-snapshot:
	go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish --config .goreleaser.yaml
