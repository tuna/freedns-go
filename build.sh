#! /bin/bash
env GOOS=linux GOARCH=amd64 go build -o freedns-go-linux-amd64

env GOOS=linux GOARCH=arm64 go build -o freedns-go-linux-arm64
env GOOS=linux GOARCH=arm go build -o freedns-go-linux-arm

env GOOS=linux GOARCH=mips go build -o freedns-go-linux-mips
env GOOS=linux GOARCH=mipsle go build -o freedns-go-linux-mipsle
env GOOS=linux GOARCH=mips64 go build -o freedns-go-linux-mips64
env GOOS=linux GOARCH=mips64le go build -o freedns-go-linux-mips64le

env GOOS=darwin GOARCH=amd64 go build -o freedns-go-macos-amd64

