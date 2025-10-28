package ssh

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/nathanlytang/rolodex/internal/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Authentication configuration options
type AuthConfig struct {
	SSHAgent           bool
	IdentityFile       string
	IdentityPassphrase string
	KeyringService     string
	KeyringAccount     string
	Password           string
}

// Creates authentication methods in priority order
// Returns array of auth methods
func buildAuthMethods(config AuthConfig) []ssh.AuthMethod {
	var authMethods []ssh.AuthMethod

	logger.Printf("Building authentication methods for %v", config)

	if config.SSHAgent {
		if agentAuth := TrySSHAgent(); agentAuth != nil {
			authMethods = append(authMethods, agentAuth)
		}
	}

	if config.IdentityFile != "" {
		if keyAuth := TryIdentityFile(config.IdentityFile, config.IdentityPassphrase); keyAuth != nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	if config.KeyringService != "" && config.KeyringAccount != "" {
		password, err := GetPasswordFromKeyring(config.KeyringService, config.KeyringAccount)
		if err == nil && password != "" {
			authMethods = append(authMethods, TryPasswordAuth(password)...)
		}
	}

	if config.Password != "" {
		authMethods = append(authMethods, TryPasswordAuth(config.Password)...)
	}

	logger.Printf("Total authentication methods configured: %d", len(authMethods))
	return authMethods
}

// Connects to an SSH server using multiple authentication methods with priority
// Returns error if connection fails
func StartSession(host string, port int, user string, authConfig AuthConfig) error {
	logger.Printf("Attempting connection to %s@%s:%d", user, host, port)

	address := host + ":" + strconv.Itoa(port)
	logger.Printf("Testing TCP connection to %s...", address)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return logger.Fatalf("Cannot reach %s - TCP connection failed: %v\nCheck firewall, DNS, and network connectivity", address, err)
	}
	conn.Close()
	logger.Printf("TCP connection successful, attempting SSH handshake...")

	authMethods := buildAuthMethods(authConfig)

	if len(authMethods) == 0 {
		return logger.Fatal("No authentication method available. Configure at least one: ssh_agent, identity_file, keyring, or password.")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		if authErr, ok := err.(*ssh.ServerAuthError); ok {
			logger.Printf("Authentication methods we tried: %d methods", len(authMethods))
			return logger.Fatalf("SSH authentication failed!\nErrors from server: %v\nFull error: %v", authErr.Errors, err)
		}
		return logger.Fatalf("SSH connection failed: %v", err)
	}
	defer client.Close()

	logger.Printf("SSH connection established successfully!")

	session, err := client.NewSession()
	if err != nil {
		return logger.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	// Put the local terminal into raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return logger.Fatalf("Failed to set raw mode: %v", err)
	}
	defer term.Restore(fd, oldState) // always restore

	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 80, 24
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return logger.Fatalf("Request for pseudo terminal failed: %v", err)
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Shell(); err != nil {
		return logger.Fatalf("Failed to start shell: %v", err)
	}
	session.Wait()
	return nil
}
