package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var tsCmd = &cobra.Command{
	Use:   "ts <timestamp_or_datetime>",
	Short: "Convert between timestamp and datetime",
	Long: `Convert between Unix timestamp and human-readable datetime.

Timestamp to datetime:
  minitool ts 1769422532              # seconds (auto-detect)
  minitool ts 1769422532000           # milliseconds
  minitool ts 1769422532000000        # microseconds
  minitool ts 1769422532000000000     # nanoseconds

Datetime to timestamp:
  minitool ts "2026-01-26 10:15:32"
  minitool ts "2026-01-26T10:15:32Z"`,
	Args: cobra.ExactArgs(1),
	RunE: runTimestamp,
}

func runTimestamp(cmd *cobra.Command, args []string) error {
	input := strings.TrimSpace(args[0])

	// Try parsing as numeric timestamp first
	if ts, err := strconv.ParseInt(input, 10, 64); err == nil {
		return convertTimestampToDatetime(ts)
	}

	// Try parsing as datetime string
	return convertDatetimeToTimestamp(input)
}

func convertTimestampToDatetime(ts int64) error {
	var t time.Time
	var precision string

	digits := len(strconv.FormatInt(ts, 10))

	switch {
	case digits <= 10:
		// Seconds
		t = time.Unix(ts, 0)
		precision = "seconds"
	case digits <= 13:
		// Milliseconds
		t = time.UnixMilli(ts)
		precision = "milliseconds"
	case digits <= 16:
		// Microseconds
		t = time.UnixMicro(ts)
		precision = "microseconds"
	default:
		// Nanoseconds
		t = time.Unix(0, ts)
		precision = "nanoseconds"
	}

	utc := t.UTC()
	local := t.Local()

	fmt.Printf("UTC:   %s\n", utc.Format("2006-01-02 15:04:05"))
	fmt.Printf("Local: %s\n", local.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Precision: %s\n", precision)

	return nil
}

func convertDatetimeToTimestamp(input string) error {
	// Common datetime formats to try
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"01-02-2006 15:04:05",
		"01/02/2006 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	var t time.Time
	var err error
	parsed := false

	for _, format := range formats {
		t, err = time.Parse(format, input)
		if err == nil {
			parsed = true
			break
		}
		// Also try parsing in local timezone
		t, err = time.ParseInLocation(format, input, time.Local)
		if err == nil {
			parsed = true
			break
		}
	}

	if !parsed {
		return fmt.Errorf("unable to parse datetime: %s", input)
	}

	fmt.Printf("Unix (seconds):      %d\n", t.Unix())
	fmt.Printf("Unix (milliseconds): %d\n", t.UnixMilli())
	fmt.Printf("Unix (microseconds): %d\n", t.UnixMicro())
	fmt.Printf("Unix (nanoseconds):  %d\n", t.UnixNano())

	return nil
}
