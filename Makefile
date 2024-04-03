build:
	cd cmd/ksef-cli && go build

windows:
	cd cmd/ksef-cli && env GOOS=windows GOARCH=amd64 go build

install:
	cp cmd/ksef-cli/ksef-cli ~/.local/bin/ksef-cli