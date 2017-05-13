default: linux windows macos

linux:
	mkdir -p bin/linux
	GOOS=linux GOARCH=amd64 go get -d -v -x ./cmd/askgod-discourse
	cd bin/linux ; GOOS=linux GOARCH=amd64 go build ../../cmd/askgod-discourse

windows:
	mkdir -p bin/windows
	GOOS=windows GOARCH=amd64 go get -d -v -x ./cmd/askgod-discourse
	cd bin/windows ; GOOS=windows GOARCH=amd64 go build ../../cmd/askgod-discourse

macos:
	mkdir -p bin/macos
	GOOS=macos GOARCH=amd64 go get -d -v -x ./cmd/askgod-discourse
	cd bin/macos ; GOOS=darwin GOARCH=amd64 go build ../../cmd/askgod-discourse

check:
	golint ./...
	go vet ./...
	go fmt ./...
	gofmt -s -w ./
