package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tongleyao/minitools/internal/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Teleport database management commands",
	Long:  `Manage and connect to Teleport databases configured in config.yaml.`,
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured databases",
	Long:  `List all Teleport databases configured in config.yaml.`,
	RunE:  runDBList,
}

var dbProxyCmd = &cobra.Command{
	Use:   "proxy <db-name|all>",
	Short: "Start a local proxy for a database (for DataGrip)",
	Long: `Start a local proxy using 'tsh proxy db' command.
This creates a local port forwarding that can be used with DataGrip or other database clients.

Use 'all' to start proxies for all configured databases simultaneously.
Note: When using 'all', each database must have a unique local_port configured.`,
	Args: cobra.ExactArgs(1),
	RunE: runDBProxy,
}

var dbConnectCmd = &cobra.Command{
	Use:   "connect <db-name>",
	Short: "Connect directly to a database",
	Long:  `Connect directly to a database using 'tsh db connect' command.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDBConnect,
}

var (
	proxyPort int
	proxyEnv  string
)

func init() {
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbProxyCmd)
	dbCmd.AddCommand(dbConnectCmd)

	// Add flags for proxy command
	dbProxyCmd.Flags().IntVarP(&proxyPort, "port", "p", 0, "Local port to listen on (0 for auto)")
	dbProxyCmd.Flags().StringVarP(&proxyEnv, "env", "e", "", "Environment to use (optional, filters databases by environment)")
}

// TshStatus represents the JSON output from 'tsh status --format=json'
type TshStatus struct {
	Active struct {
		ProfileURL string    `json:"profile_url"`
		Cluster    string    `json:"cluster"`
		ValidUntil time.Time `json:"valid_until"`
	} `json:"active"`
}

// isLoggedInToEnvironment checks if already logged into the target environment
// and if the session is still valid (with at least 5 minutes buffer)
func isLoggedInToEnvironment(env *config.TeleportEnvironment) (bool, error) {
	// Run tsh status with JSON format
	cmd := exec.Command("tsh", "status", "--format=json")
	output, err := cmd.Output()

	// If command fails, assume not logged in
	if err != nil {
		return false, nil
	}

	var status TshStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return false, fmt.Errorf("failed to parse tsh status: %w", err)
	}

	// Check if logged into the correct cluster/proxy
	expectedProxy := fmt.Sprintf("https://%s", env.Proxy)
	if status.Active.ProfileURL != expectedProxy {
		return false, nil
	}

	// Check if session is still valid with at least 5 minutes buffer
	bufferTime := 5 * time.Minute
	if time.Until(status.Active.ValidUntil) < bufferTime {
		return false, nil
	}

	return true, nil
}

// ensureTshLogin performs tsh login for the given environment if needed
// It checks if already logged in to the target environment with a valid session
// tsh supports multiple profiles, so we don't need to logout before logging into a different environment
func ensureTshLogin(env *config.TeleportEnvironment, envName string) error {
	if env == nil {
		// No environment configured, skip login (backward compatibility)
		return nil
	}

	if env.Proxy == "" || env.Cluster == "" {
		return fmt.Errorf("environment '%s' is missing proxy or cluster configuration", envName)
	}

	// Check if already logged in to the correct environment with valid session
	loggedIn, err := isLoggedInToEnvironment(env)
	if err != nil {
		fmt.Printf("Warning: failed to check login status: %v\n", err)
		// Continue with login attempt
	}

	if loggedIn {
		fmt.Printf("✓ Already logged in to environment: %s (session is valid)\n\n", envName)
		return nil
	}

	// No logout needed - tsh supports multiple profiles simultaneously
	// If the session is expired, tsh login will automatically refresh it
	// If it's a different environment, tsh will create a new profile

	fmt.Printf("Logging in to environment: %s\n", envName)
	fmt.Printf("Command: tsh login --proxy=%s %s\n\n", env.Proxy, env.Cluster)

	// Build tsh login command
	tshArgs := []string{"login", "--proxy=" + env.Proxy, env.Cluster}

	tshCmd := exec.Command("tsh", tshArgs...)
	tshCmd.Stdin = os.Stdin
	tshCmd.Stdout = os.Stdout
	tshCmd.Stderr = os.Stderr

	// Run the login command - this will open browser for authentication
	if err := tshCmd.Run(); err != nil {
		return fmt.Errorf("tsh login failed: %w", err)
	}

	fmt.Println("\n✓ Login successful")
	fmt.Println()

	return nil
}

func runDBList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	databases := cfg.GetDatabases()
	if len(databases) == 0 {
		fmt.Println("No databases configured.")
		fmt.Println("\nAdd databases to your config.yaml under the 'teleport' section:")
		fmt.Println(`
teleport:
  databases:
    - name: my-db                    # Custom alias
      service_name: my-teleport-db   # Teleport registered service name
      db_name: database-name         # Actual database name
      db_protocol: postgres
      db_user: admin
      local_port: 5432               # Local port for proxy (optional)`)
		return nil
	}

	fmt.Println("Configured Databases:")
	fmt.Println(strings.Repeat("━", 60))
	for _, db := range databases {
		fmt.Printf("\n  %-20s %s\n", db.Name, db.Description)
		fmt.Printf("    Service: %s\n", db.ServiceName)
		fmt.Printf("    DB Name: %-10s Protocol: %s\n", db.DBName, db.DBProtocol)
		fmt.Printf("    User: %-13s Cluster: %s\n", db.DBUser, db.Cluster)
		if db.Environment != "" {
			fmt.Printf("    Environment: %s\n", db.Environment)
		}
		if db.LocalPort > 0 {
			fmt.Printf("    Local Port: %d\n", db.LocalPort)
		}
		if len(db.Labels) > 0 {
			labels := make([]string, 0, len(db.Labels))
			for k, v := range db.Labels {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}
			fmt.Printf("    Labels: %s\n", strings.Join(labels, ", "))
		}
	}
	fmt.Println()

	// Show available environments
	envNames := cfg.GetEnvironmentNames()
	if len(envNames) > 0 {
		fmt.Println("Configured Environments:")
		fmt.Println(strings.Repeat("━", 60))
		for _, name := range envNames {
			env := cfg.GetEnvironment(name)
			if env != nil {
				fmt.Printf("  %-15s proxy: %s\n", name, env.Proxy)
				fmt.Printf("                cluster: %s\n", env.Cluster)
			}
		}
		fmt.Println()
	}

	return nil
}

