#!/bin/sh

port=${1:-'8080'}

python3 -m http.server $port
