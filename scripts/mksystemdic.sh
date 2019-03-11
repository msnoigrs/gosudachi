#!/bin/bash

PROXY=""
MATRIXDEF="./matrix.def"
UNIDICVER="2.1.2"
UNIDICZIP="unidic-mecab-${UNIDICVER}_src.zip"
UNIDICURL="https://unidic.ninjal.ac.jp/unidic_archive/cwj/${UNIDICVER}/${UNIDICZIP}"
SYSDIC="system_core.dic"
SYSFULLDIC="system_full.dic"
CORECSV="core_lex.csv"
NOTCORECSV="notcore_lex.csv"

if [ ! -f "${MATRIXDEF}" ]; then
    if [ -z "${PROXY}" ]; then
        curl "${UNIDICURL}" -o "${UNIDICZIP}"
    else
        curl "${UNIDICURL}" -x "${PROXY}" -o "${UNIDICZIP}"
    fi
    unzip "${UNIDICZIP}"
    cp "unidic-mecab-${UNIDICVER}_src/matrix.def" "${MATRIXDEF}"
fi

./dicbuilder -o "${SYSDIC}" -m "${MATRIXDEF}" -d "go" "${CORECSV}"
./dicbuilder -o "${SYSFULLDIC}" -m "${MATRIXDEF}" -d "go" "${CORECSV}" "${NOTCORECSV}"
