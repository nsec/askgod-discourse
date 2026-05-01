default: linux

linux:
	mkdir -p bin/linux
	GOOS=linux GOARCH=amd64 go get -d -v -x ./cmd/askgod-discourse
	cd bin/linux ; GOOS=linux GOARCH=amd64 go build -tags libsqlite3 ../../cmd/askgod-discourse

update-gomod:
	go get -t -v -u ./...
	go mod tidy --go=1.25.0
	go get toolchain@none

check:
	golangci-lint run
