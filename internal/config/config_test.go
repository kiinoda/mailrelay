package config

import (
	"flag"
	"os"
	"reflect"
	"testing"
)

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		expectedSMTP    []string
		expectedFrom    string
		expectedVerbose bool
	}{
		{
			name: "Basic environment variables",
			envVars: map[string]string{
				MailRelayEnvVar: "smtp1.example.com:25;smtp2.example.com:25",
				SenderEnvVar:    "sender@example.com",
				VerboseEnvVar:   "true",
			},
			expectedSMTP:    []string{"smtp1.example.com:25", "smtp2.example.com:25"},
			expectedFrom:    "sender@example.com",
			expectedVerbose: true,
		},
		{
			name:            "No environment variables",
			envVars:         map[string]string{},
			expectedSMTP:    nil,
			expectedFrom:    "",
			expectedVerbose: false,
		},
		{
			name: "Invalid SMTP server",
			envVars: map[string]string{
				MailRelayEnvVar: "invalid-server;smtp2.example.com:25",
				SenderEnvVar:    "sender@example.com",
			},
			expectedSMTP:    []string{"smtp2.example.com:25"},
			expectedFrom:    "sender@example.com",
			expectedVerbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv(MailRelayEnvVar)
			os.Unsetenv(SenderEnvVar)
			os.Unsetenv(VerboseEnvVar)

			// Set environment variables for the test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Create a new config and parse environment
			cfg := &Config{}
			cfg.parseEnvironment()

			// Check SMTP servers
			if !reflect.DeepEqual(cfg.SmtpAddrs, tt.expectedSMTP) {
				t.Errorf("parseEnvironment() SMTP = %v, want %v", cfg.SmtpAddrs, tt.expectedSMTP)
			}

			// Check From address
			if cfg.FromAddr != tt.expectedFrom {
				t.Errorf("parseEnvironment() From = %v, want %v", cfg.FromAddr, tt.expectedFrom)
			}

			// Check Verbose flag
			if cfg.BeVerbose != tt.expectedVerbose {
				t.Errorf("parseEnvironment() BeVerbose = %v, want %v", cfg.BeVerbose, tt.expectedVerbose)
			}
		})
	}
}

