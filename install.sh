#!/bin/bash
set -e

cd "$(dirname "$0")"

# Create ~/bin if not exists
mkdir -p ~/bin

# Build and install
go build -o ~/bin/minitool ./cmd/minitool

echo "minitool installed to ~/bin/minitool"

# Check if ~/bin is in PATH
if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    echo ""
    echo "Add this to your ~/.zshrc:"
    echo '  export PATH="$HOME/bin:$PATH"'
fi
