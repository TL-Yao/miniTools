#!/bin/bash
set -e

cd "$(dirname "$0")"

# Read config values from config.yaml if it exists
API_KEY=""
TRANSLATE_MODEL=""
if [ -f "config.yaml" ]; then
    API_KEY=$(grep -E "^anthropic_api_key:" config.yaml | sed 's/anthropic_api_key:[[:space:]]*//' | tr -d '"' | tr -d "'" || true)
    TRANSLATE_MODEL=$(grep -E "^translate_model:" config.yaml | sed 's/translate_model:[[:space:]]*//' | tr -d '"' | tr -d "'" || true)
fi

# Build ldflags
LDFLAGS=""
if [ -n "$API_KEY" ] && [ "$API_KEY" != "your-api-key-here" ]; then
    LDFLAGS="$LDFLAGS -X 'github.com/tongleyao/minitools/internal/config.EmbeddedAPIKey=$API_KEY'"
fi
if [ -n "$TRANSLATE_MODEL" ]; then
    LDFLAGS="$LDFLAGS -X 'github.com/tongleyao/minitools/internal/config.EmbeddedTranslateModel=$TRANSLATE_MODEL'"
fi

# Build and install
mkdir -p ~/.local/bin

if [ -n "$LDFLAGS" ]; then
    go build -ldflags "$LDFLAGS" -o ~/.local/bin/minitool ./cmd/minitool
    echo "minitool installed with embedded config"
else
    go build -o ~/.local/bin/minitool ./cmd/minitool
    echo "minitool installed (no config embedded)"
fi