func runDBProxy(cmd *cobra.Command, args []string) error {
	dbName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle "all" mode - start proxies for all databases
	if dbName == "all" {
		// If --env is specified, only proxy databases in that environment
		if proxyEnv != "" {
			env := cfg.GetEnvironment(proxyEnv)
			if env == nil {
				envNames := cfg.GetEnvironmentNames()
				return fmt.Errorf("environment '%s' not found in config\n\nAvailable environments: %s", proxyEnv, strings.Join(envNames, ", "))
			}
			return runAllProxies(cfg, proxyEnv, env)
		}

		// No --env specified: proxy all databases across all environments
		return runAllProxiesMultiEnv(cfg)
	}

	db := cfg.GetDatabase(dbName)
	if db == nil {
		return fmt.Errorf("database '%s' not found in config\n\nAvailable databases:\n%s",
			dbName, listAvailableDBs(cfg))
	}

	return startSingleProxy(cfg, db, proxyPort)
}

// runAllProxies starts proxies for all databases in the specified environment
func runAllProxies(cfg *config.Config, envName string, env *config.TeleportEnvironment) error {
	var databases []config.TeleportDatabase
	if envName != "" {
		databases = cfg.GetDatabasesByEnvironment(envName)
	} else {
		databases = cfg.GetDatabases()
	}

	if len(databases) == 0 {
		if envName != "" {
			return fmt.Errorf("no databases configured for environment '%s'", envName)
		}
		return fmt.Errorf("no databases configured")
	}

	// Validate all databases have local_port configured
	var missingPorts []string
	usedPorts := make(map[int]string)
	for _, db := range databases {
		if db.LocalPort == 0 {
			missingPorts = append(missingPorts, db.Name)
		} else {
			if existing, ok := usedPorts[db.LocalPort]; ok {
				return fmt.Errorf("duplicate local_port %d configured for '%s' and '%s'", db.LocalPort, existing, db.Name)
			}
			usedPorts[db.LocalPort] = db.Name
		}
	}

	if len(missingPorts) > 0 {
		return fmt.Errorf("'all' mode requires local_port for each database.\nMissing local_port: %s", strings.Join(missingPorts, ", "))
	}

	// Perform tsh login first if environment is specified
	if env != nil {
		if err := ensureTshLogin(env, envName); err != nil {
			return err
		}
	}

	// Check all ports are available
	for _, db := range databases {
		if err := checkPortAvailable(db.LocalPort); err != nil {
			return fmt.Errorf("port %d for '%s' is already in use", db.LocalPort, db.Name)
		}
	}

	if envName != "" {
		fmt.Printf("Starting proxies for %d databases in environment '%s'...\n", len(databases), envName)
	} else {
		fmt.Printf("Starting proxies for %d databases...\n", len(databases))
	}
	fmt.Println(strings.Repeat("━", 60))

	// Track all running processes for cleanup
	var processes []*exec.Cmd
	var failedProxies []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Handle signals for graceful shutdown of all proxies
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start each proxy in a goroutine
	for i := range databases {
		db := &databases[i]
		wg.Add(1)

		go func(db *config.TeleportDatabase) {
			defer wg.Done()

			tshCmd, err := startProxyProcess(db, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to start: %v\n", db.Name, err)
				mu.Lock()
				failedProxies = append(failedProxies, db.Name)
				mu.Unlock()
				return
			}

			mu.Lock()
			processes = append(processes, tshCmd)
			mu.Unlock()

			// Wait for this proxy to finish
			if err := tshCmd.Wait(); err != nil {
				// Check if proxy exited unexpectedly (not from our signal)
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Only report if it wasn't killed by signal
					if exitErr.ExitCode() != -1 && exitErr.ExitCode() != 130 {
						fmt.Fprintf(os.Stderr, "[%s] Proxy exited unexpectedly with code %d\n", db.Name, exitErr.ExitCode())
						mu.Lock()
						failedProxies = append(failedProxies, db.Name)
						mu.Unlock()
					}
				}
			}
		}(db)
	}

	// Wait a moment for all proxies to start
	time.Sleep(3 * time.Second)

	// Display summary of all connections
	fmt.Println()
	mu.Lock()
	successCount := len(databases) - len(failedProxies)
	if len(failedProxies) > 0 {
		fmt.Printf("⚠ %d/%d Proxies Started (%d failed)\n", successCount, len(databases), len(failedProxies))
		fmt.Println("\nFailed proxies:")
		for _, name := range failedProxies {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Println("✓ All Proxies Started")
	}
	mu.Unlock()
	fmt.Println()
	fmt.Println("DataGrip Connection Settings:")
	fmt.Println(strings.Repeat("━", 60))
	for _, db := range databases {
		fmt.Printf("  %-20s localhost:%-5d  (%s)\n", db.Name, db.LocalPort, db.DBName)
	}
	fmt.Println(strings.Repeat("━", 60))
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop all proxies...")

	// Wait for signal
	<-sigChan
	fmt.Println("\n\nShutting down all proxies...")

	// Kill all processes
	mu.Lock()
	for _, p := range processes {
		if p.Process != nil {
			p.Process.Signal(syscall.SIGTERM)
		}
	}
	mu.Unlock()

	// Wait for all to finish
	wg.Wait()
	fmt.Println("All proxies stopped.")

	return nil
}

