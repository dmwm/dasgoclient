#GOPATH:=$(PWD):${GOPATH}
#export GOPATH
OS := $(shell uname)
ifeq ($(OS),Darwin)
flags=-ldflags="-s -w"
else
flags=-ldflags="-s -w -extldflags -static"
endif
TAG := $(shell git tag | sort -r | head -n 1)

all: build

build:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient*; CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_all: build_osx build_osx_arm64 build_linux build_power8 build_arm64 build_riscv64 build_windows

build_osx:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_osx; GOARCH=amd64 GOOS=darwin CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_osx

build_osx_arm64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_osx_aarch64; GOARCH=arm64 GOOS=darwin CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_osx_aarch64

build_linux:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_amd64; GOOS=linux GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_amd64

build_power8:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_ppc64le; GOARCH=ppc64le GOOS=linux CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_ppc64le

build_arm64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_aarch64; GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_aarch64

build_riscv64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg dasgoclient_riscv64; GOARCH=riscv64 GOOS=linux CGO_ENABLED=0 go build ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv dasgoclient dasgoclient_riscv64

build_windows:
	go clean; rm -rf pkg dasgoclient.exe; GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	go test
