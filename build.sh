#!/bin/sh

env GOOS=windows GOARCH=amd64 go build -o PCFServiceBindingCrawler.exe main.go
env GOOS=linux GOARCH=amd64 go build -o PCFServiceBindingCrawler.out main.go
