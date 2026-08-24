#!/usr/bin/env bash
set -euo pipefail

image="$1"
platform="${2:-linux/amd64}"

docker build --platform "$platform" -t "go-annotation/${image}:${platform//\//-}" .