// startProxyProcess starts a single tsh proxy process and returns the command
func startProxyProcess(db *config.TeleportDatabase, cfg *config.Config) (*exec.Cmd, error) {
	tshArgs := []string{"proxy", "db", db.ServiceName, "--tunnel"}

	// Add --proxy flag to explicitly specify which teleport proxy to use
	// This is critical when multiple profiles exist to avoid authentication issues
	if db.Environment != "" {
		env := cfg.GetEnvironment(db.Environment)
		if env != nil && env.Proxy != "" {
			tshArgs = append(tshArgs, "--proxy", env.Proxy)
		}
	}

	if db.DBUser != "" {
		tshArgs = append(tshArgs, "--db-user", db.DBUser)
	}
	if db.DBName != "" {
		tshArgs = append(tshArgs, "--db-name", db.DBName)
	}
	if db.Cluster != "" {
		tshArgs = append(tshArgs, "--cluster", db.Cluster)
	}
	if db.LocalPort > 0 {
		tshArgs = append(tshArgs, "--port", fmt.Sprintf("%d", db.LocalPort))
	}

	fmt.Printf("[%s] Starting on port %d...\n", db.Name, db.LocalPort)

	tshCmd := exec.Command("tsh", tshArgs...)

	// Capture stderr to detect errors, but suppress stdout to avoid clutter
	stderrPipe, err := tshCmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	tshCmd.Stdout = nil

	if err := tshCmd.Start(); err != nil {
		return nil, err
	}

	// Read stderr in background and print errors if any
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			// Only print actual errors, skip warnings and info messages
			if strings.Contains(line, "ERROR") || strings.Contains(line, "error") ||
				strings.Contains(line, "failed") || strings.Contains(line, "Failed") {
				fmt.Fprintf(os.Stderr, "[%s] %s\n", db.Name, line)
			}
		}
	}()

	return tshCmd, nil
}

