package email

import (
	"crypto/tls"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kiinoda/mailrelay/internal/config"
)

// Test constants
const (
	testFromAddr = "test@example.com"
	testSMTPAddr = "smtp.example.com:587"
)

var testAllRecipients = []string{"foo@domain.tld", "bar@domain.tld", "baz@domain.tld", "waldo@domain.tld", "xyzzy@domain.tld"}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantErr  bool
		expected []string
	}{
		{
			name:     "valid email with To, Cc, Bcc",
			body:     "From: sender@example.com\nTo: Foo<" + testAllRecipients[0] + ">, Bar <" + testAllRecipients[1] + ">\nCc: Baz<" + testAllRecipients[2] + ">\nBcc: Waldo <" + testAllRecipients[3] + ">, " + testAllRecipients[4] + "\nSubject: Test\n\nBody content",
			wantErr:  false,
			expected: testAllRecipients,
		},
		{
			name:     "email with only To header",
			body:     "From: sender@example.com\nTo: single@domain.tld\nSubject: Test\n\nBody content",
			wantErr:  false,
			expected: []string{"single@domain.tld"},
		},
		{
			name:     "email with no recipients",
			body:     "From: sender@example.com\nSubject: Test\n\nBody content",
			wantErr:  false,
			expected: []string{},
		},
		{
			name:     "invalid email format",
			body:     "invalid email format",
			wantErr:  true,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				FromAddr:   testFromAddr,
				SmtpAddrs:  []string{testSMTPAddr},
				Recipients: []string{}, // Will be populated by parseRecipients
			}

			email, err := New(cfg, []byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(email.Config.Recipients, tt.expected) {
					t.Errorf("New() recipients = %v, want %v", email.Config.Recipients, tt.expected)
				}
			}
		})
	}
}

func TestNewWithTestData(t *testing.T) {
	// Read test email from testdata
	testDataPath := filepath.Join("..", "..", "testdata", "body")
	body, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	cfg := &config.Config{
		FromAddr:   testFromAddr,
		SmtpAddrs:  []string{testSMTPAddr},
		Recipients: []string{},
	}

	email, err := New(cfg, body)
	if err != nil {
		t.Errorf("New() with test data failed: %v", err)
		return
	}

	expected := testAllRecipients
	if !reflect.DeepEqual(email.Config.Recipients, expected) {
		t.Errorf("New() with test data recipients = %v, want %v", email.Config.Recipients, expected)
	}
}

func TestEmailStruct(t *testing.T) {
	cfg := &config.Config{
		FromAddr:  testFromAddr,
		SmtpAddrs: []string{testSMTPAddr},
	}
	body := []byte("test email body")

	email := &Email{
		Config: cfg,
		Body:   body,
	}

	if email.Config != cfg {
		t.Error("Email.Config not set correctly")
	}

	if string(email.Body) != "test email body" {
		t.Error("Email.Body not set correctly")
	}
}

// SMTP Client mocking implementations

// MockSMTPClient implements SMTPClient for testing
type MockSMTPClient struct {
	ShouldFailOn     string // Which method should fail: "dial", "tls", "mail", "rcpt", "data", "write", "close", "quit"
	FailOnRecipient  string // Specific recipient to fail on
	DataWriter       *MockWriteCloser
	MethodCallCount  map[string]int
}

type MockWriteCloser struct {
	ShouldFailWrite bool
	ShouldFailClose bool
	Written         []byte
}

func (m *MockWriteCloser) Write(p []byte) (n int, err error) {
	if m.ShouldFailWrite {
		return 0, errors.New("mock write error")
	}
	m.Written = append(m.Written, p...)
	return len(p), nil
}

func (m *MockWriteCloser) Close() error {
	if m.ShouldFailClose {
		return errors.New("mock close error")
	}
	return nil
}

func NewMockSMTPClient() *MockSMTPClient {
	return &MockSMTPClient{
		DataWriter: &MockWriteCloser{},
		MethodCallCount: make(map[string]int),
	}
}

func (m *MockSMTPClient) StartTLS(config *tls.Config) error {
	m.MethodCallCount["StartTLS"]++
	if m.ShouldFailOn == "tls" {
		return errors.New("mock TLS error")
	}
	return nil
}

func (m *MockSMTPClient) Mail(from string) error {
	m.MethodCallCount["Mail"]++
	if m.ShouldFailOn == "mail" {
		return errors.New("mock mail error")
	}
	return nil
}

func (m *MockSMTPClient) Rcpt(to string) error {
	m.MethodCallCount["Rcpt"]++
	if m.ShouldFailOn == "rcpt" || (m.FailOnRecipient != "" && to == m.FailOnRecipient) {
		return errors.New("mock rcpt error")
	}
	return nil
}