func TestParseArguments(t *testing.T) {
	// Save original args and flags
	originalArgs := os.Args

	// Reset flags for each test to avoid "flag redefined" errors
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	tests := []struct {
		name               string
		args               []string
		expectedConfig     *Config
		expectedExitCode   int
		expectedExitCalled bool
	}{
		{
			name: "Basic arguments",
			args: []string{"mailrelay", "-f", "sender@example.com", "-v"},
			expectedConfig: &Config{
				FromAddr:  "sender@example.com",
				BeVerbose: true,
			},
		},
		{
			name: "Special case for -f flag",
			args: []string{"mailrelay", "-fsender@example.com"},
			expectedConfig: &Config{
				FromAddr:  "sender@example.com",
				BeVerbose: false,
			},
		},
		{
			name: "Help flag",
			args: []string{"mailrelay", "-h"},
			expectedConfig: &Config{
				ShowHelp:  true,
				BeVerbose: false,
			},
			expectedExitCode:   0,
			expectedExitCalled: true,
		},
	}

	// Save the original os.Exit and restore it after the test
	oldOsExit := osExit

	// Create variables to track if exit was called and with what code
	var exitCode int
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
		exitCode = code
	}

	defer func() { osExit = oldOsExit }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset exit tracking for each test case
			exitCalled = false
			exitCode = 0

			// Reset flags for each test to avoid "flag redefined" errors
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set command line arguments
			os.Args = tt.args

			// Create a new config and parse arguments
			cfg := &Config{}
			cfg.parseArguments()

			// Check From address
			if cfg.FromAddr != tt.expectedConfig.FromAddr {
				t.Errorf("parseArguments() FromAddr = %v, want %v", cfg.FromAddr, tt.expectedConfig.FromAddr)
			}

			// Check Verbose flag
			if cfg.BeVerbose != tt.expectedConfig.BeVerbose {
				t.Errorf("parseArguments() BeVerbose = %v, want %v", cfg.BeVerbose, tt.expectedConfig.BeVerbose)
			}

			// Check Help flag
			if cfg.ShowHelp != tt.expectedConfig.ShowHelp {
				t.Errorf("parseArguments() ShowHelp = %v, want %v", cfg.ShowHelp, tt.expectedConfig.ShowHelp)
			}

			// Check Return Code and if os.Exit has been called
			if tt.name == "Help flag" {
				if !exitCalled {
					t.Errorf("parseArguments() os.Exit was not called")
				}
				if exitCode != 0 {
					t.Errorf("parseArguments() did not exit with a 0 exit code")
				}
			}
		})
	}

	// Restore original arguments
	os.Args = originalArgs
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: &Config{
				SmtpAddrs: []string{"smtp.example.com:25"},
				FromAddr:  "sender@example.com",
			},
			expectError: false,
		},
		{
			name: "Missing SMTP servers",
			config: &Config{
				SmtpAddrs: []string{},
				FromAddr:  "sender@example.com",
			},
			expectError: true,
		},
		{
			name: "Missing sender",
			config: &Config{
				SmtpAddrs: []string{"smtp.example.com:25"},
				FromAddr:  "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateSettings()

			if (err != nil) != tt.expectError {
				t.Errorf("validateSettings() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestRandomizeSMTPServers(t *testing.T) {
	// Create a config with multiple SMTP servers
	cfg := &Config{
		SmtpAddrs: []string{
			"smtp1.example.com:25",
			"smtp2.example.com:25",
			"smtp3.example.com:25",
			"smtp4.example.com:25",
			"smtp5.example.com:25",
		},
	}

	// Make a copy of the original order
	originalOrder := make([]string, len(cfg.SmtpAddrs))
	copy(originalOrder, cfg.SmtpAddrs)

	// Since randomization is based on random numbers, we can't guarantee
	// the order will change, but we can verify that the function doesn't
	// lose any servers or add new ones
	cfg.randomizeSMTPServers()

	// Verify that no servers were lost during randomization
	if len(cfg.SmtpAddrs) != len(originalOrder) {
		t.Errorf("randomizeSMTPServers() changed the number of SMTP servers: got %d, want %d",
			len(cfg.SmtpAddrs), len(originalOrder))
	}

	// Create maps to check that all original servers are still present
	// and no new servers were added
	originalServerMap := make(map[string]bool)
	for _, server := range originalOrder {
		originalServerMap[server] = true
	}

	randomizedServerMap := make(map[string]bool)
	for _, server := range cfg.SmtpAddrs {
		randomizedServerMap[server] = true
		if !originalServerMap[server] {
			t.Errorf("randomizeSMTPServers() introduced a new server: %s", server)
		}
	}

	// Check that no servers were lost
	for _, server := range originalOrder {
		if !randomizedServerMap[server] {
			t.Errorf("randomizeSMTPServers() lost a server: %s", server)
		}
	}
}

func TestNew(t *testing.T) {
	// Save original environment and args
	originalEnv := os.Getenv(MailRelayEnvVar)
	originalSender := os.Getenv(SenderEnvVar)
	originalArgs := os.Args

	// Reset flags for the test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Restore environment and args after test
	defer func() {
		os.Setenv(MailRelayEnvVar, originalEnv)
		os.Setenv(SenderEnvVar, originalSender)
		os.Args = originalArgs
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		args        []string
		expectError bool
	}{
		{
			name: "Valid configuration from environment",
			envVars: map[string]string{
				MailRelayEnvVar: "smtp.example.com:25",
				SenderEnvVar:    "sender@example.com",
			},
			args:        []string{"mailrelay"},
			expectError: false,
		},
		{
			name: "Valid configuration from args",
			envVars: map[string]string{
				MailRelayEnvVar: "smtp.example.com:25",
			},
			args:        []string{"mailrelay", "-f", "sender@example.com"},
			expectError: false,
		},
		{
			name:        "Invalid configuration - missing SMTP",
			envVars:     map[string]string{},
			args:        []string{"mailrelay", "-f", "sender@example.com"},
			expectError: true,
		},
		{
			name: "Invalid configuration - missing sender",
			envVars: map[string]string{
				MailRelayEnvVar: "smtp.example.com:25",
			},
			args:        []string{"mailrelay"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Clear environment variables
			os.Unsetenv(MailRelayEnvVar)
			os.Unsetenv(SenderEnvVar)
			os.Unsetenv(VerboseEnvVar)

			// Set environment variables for the test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Set command line arguments
			os.Args = tt.args

			// Create a new config
			cfg, err := New()

			// Check if error matches expectation
			if (err != nil) != tt.expectError {
				t.Errorf("New() error = %v, expectError %v", err, tt.expectError)
				return
			}

			// If we expect success, verify the config is not nil
			if !tt.expectError && cfg == nil {
				t.Errorf("New() returned nil config when expecting success")
			}
		})
	}
}
