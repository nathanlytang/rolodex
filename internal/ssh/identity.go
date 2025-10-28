package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathanlytang/rolodex/internal/logger"
	"golang.org/x/crypto/ssh"
)

// Attempts to load and parse an SSH private key file
// Returns nil if the file cannot be loaded or parsed
func TryIdentityFile(identityFile, passphrase string) ssh.AuthMethod {
	if identityFile == "" {
		return nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(identityFile, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Printf("Failed to get home directory: %v", err)
			return nil
		}
		identityFile = filepath.Join(home, identityFile[1:])
	}

	// Read the private key file
	keyData, err := os.ReadFile(identityFile)
	if err != nil {
		logger.Printf("Failed to read identity file %s: %v", identityFile, err)
		return nil
	}

	// Try to parse the key without passphrase first
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
			if err != nil {
				logger.Printf("Failed to parse identity file %s with passphrase: %v", identityFile, err)
				return nil
			}
			logger.Printf("Successfully loaded encrypted identity file: %s", identityFile)
		} else {
			if strings.Contains(err.Error(), "encrypted") || strings.Contains(err.Error(), "passphrase") {
				logger.Printf("Identity file %s is encrypted but no passphrase provided", identityFile)
				return nil
			}
			logger.Printf("Failed to parse identity file %s: %v", identityFile, err)
			return nil
		}
	} else {
		logger.Printf("Successfully loaded identity file: %s", identityFile)
	}

	return ssh.PublicKeys(signer)
}

// Returns the fingerprint of a public key for identification
func GetKeyFingerprint(signer ssh.Signer) string {
	return ssh.FingerprintSHA256(signer.PublicKey())
}

// Returns common SSH key file locations
func ListCommonKeyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	sshDir := filepath.Join(home, ".ssh")
	return []string{
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_dsa"),
	}
}

// Scans common locations for SSH keys
func FindAvailableKeys() []string {
	var availableKeys []string
	for _, keyPath := range ListCommonKeyPaths() {
		if _, err := os.Stat(keyPath); err == nil {
			availableKeys = append(availableKeys, keyPath)
		}
	}
	return availableKeys
}

// Checks if a key file exists and is readable
func ValidateKeyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("key file not accessible: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check permissions (should not be world-readable)
	mode := info.Mode()
	if mode.Perm()&0044 != 0 {
		logger.Printf("Warning: key file %s has overly permissive permissions: %v", path, mode.Perm())
	}

	return nil
}
