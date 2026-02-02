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

- **No sudo required**: All scripts and installations must work without sudo privileges. Install to `~/.local/bin` instead of `/usr/local/bin`.
- **Embedded config**: All config variables must be injected into the binary at build time via `-ldflags`. Update `install.sh` when adding new config values.
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
| `minitool db list` | List all configured Teleport databases and environments |
| `minitool db proxy <name>` | Start local proxy for DataGrip (auto tsh logout + login) |
| `minitool db proxy all --env <env>` | Start proxies for all databases in an environment |
| `minitool db connect <name>` | Direct connect to database CLI (auto tsh logout + login) |

### Teleport Authentication

Database commands automatically handle Teleport authentication:
1. Run `tsh logout` to clear any existing session
2. Run `tsh login --proxy=<proxy> <cluster>` based on the database's environment
3. Wait for browser authentication to complete
4. Proceed with the database operation

This ensures proper environment switching when connecting to databases in different Teleport clusters.

## Configuration

Config file: `~/.config/minitool/config.yaml`

Environment variables:
- `ANTHROPIC_API_KEY` - API key for translation feature
