package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"

	"github.com/kiinoda/mailrelay/internal/config"
)

// SMTPClient interface for dependency injection in tests
type SMTPClient interface {
	StartTLS(config *tls.Config) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

// RealSMTPClient wraps net/smtp.Client to implement SMTPClient interface
type RealSMTPClient struct {
	*smtp.Client
}

func (r *RealSMTPClient) Close() error {
	return r.Client.Close()
}

// SMTPDialer function type for creating SMTP connections
type SMTPDialer func(addr string) (SMTPClient, error)

// Email represents an email message and provides methods for reading, parsing and sending
type Email struct {
	Body   []byte
	Config *config.Config
}

// New creates a new Email instance with the provided configuration and body,
// and parses recipients from the email
func New(cfg *config.Config, body []byte) (*Email, error) {
	email := &Email{
		Config: cfg,
		Body:   body,
	}

	if err := email.parseRecipients(); err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}
	return email, nil
}

// parseRecipients parses the email message and extracts recipients
func (e *Email) parseRecipients() error {
	msg, err := mail.ReadMessage(bytes.NewReader(e.Body))
	if err != nil {
		return err
	}

	// Assume we get some To, Cc and Bcc headers like these below.
	//
	// To: Foo<foo@domain.tld>, Bar <bar@domain.tld>
	// Cc: Baz<baz@domain.tld>
	// Bcc: Waldo <waldo@domain.tld>, xyzzy@domain.tld
	//
	// Our goal is to extract the values and set the array of recipients
	// to the one below.
	//
	// []string{"foo@domain.tld", "bar@domain.tld", "baz@domain.tld", "waldo@domain.tld", "xyzzy@domain.tld"}

	for _, h := range []string{"to", "cc", "bcc"} {
		headerValue := msg.Header.Get(h)
		if headerValue == "" {
			continue
		}
		for _, part := range strings.Split(headerValue, ",") {
			trimmed := strings.Trim(part, " ")
			if trimmed == "" {
				continue
			}
			regex := regexp.MustCompile(`.*<(.*)>`)
			matches := regex.FindStringSubmatch(trimmed)
			recipient := ""
			if len(matches) > 1 {
				recipient = matches[1]
			} else {
				recipient = trimmed
			}
			e.Config.Recipients = append(e.Config.Recipients, recipient)
		}
	}
	return nil
}

// Send attempts to send the email through one of the configured SMTP servers
func (e *Email) Send() error {
	return e.sendWithDialer(DefaultSMTPDialer)
}



// DefaultSMTPDialer creates real SMTP connections
func DefaultSMTPDialer(addr string) (SMTPClient, error) {
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}
	return &RealSMTPClient{Client: client}, nil
}

// sendWithDialer allows injection of custom dialer for testing
func (e *Email) sendWithDialer(dialer SMTPDialer) error {
	var err error
	// Try each SMTP server until one succeeds
	for _, server := range e.Config.SmtpAddrs {
		if err = e.attemptRelayWithDialer(server, dialer); err == nil {
			// Email sent successfully
			if e.Config.BeVerbose {
				fmt.Println("successfully sent mail from", e.Config.FromAddr, "to", e.Config.Recipients, "via", server)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to send email to any SMTP server: %w", err)
}

// attemptRelayWithDialer attempts to send email using provided dialer
func (e *Email) attemptRelayWithDialer(server string, dialer SMTPDialer) error {
	// Create a custom TLS config that skips certificate verification
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Connect to the SMTP server using dialer
	c, err := dialer(server)
	if err != nil {
		log.Println("error connecting to", server)
		return err
	}
	defer c.Close()

	// Start TLS with our custom config
	if err = c.StartTLS(tlsConfig); err != nil {
		log.Println("error starting TLS with", server)
		return err
	}

	// Set the sender
	if err = c.Mail(e.Config.FromAddr); err != nil {
		log.Println("error setting sender:", e.Config.FromAddr)
		return err
	}

	// Set recipients
	for _, addr := range e.Config.Recipients {
		if err = c.Rcpt(addr); err != nil {
			log.Println("error setting recipient:", addr)
			return err
		}
	}

	// Send the email body
	wc, err := c.Data()
	if err != nil {
		log.Println("error getting data writer")
		return err
	}

	if _, err = wc.Write(e.Body); err != nil {
		log.Println("error writing email body")
		wc.Close()
		return err
	}

	if err = wc.Close(); err != nil {
		log.Println("error closing data writer")
		return err
	}

	// Close the connection
	if err = c.Quit(); err != nil {
		log.Println("error closing connection")
		return err
	}

	return nil
}
