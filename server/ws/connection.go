package ws

import (
	"encoding/binary"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/chacha20poly1305"

	"haven/shared"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendChSize = 256
)

// ReadPump reads messages from the WebSocket connection and dispatches them to the router.
func ReadPump(client *Client, router *Router) {
	defer client.Close()

	client.Conn.SetReadLimit(shared.MaxWSMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Debug("ws read error", "pubkey", client.PubKeyHex, "error", err)
			}
			return
		}

		// Decrypt if app-layer encryption is active
		if client.EncKey != nil {
			decrypted, err := decryptMessage(client.EncKey, client.RecvNonce, raw)
			if err != nil {
				slog.Warn("ws decrypt failed", "pubkey", client.PubKeyHex, "error", err)
				return
			}
			client.RecvNonce++
			raw = decrypted
		}

		router.Route(client, raw)
	}
}

// WritePump writes messages from the send channel to the WebSocket connection.
func WritePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Close()
	}()

	for {
		select {
		case msg, ok := <-client.SendCh:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			msgType := websocket.TextMessage
			data := msg

			// Encrypt if app-layer encryption is active
			if client.EncKey != nil {
				encrypted, err := encryptMessage(client.EncKey, client.SendNonce, msg)
				if err != nil {
					slog.Error("ws encrypt failed", "pubkey", client.PubKeyHex, "error", err)
					return
				}
				client.SendNonce++
				data = encrypted
				msgType = websocket.BinaryMessage
			}

			if err := client.Conn.WriteMessage(msgType, data); err != nil {
				slog.Debug("ws write error", "pubkey", client.PubKeyHex, "error", err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func encryptMessage(key []byte, nonce uint64, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	n := nonceBytes(nonce)
	return aead.Seal(nil, n, plaintext, nil), nil
}

func decryptMessage(key []byte, nonce uint64, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	n := nonceBytes(nonce)
	return aead.Open(nil, n, ciphertext, nil)
}

// nonceBytes encodes a counter as a 12-byte little-endian nonce for ChaCha20-Poly1305.
func nonceBytes(counter uint64) []byte {
	buf := make([]byte, chacha20poly1305.NonceSize) // 12 bytes
	binary.LittleEndian.PutUint64(buf, counter)
	return buf
}
