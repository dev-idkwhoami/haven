# Haven

A self-hosted, decentralized messaging platform. Your identity is a keypair. The server is a blind relay. All security is enforced cryptographically — not by policy, not by trusting the platform.

---

## What is Haven?

Haven is an open-source alternative to platforms like Discord and TeamSpeak, built for communities that want to own their infrastructure and not compromise on privacy. There are no accounts, no central servers, and no operator that can read your messages. You run the server, you own the data.

---

## Features

- Keypair-based identity — no registration, no passwords
- End-to-end encrypted direct messages
- Encrypted private channels with per-member key distribution
- Tamper-evident message history via hash chaining
- Voice chat via SFU — other participants never see your IP
- Self-hostable server with a minimal footprint
- Designed to run behind a reverse proxy or Cloudflare

---

## How It Works

### Identity

Every user has an asymmetric keypair. The public key *is* the identity — there is no username, no password, and no central authority that issues or owns it. Your identity string (`haven:v1:ed25519:...`) is portable and self-sovereign. The private key never leaves your device. Because identity is just a keypair, you can connect to any Haven server without registering — the server has no concept of "your account."

### Connecting to a Server

Servers also have keypairs. When a client connects, both sides exchange public keys. The client verifies that the server's identity matches what it saw on the last connection — trust-on-first-use, similar to SSH. If the server key changes unexpectedly, the client warns the user. Beyond the handshake, the server operates as a blind relay: it routes messages without being able to read them.

### Transport

All client-server communication happens over a persistent WebSocket connection wrapped in TLS. TLS protects against passive eavesdropping at the network level, but Haven does not rely on it for message confidentiality. Every message is independently signed or encrypted at the application layer before hitting the transport. Even if TLS were stripped or the server operator inspected raw traffic, message content would remain protected. The WebSocket channel carries typed frames — chat messages, presence updates, voice signaling, DM payloads — each cryptographically secured on its own.

### Channels

A server hosts multiple named text channels. Clients subscribe to the channels they want to receive messages from. Channels can be public (open to any connected user) or private (access requires holding the channel's encryption key, distributed only to members).

### Messaging

Public channel messages are signed by the sender's private key before transmission. Recipients verify the signature against the sender's known public key, confirming authorship and that the content has not been altered in transit or on the server.

Messages are hash-chained: each message includes a hash of the previous one, forming an append-only ledger. Edits are new messages that reference the original — nothing is overwritten. This makes history tamper-evident. No party, including the server operator, can silently alter or remove a message. Deletions appear as tombstone entries in the chain; the cryptographic record remains intact.

### Direct Messages

DMs use an X25519 Diffie-Hellman key exchange to derive a shared secret between two parties. No key material is transmitted — both sides independently derive the same secret from their respective keypairs. The server sees only an encrypted blob and opaque routing metadata. It cannot read DM content.

### Private Channels

When a private channel is created, a symmetric channel key is generated. That key is encrypted individually with each member's public key and stored on the server. When a new member is added, the key is re-encrypted for them. The server stores encrypted key bundles only — it never holds the plaintext channel key. Users without membership cannot decrypt the channel's history regardless of server-level access.

### Voice Chat

Haven uses an SFU (Selective Forwarding Unit) for voice rather than peer-to-peer mesh. All participants connect to the server, which forwards audio streams between them. Audio never flows directly between clients.

This was a deliberate design decision. Peer-to-peer WebRTC mesh exposes every participant's real IP address to every other participant during ICE negotiation — this is the same issue that makes TeamSpeak widely distrusted. With an SFU, other users in a voice channel never learn your IP. The SFU also scales better: mesh requires every client to maintain a connection to every other client, which degrades past 5-6 people. With an SFU, each client maintains a single connection to the server regardless of channel size.

---

## Privacy — Honest Scope

Haven draws a clear line between two threat models and is transparent about what it does and does not address.

**Protection from other users** — fully addressed. Text chat can be proxied so other users never see your real IP. Voice through the SFU means other participants never receive your IP during WebRTC negotiation either.

**Protection from the server operator** — not addressed, and Haven makes no false promises here. The server operator, like any web host, will see your IP address. This is an unavoidable property of real-time networked communication. Alternatives like Tor or i2P were considered and rejected for voice specifically: the latency and jitter they introduce make real-time audio unusable. Users who want additional protection at the transport level can route their own traffic through a VPN — Haven does not prevent this, but does not depend on it.

The practical privacy story: Haven protects you from other users. For protection from the server operator, choose operators you trust — or self-host.

---

## Deployment

Haven is designed to run behind a reverse proxy. Operators should place Nginx, Caddy, or equivalent in front of the WebSocket endpoint. Optionally, a CDN-level proxy such as Cloudflare can sit in front of that. This means the Haven process never directly handles raw internet traffic, reducing attack surface and ensuring the application layer never sees user IPs directly.

---

## Self-Hosting

> Setup documentation coming soon.

---

## License

MIT — see [LICENSE](LICENSE) for details.
