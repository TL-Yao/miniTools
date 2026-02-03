#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "🔧 Building minitool..."
echo ""

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "⚠️  Warning: config.yaml not found"
    echo "   Creating from config.yaml.example..."
    if [ -f "config.yaml.example" ]; then
        cp config.yaml.example config.yaml
        echo ""
        echo "📝 Please edit config.yaml with your actual configuration:"
        echo "   - Add your Anthropic API key"
        echo "   - Configure your Teleport databases"
        echo ""
        echo "Then run ./install.sh again"
        exit 1
    else
        echo "❌ Error: config.yaml.example not found"
        exit 1
    fi
fi

# Copy config.yaml to internal/config/embedded.yaml for embedding
echo "📦 Embedding configuration into binary..."
cp config.yaml internal/config/embedded.yaml

# Read config values from config.yaml for ldflags (backward compatibility)
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

echo "🔨 Compiling..."
if [ -n "$LDFLAGS" ]; then
    go build -ldflags "$LDFLAGS" -o ~/.local/bin/minitool ./cmd/minitool
else
    go build -o ~/.local/bin/minitool ./cmd/minitool
fi

# Clean up embedded config (keep it out of git)
rm -f internal/config/embedded.yaml
git checkout internal/config/embedded.yaml 2>/dev/null || echo "# Placeholder" > internal/config/embedded.yaml

echo ""
echo "✅ Installation complete!"
echo ""
echo "📍 Binary location: ~/.local/bin/minitool"
echo "📦 Configuration embedded: Yes (including database configs)"
echo ""
echo "You can now use 'minitool' from anywhere!"
echo ""
echo "Try: minitool db list"
