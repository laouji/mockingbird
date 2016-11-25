#!/bin/sh

GIT_VER=`git describe --tags`
echo "building ${GIT_VER}..."
go build -ldflags "-X main.version=${GIT_VER}"
echo "done"
