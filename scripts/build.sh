#!/bin/bash

SRC_DIR="${PWD}"
BUILD_DIR="${PWD}"
DIST="${BUILD_DIR}/dist"
CMDDIRS="gosudachicli dicbuilder userdicbuilder printdic printdicheader dicconv"

build() {
    cd "${SRC_DIR}/$1"
    echo -n "Building $1..."
    go build -o "${DIST}/$1"
    echo "done"
    cd "${BUILD_DIR}"
}

assets() {
    cd "${SRC_DIR}/data"
    go generate
    cd "${BUILD_DIR}"
}

assets

if [ ! -d "${DIST}" ]; then
    mkdir "${DIST}"
fi

for f in ${CMDDIRS}; do
    build "${f}"
done
