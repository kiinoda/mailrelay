package exitcode

// Standard exit codes follow Unix/Linux conventions:
// - 0: Success
// - 1-63: Application-specific error codes
// - 64-78: Command-line usage errors (based on sysexits.h)
// - 126-165: Shell/OS standard error codes
// - 255: Out of range (will be modulo'd to lower value)

const (
	// Success indicates successful completion
	Success = 0

	// ConfigError indicates an error in configuration
	ConfigError = 1

	// SendError indicates a failure to send email
	SendError = 2

	// IOError indicates a failure with input/output operations
	IOError = 3

	// ParseError indicates a failure to parse data
	ParseError = 4
)
