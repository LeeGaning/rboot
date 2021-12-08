export PATH := $(GOPATH)/bin:$(PATH)
export GO15VENDOREXPERIMENT := 1

all: fmt build

build:
	go build -ldflags "-s -w" -o robot.exe ./
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o robot ./

fmt:
	go fmt ./...
	
