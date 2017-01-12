GOPATH:=$(PWD):${GOPATH}
export GOPATH

all: build

build:
	go clean; rm -rf pkg; go build

build_all: build_osx build_linux build

build_osx:
	go clean; rm -rf pkg dasgoclient_osx; GOOS=darwin go build
	mv dasgoclient dasgoclient_osx

build_linux:
	go clean; rm -rf pkg dasgoclient_linux; GOOS=linux go build
	mv dasgoclient dasgoclient_linux

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	cd test; go test
