package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kiinoda/mailrelay/internal/config"
	"github.com/kiinoda/mailrelay/internal/email"
	"github.com/kiinoda/mailrelay/internal/exitcode"
)

func main() {
	// Load configuration
	cfg, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(exitcode.ConfigError)
	}

	// Read email from stdin
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(exitcode.IOError)
	}

	// Create email instance with body
	mail, err := email.New(cfg, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing message body: %v\n", err)
		os.Exit(exitcode.ParseError)
	}

	// Send email
	if err := mail.Send(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to send email: %v\n", err)
		os.Exit(exitcode.SendError)
	}

	// Successfully sent email
	os.Exit(exitcode.Success)
}
