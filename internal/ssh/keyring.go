package ssh

import (
	"unicode/utf16"

	"github.com/nathanlytang/rolodex/internal/logger"
	"github.com/zalando/go-keyring"
)

// Stores a password in the OS keyring
func StoreInKeyring(service, account, password string) error {
	return keyring.Set(service, account, password)
}

// Removes a password from the OS keyring
func DeleteFromKeyring(service, account string) error {
	return keyring.Delete(service, account)
}

// Retrieves a password from the OS keyring
func GetPasswordFromKeyring(service, account string) (string, error) {
	if service == "" || account == "" {
		return "", keyring.ErrNotFound
	}
	password, err := keyring.Get(service, account)
	if err != nil {
		logger.Printf("Failed to retrieve password from keyring for %s/%s: %v", service, account, err)
		return "", err
	}

	// Check if password is UTF-16LE encoded
	if len(password) > 1 && password[1] == 0 {
		// Convert UTF-16LE bytes to UTF-8 string
		passwordBytes := []byte(password)
		if len(passwordBytes)%2 != 0 {
			return password, nil // Odd length, can't be valid UTF-16LE
		}

		// Convert byte slice to uint16 slice
		utf16Slice := make([]uint16, len(passwordBytes)/2)
		for i := range utf16Slice {
			utf16Slice[i] = uint16(passwordBytes[i*2]) | uint16(passwordBytes[i*2+1])<<8
		}

		// Decode UTF-16LE to UTF-8
		runes := utf16.Decode(utf16Slice)
		password = string(runes)
	}

	logger.Printf("Successfully retrieved password from keyring for %s/%s", service, account)
	return password, nil
}
