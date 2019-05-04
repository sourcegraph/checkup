#!/bin/bash
#
# Cross-compiles the project for several platforms.
# Outputs to ./builds/
#
# Single-run usage:
#   ./build-cross.sh linux_amd64
#   
# Parallel usage:
#   cat PLATFORMS | xargs -L 1 -P 8 ./build-cross.sh

if [ -z "$1" ]; then
  echo "Error: Specify a platform"
  exit 1
fi

echo Building platform \"$1\" ...

mkdir -p ./builds/
pushd ./cmd/checkup/ > /dev/null 2>&1

OS=$(echo $1 | cut -d'_' -f1)
ARCH=$(echo $1 | cut -d'_' -f2)
GOOS=$OS GOARCH=$ARCH go build -v -ldflags '-s' -o "../../builds/checkup_$1" > /dev/null 2>&1

echo Completed platform \"$1\"
popd > /dev/null 2>&1