func (m *MockSMTPClient) Data() (io.WriteCloser, error) {
	m.MethodCallCount["Data"]++
	if m.ShouldFailOn == "data" {
		return nil, errors.New("mock data error")
	}
	if m.ShouldFailOn == "write" {
		m.DataWriter.ShouldFailWrite = true
	}
	if m.ShouldFailOn == "close" {
		m.DataWriter.ShouldFailClose = true
	}
	return m.DataWriter, nil
}

func (m *MockSMTPClient) Quit() error {
	m.MethodCallCount["Quit"]++
	if m.ShouldFailOn == "quit" {
		return errors.New("mock quit error")
	}
	return nil
}

func (m *MockSMTPClient) Close() error {
	m.MethodCallCount["Close"]++
	return nil
}

func createMockDialer(client *MockSMTPClient, shouldFailDial bool) SMTPDialer {
	return func(addr string) (SMTPClient, error) {
		if shouldFailDial {
			return nil, errors.New("mock dial error")
		}
		return client, nil
	}
}

func TestSendSuccessful(t *testing.T) {
	mockClient := NewMockSMTPClient()
	dialer := createMockDialer(mockClient, false)
	
	cfg := &config.Config{
		FromAddr:   testFromAddr,
		SmtpAddrs:  []string{testSMTPAddr},
		Recipients: []string{"test@domain.tld"},
		BeVerbose:  false,
	}
	
	email := &Email{
		Config: cfg,
		Body:   []byte("test email body"),
	}
	
	// Test successful attempt
	err := email.attemptRelayWithDialer(testSMTPAddr, dialer)
	if err != nil {
		t.Errorf("attemptRelay() failed unexpectedly: %v", err)
	}
	
	// Verify all methods were called
	expectedCalls := map[string]int{
		"StartTLS": 1,
		"Mail":     1,
		"Rcpt":     1,
		"Data":     1,
		"Quit":     1,
		"Close":    1,
	}
	
	for method, expectedCount := range expectedCalls {
		if mockClient.MethodCallCount[method] != expectedCount {
			t.Errorf("Expected %s to be called %d times, got %d", method, expectedCount, mockClient.MethodCallCount[method])
		}
	}
	
	// Verify email body was written
	if string(mockClient.DataWriter.Written) != "test email body" {
		t.Errorf("Expected email body to be written, got: %s", string(mockClient.DataWriter.Written))
	}
}

func TestSendFailureScenarios(t *testing.T) {
	tests := []struct {
		name           string
		failOn         string
		failOnDial     bool
		failRecipient  string
		expectError    bool
	}{
		{"dial failure", "", true, "", true},
		{"TLS failure", "tls", false, "", true},
		{"mail failure", "mail", false, "", true},
		{"rcpt failure", "rcpt", false, "", true},
		{"specific recipient failure", "", false, "test@domain.tld", true},
		{"data failure", "data", false, "", true},
		{"write failure", "write", false, "", true},
		{"close failure", "close", false, "", true},
		{"quit failure", "quit", false, "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockSMTPClient()
			mockClient.ShouldFailOn = tt.failOn
			mockClient.FailOnRecipient = tt.failRecipient
			dialer := createMockDialer(mockClient, tt.failOnDial)
			
			cfg := &config.Config{
				FromAddr:   testFromAddr,
				SmtpAddrs:  []string{testSMTPAddr},
				Recipients: []string{"test@domain.tld"},
			}
			
			email := &Email{
				Config: cfg,
				Body:   []byte("test email body"),
			}
			
			err := email.attemptRelayWithDialer(testSMTPAddr, dialer)
			if (err != nil) != tt.expectError {
				t.Errorf("attemptRelay() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSendWithMultipleServers(t *testing.T) {
	// First server fails, second succeeds
	failingClient := NewMockSMTPClient()
	failingClient.ShouldFailOn = "tls"
	
	successfulClient := NewMockSMTPClient()
	
	callCount := 0
	dialer := func(addr string) (SMTPClient, error) {
		callCount++
		if callCount == 1 {
			return failingClient, nil
		}
		return successfulClient, nil
	}
	
	cfg := &config.Config{
		FromAddr:   testFromAddr,
		SmtpAddrs:  []string{"smtp1.example.com:587", "smtp2.example.com:587"},
		Recipients: []string{"test@domain.tld"},
		BeVerbose:  true,
	}
	
	email := &Email{
		Config: cfg,
		Body:   []byte("test email body"),
	}
	
	err := email.sendWithDialer(dialer)
	if err != nil {
		t.Errorf("Send() should succeed with fallback server, got error: %v", err)
	}
	
	// Verify second client was used successfully
	if successfulClient.MethodCallCount["Quit"] != 1 {
		t.Error("Second server should have been used successfully")
	}
}
