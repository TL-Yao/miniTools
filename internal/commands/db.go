package commands

import (
	"bufio"
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
)

func init() {
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbProxyCmd)
	dbCmd.AddCommand(dbConnectCmd)

	// Add flags for proxy command
	dbProxyCmd.Flags().IntVarP(&proxyPort, "port", "p", 0, "Local port to listen on (0 for auto)")
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
		return runAllProxies(cfg)
	}

	db := cfg.GetDatabase(dbName)
	if db == nil {
		return fmt.Errorf("database '%s' not found in config\n\nAvailable databases:\n%s",
			dbName, listAvailableDBs(cfg))
	}

	return startSingleProxy(db, proxyPort)
}

// runAllProxies starts proxies for all configured databases
func runAllProxies(cfg *config.Config) error {
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

	// Check all ports are available
	for _, db := range databases {
		if err := checkPortAvailable(db.LocalPort); err != nil {
			return fmt.Errorf("port %d for '%s' is already in use", db.LocalPort, db.Name)
		}
	}

	fmt.Printf("Starting proxies for %d databases...\n", len(databases))
	fmt.Println(strings.Repeat("━", 60))

	// Track all running processes for cleanup
	var processes []*exec.Cmd
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

			tshCmd, err := startProxyProcess(db)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to start: %v\n", db.Name, err)
				return
			}

			mu.Lock()
			processes = append(processes, tshCmd)
			mu.Unlock()

			// Wait for this proxy to finish
			tshCmd.Wait()
		}(db)
	}

	// Wait a moment for all proxies to start
	time.Sleep(2 * time.Second)

	// Display summary of all connections
	fmt.Println()
	fmt.Println("✓ All Proxies Started")
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
func startProxyProcess(db *config.TeleportDatabase) (*exec.Cmd, error) {
	tshArgs := []string{"proxy", "db", db.ServiceName, "--tunnel"}

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
	// Redirect output to null to avoid cluttering terminal
	tshCmd.Stdout = nil
	tshCmd.Stderr = nil

	if err := tshCmd.Start(); err != nil {
		return nil, err
	}

	return tshCmd, nil
}

// startSingleProxy starts a proxy for a single database (original behavior)
func startSingleProxy(db *config.TeleportDatabase, cliPort int) error {
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

	// Note: If not logged in, tsh will automatically prompt for login
	// After login completes, tsh will continue with the database connection

	// Execute tsh db connect with stdin/stdout/stderr attached
	// This allows tsh to handle login interactively if needed
	tshCmd := exec.Command("tsh", tshArgs...)
	tshCmd.Stdin = os.Stdin
	tshCmd.Stdout = os.Stdout
	tshCmd.Stderr = os.Stderr

	// Run the command - it will handle login flow automatically
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
