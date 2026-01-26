package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var lowerCmd = &cobra.Command{
	Use:   "lower [text]",
	Short: "Convert text to lowercase",
	Long: `Convert text to lowercase.

Examples:
  minitool lower "Hello World"
  echo "HELLO" | minitool lower`,
	RunE: runLower,
}

func runLower(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		// Use command line argument
		input := strings.Join(args, " ")
		fmt.Println(strings.ToLower(input))
		return nil
	}

	// Check if there's piped input
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Print last line if it doesn't end with newline
					if len(line) > 0 {
						fmt.Print(strings.ToLower(line))
					}
					break
				}
				return err
			}
			fmt.Print(strings.ToLower(line))
		}
		return nil
	}

	return fmt.Errorf("no input provided")
}
