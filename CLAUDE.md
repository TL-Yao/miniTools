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
| `minitool db proxy <name>` | Start local proxy for DataGrip (smart session management) |
| `minitool db proxy all` | Start proxies for all databases across all environments |
| `minitool db proxy all --env <env>` | Start proxies for all databases in a specific environment |
| `minitool db connect <name>` | Direct connect to database CLI (smart session management) |

### Teleport Authentication

Database commands automatically handle Teleport authentication with smart session management:
1. Check if already logged in to the target environment with a valid session (via `tsh status --format=json`)
2. Skip login if session is valid (expires in > 5 minutes)
3. Only perform `tsh login` when needed (no unnecessary logouts)
4. Support multiple environments simultaneously via tsh profiles

Key improvements:
- **No unnecessary re-authentication**: Reuses existing valid sessions
- **Multi-environment support**: Can maintain sessions for multiple Teleport clusters simultaneously
- **Session expiry detection**: Automatically re-authenticates when session is about to expire

## Configuration

Config file: `~/.config/minitool/config.yaml`

Environment variables:
- `ANTHROPIC_API_KEY` - API key for translation feature

## Lessons Learned

### Teleport Multi-Environment Support (Feb 2026)

**Problem**: When implementing `proxy all` to support multiple Teleport environments simultaneously, production database proxies failed with "authentication handshake failed: EOF" errors, even though sessions were valid.

**Root Causes**:
1. **Session conflict**: Initial implementation called `tsh logout` before each environment login, which cleared ALL profiles (not just the current one), breaking the multi-environment support
2. **Missing --proxy flag**: When multiple tsh profiles exist, `tsh proxy db` commands default to the "active" profile (marked with `>` in `tsh status`), causing authentication failures for databases in other environments

**Solutions**:
1. **Remove unnecessary logout**: tsh natively supports multiple profiles simultaneously. Never call `tsh logout` when switching environments - just call `tsh login` directly
2. **Add --proxy and --cluster parameters**: Always read from environment config and include in `tsh proxy db` commands:
   ```go
   // Read from environment config (single source of truth)
   env := cfg.GetEnvironment(db.Environment)
   tshArgs := []string{"proxy", "db", serviceName, "--tunnel",
       "--proxy", env.Proxy,      // Critical for multi-environment
       "--cluster", env.Cluster,  // Routes to correct cluster
       "--db-user", dbUser,
       "--db-name", dbName,
       "--port", port}
   ```
   Database configs should only specify `environment` name, not duplicate proxy/cluster values
3. **Session validation**: Check session validity before login using `tsh status --format=json` and parse the `valid_until` timestamp with a 5-minute buffer
4. **Error visibility**: Capture stderr from proxy processes to display meaningful error messages instead of silently failing

**Key Insights**:
- `tsh status` shows all profiles but marks one as "active" (with `>` symbol)
- `tsh db ls` without `--cluster` only shows databases from the active profile
- Each `tsh login --proxy=<url>` creates/updates a separate profile in `~/.tsh/keys/`
- The `--proxy` flag in `tsh proxy db` is essential for routing commands to the correct profile
- Always validate by checking `tsh status` shows multiple profiles after multi-environment login

**Testing Multi-Environment Setup**:
```bash
# Login to both environments
minitool db proxy all

# Verify both profiles exist
tsh status  # Should show 2+ profiles with different proxy URLs

# Check profile directories
ls ~/.tsh/keys/  # Should show multiple cluster directories
```