// startSingleProxy starts a proxy for a single database (original behavior)
func startSingleProxy(cfg *config.Config, db *config.TeleportDatabase, cliPort int) error {
	// Perform tsh login if environment is configured
	if db.Environment != "" {
		env := cfg.GetEnvironment(db.Environment)
		if env != nil {
			if err := ensureTshLogin(env, db.Environment); err != nil {
				return err
			}
		}
	}

	// Build tsh proxy db command
	// Format: tsh proxy db <service-name> --db-user=<user> --db-name=<name> --tunnel
	// --tunnel mode allows connecting without SSL certificates
	tshArgs := []string{"proxy", "db", db.ServiceName, "--tunnel"}

	if db.DBUser != "" {
		tshArgs = append(tshArgs, "--db-user", db.DBUser)
	}
	if db.DBName != "" {
		tshArgs = append(tshArgs, "--db-name", db.DBName)
	}
	// Cluster is optional
	if db.Cluster != "" {
		tshArgs = append(tshArgs, "--cluster", db.Cluster)
	}

	// Determine which port to use: CLI flag > config file > auto
	portToUse := cliPort
	if portToUse == 0 && db.LocalPort > 0 {
		portToUse = db.LocalPort
	}

	// Check if port is available before starting proxy
	if portToUse > 0 {
		if err := checkPortAvailable(portToUse); err != nil {
			return fmt.Errorf("port %d is already in use, please choose another port with -p flag or update local_port in config", portToUse)
		}
		tshArgs = append(tshArgs, "--port", fmt.Sprintf("%d", portToUse))
	}

	fmt.Printf("Starting proxy for: %s", db.Name)
	if db.Description != "" {
		fmt.Printf(" (%s)", db.Description)
	}
	fmt.Println()
	fmt.Printf("Command: tsh %s\n\n", strings.Join(tshArgs, " "))

	// Note: If not logged in, tsh will automatically open browser for login
	// After login completes, tsh will continue with the proxy setup

	// Create command
	tshCmd := exec.Command("tsh", tshArgs...)
	tshCmd.Stdin = os.Stdin // Ensure stdin is connected for any interactive prompts

	// Create pipes to capture output (tsh may output port info to stdout or stderr)
	stdoutPipe, _ := tshCmd.StdoutPipe()
	stderrPipe, _ := tshCmd.StderrPipe()

	// Start the command
	if err := tshCmd.Start(); err != nil {
		return fmt.Errorf("failed to start tsh: %w", err)
	}

	// Track if we've displayed connection info
	connectionInfoDisplayed := false
	portRegex := regexp.MustCompile(`localhost:(\d+)`)

	// Function to scan output and display it while looking for port
	scanOutput := func(reader io.Reader, isStderr bool) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			// Print the line to appropriate output
			if isStderr {
				fmt.Fprintln(os.Stderr, line)
			} else {
				fmt.Println(line)
			}

			// Try to extract port from output if we don't already know it
			if !connectionInfoDisplayed {
				if matches := portRegex.FindStringSubmatch(line); len(matches) > 1 {
					// Use the port from tsh output if we didn't specify one
					actualPort := matches[1]
					if portToUse > 0 {
						actualPort = fmt.Sprintf("%d", portToUse)
					}
					time.Sleep(500 * time.Millisecond)
					displayConnectionInfo(db, actualPort)
					connectionInfoDisplayed = true
				}
			}
		}
	}

	// Scan both stdout and stderr in goroutines
	go scanOutput(stdoutPipe, false)
	go scanOutput(stderrPipe, true)

	// If we know the port upfront, show connection info after a delay
	// (in case tsh doesn't output the port info)
	if portToUse > 0 {
		go func() {
			time.Sleep(2 * time.Second)
			if !connectionInfoDisplayed {
				displayConnectionInfo(db, fmt.Sprintf("%d", portToUse))
				connectionInfoDisplayed = true
			}
		}()
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or command exit
	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down proxy...")
		if tshCmd.Process != nil {
			tshCmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	// Wait for command to finish
	// This will wait for login to complete if needed, then continue with proxy
	if err := tshCmd.Wait(); err != nil {
		// Check if it was killed by signal (expected)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == -1 || exitErr.ExitCode() == 130 {
				return nil // Normal exit from signal
			}
		}
		// For other errors, check if it's a login-related error
		// tsh will output login instructions to stderr, which we've already displayed
		return fmt.Errorf("tsh proxy exited with error: %w", err)
	}

	return nil
}

