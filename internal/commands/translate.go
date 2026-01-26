package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tongleyao/minitools/internal/config"
	"github.com/tongleyao/minitools/internal/translator"
)

var trCmd = &cobra.Command{
	Use:   "tr",
	Short: "Interactive Chinese-English translation",
	Long: `Start an interactive translation session.

Commands:
  /new   - Start a new conversation (clear history)
  /clear - Same as /new
  exit   - Exit the translator
  quit   - Same as exit

The translator automatically detects the language and translates between Chinese and English.
You can follow up with requests like "more casual" or "simpler words" to modify the translation.`,
	RunE: runTranslate,
}

func runTranslate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.AnthropicAPIKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set. Set it via environment variable or config file (~/.config/minitool/config.yaml)")
	}

	tr, err := translator.New(cfg.AnthropicAPIKey, cfg.TranslateModel)
	if err != nil {
		return fmt.Errorf("failed to create translator: %w", err)
	}

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nBye!")
		cancel()
		os.Exit(0)
	}()

	fmt.Println("Translator ready. Type text to translate, /new to reset, exit to quit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Bye!")
			return nil
		case "/new", "/clear":
			tr.Reset()
			fmt.Println("New conversation started.")
			fmt.Println()
			continue
		case "/help":
			printHelp()
			continue
		}

		// Translate
		result, err := tr.Translate(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(result)
		fmt.Println()
	}

	return nil
}

func printHelp() {
	fmt.Println(`
Commands:
  /new, /clear  - Start a new conversation (clear history)
  /help         - Show this help
  exit, quit    - Exit the translator

Tips:
  - Just type text to translate
  - Follow up with "more casual", "simpler", "formal" etc. to modify
  - Chinese text will be translated to English, and vice versa
`)
}
