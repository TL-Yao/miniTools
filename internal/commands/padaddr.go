package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var padAddrCmd = &cobra.Command{
	Use:   "padaddr <address>",
	Short: "Left-pad address to 32 bytes (64 hex chars)",
	Long: `Left-pad a blockchain address with zeros to make it 32 bytes (64 hex characters).

Examples:
  minitool padaddr 0x1234abcd
  minitool padaddr 1234abcd`,
	Args: cobra.ExactArgs(1),
	RunE: runPadAddr,
}

func runPadAddr(cmd *cobra.Command, args []string) error {
	addr := strings.TrimSpace(args[0])

	// Remove 0x prefix if present
	addr = strings.TrimPrefix(addr, "0x")
	addr = strings.TrimPrefix(addr, "0X")

	// Validate hex string
	for _, c := range addr {
		if !isHexChar(c) {
			return fmt.Errorf("invalid hex character: %c", c)
		}
	}

	if len(addr) > 64 {
		return fmt.Errorf("address too long: %d hex chars (max 64)", len(addr))
	}

	// Pad to 64 hex characters (32 bytes)
	padded := fmt.Sprintf("%064s", addr)
	padded = strings.ReplaceAll(padded, " ", "0")

	fmt.Printf("0x%s\n", strings.ToLower(padded))

	return nil
}

func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
