# miniTools

A collection of CLI utilities written in Go.

## Language Convention

- **Communication with user**: Chinese
- **Code, comments, documentation**: English only

## Project Structure

```
miniTools/
├── cmd/
│   └── minitool/        # Main CLI entry point
├── internal/
│   ├── commands/        # CLI command implementations
│   ├── translator/      # Claude API wrapper for translation
│   └── config/          # Configuration management
├── go.mod
└── go.sum
```

## Development

### Prerequisites

- Go 1.21+

### Build & Install

```bash
./install.sh
```

### Test

```bash
go test ./...
```

### Lint

```bash
go vet ./...
```

## Conventions

- Each CLI tool lives in its own directory under `cmd/`
- Use `cobra` or standard `flag` package for CLI argument parsing
- Follow Go naming conventions (camelCase for unexported, PascalCase for exported)
- Keep functions small and focused
- Write table-driven tests

## Code Style

- Format with `gofmt` or `goimports`
- Error messages should be lowercase without trailing punctuation
- Return errors rather than panic in library code
- Use meaningful variable names over single letters (except for loop indices)

## Commands

| Command | Description |
|---------|-------------|
| `minitool ts <input>` | Convert between timestamp and datetime |
| `minitool padaddr <addr>` | Left-pad address to 32 bytes |
| `minitool lower <text>` | Convert text to lowercase |
| `minitool tr` | Interactive Chinese-English translation |

## Configuration

Config file: `~/.config/minitool/config.yaml`

Environment variables:
- `ANTHROPIC_API_KEY` - API key for translation feature
