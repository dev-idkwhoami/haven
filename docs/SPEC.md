# Haven — Project Specification

> A decentralized, privacy-first communication platform.
> Self-hosted servers, end-to-end encryption, key-based identity.

**Version:** 0.1.0 (Draft)
**Last Updated:** 2026-02-21

---

## Table of Contents

1. [Vision & Principles](#1-vision--principles)
2. [Architecture Overview](#2-architecture-overview)
3. [Identity & Trust](#3-identity--trust)
4. [Protocol](#4-protocol)
5. [Voice Architecture](#5-voice-architecture)
6. [Encryption Model](#6-encryption-model)
7. [State Synchronization](#7-state-synchronization)
8. [Data Storage](#8-data-storage)
9. [Feature Specs](#9-feature-specs)
10. [Screen Map](#10-screen-map)
11. [Implementation Phases](#11-implementation-phases)
12. [Future Considerations](#12-future-considerations)

---

## 1. Vision & Principles

Haven is a decentralized communication platform — an alternative to Discord where users own their identity and communities own their infrastructure.

### Core Philosophy

- **Self-hosted**: Anyone can run a Haven server. No central authority, no single point of control.
- **Key-based identity**: Your identity is your Ed25519 keypair. No accounts, no passwords, no email. You are your key.
- **Privacy by default**: DMs and group chats are end-to-end encrypted. Voice is end-to-end encrypted. The server cannot read your private conversations.
- **User control**: The client decides what data it receives. Users opt in to avatars, bios, and other profile data — per server.
- **Transparent trust model**: Public channels are not end-to-end encrypted (the server can read them). DMs are. This is stated clearly, not hidden.
- **Minimal footprint**: One binary for the server, one binary for the client. Minimal dependencies, easy deployment.

### What Haven Is Not

- **Not federated** (v1): Servers do not talk to each other. Federation may come later as a plugin system.
- **Not a product**: Haven is open-source software. There is no company, no subscription, no data collection.
- **Not trying to do everything**: Video streaming, bots, and app integrations are out of scope for v1. The foundation must be solid first.

---

## 2. Architecture Overview

### Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Server | Go | All server-side logic, networking, storage |
| Client Backend | Go (Wails) | Application logic, crypto, voice, state management |
| Client Frontend | Svelte 5 | UI rendering only — minimal logic |
| Voice (both sides) | Pion (Go WebRTC) | SFU on server, WebRTC client in Go backend |
| Database (server) | SQLite or PostgreSQL | Configurable by server owner, via GORM |
| Database (client) | SQLCipher | Encrypted local storage, keyed from Ed25519 private key |
| ORM | GORM | Database abstraction supporting both SQLite and PostgreSQL |

### System Diagram

```
┌─────────────────────────────────────┐
│           Svelte Frontend           │  Pure UI. Renders state, captures input.
│   No crypto, no networking, no      │  Calls Go functions via Wails bindings.
│   business logic.                   │
├─────────────────────────────────────┤
│           Wails Bindings            │  Typed bridge between Go and Svelte
├─────────────────────────────────────┤
│         Go Client Backend           │
│  ┌───────────┐ ┌──────────────────┐ │
│  │   Pion    │ │  Crypto Engine   │ │
│  │ (WebRTC)  │ │ Ed25519, X25519  │ │
│  │  Voice    │ │ E2EE, Key Mgmt   │ │
│  └───────────┘ └──────────────────┘ │
│  ┌───────────┐ ┌──────────────────┐ │
│  │   Audio   │ │   State Sync     │ │
│  │ Capture / │ │ Version tracking │ │
│  │ Playback  │ │ Event handling   │ │
│  └───────────┘ └──────────────────┘ │
│  ┌──────────────────────────────────┐│
│  │   SQLCipher (local storage)     ││
│  │   Encrypted with user's key     ││
│  └──────────────────────────────────┘│
└──────────────┬───────────────────────┘
               │
               │  TCP :port  — WebSocket + HTTP
               │  UDP :port  — DTLS/SRTP (voice)
               │
┌──────────────┴───────────────────────┐
│            Go Server                 │
│  ┌──────────────────────────────────┐│
│  │   Single Port Listener          ││
│  │   /ws       → WebSocket handler ││
│  │   /upload/* → Tokenized uploads ││
│  │   /files/*  → Tokenized downloads││
│  └──────────────────────────────────┘│
│  ┌───────────┐ ┌──────────────────┐ │
│  │ Pion SFU  │ │  Message Router  │ │
│  │ (forward  │ │  Dispatches by   │ │
│  │  opaque   │ │  message type    │ │
│  │  packets) │ │                  │ │
│  └───────────┘ └──────────────────┘ │
│  ┌──────────────────────────────────┐│
│  │   GORM (SQLite or PostgreSQL)   ││
│  └──────────────────────────────────┘│
└──────────────────────────────────────┘
```

### Design Principle: Frontend Is Just UI

The Svelte frontend is strictly a rendering layer. It:

- Displays state provided by the Go backend via Wails bindings
- Captures user interactions and forwards them to Go functions
- Never touches networking, cryptography, audio, or business logic

All application logic lives in Go. This ensures:

- Security-critical code (crypto, auth, key management) is in one language
- The frontend can be replaced without affecting functionality
- Testing and auditing focus on a single codebase

---

## 3. Identity & Trust

### Key-Based Identity

Every Haven user is identified by an **Ed25519 keypair**.

- The **private key** never leaves the client device. It is the user's sole credential.
- The **public key** is the user's identity. It is shared with servers and other users.
- There are no usernames, passwords, or email addresses at the protocol level. Display names are cosmetic metadata.

### Server Trust — TOFU (Trust On First Use)

When a client connects to a server for the first time:

1. Client initiates WebSocket connection to the server.
2. Server presents its public key **and a signature proving it holds the corresponding private key** (signs a handshake nonce).
3. Client verifies the server's signature. If invalid → connection aborted.
4. Client has never seen this server before → displays the server's public key fingerprint to the user.
5. User accepts → client stores the server's public key, associated with that address.
6. On all subsequent connections, the client verifies the server's key matches the stored key **and** verifies the server's signature.

**Key mismatch handling:**

- If the server's key changes, the client **blocks the connection** and warns the user: *"This server's identity has changed. This could indicate a man-in-the-middle attack, or the server was reinstalled."*
- The user must explicitly choose to re-trust or cancel. No silent fallthrough.

**Why the server must prove identity:** Presenting a public key is just a claim. Without a signature, a man-in-the-middle could present the real server's public key and relay traffic. The server signature proves possession of the private key.

### Mutual Authentication — Challenge-Response

Both sides prove their identity during the handshake:

```
1. Client connects via WebSocket.
2. Server → Client:  { server_pubkey, server_nonce, server_signature(server_nonce) }
   - Client verifies server_signature against server_pubkey (TOFU check).
3. Server → Client:  { challenge_nonce }
4. Client signs: signature(challenge_nonce || server_pubkey)
   - Domain separation: binding the server's public key into the signature
     prevents cross-server replay attacks (a signature for server A is
     invalid for server B).
5. Client → Server:  { client_pubkey, signature, access_token? }
6. Server verifies the signature against client_pubkey.
7. Access control gate (see below).
8. Identity resolution:
   - Known public key → returning user, restore their state/permissions.
   - Unknown public key → new user, register with default permissions.
9. Verification fails at any step → connection rejected.
```

This makes key spoofing impossible in both directions. The server proves it holds its key, the client proves it holds its key, and signatures are bound to the specific server to prevent replay across servers.

### Access Control

Server owners configure how new users are admitted. Access control is checked **after** identity verification but **before** registration. The mode is set in the server configuration file.

| Mode | Description |
|---|---|
| `open` | Anyone can join. Default for new servers. |
| `invite` | Users must provide a valid invite code during auth. Server generates single-use or multi-use codes. |
| `password` | Users must provide a shared password during auth. Simple, for small private servers. |
| `allowlist` | Only public keys listed in the server config are accepted. Unknown keys are rejected. |

The client includes an optional `access_token` field in the auth message (step 5). Depending on the server's mode, this contains the invite code, password, or is omitted.

**Returning users** (known public key) bypass access control — they are already registered.

**Server communicates its mode** on initial connection so the client can prompt for an invite code or password before completing auth.

### Server Ownership

Server owners are defined by their public keys in the server's configuration file.

- Owners cannot be banned or kicked.
- The config file supports **hot-reloading** — changes are applied without restarting the server.
- Users can view and copy their public key in client settings, making it easy to share with server administrators.

---

## 4. Protocol

### Single Endpoint

All communication flows through a single TCP port:

```
:port
  ├── /ws        → WebSocket upgrade → message router
  ├── /upload/*  → Tokenized file uploads (HTTP PUT)
  └── /files/*   → Tokenized file downloads (HTTP GET)
```

Voice media uses the same port number over UDP (DTLS/SRTP via Pion).

No additional ports need to be opened. One port, one firewall rule.

### Connection UX

Users type only a server address — the client handles the rest:

| User types | Client connects to |
|---|---|
| `myserver.com` | `wss://myserver.com/ws` |
| `myserver.com:8443` | `wss://myserver.com:8443/ws` |
| `192.168.1.50:9090` | `ws://192.168.1.50:9090/ws` |

- Domain names default to `wss://` (TLS).
- Raw IP addresses default to `ws://` (no TLS) — the client displays a **security warning** and requires explicit confirmation.
- The `/ws` path is always appended automatically.

### Application-Layer Encryption (ws:// Fallback)

When TLS is not available (`ws://` connections), Haven provides **application-layer encryption** using the keys both parties already possess.

During the WebSocket handshake, after mutual authentication:

1. Both sides convert their Ed25519 keys to X25519.
2. Perform a Diffie-Hellman key exchange → derive a shared symmetric key.
3. All subsequent WebSocket payloads are encrypted with this key (ChaCha20-Poly1305).

This provides **confidentiality and integrity** over plaintext WebSocket connections. It does **not** replace TLS — it has the same TOFU limitation (the very first connection to an unknown server is vulnerable to MITM). But after trust is established, a network observer sees only opaque bytes.

**This mode is automatically activated when the connection is `ws://`.** No user configuration required.

HTTP file transfer endpoints (`/upload/*`, `/files/*`) over non-TLS connections are **disabled** — files can only be transferred over `wss://` or via the encrypted WebSocket as base64 payloads (with a size penalty). This prevents session tokens from being exposed on the wire.

### WebSocket Message Format

All WebSocket messages follow a unified structure:

```json
{
  "type": "namespace.action",
  "id": "msg_unique_id",
  "payload": { }
}
```

- **`type`**: Dot-namespaced string that routes to the correct handler module.
- **`id`**: Unique message ID for request/response correlation.
- **`payload`**: Action-specific data.

#### Response Messages

Successful responses:

```json
{
  "type": "namespace.action.ok",
  "id": "msg_unique_id",
  "payload": { }
}
```

Error responses:

```json
{
  "type": "namespace.action.error",
  "id": "msg_unique_id",
  "payload": {
    "code": "PERMISSION_DENIED",
    "message": "You do not have permission to delete this channel."
  }
}
```

#### Server-Initiated Events

The server pushes events through the same WebSocket:

```json
{
  "type": "event.message.new",
  "payload": {
    "channel_id": "ch_1",
    "message": { }
  }
}
```

Events have no `id` field — they are fire-and-forget from the server.

#### Message Type Namespaces

```
auth.*              Authentication and challenge-response
server.*            Server settings, info, management
channel.*           Channel CRUD, listing
category.*          Category CRUD
message.*           Send, edit, delete, search messages
user.*              Profile updates, presence
voice.*             Join, leave, signaling
dm.*                Direct messages and group chats
file.*              Upload/download token requests
sync.*              State synchronization
```

### File Transfer Protocol

File uploads use **single-use tokens** obtained via WebSocket. File downloads and media viewing use **session-based authentication** with persistent URLs.

#### Session Authentication for HTTP

During the WebSocket handshake, the server issues a session token. All HTTP requests include this token:

```
Authorization: Bearer <session_token>
```

The server verifies: is the session valid? Is this user a member of the channel this file belongs to? If not, return 403.

**Session token lifecycle:**

- Issued on successful WebSocket authentication.
- Valid for the duration of the WebSocket connection plus a **grace period** (configurable, default: 5 minutes) after disconnect — allows reconnects without re-auth.
- Explicitly invalidated on: user logout, server-side kick/ban, or grace period expiry.
- Long-lived during active connections — no periodic rotation to avoid unnecessary overhead.

#### Upload Flow (Tokenized)

Uploads use single-use tokens to prevent replay attacks and enable rate limiting:

```
1. Client → WS:   file.upload.request   { name: "photo.png", size: 524288, channel_id: "ch_1" }
2. Server → WS:   file.upload.token     { token: "abc123", url: "/upload/abc123", expires_in: 60 }
3. Client → HTTP:  PUT /upload/abc123    (binary data, with session auth header)
4. Server generates thumbnail (images/videos), stores original + thumbnail.
5. Server → WS:   file.upload.complete  { file_id: "f_xyz" }
6. Client → WS:   message.send          { channel_id: "ch_1", content: "", files: ["f_xyz"] }
```

Upload tokens are:
- Single-use (consumed on first request)
- Short-lived (default: 60 seconds)
- Tied to the authenticated WebSocket session

#### Download / Viewing Flow (Session-Authenticated)

Downloads and media viewing use persistent, stable URLs authenticated by the session token. No per-request token dance required.

```
Thumbnail:  GET /files/{file_id}/thumb   (auto-loaded when message scrolls into view)
Full file:  GET /files/{file_id}         (loaded on user interaction)
```

Both endpoints verify session authentication and channel membership.

#### Thumbnails

The server generates thumbnails **on upload** (not on-the-fly) to avoid CPU spikes during serving:

- Images: low-resolution preview (~5-10 KB)
- Videos: preview frame
- Files (non-media): no thumbnail, show file icon and metadata

Thumbnails are stored alongside the original and served instantly on request.

**Security note:** Thumbnail generation involves parsing untrusted image/video data. Image processing libraries are a historically common attack surface. The server must use memory-safe image libraries and should process uploads in a sandboxed subprocess to contain potential exploits.

#### Media Loading Strategy

| Content | When loaded | Cached on client? |
|---|---|---|
| Thumbnails | Auto, when message scrolls into view | In-memory (session lifetime) |
| Full images | On user click/tap | Optional (client setting) |
| Videos | Stream on play | No (re-stream each time) |
| Files | On explicit download | Yes (user chose to save) |

This minimizes bandwidth — viewing a chat only loads tiny thumbnails. Full media transfers only when the user actively engages.

### Rate Limiting & Message Size Limits

The server enforces limits to prevent abuse and resource exhaustion.

**WebSocket message limits:**

| Limit | Default | Description |
|---|---|---|
| Max message size | 64 KB | Maximum size of a single WebSocket JSON payload. Messages exceeding this are dropped and the client receives an error. |
| Messages per second | 10 msg/s | Per-client rate limit for outgoing messages. Burst allowance: 20 messages. |
| Auth attempts per minute | 5 | Per-IP rate limit for authentication attempts (prevents brute-force). |

**HTTP transfer limits:**

| Limit | Default | Description |
|---|---|---|
| Max file size | Configurable (server) | See Section 7 — File & Storage Limits. |
| Concurrent uploads | 3 | Per-client concurrent upload limit. |
| Upload bandwidth | Configurable | Optional per-client upload bandwidth throttle. |

**New registration limits:**

| Limit | Default | Description |
|---|---|---|
| Registrations per IP per hour | 5 | Prevents automated mass account creation. |

All limits are configurable in the server configuration file. The server communicates applicable limits to the client on connection.

---

## 5. Voice Architecture

### SFU Model

Haven uses a **Selective Forwarding Unit (SFU)** for voice. The server does not decode, mix, or process audio — it forwards encrypted packets between participants.

```
Client A ──audio──→ SFU ──audio──→ Client B
                        ──audio──→ Client C
Client B ──audio──→ SFU ──audio──→ Client A
                        ──audio──→ Client C
```

The SFU runs on the server using **Pion** (pure Go WebRTC).
The client also uses **Pion** in the Go backend — not the browser's WebRTC.

Audio capture and playback happen in Go (via system audio libraries), not in the Svelte frontend.

### Audio Device Management

Audio device selection and volume control are handled entirely in the Go backend:

- **Device enumeration**: The audio library enumerates system input/output devices and exposes them via Wails bindings.
- **Device selection**: User picks input (microphone) and output (speakers/headphones) from dropdowns in settings. Stored per-client.
- **Master volume**: A gain multiplier applied to the audio playback stream in Go. The frontend only renders the slider.
- **Per-user volume** (future): Adjust the volume of individual participants.

### Codec: Opus Only

Haven uses **Opus** exclusively. No other codecs are supported.

Opus is the industry standard for real-time audio. It was specifically designed to replace older codecs (Speex, CELT — which Opus actually contains internally). Supporting multiple codecs adds negotiation complexity and multiple decode paths for zero benefit.

Opus covers the full quality range via bitrate adjustment:

| Bitrate | Quality | Use case |
|---|---|---|
| 32 kbps | Good voice | Default, bandwidth-constrained |
| 64 kbps | Great voice | Standard quality |
| 96-128 kbps | Near-transparent | High-quality, music-friendly |

Voice quality is configurable **per channel** by the server administrator. This allows bandwidth-conscious setups — e.g., large public channels at 32 kbps, small private channels at 96 kbps.

### Why SFU

| | SFU | MCU | Peer-to-Peer |
|---|---|---|---|
| Server CPU | Minimal (packet routing) | Heavy (decode/mix/encode) | None |
| E2EE compatible | Yes | No (server must decrypt to mix) | Yes |
| Client IP exposure | No (only server IP visible) | No | Yes (DDoS risk) |
| Scales to | 50-100+ for audio | Depends on server hardware | 4-5 max |

MCU is ruled out because it is incompatible with end-to-end voice encryption.
Peer-to-peer is ruled out because it exposes client IP addresses.

### Bandwidth Optimization

**Voice Activity Detection (VAD) + Selective Forwarding:**

In a 20-person channel, typically 1-3 people speak at once. The SFU only forwards streams from **active speakers**, not silent participants.

- 20 participants, 3 speaking → client receives 3 streams (~100-200 kbps), not 19.
- Bandwidth stays low regardless of channel size.

**Dynamic Opus Bitrate:**

When more speakers are active simultaneously, the server can signal clients to lower their Opus encoding bitrate, reducing per-stream bandwidth.

### Video Streaming

**Out of scope for v1.** The SFU + Pion architecture natively supports video. No architectural decisions in v1 block adding webcam or screen sharing later.

---

## 6. Encryption Model

### Privacy Tiers

| Context | Encryption | Signed by sender? | Server can read? |
|---|---|---|---|
| Public channel messages | Encrypted in transit (WSS/TLS) | Yes (Ed25519) | Yes |
| Direct messages (1:1) | End-to-end encrypted (E2EE) | Yes (Ed25519) | No |
| Group DMs | End-to-end encrypted (E2EE) | Yes (Ed25519) | No |
| Voice (all channels) | End-to-end encrypted (E2EE) | N/A (real-time) | No |
| File transfers | Encrypted in transit (HTTPS) | No | Depends on context |

This model is **transparently communicated** to users. No false sense of security.

### Message Signing (All Channels)

**Every text message** — public and private — is signed by the sender's Ed25519 key.

```
message_signature = Ed25519_Sign(private_key, message_content || channel_id || timestamp || nonce)
```

The signature is stored alongside the message. This provides:

- **Integrity**: The server (or any intermediary) cannot modify message content without invalidating the signature.
- **Attribution**: A message provably came from the holder of a specific key. The server cannot fabricate messages that appear to come from a user.
- **Non-repudiation**: In public channels, where the server can read content, it still cannot forge messages from users.

Clients verify signatures on received messages. A failed verification flags the message as tampered.

### 1:1 DM Encryption

Uses **X25519 key exchange** with **ephemeral session keys** for forward secrecy.

**Identity verification (one-time):**

1. Both users have Ed25519 identity keys.
2. Derive X25519 key pairs from the Ed25519 keys.
3. Perform Diffie-Hellman key exchange → **identity shared secret** (used only for initial authentication, not for message encryption).

**Per-session forward secrecy:**

```
When a DM session starts (either user comes online):
  1. Both sides generate ephemeral X25519 keypairs.
  2. Exchange ephemeral public keys (encrypted with the identity shared secret).
  3. DH on ephemeral keys → session_key.
  4. All messages in this session are encrypted with session_key (ChaCha20-Poly1305).
  5. When the session ends, ephemeral keys are deleted.
```

**Why this matters:** If a user's long-term Ed25519 key is ever compromised (stolen device, malware, seizure), past DM conversations cannot be decrypted — the ephemeral session keys that encrypted them no longer exist. Only future sessions (where the attacker actively uses the stolen key) are at risk.

**Future enhancement:** A full Double Ratchet protocol (per-message keys) can be layered on top for even stronger forward secrecy. Per-session keys are the v1 baseline.

### Group DM Encryption — Shared Group Key

Group DMs use a single **shared symmetric key** known to all group members.

**Group creation:**

```
Creator (A) creates group with B and C:
  1. A generates a random symmetric group_key.
  2. A encrypts group_key for B (using B's public key) → sends via relay.
  3. A encrypts group_key for C (using C's public key) → sends via relay.
  4. All members now share group_key.
```

**Sending a message:**

```
  1. Encrypt message with group_key (confidentiality).
  2. Sign message with sender's Ed25519 key (attribution/authenticity).
  3. Send via relay server.
```

**Member leaves (C removed):**

```
  1. A generates a new group_key.
  2. A distributes new group_key to B only.
  3. C still has the old key but no future messages use it.
```

**Member joins (D added):**

```
  1. A encrypts current group_key for D → sends via relay.
  2. D can read new messages but NOT old ones (didn't have the previous key).
```

**Key management delegation:**

The group creator is the initial key manager, but any member can be designated as a key manager. This prevents a single point of failure:

- If the creator goes offline permanently, another key manager can rotate the group key and add/remove members.
- Key manager status is part of the group metadata, distributed alongside the group key.
- At least one active member must always be a key manager. If all key managers leave, the longest-tenured remaining member is automatically promoted.

### Voice Encryption (E2EE)

Voice packets are encrypted at the RTP level in the Go client backend before being sent to the SFU:

1. Participants in a voice channel perform a key exchange (same shared key model as group DMs).
2. Each audio frame is encrypted with the shared key before transmission.
3. The SFU receives and forwards **opaque encrypted bytes** — it cannot decode or listen to the audio.
4. Recipients decrypt in their Go client backend before playback.

### DM Relay Model

DMs are routed through shared servers. The server acts as a **blind relay** — it forwards encrypted blobs it cannot read.

**Server selection:** The conversation initiator selects the relay server based on lowest latency (ping) among mutually shared servers.

**Offline delivery:** The relay server stores encrypted message blobs until the recipient connects and retrieves them.

**Server unavailability (v1):** If the relay server goes down, DMs routed through it are unavailable until it returns. Multi-server failover is a future enhancement.

### Server Departure & Data Rights

When a user leaves a server, they choose one of three departure modes:

#### 1. Leave

Simple disconnect. User's identity, messages, files, and all data remain on the server. The user can rejoin and everything is restored.

#### 2. Ghost

The user's identity is **anonymized**. All content (messages, files, uploads) is reassigned to a generic placeholder identity ("Deleted User"). The user's public key is removed from the server's user records.

- Messages remain readable but are no longer attributable to the original author.
- The server must maintain anonymous placeholder identities for this purpose.
- This is irreversible — rejoining creates a fresh identity, not a restoration.

#### 3. Forget Me

**Complete deletion** of all data tied to the user's public key:

- All messages authored by the user are deleted.
- All files uploaded by the user are deleted.
- All DM blobs stored on this server for this user are deleted.
- The user's public key and profile data are purged.
- **All relay connections through this server are force-terminated.** DM conversations that relied on this server as their only relay are lost. The user is warned and must confirm.
- This is irreversible.

**Impact on group DMs:** Messages sent by the user in group DMs are deleted. Other participants see "[message deleted]" or similar placeholder. The group conversation continues without gaps in context becoming confusing.

**Client-side erasure propagation:** When a user chooses Ghost or Forget Me, the server records a `{ pubkey, erased_at, mode }` entry. On future sync, other clients receive this event and purge locally cached data (messages, profile, avatar) for that user. This is best-effort — a client that never reconnects retains cached data — but covers the common case.

### Relay-Only Server Mode

When a user leaves a server but still has active DM connections routed through it (and no alternative shared server exists), the server transitions to **relay-only mode** for that user.

**Relay-only means:**
- The server does **not** appear in the client's server list.
- The server appears in **Client Settings → Relay Servers** as a separate list.
- The user cannot see or post in channels.
- The user cannot join voice channels.
- The user is not visible in the server's member list.
- The server only routes DM traffic for the user's active conversations.
- Minimal server resources are consumed.

**Lifecycle:**

```
User leaves Server X, but has active DMs with User B through Server X:
  → No alternative shared server with User B exists.
  → Server X becomes a relay for User B's DMs.
  → Client settings shows:
      Relay Servers:
        Server X — relaying DMs with: User B
        [Remove relay] ⚠ "You will lose contact with User B unless you share another server."
```

If the user later joins another server shared with User B, the relay can be removed (manually or automatically) since DMs can route through the new shared server.

**Relay mode + departure mode interaction:**
- **Leave + relay**: server retains full data, user is relay-only.
- **Ghost + relay**: server anonymizes public data, user is relay-only. Private data (DM relay) persists.
- **Forget Me**: **no relay**. All data is deleted and all relay connections through this server are force-terminated. The user accepts losing DM conversations that depended on this server.

---

## 7. State Synchronization

### Sync Model

State is tracked using **version numbers**. Every entity (user profile, channel, server settings) carries a monotonically increasing version.

**While connected — event-driven push:**

Any state change broadcasts an event immediately to connected clients:

```json
{
  "type": "event.user.updated",
  "payload": {
    "user_id": "user_abc",
    "version": 7,
    "display_name": "NewName",
    "avatar_hash": "a1b2c3"
  }
}
```

**On connect/reconnect — version-based diff sync:**

The client sends its known state versions:

```json
{
  "type": "sync.request",
  "payload": {
    "users": { "user_abc": 5, "user_def": 3 },
    "channels": { "ch_1": 12, "ch_2": 8 },
    "server": 4
  }
}
```

The server compares and responds with **only what changed**:

```json
{
  "type": "sync.response",
  "payload": {
    "users": [
      { "id": "user_abc", "version": 7, "display_name": "NewName", "avatar_hash": "a1b2c3" }
    ],
    "channels": [],
    "server": null
  }
}
```

**First connection** (no local state): Server sends a full state dump. All subsequent syncs are diffs.

### File & Storage Limits

Server owners configure hard limits for file storage:

- **Max file size**: Maximum size for a single upload (e.g., 10 MB). Uploads exceeding this are rejected.
- **Total storage limit**: Maximum total disk usage for all uploaded files on the server (e.g., 5 GB). When exceeded, new uploads are rejected until space is freed.

Both are server configuration options. The server communicates these limits to clients on connection so the UI can show them before an upload attempt.

**Future:** Per-role file size limits via the permission system (e.g., a role could allow uploads up to 50 MB while the default is 10 MB).

### Avatar and Binary Data

Binary data (avatars, attachments) is separated from metadata:

1. Metadata (including `avatar_hash`) syncs via WebSocket — instant, tiny.
2. Binary data is fetched via HTTP only when the client doesn't have the hash cached.
3. Client maintains a local cache: `hash → binary data`.

```
New avatar_hash received → check local cache:
  Cache hit  → display immediately
  Cache miss → HTTP fetch → cache → display
```

### Client Field Selection

Clients control **what data they receive** from each server. On connection, the client sends its subscription preferences:

```json
{
  "type": "sync.subscribe",
  "payload": {
    "users": ["display_name", "status"],
    "channels": ["name", "category", "type"],
    "messages": ["content", "author_id", "timestamp"]
  }
}
```

Fields not listed are never sent by the server. This is:

- **A safety measure**: opt out of avatars to avoid caching unwanted images.
- **A bandwidth optimization**: only receive data you display.
- **Per-server configurable**: trust a friends server with full data, restrict public servers to minimal data.
- **Controlled by the client, not the server**: the server respects the field mask regardless.

The client stores a **default field profile** and allows **per-server overrides** in settings.

---

## 8. Data Storage

### Server Database

The server supports **both SQLite and PostgreSQL**, configurable by the server owner.

| | SQLite | PostgreSQL |
|---|---|---|
| Setup | Zero config, embedded in binary | Requires separate service |
| Best for | Small-medium communities, easy self-hosting | Large servers, advanced features |
| Full-text search | FTS5 (exact + prefix matching) | tsvector + pg_trgm (stemming, fuzzy) |
| Semantic search | Not supported | pgvector (future) |
| Concurrency | Limited | Excellent |

**Database abstraction via GORM** ensures the application logic is database-agnostic. Database-specific features (e.g., fuzzy search) use provider-specific implementations behind a common interface.

**Database encryption at rest** is optional, configured by the server owner.

**Message persistence** is permanent by default. Messages are stored indefinitely.

### Client Database

The client uses **SQLCipher** (encrypted SQLite) for local storage:

- Message cache, user profiles, channel state, trust store (known server keys).
- Encrypted at rest using a key derived from the user's Ed25519 private key.
- When the client is not running, the database file is unreadable without the key.

### Private Key Storage

The user's Ed25519 private key is stored in the **OS-provided secure credential store**, not as a plaintext file:

| OS | Storage |
|---|---|
| Windows | Credential Manager (DPAPI-backed) |
| macOS | Keychain Services |
| Linux | Secret Service API (GNOME Keyring / KDE Wallet) |

The private key never touches the filesystem in plaintext. On first launch, the key is generated and immediately stored in the OS keychain. On subsequent launches, it is retrieved from the keychain, used to derive the SQLCipher encryption key, and held in memory only while the application is running.

**Fallback:** If no OS keychain is available (e.g., headless Linux), the key is encrypted with a user-provided passphrase and stored in the application data directory.

### Message Search

Haven supports **Discord-style filtered search**:

```
from:Alice has:file before:2026-01-01 hello world
```

The client parses filter syntax into structured queries sent via WebSocket:

```json
{
  "type": "message.search",
  "payload": {
    "text": "hello world",
    "from_user": "user_abc",
    "has": ["file"],
    "before": "2026-01-01",
    "channel_id": "ch_1"
  }
}
```

Supported filters:
- `from:<user>` — messages by a specific user
- `has:file` / `has:image` / `has:link` — messages with attachments
- `before:<date>` / `after:<date>` — date range
- `in:<channel>` — specific channel
- Free text — full-text search (quality scales with database backend)

---

## 9. Feature Specs

Each feature includes a description, acceptance criteria, and UI reference to the design file (`haven.pen`).

---

### 9.1 Client — Loading Screen

**Description:** Splash screen shown while the client initializes, loads local data, and attempts to connect.

**UI Reference:** `haven.pen` → `Client - Loading` (`uyao0`)

**Acceptance Criteria:**
- [ ] Display Haven logo and progress bar.
- [ ] Progress bar reflects actual loading stages (key loading, DB init, connection attempts).
- [ ] Display "Loading" text below progress bar.
- [ ] Transition to Setup screen (first launch) or main screen (returning user).

---

### 9.2 Client — Profile Setup

**Description:** First-run experience where the user creates their profile. The Ed25519 keypair is generated automatically — the user sets cosmetic details.

**UI Reference:** `haven.pen` → `Client - Setup` (`dlYLL`)

**Acceptance Criteria:**
- [ ] Generate Ed25519 keypair on first launch (transparent to user).
- [ ] Allow user to upload an avatar image.
- [ ] Allow user to set a display name.
- [ ] Allow user to set a bio.
- [ ] "New Profile" button saves profile and transitions to the main screen.
- [ ] Display name is required. Avatar and bio are optional.

---

### 9.3 Client — No Server State

**Description:** Main screen shown when the user has no servers. Prompts them to join one.

**UI Reference:** `haven.pen` → `Client - No Server` (`aFBop`)

**Acceptance Criteria:**
- [ ] Display server list sidebar (empty).
- [ ] Display channel list sidebar with placeholder text ("No Server Connected").
- [ ] Main area shows prompt: "Chat or talk to others by adding your first server."
- [ ] "Add Server" button opens the Join Server modal.
- [ ] Profile bar visible at bottom with user info and controls.

---

### 9.4 Client — Join Server

**Description:** Modal where the user enters a server address to connect.

**UI Reference:** `haven.pen` → `Client - Join Server` (`5168M`)

**Acceptance Criteria:**
- [ ] Modal overlay with backdrop blur.
- [ ] Single input field: server address (domain or IP, optional port).
- [ ] Client auto-prepends `wss://` (domain) or `ws://` (IP) and appends `/ws`.
- [ ] "Connect" button initiates connection.
- [ ] On first connection to unknown server: show Server Trust modal (see 9.5).
- [ ] On successful connection: server appears in server list sidebar.
- [ ] On connection failure: display error message in modal.

---

### 9.5 Client — Server Trust Prompt

**Description:** Modal shown when connecting to a server whose public key is not yet trusted. Implements TOFU (Trust On First Use).

**UI Reference:** Not yet designed in `haven.pen` — modal component exists, content defined here.

**Acceptance Criteria:**
- [ ] Display server address.
- [ ] Display server's public key fingerprint (human-readable format).
- [ ] Explanatory text: "This is the first time you're connecting to this server. Verify the server's identity before trusting it."
- [ ] "Trust" button → store server key, proceed with connection.
- [ ] "Cancel" button → abort connection, return to previous screen.

---

### 9.6 Client — Server Trust Warning (Key Mismatch)

**Description:** Warning shown when a known server presents a different public key than previously stored.

**UI Reference:** Not yet designed — uses warning modal variant.

**Acceptance Criteria:**
- [ ] **Block the connection** — do not connect until the user makes a choice.
- [ ] Display warning: "This server's identity has changed since you last connected. This could indicate a man-in-the-middle attack, or the server may have been reinstalled."
- [ ] Display both the stored key and the new key.
- [ ] "Re-trust" button → update stored key, proceed with connection.
- [ ] "Cancel" button → abort connection.
- [ ] Default action is Cancel (prevent accidental re-trust).

---

### 9.7 Server — Main View

**Description:** The primary server view showing the channel list and a prompt to select a channel.

**UI Reference:** `haven.pen` → `Server` (`1YKmD`)

**Acceptance Criteria:**
- [ ] Server list sidebar on the far left with server icons.
- [ ] Channel list sidebar showing categories with text and voice channels.
- [ ] Server name and settings icons in the channel list header.
- [ ] Main content area: "Select any channel to start chatting."
- [ ] "Create Category" and "Create Channel" buttons at the bottom of the channel list.
- [ ] Profile bar at the bottom with avatar, display name, and controls (settings, mute, deafen, end call).

---

### 9.8 Server — Text Channel

**Description:** View for a text channel with message history and input.

**UI Reference:** `haven.pen` → `Server - View Text Channel` (`agRZp`)

**Acceptance Criteria:**
- [ ] Channel name displayed in header with `#` prefix.
- [ ] Message list showing messages with: author avatar, display name, timestamp, content.
- [ ] Messages grouped by author for consecutive messages.
- [ ] Message input bar at the bottom with text field and send button.
- [ ] Attachment button for file uploads.
- [ ] Auto-scroll to newest messages on load and on new message received.
- [ ] Scrollback loads older messages on demand.

---

### 9.9 Server — Voice Channel

**Description:** View for a voice channel showing participants and an optional text chat.

**UI Reference:** `haven.pen` → `Server - View Voice Channel` (`6OJLm`)

**Acceptance Criteria:**
- [ ] Voice channel name displayed in header.
- [ ] Participant grid showing avatars with visual indicators (green ring = speaking).
- [ ] Display name and mute/deafen status per participant.
- [ ] Text chat area below the participant grid (voice channels have text too).
- [ ] Joining a voice channel initiates Pion WebRTC connection via Go backend.
- [ ] Profile bar controls update: mute, deafen, end call buttons become active.

---

### 9.10 Server — Create Channel

**Description:** Modal for creating a new text or voice channel within a category.

**UI Reference:** `haven.pen` → `Server - Create Channel` (`gXHj1`)

**Acceptance Criteria:**
- [ ] Modal with fields: Channel Name, Category (dropdown), Private Channel (toggle).
- [ ] Channel name is required.
- [ ] Category dropdown lists existing categories.
- [ ] Private channel toggle restricts visibility to permitted users/roles.
- [ ] "Create Channel" button sends `channel.create` message via WebSocket.
- [ ] New channel appears in the channel list immediately.

---

### 9.11 Server — Create Category

**Description:** Modal for creating a new category to organize channels.

**UI Reference:** `haven.pen` → `Server - Create Category` (`YJU7H`)

**Acceptance Criteria:**
- [ ] Modal with fields: Category Name, Type toggle (Text / Voice).
- [ ] Category name is required.
- [ ] "Create Category" button sends `category.create` message via WebSocket.
- [ ] New category appears in the channel list immediately.

---

### 9.12 Server — Administration

**Description:** Server settings panel for administrators. Sidebar navigation with multiple settings sections.

**UI Reference:** `haven.pen` → `Server - Administration` (`8i5M5`)

**Acceptance Criteria:**
- [ ] Sidebar navigation: Overview, Channels, Roles, Audit Log, Invitations, Webhooks.
- [ ] **Overview section:**
  - [ ] Server avatar upload.
  - [ ] Server name field (editable).
  - [ ] Server description field (editable).
  - [ ] Notification settings toggle.
  - [ ] System channel configuration (welcome messages, default channel).
  - [ ] "Reset Server" button in danger zone (with confirmation).
- [ ] Only accessible to users with admin permissions.
- [ ] Changes send appropriate `server.*` messages via WebSocket.

---

### 9.13 Client — Settings

**Description:** Client-side settings panel. Configures user preferences, not server settings.

**UI Reference:** `haven.pen` → `Client - Settings` (`Rt0xJ`)

**Acceptance Criteria:**
- [ ] Sidebar navigation for settings categories.
- [ ] **Profile settings:** edit display name, avatar, bio.
- [ ] **Public key display:** show the user's public key with copy button.
- [ ] **Sync preferences:** default field selection (avatars, bios, etc.).
- [ ] **Per-server overrides:** customize field selection per server.
- [ ] **Audio settings:** input/output device selection, volume.
- [ ] **Appearance:** theme settings (if applicable).
- [ ] Settings are stored locally in SQLCipher database.

---

## 10. Screen Map

Maps each `haven.pen` design frame to its feature spec.

| Screen Name | Frame ID | Feature Spec |
|---|---|---|
| Client - Loading | `uyao0` | 9.1 Loading Screen |
| Client - Setup | `dlYLL` | 9.2 Profile Setup |
| Client - No Server | `aFBop` | 9.3 No Server State |
| Client - Join Server | `5168M` | 9.4 Join Server |
| Server | `1YKmD` | 9.7 Server Main View |
| Server - View Text Channel | `agRZp` | 9.8 Text Channel |
| Server - View Voice Channel | `6OJLm` | 9.9 Voice Channel |
| Server - Create Channel | `gXHj1` | 9.10 Create Channel |
| Server - Create Category | `YJU7H` | 9.11 Create Category |
| Server - Administration | `8i5M5` | 9.12 Server Administration |
| Client - Settings | `Rt0xJ` | 9.13 Client Settings |

**Screens not yet in `haven.pen`:**

| Feature | Spec | Notes |
|---|---|---|
| Server Trust Prompt | 9.5 | Uses existing modal component |
| Server Trust Warning | 9.6 | Uses existing warning modal variant |
| DM Conversation | — | To be designed |
| Group DM | — | To be designed |

---

## 11. Implementation Phases

### Phase 1 — Foundation

Core infrastructure that everything else depends on.

- [ ] Go server skeleton with single-port listener (WebSocket + HTTP).
- [ ] WebSocket message router with namespace-based dispatch.
- [ ] Ed25519 key generation, challenge-response authentication.
- [ ] GORM database layer with SQLite and PostgreSQL support.
- [ ] Wails client skeleton with Svelte frontend.
- [ ] Client-side SQLCipher storage.
- [ ] Server trust (TOFU) — store, verify, and warn on mismatch.
- [ ] Basic server config with hot-reload (owner public keys, port, DB settings).

### Phase 2 — Channels & Messaging

Text communication within a server.

- [ ] Channel and category CRUD.
- [ ] Message send, receive, edit, delete.
- [ ] Message persistence in server database.
- [ ] State sync — version-based diffing on connect, event-driven push while connected.
- [ ] Client field selection (sync.subscribe).
- [ ] File upload/download with tokenized HTTP endpoints.
- [ ] Message search with filters (from:, has:, before:, etc.).

### Phase 3 — Voice

Real-time voice communication.

- [ ] Pion SFU on server — receive and forward RTP packets.
- [ ] Pion client in Go backend — WebRTC connection, audio capture/playback.
- [ ] Voice channel join/leave signaling via WebSocket.
- [ ] Voice Activity Detection (VAD) + selective forwarding.
- [ ] E2EE voice — shared key exchange, RTP-level encryption.
- [ ] UI: participant grid, speaking indicators, mute/deafen controls.

### Phase 4 — DMs & Group Chats

Private end-to-end encrypted conversations.

- [ ] DM initiation between users sharing a server.
- [ ] Relay server selection (lowest latency).
- [ ] 1:1 E2EE — X25519 key exchange, symmetric encryption.
- [ ] Group DM — shared group key, distribution, rotation on member change.
- [ ] Offline message storage (encrypted blobs on relay server).
- [ ] Message signing for attribution in group chats.

### Phase 5 — Permissions & Administration

Server management and access control.

- [ ] Role system with configurable permissions.
- [ ] Private channels with role-based access.
- [ ] Server administration panel (all sections from 9.12).
- [ ] Moderation tools: kick, ban, message deletion.
- [ ] Audit log.

### Phase 6 — Polish & Hardening

- [ ] Dynamic Opus bitrate adjustment.
- [ ] Notification system.
- [ ] Appearance/theme settings.
- [ ] Performance optimization.
- [ ] Security audit of crypto implementations.
- [ ] Documentation for server administrators.

---

## 12. Future Considerations

These are explicitly **out of scope for v1** but acknowledged as potential future work.

- **Video streaming** (webcam + screen share): Architecture supports it via SFU + Pion. No v1 decisions block it.
- **Federation / server-to-server**: Possibly via a plugin system. Servers could share ban lists or bridge channels.
- **Plugin system**: Extensibility for community-built features.
- **Multi-server DM failover**: If the relay server goes down, automatically route through another shared server.
- **Multi-server DM redundancy**: Store encrypted DM blobs on multiple shared servers by default.
- **Bot/integration API**: Programmatic access for bots and integrations.
- **Semantic search**: pgvector-based embedding search for PostgreSQL deployments.
- **Mobile clients**: The Go backend + web frontend model could target mobile via WebView.
