package config

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// Configuration constants
const (
	MailRelayEnvVar = "MAILRELAY_SERVERS"
	SenderEnvVar    = "MAILRELAY_FROM"
	VerboseEnvVar   = "MAILRELAY_VERBOSE"
)

// Config holds all the program configuration
type Config struct {
	BeVerbose  bool
	ShowHelp   bool
	FromAddr   string
	SmtpAddrs  []string
	Recipients []string
}

// New creates and initializes a new Config with values from
// environment variables and command-line arguments
func New() (*Config, error) {
	cfg := &Config{}

	cfg.parseArguments()
	cfg.parseEnvironment()

	if err := cfg.validateSettings(); err != nil {
		return nil, err
	}

	cfg.randomizeSMTPServers()

	return cfg, nil
}

// parseEnvironment reads configuration from environment variables
func (cfg *Config) parseEnvironment() {
	// Read SMTP servers
	if envServers := os.Getenv(MailRelayEnvVar); len(envServers) > 0 {
		relays := strings.Split(strings.Trim(envServers, "\""), ";")
		for _, s := range relays {
			_, _, err := net.SplitHostPort(s)
			if err != nil {
				fmt.Printf("invalid SMTP address: %s", s)
				continue
			}
			cfg.SmtpAddrs = append(cfg.SmtpAddrs, s)
		}
	}

	// Read sender address
	if envFrom := os.Getenv(SenderEnvVar); len(envFrom) > 0 {
		cfg.FromAddr = envFrom
	}

	// Read verbosity setting
	if len(os.Getenv(VerboseEnvVar)) > 0 {
		cfg.BeVerbose = true
	}
}

// parseArguments processes command line arguments
func (cfg *Config) parseArguments() {
	processedArgs := []string{}

	// Handle special case for -f flag
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-f") && len(arg) > 2 {
			processedArgs = append(processedArgs, "-f", arg[2:])
		} else {
			processedArgs = append(processedArgs, arg)
		}
	}

	// Define flags
	flag.BoolVar(&cfg.BeVerbose, "v", false, "set verbose output")
	flag.StringVar(&cfg.FromAddr, "f", "", "set sender")
	flag.BoolVar(&cfg.ShowHelp, "h", false, "show help")

	// Parse flags
	flag.CommandLine.Parse(processedArgs[1:])

	// Handle help flag
	if cfg.ShowHelp {
		flag.CommandLine.Usage()
		os.Exit(0)
	}
}

// validateSettings ensures all required settings are provided
func (cfg *Config) validateSettings() error {
	if len(cfg.SmtpAddrs) == 0 {
		return fmt.Errorf("at least one SMTP address is required to continue, set %s", MailRelayEnvVar)
	}

	if cfg.FromAddr == "" {
		return fmt.Errorf("either pass sender using -f or set %s", SenderEnvVar)
	}

	return nil
}

// randomizeSMTPServers randomly shuffles the list of SMTP servers
func (cfg *Config) randomizeSMTPServers() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for k := range cfg.SmtpAddrs {
		idx1 := (r.Int() % (k + 1))
		idx2 := (r.Int() % (k + 1))
		cfg.SmtpAddrs[idx1], cfg.SmtpAddrs[idx2] = cfg.SmtpAddrs[idx2], cfg.SmtpAddrs[idx1]
	}
}
