package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"

	"crypto/sha256"
	"io"
)

// GenerateKeyPair creates a new Ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate keypair: %w", err)
	}
	return pub, priv, nil
}

// Sign signs a message with the Ed25519 private key.
func Sign(privKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privKey, message)
}

// Verify checks an Ed25519 signature.
func Verify(pubKey ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pubKey, message, sig)
}

// SignMessage creates a signature for a channel message: content || channelID || timestamp || nonce.
func SignMessage(privKey ed25519.PrivateKey, content, channelID string, timestamp int64, nonce []byte) []byte {
	var buf []byte
	buf = append(buf, []byte(content)...)
	buf = append(buf, []byte(channelID)...)
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(timestamp))
	buf = append(buf, ts...)
	buf = append(buf, nonce...)
	return Sign(privKey, buf)
}

// VerifyMessage verifies a channel message signature.
func VerifyMessage(pubKey ed25519.PublicKey, content, channelID string, timestamp int64, nonce, sig []byte) bool {
	var buf []byte
	buf = append(buf, []byte(content)...)
	buf = append(buf, []byte(channelID)...)
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(timestamp))
	buf = append(buf, ts...)
	buf = append(buf, nonce...)
	return Verify(pubKey, buf, sig)
}

// SignChallenge signs challengeNonce || serverPubKey for the auth handshake.
func SignChallenge(privKey ed25519.PrivateKey, challengeNonce, serverPubKey []byte) []byte {
	msg := append(challengeNonce, serverPubKey...)
	return Sign(privKey, msg)
}

// RandomNonce generates n random bytes.
func RandomNonce(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("random nonce: %w", err)
	}
	return buf, nil
}

// Ed25519PrivateToX25519 converts an Ed25519 private key to an X25519 private key.
func Ed25519PrivateToX25519(privKey ed25519.PrivateKey) []byte {
	h := sha512.Sum512(privKey.Seed())
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	return h[:32]
}

// Ed25519PublicToX25519 converts an Ed25519 public key to an X25519 public key.
func Ed25519PublicToX25519(pubKey ed25519.PublicKey) ([]byte, error) {
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key length: %d", len(pubKey))
	}
	// Use the extra25519 approach: Edwards -> Montgomery conversion
	// For Ed25519, there's a direct conversion from the Edwards y-coordinate.
	edPoint := make([]byte, 32)
	copy(edPoint, pubKey)

	// Convert Edwards y-coordinate to Montgomery u-coordinate:
	// u = (1 + y) / (1 - y) mod p
	// We use the golang.org/x/crypto/curve25519 internal representation.
	return edwardsToMontgomery(edPoint)
}

// DeriveWSEncryptionKey derives the ChaCha20-Poly1305 key for WS app-layer encryption.
func DeriveWSEncryptionKey(myPrivKey ed25519.PrivateKey, peerPubKey ed25519.PublicKey, serverNonce, challengeNonce []byte) ([]byte, error) {
	myX25519 := Ed25519PrivateToX25519(myPrivKey)

	peerX25519, err := Ed25519PublicToX25519(peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("convert peer pubkey: %w", err)
	}

	shared, err := curve25519.X25519(myX25519, peerX25519)
	if err != nil {
		return nil, fmt.Errorf("x25519 DH: %w", err)
	}

	salt := append(serverNonce, challengeNonce...)
	hkdfReader := hkdf.New(sha256.New, shared, salt, []byte("haven-ws-encryption"))

	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("hkdf derive: %w", err)
	}
	return key, nil
}

// DeriveSQLCipherKey derives the SQLCipher encryption key from an Ed25519 private key seed.
func DeriveSQLCipherKey(privKey ed25519.PrivateKey) (string, error) {
	seed := privKey.Seed()
	hkdfReader := hkdf.New(sha256.New, seed, []byte("haven-sqlcipher-v1"), []byte("database-encryption-key"))

	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return "", fmt.Errorf("hkdf derive sqlcipher key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// DeriveDMSharedKey derives a shared key for 1:1 DM encryption via X25519 + HKDF.
func DeriveDMSharedKey(myPrivKey ed25519.PrivateKey, peerPubKey ed25519.PublicKey) ([]byte, error) {
	myX25519 := Ed25519PrivateToX25519(myPrivKey)

	peerX25519, err := Ed25519PublicToX25519(peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("convert peer pubkey: %w", err)
	}

	shared, err := curve25519.X25519(myX25519, peerX25519)
	if err != nil {
		return nil, fmt.Errorf("x25519 DH: %w", err)
	}

	hkdfReader := hkdf.New(sha256.New, shared, nil, []byte("haven-dm-encryption"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("hkdf derive dm key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext with ChaCha20-Poly1305 using the given key and nonce counter.
func Encrypt(key []byte, nonceCounter uint64, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	binary.LittleEndian.PutUint64(nonce, nonceCounter)

	return aead.Seal(nil, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext with ChaCha20-Poly1305 using the given key and nonce counter.
func Decrypt(key []byte, nonceCounter uint64, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	binary.LittleEndian.PutUint64(nonce, nonceCounter)

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptBlob encrypts plaintext with a random nonce prepended (for DM messages).
func EncryptBlob(key, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("random nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBlob decrypts ciphertext that has the nonce prepended (for DM messages).
func DecryptBlob(key, blob []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	if len(blob) < chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := blob[:chacha20poly1305.NonceSize]
	ciphertext := blob[chacha20poly1305.NonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt blob: %w", err)
	}
	return plaintext, nil
}

// HexEncode encodes bytes to a hex string.
func HexEncode(b []byte) string {
	return hex.EncodeToString(b)
}

// HexDecode decodes a hex string to bytes.
func HexDecode(s string) ([]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("hex decode: %w", err)
	}
	return b, nil
}
