package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
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

Usage:
  1. First input: the text you want to translate
  2. Follow-up inputs: modification requests (e.g., "more formal", "shorter")
  3. To translate NEW text, use /new to start a fresh session

The translator uses casual, conversational tone by default (like texting).
Specify "formal" or "academic" if you need a different style.`,
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

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	defer rl.Close()

	for {
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("Bye!")
				return nil
			}
			return err
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
}

func printHelp() {
	fmt.Print(`
Commands:
  /new, /clear  - Start a new conversation (clear history)
  /help         - Show this help
  exit, quit    - Exit the translator

Usage:
  1. First input: the text you want to translate
  2. Follow-up inputs: modification requests (e.g., "more formal", "shorter")
  3. To translate NEW text, use /new to start a fresh session

Style:
  - Default: casual, conversational (like texting/IM)
  - Say "formal" or "academic" if needed
`)
}
