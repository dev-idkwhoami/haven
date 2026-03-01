package crypto

import (
	"crypto/ed25519"
	"fmt"

	"filippo.io/edwards25519"
)

// edwardsToMontgomery converts an Ed25519 public key (Edwards form) to an
// X25519 public key (Montgomery form) using the birational map.
func edwardsToMontgomery(edPub ed25519.PublicKey) ([]byte, error) {
	p, err := new(edwards25519.Point).SetBytes(edPub)
	if err != nil {
		return nil, fmt.Errorf("invalid edwards25519 point: %w", err)
	}
	return p.BytesMontgomery(), nil
}
