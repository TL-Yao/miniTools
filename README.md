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
minitool db proxy <db-name>           # Start local proxy for DataGrip (auto tsh login)
minitool db proxy <db-name> -p 5432   # Use specific local port (overrides config)
minitool db proxy all --env <env>     # Start proxies for all databases in an environment
minitool db connect <db-name>         # Direct connect to database CLI (auto tsh login)
```

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
      environment: production                              # Environment for tsh login
      service_name: prod-xxx-20230724-rds-aurora-xxx-1    # Teleport service name
      db_name: myapp                                       # Actual database name
      db_protocol: postgres
      db_user: admin
      local_port: 5432                                     # Local port for proxy
```

The tool automatically runs `tsh login --proxy=<proxy> <cluster>` before connecting to databases based on the environment configuration.

#### Using Proxy Mode with DataGrip

**Step 1: Start the proxy**

```bash
minitool db proxy prod-db
```

You will see output like:

```
Starting proxy for: prod-db (Production database)
Command: tsh proxy db prod-xxx-20230724-rds-aurora-xxx-1 --db-user admin --db-name myapp --port 5432

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

#### Running Multiple Proxies by Environment

To connect to multiple databases in the same environment, use the `--env` flag with `proxy all`:

```yaml
teleport:
  environments:
    production:
      proxy: "general-prod.xxx.com:443"
      cluster: "general.xxx.prod.cdcinternal.com"

  databases:
    - name: prod-db-1
      environment: production
      service_name: prod-xxx-rds-1
      db_name: myapp
      db_user: admin
      local_port: 5432

    - name: prod-db-2
      environment: production
      service_name: prod-xxx-rds-2
      db_name: analytics
      db_user: admin
      local_port: 5433
```

**Start all proxies for an environment:**

```bash
minitool db proxy all --env production
```

This will:
1. Run `tsh login` for the production environment (opens browser for auth)
2. Start proxies for all databases in that environment

```
Logging in to environment: production
Command: tsh login --proxy=general-prod.xxx.com:443 general.xxx.prod.cdcinternal.com

✓ Login successful

Starting proxies for 2 databases in environment 'production'...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[prod-db-1] Starting on port 5432...
[prod-db-2] Starting on port 5433...

✓ All Proxies Started

DataGrip Connection Settings:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  prod-db-1            localhost:5432   (myapp)
  prod-db-2            localhost:5433   (analytics)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Press Ctrl+C to stop all proxies...
```

**Run proxies separately (each with its own tsh login):**

```bash
# Terminal 1
minitool db proxy prod-db-1

# Terminal 2
minitool db proxy prod-db-2
```

> **Note**: The tool will check if the port is already in use before starting the proxy.

## Configuration

Copy `config.yaml.example` to `config.yaml` and add your API key:

```yaml
anthropic_api_key: your-key-here
```

Config values are embedded into the binary during `./install.sh`, so `minitool tr` works from any directory.

**Note:** Re-run `./install.sh` after changing `config.yaml` to apply updates.
