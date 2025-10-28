package ssh

import (
	"github.com/nathanlytang/rolodex/internal/logger"
	"golang.org/x/crypto/ssh"
)

// Adds password and keyboard-interactive authentication methods
// Password is tried first, keyboard-interactive as fallback for PAM
// Returns array of auth methods
func TryPasswordAuth(password string) []ssh.AuthMethod {
	logger.Printf("Adding password and keyboard-interactive authentication methods")

	var authMethods []ssh.AuthMethod

	authMethods = append(authMethods, ssh.Password(password))
	authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, len(questions))
		for i := range questions {
			answers[i] = password
		}
		return answers, nil
	}))

	return authMethods
}
