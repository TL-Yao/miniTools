# miniTools

A collection of CLI utilities.

## Prerequisites

- Go 1.21+
- Anthropic API key (for translation only)

## Install

```bash
./install.sh
```

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

Commands in translation mode:
- `/new` - start new conversation
- `exit` - quit

## Configuration

Copy `config.yaml.example` to `config.yaml` and add your API key:

```yaml
anthropic_api_key: your-key-here
```
