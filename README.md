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
minitool db list                    # List all configured databases
minitool db proxy <db-name>         # Start local proxy for DataGrip
minitool db proxy <db-name> -p 5432 # Use specific local port (overrides config)
minitool db proxy all               # Start proxies for ALL configured databases
minitool db connect <db-name>       # Direct connect to database CLI
```

Configure databases in `config.yaml`:

```yaml
teleport:
  databases:
    - name: prod-db                                        # Custom alias
      description: Production database
      service_name: prod-xxx-20230724-rds-aurora-xxx-1    # Teleport service name
      db_name: myapp                                       # Actual database name
      db_protocol: postgres
      db_user: admin
      local_port: 5432                                     # Local port for proxy
```

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

#### Running Multiple Proxies

To connect to multiple databases simultaneously, configure different `local_port` for each:

```yaml
teleport:
  databases:
    - name: prod-db
      service_name: prod-xxx-rds
      db_name: myapp
      db_user: admin
      local_port: 5432      # Port for prod

    - name: staging-db
      service_name: staging-xxx-rds
      db_name: myapp
      db_user: admin
      local_port: 5433      # Different port for staging
```

**Option 1: Start all proxies at once (recommended)**

```bash
minitool db proxy all
```

This starts proxies for all configured databases in one terminal:

```
Starting proxies for 2 databases...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[prod-db] Starting on port 5432...
[staging-db] Starting on port 5433...

✓ All Proxies Started

DataGrip Connection Settings:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  prod-db              localhost:5432   (myapp)
  staging-db           localhost:5433   (myapp)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Press Ctrl+C to stop all proxies...
```

**Option 2: Run each proxy separately**

```bash
# Terminal 1
minitool db proxy prod-db

# Terminal 2
minitool db proxy staging-db
```

> **Note**: The tool will check if the port is already in use before starting the proxy.

## Configuration

Copy `config.yaml.example` to `config.yaml` and add your API key:

```yaml
anthropic_api_key: your-key-here
```

Config values are embedded into the binary during `./install.sh`, so `minitool tr` works from any directory.

**Note:** Re-run `./install.sh` after changing `config.yaml` to apply updates.
