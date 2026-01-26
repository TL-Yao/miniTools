#!/bin/bash
set -e

cd "$(dirname "$0")"

# Build and install to ~/.local/bin (no sudo required)
mkdir -p ~/.local/bin
go build -o ~/.local/bin/minitool ./cmd/minitool

echo "minitool installed to ~/.local/bin/minitool"
