package ssh

import (
	"net"
	"os"

	"github.com/nathanlytang/rolodex/internal/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Attempts to connect to the SSH agent and returns an AuthMethod if successful
func TrySSHAgent() ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		logger.Printf("SSH agent not available (SSH_AUTH_SOCK not set)")
		return nil
	}

	// On Windows, SSH agent uses named pipes; on Unix, it uses Unix sockets
	network := "unix"
	if len(socket) > 0 && socket[0] == '\\' {
		network = "pipe" // Windows named pipe
	}

	conn, err := net.Dial(network, socket)
	if err != nil {
		logger.Printf("Failed to connect to SSH agent: %v", err)
		return nil
	}

	agentClient := agent.NewClient(conn)
	logger.Printf("Successfully connected to SSH agent")
	return ssh.PublicKeysCallback(agentClient.Signers)
}
