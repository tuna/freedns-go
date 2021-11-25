all: build_all

build_all:
	mkdir -p ./build
	env GOOS=linux GOARCH=amd64    go build -o ./build/freedns-go-linux-amd64
	env GOOS=linux GOARCH=arm64    go build -o ./build/freedns-go-linux-arm64
	env GOOS=linux GOARCH=arm      go build -o ./build/freedns-go-linux-arm
	env GOOS=linux GOARCH=mips     go build -o ./build/freedns-go-linux-mips
	env GOOS=linux GOARCH=mipsle   go build -o ./build/freedns-go-linux-mipsle
	env GOOS=linux GOARCH=mips64   go build -o ./build/freedns-go-linux-mips64
	env GOOS=linux GOARCH=mips64le go build -o ./build/freedns-go-linux-mips64le
	env GOOS=darwin GOARCH=amd64   go build -o ./build/freedns-go-macos-amd64
	env GOOS=darwin GOARCH=arm64   go build -o ./build/freedns-go-macos-arm64

update_db:
	curl -s 'https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt' > chinaip/china.txt

test:
	go test ./chinaip
	go test ./freedns

.PHONY: build_all update_db test