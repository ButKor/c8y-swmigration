#!/bin/sh

build_bin()
{
    FILENAME=swmigration_$1_$2.$3
    env GOOS=$1 GOARCH=$2 go build -o target/$FILENAME ./src
    chmod +x target/$FILENAME
    echo "Built binary: OS=$1;ARCH=$2;FILENAME=$FILENAME"
}

rm target/*

build_bin darwin amd64 deb
build_bin darwin arm64 deb

build_bin linux arm deb
build_bin linux arm64 deb
build_bin linux 386 deb
build_bin linux amd64 deb

build_bin windows 386 exe
build_bin windows amd64 exe