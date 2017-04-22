default: linux windows macos

linux:
	mkdir -p bin/linux
	GOOS=linux GOARCH=amd64 go get -d -v -x ./
	cd bin/linux ; GOOS=linux GOARCH=amd64 go build ../../

windows:
	mkdir -p bin/windows
	GOOS=windows GOARCH=amd64 go get -d -v -x ./
	cd bin/windows ; GOOS=windows GOARCH=amd64 go build ../../

macos:
	mkdir -p bin/macos
	GOOS=macos GOARCH=amd64 go get -d -v -x ./
	cd bin/macos ; GOOS=darwin GOARCH=amd64 go build ../../

check:
	golint ./...
	go vet ./...
	go fmt ./...
	gofmt -s -w ./
