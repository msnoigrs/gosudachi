#!/bin/bash

PROXY=""
VERSION=""
UNIDICVER="2.1.2"
UNIDICZIP="unidic-mecab-${UNIDICVER}_src.zip"
UNIDICURL="https://unidic.ninjal.ac.jp/unidic_archive/cwj/${UNIDICVER}/${UNIDICZIP}"
POMXML="pom.xml"
MATRIXDEF="matrix.def"
SYSSMALLDIC="system_small.dic"
SYSCOREDIC="system_core.dic"
SYSFULLDIC="system_full.dic"
SMALLCSV="small_lex.csv"
CORECSV="core_lex.csv"
NOTCORECSV="notcore_lex.csv"

init_env() {
    if [ -f "${1}/pom.xml" ]; then
        POMXML="${1}/pom.xml"
    fi
    if [ -f "${1}/src/main/text/${MATRIXDEF}.zip" ]; then
        cp "${1}/src/main/text/${MATRIXDEF}.zip" .
    fi
    if [ -f "${1}/src/main/text/${SMALLCSV}" ]; then
        SMALLCSV="${1}/src/main/text/${SMALLCSV}"
    fi
    if [ -f "${1}/src/main/text/${CORECSV}" ]; then
        CORECSV="${1}/src/main/text/${CORECSV}"
    fi
    if [ -f "${1}/src/main/text/${NOTCORECSV}" ]; then
        NOTCORECSV="${1}/src/main/text/${NOTCORECSV}"
    fi
}

if [ -n "${1}" ]; then
    init_env "${1}"
elif [ -d "../SudachiDict" ]; then
    init_env "../SudachiDict"
fi

if [ ! -f "${MATRIXDEF}" ]; then
    if [ ! -f "${MATRIXDEF}.zip" ]; then
        if [ -z "${PROXY}" ]; then
            curl "${UNIDICURL}" -o "${UNIDICZIP}"
        else
            curl "${UNIDICURL}" -x "${PROXY}" -o "${UNIDICZIP}"
        fi
        unzip "${UNIDICZIP}"
        cp "unidic-mecab-${UNIDICVER}_src/matrix.def" "${MATRIXDEF}"
    else
        unzip "${MATRIXDEF}.zip"
    fi
fi

if [ -f "${POMXML}" ]; then
    VERSION=$(grep -oP -m 1 '<version>\K([^<]+)' "${POMXML}")
fi

if [ -z "${VERSION}" ]; then
    VERSION="go"
fi

if [ ! -f "${SMALLCSV}" -o ! -f "${CORECSV}" -o ! -f "${NOTCORECSV}" ]; then
    echo "dictionary files are needed: ${SMALLCSV}, ${CORECSV}, ${NOTCORECSV}" 1>&2
fi

./dicbuilder -o "${SYSSMALLDIC}" -m "${MATRIXDEF}" -d "${VERSION}" "${SMALLCSV}"
./dicbuilder -o "${SYSCOREDIC}" -m "${MATRIXDEF}" -d "${VERSION}" "${SMALLCSV}" "${CORECSV}"
./dicbuilder -o "${SYSFULLDIC}" -m "${MATRIXDEF}" -d "${VERSION}" "${SMALLCSV}" "${CORECSV}" "${NOTCORECSV}"
