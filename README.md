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

### Teleport Database Management

```bash
minitool db list                      # List all configured databases and environments
minitool db proxy <db-name>           # Start local proxy for DataGrip
minitool db proxy <db-name> -p 5432   # Use specific local port (overrides config)
minitool db proxy all                 # Start proxies for ALL databases across all environments
minitool db proxy all --env <env>     # Start proxies for all databases in a specific environment
minitool db connect <db-name>         # Direct connect to database CLI
```

#### Smart Session Management

The tool features intelligent Teleport session management:
- **Reuses existing sessions**: Checks if you're already logged in before re-authenticating
- **Multi-environment support**: Can maintain sessions for multiple Teleport clusters simultaneously (e.g., production + staging)
- **Automatic expiry detection**: Only re-authenticates when session expires (< 5 minutes remaining)
- **No unnecessary browser prompts**: Skips login if session is still valid

Configure environments and databases in `config.yaml`:

```yaml
teleport:
  # Environment configurations for tsh login
  environments:
    production:
      proxy: "general-prod.xxx.com:443"
      cluster: "general.xxx.prod.cdcinternal.com"
    staging:
      proxy: "general-staging.xxx.com:443"
      cluster: "general.xxx.staging.cdcinternal.com"

  databases:
    - name: prod-db                                        # Custom alias
      description: Production database
      environment: production                              # Environment name (proxy & cluster are read from environments)
      service_name: prod-xxx-20230724-rds-aurora-xxx-1    # Teleport service name
      db_name: myapp                                       # Actual database name
      db_protocol: postgres
      db_user: admin
      local_port: 5432                                     # Local port for proxy (required for 'all' mode)
```

**Important**: 
- Each database only needs to specify `environment` - the `proxy` and `cluster` are automatically read from the corresponding environment configuration
- All databases must have unique `local_port` values for `proxy all` mode to work

#### Using Proxy Mode with DataGrip

**Step 1: Start the proxy**

```bash
minitool db proxy prod-db
```

You will see output like:

```
✓ Already logged in to environment: production (session is valid)

Starting proxy for: prod-db (Production database)
Command: tsh proxy db prod-xxx-xxx-rds-xxx-xxx-1 --proxy=... --cluster=... --db-user admin --db-name myapp --port 5432

✓ Teleport Proxy Started

Database: prod-db (Production database)

DataGrip Connection Settings:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Host:     localhost
  Port:     5432
  Database: myapp
  User:     admin
  Protocol: postgres
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Press Ctrl+C to stop the proxy...
```

> **Note**: If this is your first time or the session expired, it will prompt for browser authentication.

**Step 2: Configure DataGrip**

1. Open DataGrip and click **+** → **Data Source** → **PostgreSQL** (or your database type)

2. In the **General** tab, fill in:
   - **Host**: `localhost`
   - **Port**: `5432` (or the port shown in proxy output)
   - **Database**: `myapp` (your db_name)
   - **User**: `admin` (your db_user)
   - **Authentication**: Select **No auth** (Teleport handles authentication)

3. Click **Test Connection** to verify

4. Click **OK** to save

> **Note**: The proxy runs in `--tunnel` mode, so no SSL configuration is needed in DataGrip.

**Step 3: Keep proxy running**

Keep the terminal with proxy running while using DataGrip. Press `Ctrl+C` to stop when done.

#### Running Multiple Proxies

**Option 1: All databases across all environments (recommended)**

```bash
minitool db proxy all
```

This will:
1. Login to each environment (if not already logged in or session expired)
2. Start proxies for ALL databases in ALL environments simultaneously
3. Maintain separate sessions for each environment

Example output:
```
✓ Already logged in to environment: production (session is valid)

Logging in to environment: staging
Command: tsh login --proxy=general-xxx.xxx.com:443 general.xxx.staging.xxx.com

✓ Login successful

Starting proxies for 15 databases across 2 environment(s)...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [production] 11 database(s)
  [staging] 4 database(s)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[prod-db-1] Starting on port 5432...
[staging-db-1] Starting on port 6432...
...

✓ All Proxies Started

DataGrip Connection Settings:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [production]
  prod-db-1            localhost:5432   (myapp)
  prod-db-2            localhost:5433   (analytics)

  [staging]
  staging-db-1         localhost:6432   (myapp)
  staging-db-2         localhost:6433   (analytics)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Press Ctrl+C to stop all proxies...
```

**Option 2: All databases in a specific environment**

```bash
minitool db proxy all --env production
```

This will only start proxies for databases in the production environment.

**Option 3: Single database**

```bash
minitool db proxy prod-db-1
```

**Configuration requirements for `proxy all`:**
- Each database must have a unique `local_port` configured
- Each database should specify `environment` (proxy and cluster are read from environment config)
- Port numbers must not conflict across environments

**Verifying multi-environment sessions:**

```bash
# Check active sessions
tsh status

# Should show multiple profiles:
# > Profile URL:   https://xxx-staging.xxx.com:443
#   ...
#   Profile URL:   https://xxx-prod.xxx.com:443
#   ...
```

> **Note**: The tool automatically checks if ports are available before starting proxies and will report any failures with detailed error messages.

## Configuration

Copy `config.yaml.example` to `config.yaml` and add your API key:

```yaml
anthropic_api_key: your-key-here
```

Config values are embedded into the binary during `./install.sh`, so `minitool tr` works from any directory.

**Note:** Re-run `./install.sh` after changing `config.yaml` to apply updates.