func runDBConnect(cmd *cobra.Command, args []string) error {
	dbName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db := cfg.GetDatabase(dbName)
	if db == nil {
		return fmt.Errorf("database '%s' not found in config\n\nAvailable databases:\n%s",
			dbName, listAvailableDBs(cfg))
	}

	// Perform tsh login if environment is configured
	if db.Environment != "" {
		env := cfg.GetEnvironment(db.Environment)
		if env != nil {
			if err := ensureTshLogin(env, db.Environment); err != nil {
				return err
			}
		}
	}

	// Build tsh db connect command
	// Format: tsh db connect <service-name> --db-user=<user> --db-name=<name>
	tshArgs := []string{"db", "connect", db.ServiceName}

	if db.DBUser != "" {
		tshArgs = append(tshArgs, "--db-user", db.DBUser)
	}
	if db.DBName != "" {
		tshArgs = append(tshArgs, "--db-name", db.DBName)
	}
	// Cluster is optional
	if db.Cluster != "" {
		tshArgs = append(tshArgs, "--cluster", db.Cluster)
	}

	fmt.Printf("Connecting to: %s", db.Name)
	if db.Description != "" {
		fmt.Printf(" (%s)", db.Description)
	}
	fmt.Println()
	fmt.Printf("Command: tsh %s\n\n", strings.Join(tshArgs, " "))

	// Execute tsh db connect with stdin/stdout/stderr attached
	tshCmd := exec.Command("tsh", tshArgs...)
	tshCmd.Stdin = os.Stdin
	tshCmd.Stdout = os.Stdout
	tshCmd.Stderr = os.Stderr

	// Run the command
	if err := tshCmd.Run(); err != nil {
		return fmt.Errorf("tsh db connect failed: %w", err)
	}

	return nil
}

