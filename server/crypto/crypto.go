package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"os"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"

	"crypto/sha256"
	"io"

	"haven/shared"
)

// LoadOrGenerateKey loads an Ed25519 private key from file, or generates and saves a new one.
func LoadOrGenerateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid key file: expected %d bytes, got %d", ed25519.PrivateKeySize, len(data))
		}
		return ed25519.PrivateKey(data), nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	if err := os.WriteFile(path, priv, 0600); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}
	return priv, nil
}

// SignNonce signs a nonce with the given Ed25519 private key.
func SignNonce(key ed25519.PrivateKey, nonce []byte) []byte {
	return ed25519.Sign(key, nonce)
}

// VerifySignature verifies an Ed25519 signature.
func VerifySignature(pubKey ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pubKey, message, sig)
}

// Ed25519PrivateToX25519 converts an Ed25519 private key to an X25519 private key.
func Ed25519PrivateToX25519(edPriv ed25519.PrivateKey) []byte {
	h := sha512.Sum512(edPriv.Seed())
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	return h[:32]
}

// Ed25519PublicToX25519 converts an Ed25519 public key to an X25519 public key.
func Ed25519PublicToX25519(edPub ed25519.PublicKey) ([]byte, error) {
	if len(edPub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key length: %d", len(edPub))
	}
	// Use the Montgomery form conversion via the edwards25519 field operations.
	// The standard approach: compute the birational map from Edwards to Montgomery.
	return edwardsToMontgomery(edPub)
}

// DeriveSharedKey performs X25519 DH and derives a 32-byte ChaCha20-Poly1305 key via HKDF-SHA256.
func DeriveSharedKey(privKey ed25519.PrivateKey, peerPubKey ed25519.PublicKey, serverNonce, challengeNonce []byte) ([]byte, error) {
	x25519Priv := Ed25519PrivateToX25519(privKey)
	x25519Pub, err := Ed25519PublicToX25519(peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("convert peer public key: %w", err)
	}

	rawShared, err := curve25519.X25519(x25519Priv, x25519Pub)
	if err != nil {
		return nil, fmt.Errorf("x25519 key exchange: %w", err)
	}

	salt := make([]byte, 0, len(serverNonce)+len(challengeNonce))
	salt = append(salt, serverNonce...)
	salt = append(salt, challengeNonce...)

	hkdfReader := hkdf.New(sha256.New, rawShared, salt, []byte(shared.HKDFInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("hkdf derive: %w", err)
	}
	return key, nil
}

// GenerateNonce creates a 32-byte random nonce.
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

// GenerateSessionToken creates a cryptographically random base64 session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}
