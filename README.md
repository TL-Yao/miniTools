# miniTools

A collection of CLI utilities.

## Prerequisites

- Go 1.21+
- Anthropic API key (for translation only)

## Install

```bash
./install.sh
```

If `minitool` command not found after install, add this to your `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then run `source ~/.zshrc` or restart terminal.

## Commands

### Timestamp Conversion

```bash
minitool ts 1769422532              # timestamp to datetime
minitool ts "2026-01-26 10:15:32"   # datetime to timestamp
```

### Address Padding

```bash
minitool padaddr 0x1234abcd         # pad to 32 bytes (64 hex chars)
```

### Lowercase

```bash
minitool lower "Hello World"
echo "HELLO" | minitool lower
```

### Translation (Interactive)

```bash
minitool tr
```

Usage:
1. First input: text to translate (auto-detects Chinese/English)
2. Follow-up inputs: modification requests (e.g., "more formal", "shorter")
3. Use `/new` to translate different text

Default style is casual/conversational (like texting). Say "formal" if needed.

Commands: `/new`, `/help`, `exit`

## Configuration

Copy `config.yaml.example` to `config.yaml` and add your API key:

```yaml
anthropic_api_key: your-key-here
```