func displayConnectionInfo(db *config.TeleportDatabase, port string) {
	fmt.Println()
	fmt.Println("✓ Teleport Proxy Started")
	fmt.Println()
	fmt.Printf("Database: %s", db.Name)
	if db.Description != "" {
		fmt.Printf(" (%s)", db.Description)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("DataGrip Connection Settings:")
	fmt.Println(strings.Repeat("━", 30))
	fmt.Printf("  Host:     localhost\n")
	fmt.Printf("  Port:     %s\n", port)
	fmt.Printf("  Database: %s\n", db.DBName)
	fmt.Printf("  User:     %s\n", db.DBUser)
	fmt.Printf("  Protocol: %s\n", db.DBProtocol)
	fmt.Println(strings.Repeat("━", 30))
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the proxy...")
}

// checkPortAvailable checks if a port is available for use
func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

func listAvailableDBs(cfg *config.Config) string {
	databases := cfg.GetDatabases()
	if len(databases) == 0 {
		return "  (no databases configured)"
	}

	var sb strings.Builder
	for _, db := range databases {
		sb.WriteString(fmt.Sprintf("  - %s", db.Name))
		if db.Description != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", db.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// runAllProxiesMultiEnv starts proxies for all databases across all environments
// This supports multiple environments running simultaneously
func runAllProxiesMultiEnv(cfg *config.Config) error {
	databases := cfg.GetDatabases()
	if len(databases) == 0 {
		return fmt.Errorf("no databases configured")
	}

	// Validate all databases have local_port configured
	var missingPorts []string
	usedPorts := make(map[int]string)
	for _, db := range databases {
		if db.LocalPort == 0 {
			missingPorts = append(missingPorts, db.Name)
		} else {
			if existing, ok := usedPorts[db.LocalPort]; ok {
				return fmt.Errorf("duplicate local_port %d configured for '%s' and '%s'", db.LocalPort, existing, db.Name)
			}
			usedPorts[db.LocalPort] = db.Name
		}
	}

	if len(missingPorts) > 0 {
		return fmt.Errorf("'all' mode requires local_port for each database.\nMissing local_port: %s", strings.Join(missingPorts, ", "))
	}

	// Group databases by environment
	envDatabases := make(map[string][]config.TeleportDatabase)
	for _, db := range databases {
		envName := db.Environment
		if envName == "" {
			envName = "(no environment)"
		}
		envDatabases[envName] = append(envDatabases[envName], db)
	}

	// Login to each environment
	loginedEnvs := make(map[string]bool)
	for envName := range envDatabases {
		if envName == "(no environment)" {
			continue
		}

		env := cfg.GetEnvironment(envName)
		if env != nil {
			if err := ensureTshLogin(env, envName); err != nil {
				return fmt.Errorf("failed to login to environment '%s': %w", envName, err)
			}
			loginedEnvs[envName] = true
		}
	}

	// Check all ports are available
	for _, db := range databases {
		if err := checkPortAvailable(db.LocalPort); err != nil {
			return fmt.Errorf("port %d for '%s' is already in use", db.LocalPort, db.Name)
		}
	}

	// Display summary of what we're starting
	fmt.Printf("Starting proxies for %d databases across %d environment(s)...\n", len(databases), len(envDatabases))
	fmt.Println(strings.Repeat("━", 60))
	for envName, dbs := range envDatabases {
		fmt.Printf("  [%s] %d database(s)\n", envName, len(dbs))
	}
	fmt.Println(strings.Repeat("━", 60))

	// Track all running processes for cleanup
	var processes []*exec.Cmd
	var failedProxies []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Handle signals for graceful shutdown of all proxies
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start each proxy in a goroutine
	for i := range databases {
		db := &databases[i]
		wg.Add(1)

		go func(db *config.TeleportDatabase) {
			defer wg.Done()

			tshCmd, err := startProxyProcess(db, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to start: %v\n", db.Name, err)
				mu.Lock()
				failedProxies = append(failedProxies, db.Name)
				mu.Unlock()
				return
			}

			mu.Lock()
			processes = append(processes, tshCmd)
			mu.Unlock()

			// Wait for this proxy to finish
			if err := tshCmd.Wait(); err != nil {
				// Check if proxy exited unexpectedly (not from our signal)
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Only report if it wasn't killed by signal
					if exitErr.ExitCode() != -1 && exitErr.ExitCode() != 130 {
						fmt.Fprintf(os.Stderr, "[%s] Proxy exited unexpectedly with code %d\n", db.Name, exitErr.ExitCode())
						mu.Lock()
						failedProxies = append(failedProxies, db.Name)
						mu.Unlock()
					}
				}
			}
		}(db)
	}

	// Wait a moment for all proxies to start
	time.Sleep(3 * time.Second)

	// Display summary of all connections
	fmt.Println()
	mu.Lock()
	successCount := len(databases) - len(failedProxies)
	if len(failedProxies) > 0 {
		fmt.Printf("⚠ %d/%d Proxies Started (%d failed)\n", successCount, len(databases), len(failedProxies))
		fmt.Println("\nFailed proxies:")
		for _, name := range failedProxies {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Println("✓ All Proxies Started")
	}
	mu.Unlock()
	fmt.Println()
	fmt.Println("DataGrip Connection Settings:")
	fmt.Println(strings.Repeat("━", 60))

	// Group by environment for display
	for envName, dbs := range envDatabases {
		if len(envDatabases) > 1 {
			fmt.Printf("\n  [%s]\n", envName)
		}
		for _, db := range dbs {
			fmt.Printf("  %-20s localhost:%-5d  (%s)\n", db.Name, db.LocalPort, db.DBName)
		}
	}

	fmt.Println(strings.Repeat("━", 60))
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop all proxies...")

	// Wait for signal
	<-sigChan
	fmt.Println("\n\nShutting down all proxies...")

	// Kill all processes
	mu.Lock()
	for _, p := range processes {
		if p.Process != nil {
			p.Process.Signal(syscall.SIGTERM)
		}
	}
	mu.Unlock()

	// Wait for all to finish
	wg.Wait()
	fmt.Println("All proxies stopped.")

	return nil
}
