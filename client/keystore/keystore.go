package keystore

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "haven-client"
	keyName     = "identity-key"
)

// Store saves the Ed25519 private key to the OS credential store.
func Store(privKey ed25519.PrivateKey) error {
	encoded := hex.EncodeToString(privKey)
	if err := keyring.Set(serviceName, keyName, encoded); err != nil {
		return fmt.Errorf("keystore store: %w", err)
	}
	return nil
}

// Load retrieves the Ed25519 private key from the OS credential store.
// Returns nil, nil if no key is stored.
func Load() (ed25519.PrivateKey, error) {
	encoded, err := keyring.Get(serviceName, keyName)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("keystore load: %w", err)
	}

	b, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("keystore decode: %w", err)
	}

	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("keystore: invalid key length %d", len(b))
	}

	return ed25519.PrivateKey(b), nil
}

// Delete removes the stored private key from the OS credential store.
func Delete() error {
	if err := keyring.Delete(serviceName, keyName); err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("keystore delete: %w", err)
	}
	return nil
}

// Exists checks whether a private key is stored.
func Exists() (bool, error) {
	_, err := keyring.Get(serviceName, keyName)
	if err != nil {
		if err == keyring.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("keystore exists: %w", err)
	}
	return true, nil
}
