// backend/internal/email/mock_email.go

package email

import (
	"sync"
)

// MockEmailService is a mock implementation of the email service for testing
type MockEmailService struct {
	sentEmails     []MockEmail
	sendEmailError error
	mu             sync.Mutex
}

// MockEmail represents a sent email for testing
type MockEmail struct {
	To          string
	Token       string
	Username    string
	DisplayName string
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sentEmails: make([]MockEmail, 0),
	}
}

// SendPasswordResetEmail records the email that would be sent
func (m *MockEmailService) SendPasswordResetEmail(to, token, username, displayName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentEmails = append(m.sentEmails, MockEmail{
		To:          to,
		Token:       token,
		Username:    username,
		DisplayName: displayName,
	})

	return m.sendEmailError
}

// SetSendEmailError sets an error to be returned by SendPasswordResetEmail
func (m *MockEmailService) SetSendEmailError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendEmailError = err
}

// GetSentEmails returns all emails that have been "sent"
func (m *MockEmailService) GetSentEmails() []MockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid race conditions
	result := make([]MockEmail, len(m.sentEmails))
	copy(result, m.sentEmails)
	return result
}

// ClearSentEmails clears the list of sent emails
func (m *MockEmailService) ClearSentEmails() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = make([]MockEmail, 0)
}
