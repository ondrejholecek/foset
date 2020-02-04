#!/bin/sh

version=`cat VERSION.txt`

cat <<EOF
Package: foset
Version: $version
Section: base
Priority: optional
Architecture: amd64
Suggests: foset
Maintainer: Ondrej Holecek <ondrej@holecek.eu>
Homepage: https://github.com/ondrejholecek/foset 
Description: FOrtigate SEssion Tool
 Command line utility to parse session dump from FortiGate
 device. It can also filter the sessions to only display
 the interresting ones and then format the output to be
 more suitable for further analysis.
EOF
