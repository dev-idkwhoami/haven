# Haven — Implementation Data Models

> Concrete schemas for every entity in the Haven system.
> Derived from [SPEC.md](./SPEC.md). Used by implementation agents.

**Version:** 0.1.0 (Draft)
**Last Updated:** 2026-02-22

---

## Conventions

- **Types** use Go syntax (`string`, `int64`, `[]byte`, `time.Time`, etc.)
- **PK** = Primary Key, **FK** = Foreign Key, **UQ** = Unique, **IDX** = Indexed
- All timestamps are UTC `time.Time`
- `PublicKey` is always the raw 32-byte Ed25519 public key, stored as `[]byte` (hex-encoded in JSON)
- GORM tags are implied; these schemas are the source of truth for GORM model structs
- Server-side IDs use `string` (ULID via `oklog/ulid` — sortable, timestamp-embedded, 26 chars)
- Client-side IDs use `int64` (SQLCipher auto-increment)

---

## Server-Side Models

### 1. User

Identity and profile for an authenticated user on this server.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| PublicKey | []byte | UQ, IDX | Ed25519 public key (32 bytes) |
| DisplayName | string | | Cosmetic, user-chosen |
| Avatar | string | FK(File.ID), nullable | Reference to uploaded avatar file |
| AvatarHash | string | | Hash for client-side cache invalidation |
| Bio | string | nullable | Optional user bio |
| Status | string | | `"online"` \| `"idle"` \| `"dnd"` \| `"offline"` |
| Version | int64 | | Monotonic, bumped on any profile change |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- `IsOwner` is **not** a DB column — resolved at runtime by checking the server config's owner pubkey list (hot-reloadable).
- `Status` defaults to `"offline"`, set to `"online"` on WebSocket connect. Not persisted across server restarts.
- `Version` drives the sync system — clients track this to detect profile changes.

---

### 2. Server

Server metadata and runtime configuration (singleton row). Static config (owner pubkeys, port, DB connection) lives in the config file.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| Name | string | | Server display name |
| Description | string | nullable | Server description |
| Icon | string | FK(File.ID), nullable | Server avatar/icon |
| IconHash | string | | Cache invalidation hash |
| AccessMode | string | | `"open"` \| `"invite"` \| `"password"` \| `"allowlist"` |
| AccessPassword | string | nullable | Bcrypt hash, only when AccessMode = `"password"` |
| MaxFileSize | int64 | | Bytes, max single upload size |
| TotalStorageLimit | int64 | | Bytes, max total file storage |
| DefaultChannelID | string | FK(Channel.ID), nullable | Channel new users land in |
| WelcomeMessage | string | nullable | Sent to new users on join |
| Version | int64 | | Monotonic, for sync |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- Singleton — only one row ever exists. Created on first server boot.
- Owner pubkeys, port, DB connection string live in the **config file** (needed before DB init).
- `AccessPassword` is bcrypt-hashed, never plaintext.
- `MaxFileSize` and `TotalStorageLimit` are communicated to clients on connect.

---

### 3. Category

Channel grouping container, as seen in the sidebar.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| Name | string | | Category display name |
| Position | int | | Sort order in sidebar (zero-indexed, gaps allowed) |
| Type | string | | `"text"` \| `"voice"` — default channel type hint |
| Version | int64 | | Monotonic, for sync |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- `Type` is a hint from the Create Category modal (spec 9.11) — doesn't restrict what channels can be placed inside.
- `Position` uses gapped integers (0, 10, 20) to allow insertions without rewriting all rows.

---

### 4. Channel

Text or voice channel, belongs to a category.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| CategoryID | string | FK(Category.ID) | Parent category |
| Name | string | | Channel display name |
| Type | string | | `"text"` \| `"voice"` |
| Position | int | | Sort order within category (gapped integers) |
| OpusBitrate | int | nullable | Voice only: kbps (32/64/96/128), null = server default |
| Version | int64 | | Monotonic, for sync |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

### 4a. ChannelRoleAccess

Join table controlling which roles can access a channel.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ChannelID | string | PK, FK(Channel.ID) | |
| RoleID | string | PK, FK(Role.ID) | |

**Notes:**
- **No rows** for a channel → open to everyone. **Any rows** → only users holding one of those roles can see/join.
- Server owners (from config) always have access regardless of roles.
- Voice channels also have text chat (spec 9.9) — `Type` only affects the primary UX and whether voice infrastructure is spun up.
- `OpusBitrate` is per-channel voice quality config (spec section 5). Null = server default.

---

### 5. Message

A message in a text or voice channel.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID (sortable = chronological order for free) |
| ChannelID | string | FK(Channel.ID), IDX | Channel this message belongs to |
| AuthorID | string | FK(User.ID), IDX | Message author |
| Content | string | | Message text (max 64KB per WS limit) |
| Signature | []byte | | Ed25519 signature of (content \|\| channel_id \|\| timestamp \|\| nonce) |
| Nonce | []byte | | Random bytes used in signature |
| EditedAt | time.Time | nullable | Null if never edited |
| Version | int64 | | Monotonic, for sync |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- ULID as PK means messages are inherently sorted by creation time — no separate ordering column needed.
- `Signature` covers `content || channel_id || timestamp || nonce` (spec section 6). Clients verify on receipt.
- `EditedAt` distinguishes original vs edited. On edit, `Signature` is re-signed and `EditedAt` is set.
- File attachments linked via `MessageFile` join table (see model 6a).
- "Forget Me" → hard-delete all messages. "Ghost" → reassign `AuthorID` to placeholder user.

---

### 6. File

Uploaded file metadata. Created on upload completion, before the message referencing it is sent.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| UploaderID | string | FK(User.ID), IDX | User who uploaded the file |
| ChannelID | string | FK(Channel.ID), IDX, nullable | Channel it was uploaded to (null for avatars/icons) |
| Name | string | | Original filename |
| MimeType | string | | e.g. `"image/png"` |
| Size | int64 | | File size in bytes |
| StoragePath | string | | Server filesystem path to original (never exposed to clients) |
| ThumbPath | string | nullable | Path to generated thumbnail (images/videos only) |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

### 6a. MessageFile

Join table linking messages to their file attachments (many-to-many).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| MessageID | string | PK, FK(Message.ID) | |
| FileID | string | PK, FK(File.ID) | |

**Notes:**
- Clients access files via `/files/{file_id}` and `/files/{file_id}/thumb` — `StoragePath` is internal only.
- `ThumbPath` populated server-side on upload for images/videos. Non-media files have no thumbnail.
- `ChannelID` on File enables download permission checks — server verifies user has channel access. Null for avatars/icons (permission check falls back to server membership).
- "Forget Me" → delete all files where `UploaderID` matches. "Ghost" → reassign `UploaderID` to placeholder.

---

### 7. Role

Named permission set assigned to users. Permissions are stored as a bitfield.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| Name | string | UQ | Role display name (e.g. "Moderator", "Member") |
| Color | string | nullable | Hex color for display name tinting |
| Position | int | | Hierarchy: higher position = higher priority |
| IsDefault | bool | | Auto-assigned to new users on join |
| Permissions | int64 | | Bitfield of permission flags |
| Version | int64 | | Monotonic, for sync |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Permission bitfield:**

| Bit | Flag | Description |
|-----|------|-------------|
| 0 | ManageServer | Edit server settings |
| 1 | ManageChannels | Create/edit/delete channels and categories |
| 2 | ManageRoles | Create/edit/delete roles (below own position) |
| 3 | ManageMessages | Delete/pin others' messages |
| 4 | KickUsers | Kick users from server |
| 5 | BanUsers | Ban users from server |
| 6 | ManageInvites | Create/revoke invite codes |
| 7 | SendMessages | Send messages in text channels |
| 8 | AttachFiles | Upload files |
| 9 | JoinVoice | Connect to voice channels |
| 10 | Speak | Transmit audio in voice channels |

**Notes:**
- `Position` establishes hierarchy — users can only manage roles with a lower position than their highest role. Prevents privilege escalation.
- `IsDefault` marks the role auto-assigned to every new user. Typically "Member" with basic permissions.
- Bitfield check: `role.Permissions & PermSendMessages != 0`.
- Server owners (from config) implicitly have all permissions — no role check needed.

---

### 8. UserRole

Join table assigning roles to users (many-to-many).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| UserID | string | PK, FK(User.ID) | |
| RoleID | string | PK, FK(Role.ID) | |

**Notes:**
- A user can have multiple roles. Effective permissions = bitwise OR of all assigned roles' permission fields.
- Default role (`IsDefault = true`) is assigned automatically on registration — a UserRole row is created.
- "Forget Me" / "Ghost" → delete all UserRole rows for that user.

---

### 9. Ban

Tracks banned users. Banned public keys are rejected during the auth access control gate.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| PublicKey | []byte | IDX | Banned user's Ed25519 public key |
| Reason | string | nullable | Admin-provided reason |
| BannedBy | string | FK(User.ID), nullable | Admin who issued the ban (ON DELETE SET NULL) |
| ExpiresAt | time.Time | nullable | Null = permanent ban |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- Keyed on `PublicKey` (not `UserID`) because a banned user's User record may be deleted ("Forget Me"). The ban must persist regardless.
- `ExpiresAt` enables temp bans. Server checks expiry during auth.
- Server owners (from config) cannot be banned — auth flow skips ban checks for owner pubkeys.
- Checked during step 7 of the handshake (spec section 3), after identity verification but before session creation.

---

### 10. AuditLogEntry

Records admin/moderator actions for accountability. Append-only.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID (sortable = chronological for free) |
| ActorID | string | FK(User.ID), nullable | User who performed the action (ON DELETE SET NULL) |
| Action | string | | Action type (see below) |
| TargetType | string | | Entity type: `"user"` \| `"channel"` \| `"category"` \| `"role"` \| `"message"` \| `"server"` \| `"invite"` |
| TargetID | string | | ID of the affected entity |
| Details | string | nullable | JSON blob with action-specific context |
| CreatedAt | time.Time | | GORM auto |

**Action types:**

| Namespace | Actions |
|-----------|---------|
| user | `user.kick`, `user.ban`, `user.unban`, `user.role.add`, `user.role.remove` |
| channel | `channel.create`, `channel.update`, `channel.delete` |
| category | `category.create`, `category.update`, `category.delete` |
| role | `role.create`, `role.update`, `role.delete` |
| message | `message.delete` (moderator deleting another user's message) |
| server | `server.update` |
| invite | `invite.create`, `invite.revoke` |

**Notes:**
- Read-only, append-only. No `UpdatedAt` — entries are never modified.
- `Details` is a JSON string for flexibility — e.g. `{"reason": "spam", "expires_at": "..."}` for bans, `{"field": "name", "old": "general", "new": "main"}` for updates.
- Only actions requiring elevated permissions generate audit entries. Normal user actions do not.

---

### 11. InviteCode

For servers with `AccessMode = "invite"`. Single-use or multi-use codes.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| Code | string | UQ, IDX | The invite code string (short, human-typeable) |
| CreatedBy | string | FK(User.ID), nullable | Admin who created it (ON DELETE SET NULL) |
| UsesLeft | int | nullable | Remaining uses. Null = unlimited. Decremented on use, rejected at 0 |
| ExpiresAt | time.Time | nullable | Null = never expires |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- `Code` is the string users share (e.g. "ABC-XYZ-123"). Generated server-side.
- Valid when: `UsesLeft` is null OR `UsesLeft > 0`, AND `ExpiresAt` is null OR `ExpiresAt > now`.
- Used during auth step 5 (spec section 3) — client sends code as `access_token`, server validates and decrements `UsesLeft`.
- Creating/revoking invites requires the `ManageInvites` permission.

---

### 12. DMConversation

Unified model for both 1:1 DMs and group DMs. The server acts as a blind relay — stores encrypted blobs but cannot read them.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| IsGroup | bool | | `false` = 1:1 DM, `true` = group DM |
| Name | string | nullable | Group name (null for 1:1 DMs) |
| CreatedBy | string | FK(User.ID), nullable | Conversation initiator (ON DELETE SET NULL) |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

### 12a. DMParticipant

Tracks membership in a DM conversation.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ConversationID | string | PK, FK(DMConversation.ID) | |
| UserID | string | PK, FK(User.ID) | |
| IsKeyManager | bool | | Can rotate group key, add/remove members |
| JoinedAt | time.Time | | When they were added |
| LeftAt | time.Time | nullable | Null = still active member |

### 12b. DMMessage

Encrypted DM message blob. The server stores and forwards but cannot read.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID (sortable = chronological) |
| ConversationID | string | FK(DMConversation.ID), IDX | |
| SenderID | string | FK(User.ID) | |
| EncryptedPayload | []byte | | E2EE blob (ChaCha20-Poly1305) |
| CreatedAt | time.Time | | GORM auto |

**Notes:**
- 1:1 DMs: `IsGroup = false`, exactly 2 participants, `IsKeyManager` irrelevant (both derive shared secret via X25519).
- Group DMs: `IsGroup = true`, 2+ participants, `IsKeyManager` tracks who can rotate the group key (spec section 6).
- `LeftAt` enables soft-leave — participant history preserved for key rotation audit. When set, user no longer receives messages.
- `Name` is only for group DMs. 1:1 DMs display the other participant's name client-side.
- Voice calls use the same `ConversationID` for signaling — no separate voice model needed.
- DMMessage is read-only server-side — no `UpdatedAt`. Edits/deletes happen inside encrypted payloads as client-side logic.
- Offline delivery: messages accumulate until recipient connects and retrieves them.
- "Forget Me" → delete all DMMessage rows where `SenderID` matches, remove from all DMParticipant rows, delete DMConversation if no participants remain.

---

### 13. ErasureRecord

Tracks Ghost/Forget Me departures for client-side cache propagation. Append-only.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| PublicKey | []byte | IDX | Departed user's Ed25519 public key |
| Mode | string | | `"ghost"` \| `"forget"` |
| ErasedAt | time.Time | | When the erasure was performed |
| CreatedAt | time.Time | | GORM auto |

**Notes:**
- Append-only, never modified or deleted.
- On `sync.request`, clients receive ErasureRecords newer than their last sync, then purge local caches.
- `"ghost"` → client replaces cached name/avatar with "Deleted User" placeholder. `"forget"` → client hard-deletes all cached data for that pubkey.
- Best-effort propagation — a client that never reconnects retains cached data (spec section 6).
- Keyed on `PublicKey` (not UserID) because the User record may already be deleted.

### Ghost Mode — Sentinel User & Procedure

When a user departs with Ghost mode, their content is reassigned to a **sentinel placeholder user**. One sentinel row exists per server, created on first Ghost departure.

**Sentinel User row:**

| Field | Value |
|-------|-------|
| ID | `"00000000000000000000000000"` (26-char zero ULID) |
| PublicKey | 32 zero bytes (`0x00...00`) |
| DisplayName | `"Deleted User"` |
| Avatar | null |
| Bio | null |
| Status | `"offline"` (always) |
| Version | 0 (never synced) |

**Why a real row, not null?** Keeping a real FK target avoids nullable `AuthorID` on Message and lets the client render "Deleted User" naturally from the user list — no special-casing in the UI.

**Ghost procedure (server-side, atomic transaction):**

1. Create sentinel User row if it doesn't already exist (idempotent).
2. `UPDATE messages SET author_id = sentinel WHERE author_id = departing_user`.
3. `UPDATE files SET uploader_id = sentinel WHERE uploader_id = departing_user`.
4. `DELETE FROM user_roles WHERE user_id = departing_user`.
5. `UPDATE dm_participants SET left_at = now() WHERE user_id = departing_user AND left_at IS NULL`.
6. `DELETE FROM users WHERE id = departing_user`.
7. `INSERT INTO erasure_records (public_key, mode, erased_at) VALUES (departing_pubkey, 'ghost', now())`.
8. Broadcast `event.user.erased` with `mode = "ghost"`.
9. Close the departing user's WebSocket.

**Forget Me procedure (server-side, atomic transaction):**

1. `DELETE FROM messages WHERE author_id = departing_user`.
2. Delete all files where `uploader_id = departing_user` (both DB rows and disk files).
3. `DELETE FROM dm_messages WHERE sender_id = departing_user`.
4. `UPDATE dm_participants SET left_at = now() WHERE user_id = departing_user AND left_at IS NULL`.
5. Delete DMConversations with no remaining active participants.
6. `DELETE FROM user_roles WHERE user_id = departing_user`.
7. `DELETE FROM users WHERE id = departing_user`.
8. `INSERT INTO erasure_records (public_key, mode, erased_at) VALUES (departing_pubkey, 'forget', now())`.
9. Broadcast `event.user.erased` with `mode = "forget"`.
10. Close the departing user's WebSocket.

---

### 14. Session

Active session tokens for HTTP request authentication.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | string | PK | ULID |
| UserID | string | FK(User.ID), IDX, nullable | Owning user (nullable, ON DELETE SET NULL) |
| Token | string | UQ, IDX | Opaque session token (crypto/rand, base64) |
| ExpiresAt | time.Time | | Absolute expiry (disconnect time + grace period) |
| CreatedAt | time.Time | | GORM auto |
| UpdatedAt | time.Time | | GORM auto |

**Notes:**
- Created on successful WebSocket authentication. Token returned to client.
- `ExpiresAt` initially set far into future (while WS active). On disconnect, updated to `now + grace_period` (default 5 min).
- Validates HTTP requests: `Authorization: Bearer <token>` → look up by Token, check `ExpiresAt > now`, resolve UserID.
- On reconnect within grace period: client reuses token, server finds valid session, skips full re-auth.
- Invalidated on: logout, kick, ban, grace period expiry. Expired sessions cleaned up periodically.

---

## Client-Side Models (SQLCipher)

### 15. TrustedServer

TOFU trust store — maps server addresses to their public keys (like SSH `known_hosts`).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| Address | string | UQ, IDX | Server address as typed by user (e.g. "myserver.com:8443") |
| PublicKey | []byte | | Server's Ed25519 public key (32 bytes) |
| Name | string | nullable | Cached server display name |
| Icon | []byte | nullable | Cached server icon binary |
| IconHash | string | nullable | For cache invalidation |
| SessionToken | string | nullable | Current session token (valid while connected + grace period) |
| IsRelayOnly | bool | | True if server is in relay-only mode for this user |
| FirstTrustedAt | time.Time | | When the user first accepted this server's key |
| LastConnectedAt | time.Time | nullable | Last successful connection |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Notes:**
- `Address` is what the user typed — canonical identifier for this server from the client's perspective.
- Key mismatch detection: on connect, compare presented key against `PublicKey`. Mismatch → block + warn (spec 9.6).
- `SessionToken` persists across app restarts to allow reconnects within the grace period without re-auth.
- `IsRelayOnly` marks servers in relay-only mode (spec section 6) — appear in Settings → Relay Servers, not the main server list.
- `Name` and `Icon` are cached metadata so the server list renders instantly before connecting.

---

### 16. LocalProfile

The user's own identity and profile. Singleton row.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment (singleton, always row 1) |
| PublicKey | []byte | UQ | Ed25519 public key (32 bytes) |
| DisplayName | string | | User's display name |
| Avatar | []byte | nullable | Avatar image binary |
| AvatarHash | string | nullable | Hash for sync |
| Bio | string | nullable | User bio |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Notes:**
- Singleton — only one row. The user has one identity.
- The **private key** is NOT stored here — it lives in the OS credential store (spec section 8).
- Profile data is sent to each server on connect. Servers cache it in their own User table.
- `AvatarHash` is used during sync — if the server's cached hash differs, the client re-uploads.

---

### 17. CachedUser

Cached remote user profiles from servers. Allows offline display of user info.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ServerID | int64 | FK(TrustedServer.ID), IDX | Which server this cache is from |
| PublicKey | []byte | IDX | Ed25519 public key |
| DisplayName | string | | Cached display name |
| Avatar | []byte | nullable | Cached avatar binary |
| AvatarHash | string | nullable | For cache invalidation |
| Bio | string | nullable | Cached bio |
| Version | int64 | | Last known version (for sync diffing) |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Unique constraint:** `(ServerID, PublicKey)`

**Notes:**
- Keyed on `(ServerID, PublicKey)` — same pubkey can appear on multiple servers.
- Server sync payloads reference users by pubkey, not internal ULID.
- `Version` compared during `sync.request` to determine if update is needed.
- `Avatar` only cached if the client's field selection includes avatars for this server.
- On `ErasureRecord`: `"ghost"` → replace with placeholder. `"forget"` → delete the row.

---

### 18. CachedMessage

Locally cached messages from server channels. Enables offline browsing and instant display on reconnect.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ServerID | int64 | FK(TrustedServer.ID), IDX | Which server this message is from |
| RemoteMessageID | string | IDX | Remote Message ULID (for sync correlation) |
| ChannelID | string | IDX | Remote channel ULID |
| AuthorPubKey | []byte | IDX | Message author's public key |
| Content | string | | Message text |
| Signature | []byte | | Ed25519 signature |
| Nonce | []byte | | Signature nonce |
| EditedAt | time.Time | nullable | Null if never edited |
| RemoteCreatedAt | time.Time | | Original timestamp from server |
| Version | int64 | | For sync diffing |
| CreatedAt | time.Time | | Auto (when cached locally) |
| UpdatedAt | time.Time | | Auto |

**Notes:**
- `RemoteMessageID` enables sync correlation — server sends "message X edited" → client finds local row by `(ServerID, RemoteMessageID)`.
- `ChannelID` stored as the remote ULID string.
- `AuthorPubKey` instead of user ID — consistent with pubkey-based identity on the client.
- `RemoteCreatedAt` preserves the original message timestamp. Local `CreatedAt` is when it was cached.
- Signature is cached so the client can re-verify integrity at any time.
- On `ErasureRecord`: `"ghost"` → null out `AuthorPubKey`. `"forget"` → delete the row.
- Old messages can be evicted by the client based on age or storage limits (client-side policy).

---

### 19. PerServerConfig

Per-server client preferences — field selection overrides, sync settings (spec section 7).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ServerID | int64 | FK(TrustedServer.ID), UQ | One config per server |
| SyncAvatars | bool | | Receive avatar data from this server |
| SyncBios | bool | | Receive bio data |
| SyncStatus | bool | | Receive online/idle/dnd status |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Notes:**
- One row per server. Created with defaults when user first connects.
- Booleans map to the `sync.subscribe` payload fields. Client builds the field list from these flags before sending.
- Defaults are defined in a separate DefaultSyncConfig singleton (or hardcoded). Per-server rows override.
- Extensible — new fields added as Haven gains more syncable data types.

---

### 20. CachedCategory

Cached server categories for offline sidebar rendering.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ServerID | int64 | FK(TrustedServer.ID), IDX | Which server this is from |
| RemoteCategoryID | string | | Remote ULID |
| Name | string | | Cached category name |
| Position | int | | Sort order |
| Type | string | | `"text"` \| `"voice"` |
| Version | int64 | | For sync diffing |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Unique constraint:** `(ServerID, RemoteCategoryID)`

---

### 21. CachedChannel

Cached server channels for offline sidebar rendering.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ServerID | int64 | FK(TrustedServer.ID), IDX | Which server this is from |
| RemoteChannelID | string | | Remote ULID |
| RemoteCategoryID | string | | Remote category ULID (for grouping) |
| Name | string | | Cached channel name |
| Type | string | | `"text"` \| `"voice"` |
| Position | int | | Sort order within category |
| LastReadMessageID | string | nullable | Remote ULID of last read message (local unread watermark) |
| Version | int64 | | For sync diffing |
| CreatedAt | time.Time | | Auto |
| UpdatedAt | time.Time | | Auto |

**Unique constraint:** `(ServerID, RemoteChannelID)`

**Notes:**
- Enable offline sidebar rendering — channel names, categories, ordering available without connecting.
- `Version` compared during `sync.request` to get only changed channels/categories.
- `RemoteCategoryID` on CachedChannel links to CachedCategory by remote ID (resolved by matching `ServerID + RemoteCategoryID`, not a local FK).
- `LastReadMessageID` is the local read watermark — updated when the user views a channel. Unread count = messages with ULID > this value. Purely client-side, never sent to server unless `ShowReadReceipts` is enabled.
- Deleted when the server's TrustedServer entry is removed.

---

### 22. CachedDMMessage

Locally cached decrypted DM messages. Safe because SQLCipher encrypts the entire DB at rest.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| ID | int64 | PK | Auto-increment |
| ConversationID | string | IDX | Remote DMConversation ULID |
| SenderPubKey | []byte | IDX | Sender's public key |
| Content | string | | Decrypted message text |
| Signature | []byte | | Ed25519 signature (from inside the E2EE blob) |
| Nonce | []byte | | Signature nonce |
| RemoteCreatedAt | time.Time | | Original timestamp from server |
| CreatedAt | time.Time | | Auto (when cached locally) |
| UpdatedAt | time.Time | | Auto |

**Notes:**
- Stores **decrypted** content. SQLCipher encrypts the entire DB at rest with the user's key.
- `ConversationID` is the remote ULID — not a local FK. Groups messages by conversation.
- `SenderPubKey` for author identity, consistent with `CachedMessage.AuthorPubKey`.
- No `ServerID` — DMs are identified by conversation, not by which server relayed them.
- `Signature` preserved for re-verification of message integrity/attribution.
- On erasure: `"ghost"` → null out `SenderPubKey`. `"forget"` → delete matching rows.

---

## Server Configuration File

The server reads `haven-server.toml` on startup. Fields marked **[HOT]** are hot-reloadable (file watcher applies changes without restart). All other fields require a server restart.

The `[defaults]` section is **only read on first boot** to seed the database. After that, the admin panel / DB is the source of truth for runtime settings.

```toml
# haven-server.toml — Haven Server Configuration

# ─── Identity ───────────────────────────────────────────────────
[identity]
# Path to the server's Ed25519 private key file.
# Generated automatically on first boot if it doesn't exist.
private_key_path = "data/server.key"

# ─── Network ────────────────────────────────────────────────────
[network]
# Address and port to listen on. Single port for WS + HTTP + UDP voice.
listen_address = "0.0.0.0"
port = 9090

# ─── Database ────────────────────────────────────────────────────
[database]
# "sqlite" or "postgres"
driver = "sqlite"
# SQLite: file path. Postgres: connection string.
dsn = "data/haven.db"

# ─── Owners ──────────────────────────────────────────────────────
# [HOT] List of owner public keys (hex-encoded Ed25519).
# Owners have all permissions and cannot be banned or kicked.
[owners]
public_keys = []

# ─── Defaults ────────────────────────────────────────────────────
# Initial values seeded into the DB on first boot only.
# After first boot, the DB is the source of truth (admin panel).
[defaults]
server_name = "My Haven Server"
access_mode = "open"                # "open" | "invite" | "password" | "allowlist"
max_file_size = 52428800            # 50 MB
total_storage_limit = 21474836480   # 20 GB

# ─── Rate Limits ─────────────────────────────────────────────────
# [HOT] Relaxed by default. Only tighten if abuse occurs.
[rate_limits]
messages_per_second = 20            # per-client WS message rate
message_burst = 50                  # burst allowance
auth_attempts_per_minute = 10       # per-IP auth rate limit
registrations_per_ip_per_hour = 20  # per-IP new user registration limit
concurrent_uploads = 5              # per-client concurrent upload limit

# ─── Session ─────────────────────────────────────────────────────
[session]
grace_period_seconds = 300          # 5 min after WS disconnect before session expires

# ─── Allowlist ───────────────────────────────────────────────────
# [HOT] Only used when access_mode = "allowlist" (in DB).
# Listed public keys are the only ones permitted to connect.
[allowlist]
public_keys = []
```

**Hot-reloadable sections:** `owners`, `rate_limits`, `allowlist`.

**Not hot-reloadable (require restart):** `identity`, `network`, `database`.

**Notes:**
- Private key is auto-generated on first boot if the file doesn't exist.
- `owners.public_keys` starts empty — the admin must add their own pubkey after first launch.
- `allowlist.public_keys` is only consulted when the DB's `AccessMode = "allowlist"`.
- Rate limits are intentionally relaxed — normal users should never hit them. Admins tighten if abuse occurs.

---

## Auth Handshake Sequence

Concrete message payloads for the mutual authentication flow (spec section 3).

### Step 1 — Client Opens WebSocket

```
Client connects to wss://server.com/ws (or ws://IP:port/ws)
No application-level message — just the WS upgrade.
```

### Step 2 — Server Hello

Server proves its identity and challenges the client in a single message:

```json
{
  "type": "auth.hello",
  "payload": {
    "server_pubkey": "hex-encoded Ed25519 public key (64 hex chars)",
    "server_nonce": "hex-encoded 32 random bytes",
    "server_signature": "hex-encoded Ed25519 signature of server_nonce",
    "challenge_nonce": "hex-encoded 32 random bytes",
    "access_mode": "open",
    "server_name": "My Haven Server",
    "server_version": "0.1.0"
  }
}
```

**Client-side processing:**

1. Verify `server_signature` against `server_pubkey` — proves server holds the private key.
2. **TOFU check**: compare `server_pubkey` against stored key for this address.
   - First connection → show trust prompt (spec 9.5).
   - Known + matches → proceed.
   - Known + mismatch → block, show warning (spec 9.6).
3. If `access_mode` is `"invite"` or `"password"` and this is a new user → prompt for input.

**UX note:** The Join Server modal (spec 9.4) does **not** open a separate modal for invite codes or passwords. Instead, the same modal dynamically shows the required field based on the `access_mode` received from the server. Flow: user enters address → client connects → receives `auth.hello` → modal updates in-place to show an invite code or password field if needed → user fills it in → client sends `auth.respond`. One modal, no chaining.

### Step 3 — Client Response

Client proves its identity and provides access credentials:

```json
{
  "type": "auth.respond",
  "payload": {
    "client_pubkey": "hex-encoded Ed25519 public key",
    "signature": "hex-encoded Ed25519 signature of (challenge_nonce || server_pubkey)",
    "access_token": "invite-code-or-password-or-null",
    "session_token": "existing-session-token-or-null",
    "profile": {
      "display_name": "Alice",
      "avatar_hash": "a1b2c3",
      "bio": "Hello!"
    }
  }
}
```

**Notes:**
- `signature` signs `challenge_nonce || server_pubkey` — domain separation prevents cross-server replay.
- `access_token`: invite code (`mode=invite`), password (`mode=password`), or null (`mode=open/allowlist`).
- `session_token`: if reconnecting within grace period, send stored token to skip full re-auth.
- `profile`: current profile data so the server can update its cached User record if changed.

### Step 4 — Server Verification

Server-side processing (no message sent yet):

1. If `session_token` is valid and not expired → **fast path**: restore session, skip to Step 5 success.
2. Verify `signature` against `client_pubkey` — proves client holds the private key.
3. **Ban check**: is `client_pubkey` in the Ban table (and not expired)? → reject.
4. **Identity resolution**: known pubkey → returning user. Unknown → new user.
5. **Access control** (new users only):
   - `open` → allow.
   - `invite` → validate `access_token` against InviteCode table, decrement `UsesLeft`.
   - `password` → compare `access_token` against bcrypt hash in `Server.AccessPassword`.
   - `allowlist` → check `client_pubkey` against config allowlist.
6. **Registration** (new users only): create User record, assign default role.
7. **Create session**: generate Session token, store in Session table.
8. Update User profile if `profile` data differs from stored.

### Step 5a — Auth Success

```json
{
  "type": "auth.success",
  "payload": {
    "session_token": "opaque-session-token",
    "user_id": "01JXYZ...",
    "encryption_required": false,
    "server_info": {
      "max_file_size": 52428800,
      "total_storage_limit": 21474836480,
      "rate_limits": {
        "messages_per_second": 20,
        "message_burst": 50,
        "concurrent_uploads": 5
      }
    }
  }
}
```

- `session_token`: client stores for HTTP auth and reconnection.
- `encryption_required`: `true` if connection is `ws://` — triggers app-layer encryption setup (Step 6).
- `server_info`: limits the client needs for UI and local enforcement.

### Step 5b — Auth Error

```json
{
  "type": "auth.error",
  "payload": {
    "code": "BANNED",
    "message": "You are banned from this server."
  }
}
```

**Error codes:** `INVALID_SIGNATURE`, `BANNED`, `INVALID_INVITE`, `INVALID_PASSWORD`, `NOT_ALLOWLISTED`, `RATE_LIMITED`, `SESSION_EXPIRED`.

Server closes the WebSocket after sending `auth.error`.

### Step 6 — App-Layer Encryption (ws:// only)

Only when `encryption_required = true` in `auth.success`. No additional messages — both sides derive the same key from known material:

```
Both sides independently:
  1. Convert their Ed25519 private key → X25519 private key.
  2. Convert the peer's Ed25519 public key → X25519 public key.
  3. X25519 DH → raw shared secret.
  4. Derive symmetric key:
     HKDF-SHA256(
       ikm  = raw_shared_secret,
       salt = server_nonce || challenge_nonce,
       info = "haven-ws-encryption"
     ) → 32-byte ChaCha20-Poly1305 key.
  5. All subsequent WS frames are encrypted with this key.
     Each frame uses an incrementing nonce counter
     (uint64, little-endian, zero-padded to 12 bytes).
```

The nonces from the handshake provide session uniqueness — same key pair produces a different symmetric key each connection.

### Reconnection Flow

```
Client has stored session_token + server pubkey:
  1. Open WS → receive auth.hello
  2. Verify server identity (TOFU — should match stored key)
  3. Send auth.respond with session_token set
  4. Server validates token → fast-path auth.success
  5. If token expired → auth.error with SESSION_EXPIRED
     → client falls back to full auth (new auth.respond without session_token)
```

---

## WebSocket Message Catalog

Concrete payloads for every message type. Auth messages are defined in the handshake section above.

**Conventions:**
- Request: `{ "type": "namespace.action", "id": "msg_id", "payload": {} }`
- Success: `{ "type": "namespace.action.ok", "id": "msg_id", "payload": {} }`
- Error: `{ "type": "namespace.action.error", "id": "msg_id", "payload": { "code": "...", "message": "..." } }`
- Event: `{ "type": "event.namespace.action", "payload": {} }` (no `id` — fire-and-forget)
- All fields marked `?` are optional/nullable.

---

### server.*

#### server.info

Get current server metadata. Called after auth for initial load.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| name | string | Server display name |
| description | string? | Server description |
| icon_id | string? | File ID of server icon |
| icon_hash | string? | Icon cache hash |
| access_mode | string | `"open"` \| `"invite"` \| `"password"` \| `"allowlist"` |
| member_count | int | Current member count |
| version | int64 | Server entity version |

#### server.update

Edit server settings. Requires `ManageServer` permission.

**Request payload:** *(all fields optional — only send what changed)*

| Field | Type | Description |
|-------|------|-------------|
| name | string? | New server name |
| description | string? | New description |
| icon_id | string? | New icon file ID |
| access_mode | string? | New access mode |
| access_password | string? | New password (plaintext, server bcrypt-hashes it) |
| max_file_size | int64? | New max file size (bytes) |
| total_storage_limit | int64? | New total storage limit (bytes) |
| default_channel_id | string? | New default channel |
| welcome_message | string? | New welcome message |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New server version |

**Event broadcast:** `event.server.updated` — full snapshot of server metadata (same shape as `server.info` response), sent to all connected clients.

**Notes:**
- Triggers audit log entry (`server.update`).
- `access_password` sent plaintext over the already-encrypted WS — server hashes before storing.

---

### category.*

#### category.list

List all categories on the server.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| categories | array | List of category objects |

**Category object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| name | string | Category name |
| position | int | Sort order |
| type | string | `"text"` \| `"voice"` |
| version | int64 | Entity version |

#### category.create

Create a new category. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| name | string | Category name |
| type | string | `"text"` \| `"voice"` |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | New category ULID |
| position | int | Assigned position (appended at end) |
| version | int64 | Entity version |

**Event broadcast:** `event.category.created` — full category object.

#### category.update

Update a category. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Category ULID |
| name | string? | New name |
| position | int? | New position |
| type | string? | New type |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New version |

**Event broadcast:** `event.category.updated` — full category object.

#### category.delete

Delete a category and all its channels. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Category ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.category.deleted`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Deleted category ULID |
| deleted_channel_ids | string[] | IDs of channels deleted by cascade |

**Notes:**
- Deleting a category cascades to all channels inside it. Event includes deleted channel IDs so clients can clean up.
- All mutations trigger audit log entries.
- `position` on create is auto-assigned (appended at end). Reorder via `category.update`.

---

### channel.*

#### channel.list

List channels, optionally filtered by category.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| category_id | string? | Filter by category (omit = all channels) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| channels | array | List of channel objects |

**Channel object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| category_id | string | Parent category |
| name | string | Channel name |
| type | string | `"text"` \| `"voice"` |
| position | int | Sort order within category |
| opus_bitrate | int? | Voice only, null = server default |
| role_ids | string[] | Roles with access (empty = open to all) |
| version | int64 | Entity version |

#### channel.create

Create a new channel. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| category_id | string | Parent category |
| name | string | Channel name |
| type | string | `"text"` \| `"voice"` |
| role_ids | string[]? | Restrict to these roles (omit/empty = open) |
| opus_bitrate | int? | Voice only: kbps |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | New channel ULID |
| position | int | Assigned position (appended at end) |
| version | int64 | Entity version |

**Event broadcast:** `event.channel.created` — full channel object.

#### channel.update

Update a channel. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Channel ULID |
| name | string? | New name |
| category_id | string? | Move to different category |
| position | int? | New position |
| role_ids | string[]? | New role access list (full replace, not merge) |
| opus_bitrate | int? | New bitrate (voice only) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New version |

**Event broadcast:** `event.channel.updated` — full channel object.

#### channel.delete

Delete a channel. Requires `ManageChannels` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Channel ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.channel.deleted`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Deleted channel ULID |

**Notes:**
- `role_ids` maps to the `ChannelRoleAccess` join table. Empty array = no restrictions.
- Moving a channel to a different category via `category_id` resets its position to end of target category.
- Clients without a required role won't receive the channel in `channel.list` or events — server filters.
- All mutations trigger audit log entries.

---

### message.*

#### Message Object

Used in events and responses throughout this namespace.

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| channel_id | string | Channel ULID |
| author_pubkey | bytes | Author's Ed25519 public key |
| content | string | Message text |
| signature | bytes | Ed25519 signature |
| nonce | bytes | Signature nonce |
| file_ids | string[] | Attached file IDs |
| edited_at | timestamp? | Null if never edited |
| created_at | timestamp | Server timestamp |
| version | int64 | Entity version |

#### message.send

Send a message to a channel. Requires `SendMessages` permission. `AttachFiles` required if `file_ids` is non-empty.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Target channel |
| content | string | Message text |
| signature | bytes | Ed25519 sig of (content \|\| channel_id \|\| timestamp \|\| nonce) |
| nonce | bytes | Random bytes used in signature |
| file_ids | string[]? | Attached file IDs (from file.upload flow) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | New message ULID |
| created_at | timestamp | Server-assigned timestamp |
| version | int64 | Entity version |

**Event broadcast:** `event.message.new`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel the message was sent to |
| message | object | Full message object |

#### message.edit

Edit own message. Re-signs with updated content.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Message ULID |
| content | string | New content |
| signature | bytes | Re-signed with new content |
| nonce | bytes | New nonce |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New version |
| edited_at | timestamp | Edit timestamp |

**Event broadcast:** `event.message.edited`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |
| id | string | Message ULID |
| content | string | New content |
| signature | bytes | New signature |
| nonce | bytes | New nonce |
| edited_at | timestamp | Edit timestamp |
| version | int64 | New version |

#### message.delete

Delete a message. Author can delete own; `ManageMessages` permission can delete anyone's.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Message ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.message.deleted`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |
| id | string | Deleted message ULID |

#### message.history

Fetch message history with cursor-based pagination.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Target channel |
| before | string? | Message ULID — fetch messages older than this (cursor) |
| limit | int? | Max messages to return (default 50, max 100) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| messages | array | List of full message objects |
| has_more | bool | True if older messages exist |

#### message.search

Search messages with Discord-style filters.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| text | string? | Full-text search query |
| channel_id | string? | Filter by channel |
| from_pubkey | bytes? | Filter by author public key |
| has | string[]? | `"file"` \| `"image"` \| `"link"` |
| before | timestamp? | Messages before this date |
| after | timestamp? | Messages after this date |
| limit | int? | Max results (default 25, max 50) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| messages | array | List of full message objects |
| total_count | int | Total matches (for pagination UI) |

#### message.typing

Notify that the current user is typing in a channel.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.message.typing`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |
| pubkey | bytes | Typing user's public key |

**Notes:**
- Fire-and-forget — no response, no error. If rate-limited, silently dropped.
- Client sends on first keystroke, then no more often than every 5 seconds while still typing.
- Server broadcasts to all channel members whose field selection includes typing indicators.
- Receiving clients show "User is typing..." for 6 seconds (or until a message arrives from that user).
- No explicit "stopped typing" message — the indicator simply times out.

#### message.read

Mark messages as read up to a given point in a channel. Opt-in broadcast for "seen by" indicators.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |
| last_read_id | string | ULID of the newest message the client has read |

**Response payload:** *(none)*

**Event broadcast:** `event.message.read`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Channel ULID |
| pubkey | bytes | Reader's public key |
| last_read_id | string | ULID they've read up to |

**Notes:**
- Watermark-based — "I've read up to message X" rather than per-message acks. Simpler and less traffic.
- Only sent if the user's `ShowReadReceipts` setting is enabled. If off, nothing is sent — server never knows what they've read.
- Broadcast only to clients whose field selection includes read receipts.
- Client sends on channel focus/scroll, debounced (no more than once per second).
- **Local unread tracking** is purely client-side — `CachedChannel.LastReadMessageID` stores the watermark in SQLCipher. Unread counts are computed locally. No server involvement.
- **"Seen by" broadcast** is the opt-in WS message above — server relays to others who also opted in.

**message.* general notes:**
- `message.edit` — only the author can edit their own messages.
- `message.delete` — mod deletes trigger audit log entry.
- `message.history` uses cursor pagination via `before` (ULID of oldest message in current view). First load omits `before`.
- `message.search` uses `from_pubkey` (not user ID) — server resolves pubkey → user internally.
- Message objects include `author_pubkey` so clients can identify users cross-server by key.

---

### user.*

#### user.profile

Get a user's profile. Fields returned respect the requesting client's field selection.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes? | Target user's pubkey (omit = self) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Ed25519 public key |
| display_name | string | Display name |
| avatar_id | string? | File ID |
| avatar_hash | string? | Cache hash |
| bio | string? | User bio |
| status | string | `"online"` \| `"idle"` \| `"dnd"` \| `"offline"` |
| roles | string[] | Role IDs assigned to this user |
| version | int64 | Entity version |

#### user.update

Update own profile.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| display_name | string? | New display name |
| avatar_id | string? | New avatar file ID |
| bio | string? | New bio |
| status | string? | `"online"` \| `"idle"` \| `"dnd"` |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New version |

**Event broadcast:** `event.user.updated` — only changed fields included:

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | User's public key |
| display_name | string? | New display name |
| avatar_id | string? | New avatar file ID |
| avatar_hash | string? | New avatar hash |
| bio | string? | New bio |
| status | string? | New status |
| version | int64 | New version |

#### user.list

List all members visible to the requesting user.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| users | array | List of user summary objects |

**User summary object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Ed25519 public key |
| display_name | string | Display name |
| avatar_hash | string? | Cache hash |
| status | string | Online status |
| roles | string[] | Role IDs |
| version | int64 | Entity version |

#### user.kick

Kick a user from the server. Requires `KickUsers` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Target user's public key |

**Response payload:** *(none)*

**Event broadcast:** `event.user.kicked`

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Kicked user's public key |

#### user.leave

Leave the server with a chosen departure mode.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| mode | string | `"leave"` \| `"ghost"` \| `"forget"` |

**Response payload:** *(none)* — server closes WebSocket after processing.

**Event broadcast:** `event.user.erased` (only for `"ghost"` and `"forget"` modes)

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Departed user's public key |
| mode | string | `"ghost"` \| `"forget"` |

**Notes:**
- `user.profile` fields are filtered by the requesting client's field selection (PerServerConfig).
- `user.update` only modifies own profile. Server bumps Version and broadcasts.
- `user.list` respects field selection — if client didn't subscribe to bios, bios are omitted.
- `user.kick` disconnects the target's WebSocket. They can rejoin (unless also banned).
- `user.leave` with `"ghost"`/`"forget"` is irreversible. Server creates ErasureRecord, performs data cleanup (spec section 6), closes connection.
- Kick and leave trigger audit log entries.

---

### role.*

#### role.list

List all roles on the server.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| roles | array | List of role objects |

**Role object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| name | string | Role name |
| color | string? | Hex color |
| position | int | Hierarchy position |
| is_default | bool | Auto-assigned to new users |
| permissions | int64 | Permission bitfield |
| version | int64 | Entity version |

#### role.create

Create a new role. Requires `ManageRoles` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| name | string | Role name |
| color | string? | Hex color |
| permissions | int64 | Permission bitfield |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | New role ULID |
| position | int | Assigned position |
| version | int64 | Entity version |

**Event broadcast:** `event.role.created` — full role object.

#### role.update

Update a role. Requires `ManageRoles` permission. Can only modify roles below own position.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Role ULID |
| name | string? | New name |
| color | string? | New color |
| position | int? | New position |
| permissions | int64? | New bitfield (full replace) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| version | int64 | New version |

**Event broadcast:** `event.role.updated` — full role object.

#### role.delete

Delete a role. Requires `ManageRoles` permission. Cannot delete the default role.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Role ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.role.deleted`

| Field | Type | Description |
|-------|------|-------------|
| id | string | Deleted role ULID |

#### role.assign

Assign a role to a user. Requires `ManageRoles` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Target user's public key |
| role_id | string | Role to assign |

**Response payload:** *(none)*

**Event broadcast:** `event.user.role.added`

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | User's public key |
| role_id | string | Assigned role ID |

#### role.revoke

Remove a role from a user. Requires `ManageRoles` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Target user's public key |
| role_id | string | Role to remove |

**Response payload:** *(none)*

**Event broadcast:** `event.user.role.removed`

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | User's public key |
| role_id | string | Removed role ID |

**Notes:**
- Users can only create/update/delete roles with a position lower than their own highest role. Server enforces this.
- `role.update` with `permissions` does a full replace of the bitfield, not a merge.
- Cannot delete the default role (`is_default = true`). Server rejects with error.
- All mutations trigger audit log entries.

---

### voice.*

#### voice.join

Join a voice channel. Requires `JoinVoice` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| participants | array | Current participants (see below) |
| voice_key | bytes | Shared encryption key for E2EE voice |
| sdp_offer | string | SFU's SDP offer for WebRTC negotiation |

**Participant object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Participant's public key |
| display_name | string | Display name |
| is_muted | bool | Currently muted |
| is_deafened | bool | Currently deafened |

**Event broadcast:** `event.voice.joined`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| pubkey | bytes | Joining user's public key |
| display_name | string | Display name |

#### voice.leave

Leave a voice channel.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.voice.left`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| pubkey | bytes | Leaving user's public key |

#### voice.signal

WebRTC signaling exchange (bidirectional). Server also sends `event.voice.signal` to the client with the same shape.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| sdp_answer | string? | WebRTC SDP answer |
| ice_candidate | object? | ICE candidate |

**Response payload:** *(none)*

#### voice.mute

Toggle mute state. Client stops sending audio when muted.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| muted | bool | New mute state |

**Response payload:** *(none)*

**Event broadcast:** `event.voice.mute`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| pubkey | bytes | User's public key |
| muted | bool | New mute state |

#### voice.deafen

Toggle deafen state. Client stops playing audio when deafened.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| deafened | bool | New deafen state |

**Response payload:** *(none)*

**Event broadcast:** `event.voice.deafen`

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| pubkey | bytes | User's public key |
| deafened | bool | New deafen state |

#### event.voice.speaking (server → client only)

VAD-detected speaking state. No client request — server pushes this.

| Field | Type | Description |
|-------|------|-------------|
| channel_id | string | Voice channel ULID |
| pubkey | bytes | Speaker's public key |
| speaking | bool | Currently speaking |

**Notes:**
- Speaking requires `Speak` permission (checked when sending audio, not on join).
- `voice_key` is the shared E2EE key. First joiner generates it; subsequent joiners receive it from a current key holder. Same shared-key model as group DMs (spec section 6).
- `voice.signal` handles WebRTC negotiation — the SFU is the WebRTC peer, not other clients.
- `event.voice.speaking` is driven by server-side VAD detecting RTP packet energy/timing (encrypted content is opaque, but activity patterns are visible).
- Mute/deafen are client-side actions — server just broadcasts the state to other participants.

---

### dm.*

#### dm.create

Create a DM conversation. 1 participant = 1:1 DM, 2+ = group DM.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| participants | bytes[] | Pubkeys of other participants |
| name | string? | Group name (null for 1:1) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | DMConversation ULID |

**Notes:** For 1:1, if a conversation already exists between the pair, returns the existing ID (idempotent).

#### dm.list

List all DM conversations for the current user.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversations | array | List of conversation objects |

**Conversation object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Conversation ULID |
| is_group | bool | 1:1 or group |
| name | string? | Group name (null for 1:1) |
| participants | array | Participant list (see below) |
| last_message_at | timestamp? | For sorting by recency |

**Participant object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Public key |
| display_name | string | Display name |
| is_key_manager | bool | Can rotate group key |

#### dm.send

Send an encrypted message to a DM conversation.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| encrypted_payload | bytes | E2EE blob |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | New DMMessage ULID |
| created_at | timestamp | Server timestamp |

**Event broadcast:** `event.dm.new` — sent to all online participants.

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| id | string | Message ULID |
| sender_pubkey | bytes | Sender's public key |
| encrypted_payload | bytes | E2EE blob |
| created_at | timestamp | Server timestamp |

#### dm.history

Fetch DM message history with cursor-based pagination.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| before | string? | DMMessage ULID cursor |
| limit | int? | Default 50, max 100 |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| messages | array | List of DM message objects |
| has_more | bool | True if older messages exist |

**DM message object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| sender_pubkey | bytes | Sender's public key |
| encrypted_payload | bytes | E2EE blob |
| created_at | timestamp | Server timestamp |

#### dm.add_member

Add a member to a group DM. Requires `IsKeyManager` on the conversation.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | New member's public key |

**Response payload:** *(none)*

**Event broadcast:** `event.dm.member.added`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | New member's public key |
| display_name | string | Display name |

#### dm.remove_member

Remove a member from a group DM. Requires `IsKeyManager`. Key manager must rotate group key after removal.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | Member to remove |

**Response payload:** *(none)*

**Event broadcast:** `event.dm.member.removed`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | Removed member's public key |

**Notes:** Sets `LeftAt` on DMParticipant. Key manager must distribute a new group key via `dm.key.distribute` after removal.

#### dm.leave

Leave a DM conversation voluntarily.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.dm.member.removed` (same as remove).

#### dm.key.distribute

Distribute encrypted group keys to participants. Server blindly relays — cannot read the key blobs.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| recipients | array | List of recipient key blobs |

**Recipient object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Recipient's public key |
| encrypted_key | bytes | Group key encrypted for this recipient |

**Response payload:** *(none)*

**Notes:** Used after group creation, member add/remove, and periodic key rotation.

#### dm.voice.start

Initiate a voice call in a DM conversation. Rings all participants.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| voice_key | bytes | E2EE voice key |
| sdp_offer | string | SFU SDP offer |

**Event broadcast:** `event.dm.voice.ringing`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| caller_pubkey | bytes | Who initiated the call |

#### dm.voice.accept

Accept an incoming DM voice call.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| sdp_answer | string | WebRTC SDP answer |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| participants | array | Who's already in the call |

**Participant object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Public key |
| display_name | string | Display name |

**Event broadcast:** `event.dm.voice.joined`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | Joining user's public key |

#### dm.voice.reject

Decline an incoming DM voice call.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.dm.voice.declined`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | Declining user's public key |

#### dm.voice.leave

Leave an active DM voice call.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |

**Response payload:** *(none)*

**Event broadcast:** `event.dm.voice.left`

| Field | Type | Description |
|-------|------|-------------|
| conversation_id | string | Conversation ULID |
| pubkey | bytes | Leaving user's public key |

**Notes:** When last participant leaves, call ends. `event.dm.voice.ended` broadcast to all participants.

**dm.* general notes:**
- `dm.send` — server stores E2EE blob and relays to online participants. Offline participants retrieve via `dm.history` on reconnect.
- `dm.key.distribute` — server cannot read key blobs, only routes them.
- DM voice calls reuse the SFU but signal via `dm.voice.*` instead of `voice.*`. The `conversation_id` identifies the call session.
- `dm.voice.start` rings all participants. Call begins when at least one accepts.
- All DM messages are encrypted client-side — server only sees opaque blobs.

---

### file.*

#### file.upload.request

Request a single-use upload token. Requires `AttachFiles` permission for channel uploads.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| name | string | Original filename |
| size | int64 | File size in bytes |
| mime_type | string | MIME type |
| channel_id | string? | Target channel (null for avatar/icon uploads) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| token | string | Single-use upload token |
| url | string | Upload URL path (e.g. `"/upload/abc123"`) |
| expires_in | int | Seconds until token expires (default 60) |

**Error codes:** `FILE_TOO_LARGE`, `STORAGE_FULL`, `RATE_LIMITED`.

#### event.file.upload.complete (server → client)

Pushed after the HTTP PUT upload completes and the server finishes processing (thumbnail generation, etc.).

| Field | Type | Description |
|-------|------|-------------|
| file_id | string | New File ULID |
| name | string | Original filename |
| mime_type | string | MIME type |
| size | int64 | File size in bytes |
| has_thumbnail | bool | Whether a thumbnail was generated |

**Upload flow:**
1. `file.upload.request` → get token + URL.
2. `POST /upload` with multipart body. Auth via `Authorization: Bearer <upload_token>` header or `?token=<upload_token>` query param.
3. Server processes file, generates thumbnail (images/videos).
4. Server pushes `event.file.upload.complete` via WS.
5. Client references `file_id` in `message.send` (`file_ids`), `user.update` (`avatar_id`), or `server.update` (`icon_id`).

**Download flow:**
1. `file.download.request` → get single-use download token.
2. `GET /files/{file_id}` or `GET /files/{file_id}/thumb`. Auth via `Authorization: Bearer <download_token>` header or `?token=<download_token>` query param.

**Progress tracking:**
- **Upload**: Client-side only. Go's HTTP client wraps the request body in a counting `io.Reader` → reports bytes sent / total to the UI via Wails binding. No WS messages.
- **Download**: Client-side only. Go reads `Content-Length` from HTTP response, tracks bytes received → reports to UI via Wails binding. No WS messages.
- Svelte frontend renders progress bars from these Wails-bound progress callbacks.

#### file.download.request

Request a single-use download token for a file.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| file_id | string | File ULID to download |
| thumbnail | bool? | If true, token is for the thumbnail endpoint (default false) |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| token | string | Single-use download token |
| url | string | Download URL path (e.g. `"/files/01JABCDEF..."` or `"/files/01JABCDEF.../thumb"`) |
| expires_in | int | Seconds until token expires (default 60) |

**Notes:**
- The Go client requests tokens transparently when Svelte renders a message with attachments. No frontend involvement.
- One token per file — if the same file needs to be re-downloaded (e.g. token expired), a new request is needed.
- Server verifies the requesting user has access to the channel the file belongs to. Avatars/icons (ChannelID = null) require server membership.

**file.* general notes:**
- Upload tokens are single-use, short-lived (60s), tied to the session.
- Over `ws://` (no TLS), HTTP file endpoints are disabled. Files must be sent as base64 inside WS messages (size penalty acknowledged in spec).

---

### sync.*

#### sync.subscribe

Set the client's field selection for this session. Sent once after auth.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| users | string[] | Fields to receive: `"display_name"`, `"avatar"`, `"bio"`, `"status"` |
| channels | string[] | `"name"`, `"category"`, `"type"` |
| messages | string[] | `"content"`, `"author"`, `"timestamp"` |

**Response payload:** *(none)*

**Notes:** Server filters all future events and responses to only include requested fields. Persisted for the session duration. Maps to PerServerConfig on the client side.

#### sync.request

Request a state diff based on locally known versions. Sent on connect/reconnect.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| users | object | `{ pubkey_hex: version, ... }` |
| channels | object | `{ channel_ulid: version, ... }` |
| categories | object | `{ category_ulid: version, ... }` |
| roles | object | `{ role_ulid: version, ... }` |
| server | int64? | Server entity version (null = first sync) |
| erasure_since | timestamp? | Last known erasure record time |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| users | array | Full user objects with newer versions |
| channels | array | Full channel objects, only changed |
| categories | array | Full category objects, only changed |
| roles | array | Full role objects, only changed |
| server | object? | Server metadata (null if unchanged) |
| deleted_users | bytes[] | Pubkeys of users no longer on server |
| deleted_channels | string[] | ULIDs of deleted channels |
| deleted_categories | string[] | ULIDs of deleted categories |
| deleted_roles | string[] | ULIDs of deleted roles |
| erasure_records | array | ErasureRecords newer than `erasure_since` |

**Erasure record object:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Departed user's public key |
| mode | string | `"ghost"` \| `"forget"` |
| erased_at | timestamp | When the erasure occurred |

**Notes:**
- First connection (all versions null/empty) triggers a full state dump.
- `deleted_*` arrays tell the client what to remove from its cache — entities in the client's version map that no longer exist on server.
- After initial sync, all updates come via real-time events (`event.*`). No polling.

---

### ban.*

#### ban.create

Ban a user. Requires `BanUsers` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Target user's public key |
| reason | string? | Ban reason |
| expires_at | timestamp? | Null = permanent |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Ban ULID |

**Event broadcast:** `event.user.banned`

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Banned user's public key |
| reason | string? | Ban reason |

**Notes:** Server immediately disconnects the banned user's WebSocket.

#### ban.remove

Unban a user. Requires `BanUsers` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Banned user's public key |

**Response payload:** *(none)*

**Event broadcast:** `event.user.unbanned`

| Field | Type | Description |
|-------|------|-------------|
| pubkey | bytes | Unbanned user's public key |

#### ban.list

List all bans. Requires `BanUsers` permission.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| bans | array | List of ban objects |

**Ban object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | Ban ULID |
| pubkey | bytes | Banned user's public key |
| reason | string? | Ban reason |
| banned_by_pubkey | bytes? | Banner's pubkey (null if they did Forget Me) |
| expires_at | timestamp? | Null = permanent |
| created_at | timestamp | When the ban was issued |

---

### invite.*

#### invite.create

Create an invite code. Requires `ManageInvites` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| uses_left | int? | Null = unlimited |
| expires_at | timestamp? | Null = never expires |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | InviteCode ULID |
| code | string | The invite code string (generated server-side) |

#### invite.list

List all invite codes. Requires `ManageInvites` permission.

**Request payload:** *(none)*

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| invites | array | List of invite objects |

**Invite object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | InviteCode ULID |
| code | string | Invite code string |
| uses_left | int? | Remaining uses (null = unlimited) |
| expires_at | timestamp? | Null = never |
| created_by_pubkey | bytes? | Creator's pubkey (null if they did Forget Me) |
| created_at | timestamp | When the invite was created |

#### invite.revoke

Revoke an invite code. Requires `ManageInvites` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | InviteCode ULID |

**Response payload:** *(none)*

---

### audit.*

#### audit.list

List audit log entries. Requires `ManageServer` permission.

**Request payload:**

| Field | Type | Description |
|-------|------|-------------|
| before | string? | AuditLogEntry ULID cursor |
| limit | int? | Default 50, max 100 |
| action | string? | Filter by action type |
| actor_pubkey | bytes? | Filter by actor |

**Response payload:**

| Field | Type | Description |
|-------|------|-------------|
| entries | array | List of audit log entries |
| has_more | bool | True if older entries exist |

**Audit entry object:**

| Field | Type | Description |
|-------|------|-------------|
| id | string | ULID |
| actor_pubkey | bytes? | Actor's pubkey (null if they did Forget Me) |
| action | string | Action type (e.g. `"user.ban"`, `"channel.create"`) |
| target_type | string | Entity type affected |
| target_id | string | ID of affected entity |
| details | string? | JSON blob with action-specific context |
| created_at | timestamp | When the action occurred |

**Notes:**
- All ban/invite mutations trigger audit log entries.
- `ban.list` and `audit.list` use pubkeys (not user IDs) for actor/banner fields — the User record may be deleted.
- `audit.list` uses cursor pagination (same pattern as `message.history`).

---

## Wails Bindings (Go → Svelte Contract)

Each service is a Go struct bound via `Bind: []interface{}{...}` in `wails.Run()`. Public methods become callable from Svelte as `Promise<T>`. Go `error` returns → Promise reject. Structs with `json` tags auto-generate TypeScript models in `wailsjs/go/models.ts`.

**Go→Svelte push** uses Wails runtime events: `runtime.EventsEmit(ctx, "event-name", data)`. Frontend listens via `runtime.EventsOn("event-name", callback)`.

**Package:** `services` — frontend imports: `import { Method } from "../wailsjs/go/services/StructName"`

---

### AppService

Lifecycle and loading state.

```go
type AppState struct {
    Phase      string `json:"phase"`      // "loading" | "setup" | "ready"
    LoadingMsg string `json:"loadingMsg"` // current loading stage description
    Progress   int    `json:"progress"`   // 0-100
}

func (a *AppService) GetState() AppState
func (a *AppService) Shutdown() error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `app:stateChanged` | `AppState` | During loading stages |

---

### ProfileService

Local profile management. Private key lives in OS credential store — never exposed.

```go
type Profile struct {
    PublicKey   string `json:"publicKey"`   // hex-encoded
    DisplayName string `json:"displayName"`
    AvatarHash string `json:"avatarHash"`
    Bio        string `json:"bio"`
}

func (p *ProfileService) GetProfile() (Profile, error)
func (p *ProfileService) UpdateProfile(displayName string, bio string) error
func (p *ProfileService) SetAvatar(filePath string) error
func (p *ProfileService) RemoveAvatar() error
func (p *ProfileService) GetPublicKey() string
func (p *ProfileService) ExportIdentity(filePath string) error
func (p *ProfileService) ImportIdentity(filePath string) error
```

---

### ServerService

Server connections, TOFU trust, and server metadata.

```go
type ServerEntry struct {
    ID              int64  `json:"id"`
    Address         string `json:"address"`
    Name            string `json:"name"`
    IconHash        string `json:"iconHash"`
    IsRelayOnly     bool   `json:"isRelayOnly"`
    Connected       bool   `json:"connected"`
    LastConnectedAt string `json:"lastConnectedAt"`
}

type ServerHello struct {
    ServerPubKey string `json:"serverPubKey"` // hex
    ServerName   string `json:"serverName"`
    AccessMode   string `json:"accessMode"`
    TrustStatus  string `json:"trustStatus"`  // "new" | "trusted" | "mismatch"
    StoredPubKey string `json:"storedPubKey"` // hex, only if mismatch
}

type ServerInfo struct {
    Name              string `json:"name"`
    Description       string `json:"description"`
    IconID            string `json:"iconId"`
    IconHash          string `json:"iconHash"`
    AccessMode        string `json:"accessMode"`
    MemberCount       int    `json:"memberCount"`
    MaxFileSize       int64  `json:"maxFileSize"`
    TotalStorageLimit int64  `json:"totalStorageLimit"`
}

func (s *ServerService) GetServers() []ServerEntry
func (s *ServerService) GetRelayServers() []ServerEntry
func (s *ServerService) Connect(address string) (ServerHello, error)
func (s *ServerService) TrustAndAuth(accessToken string) error
func (s *ServerService) RejectTrust()
func (s *ServerService) Disconnect(serverID int64) error
func (s *ServerService) Reconnect(serverID int64) error
func (s *ServerService) LeaveServer(serverID int64, mode string) error
func (s *ServerService) RemoveRelay(serverID int64) error
func (s *ServerService) GetServerInfo(serverID int64) (ServerInfo, error)
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `server:connected` | `ServerEntry` | Server connection established |
| `server:disconnected` | `{ serverID }` | Server disconnected |
| `server:updated` | `ServerInfo` | Server metadata changed |

**Notes:**
- `Connect()` returns `ServerHello` — tells the frontend what to render in the Join Server modal: `trustStatus` determines trust prompt vs warning, `accessMode` determines invite/password field.
- `TrustAndAuth()` called after user approves trust / enters credentials. Go backend completes handshake.
- `RejectTrust()` aborts connection if user cancels trust prompt or key mismatch warning.

---

### ChannelService

Channel and category operations. Routes through the specified server connection.

```go
type Category struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Position int    `json:"position"`
    Type     string `json:"type"`
}

type Channel struct {
    ID          string   `json:"id"`
    CategoryID  string   `json:"categoryId"`
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    Position    int      `json:"position"`
    OpusBitrate int      `json:"opusBitrate"`
    RoleIDs     []string `json:"roleIds"`
}

// Categories
func (c *ChannelService) GetCategories(serverID int64) ([]Category, error)
func (c *ChannelService) CreateCategory(serverID int64, name string, typ string) (Category, error)
func (c *ChannelService) UpdateCategory(serverID int64, id string, name string, position int) error
func (c *ChannelService) DeleteCategory(serverID int64, id string) error

// Channels
func (c *ChannelService) GetChannels(serverID int64, categoryID string) ([]Channel, error)
func (c *ChannelService) CreateChannel(serverID int64, categoryID string, name string, typ string, roleIDs []string) (Channel, error)
func (c *ChannelService) UpdateChannel(serverID int64, id string, name string, categoryID string, position int, roleIDs []string) error
func (c *ChannelService) DeleteChannel(serverID int64, id string) error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `channel:created` | `Channel` | New channel on this server |
| `channel:updated` | `Channel` | Channel modified |
| `channel:deleted` | `{ serverID, channelID }` | Channel removed |
| `category:created` | `Category` | New category |
| `category:updated` | `Category` | Category modified |
| `category:deleted` | `{ serverID, categoryID, deletedChannelIDs }` | Category + children removed |

---

### MessageService

Message operations. Signing is handled internally — Svelte never touches crypto.

```go
type Message struct {
    ID           string   `json:"id"`
    ChannelID    string   `json:"channelId"`
    AuthorPubKey string   `json:"authorPubKey"` // hex
    Content      string   `json:"content"`
    FileIDs      []string `json:"fileIds"`
    EditedAt     string   `json:"editedAt"`     // ISO 8601 or empty
    CreatedAt    string   `json:"createdAt"`    // ISO 8601
}

type MessageSearchParams struct {
    Text       string `json:"text"`
    ChannelID  string `json:"channelId"`
    FromPubKey string `json:"fromPubKey"`
    Has        string `json:"has"`
    Before     string `json:"before"`
    After      string `json:"after"`
}

type MessagePage struct {
    Messages []Message `json:"messages"`
    HasMore  bool      `json:"hasMore"`
}

type SearchResult struct {
    Messages   []Message `json:"messages"`
    TotalCount int       `json:"totalCount"`
}

func (m *MessageService) Send(serverID int64, channelID string, content string, fileIDs []string) (Message, error)
func (m *MessageService) Edit(serverID int64, messageID string, content string) error
func (m *MessageService) Delete(serverID int64, messageID string) error
func (m *MessageService) GetHistory(serverID int64, channelID string, beforeID string, limit int) (MessagePage, error)
func (m *MessageService) Search(serverID int64, params MessageSearchParams) (SearchResult, error)
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `message:new` | `Message` | New message received |
| `message:edited` | `Message` | Message was edited |
| `message:deleted` | `{ serverID, channelID, messageID }` | Message removed |

**Notes:**
- `Send()` signs the message with the user's private key internally before sending over WS.
- `GetHistory()` with empty `beforeID` fetches most recent messages. Pass oldest message's ID for pagination.
- `serverID` (int64) is the local `TrustedServer.ID` — identifies which connection to route through.
- Timestamps are ISO 8601 strings for easy frontend formatting.

---

### UserService

User profiles, presence, and moderation actions.

```go
type User struct {
    PubKey      string   `json:"pubKey"`      // hex
    DisplayName string   `json:"displayName"`
    AvatarHash  string   `json:"avatarHash"`
    Bio         string   `json:"bio"`
    Status      string   `json:"status"`
    RoleIDs     []string `json:"roleIds"`
}

type Ban struct {
    ID             string `json:"id"`
    PubKey         string `json:"pubKey"`
    Reason         string `json:"reason"`
    BannedByPubKey string `json:"bannedByPubKey"`
    ExpiresAt      string `json:"expiresAt"`
    CreatedAt      string `json:"createdAt"`
}

func (u *UserService) GetUsers(serverID int64) ([]User, error)
func (u *UserService) GetUser(serverID int64, pubKey string) (User, error)
func (u *UserService) KickUser(serverID int64, pubKey string) error
func (u *UserService) BanUser(serverID int64, pubKey string, reason string, expiresAt string) error
func (u *UserService) UnbanUser(serverID int64, pubKey string) error
func (u *UserService) GetBans(serverID int64) ([]Ban, error)
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `user:updated` | `User` | Profile or status change |
| `user:joined` | `User` | New member on server |
| `user:kicked` | `{ serverID, pubKey }` | User was kicked |
| `user:banned` | `{ serverID, pubKey, reason }` | User was banned |
| `user:unbanned` | `{ serverID, pubKey }` | Ban removed |
| `user:erased` | `{ serverID, pubKey, mode }` | Ghost or Forget Me |

---

### RoleService

Role CRUD and assignment.

```go
type Role struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Color       string `json:"color"`
    Position    int    `json:"position"`
    IsDefault   bool   `json:"isDefault"`
    Permissions int64  `json:"permissions"`
}

func (r *RoleService) GetRoles(serverID int64) ([]Role, error)
func (r *RoleService) CreateRole(serverID int64, name string, color string, permissions int64) (Role, error)
func (r *RoleService) UpdateRole(serverID int64, id string, name string, color string, position int, permissions int64) error
func (r *RoleService) DeleteRole(serverID int64, id string) error
func (r *RoleService) AssignRole(serverID int64, pubKey string, roleID string) error
func (r *RoleService) RevokeRole(serverID int64, pubKey string, roleID string) error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `role:created` | `Role` | New role created |
| `role:updated` | `Role` | Role modified |
| `role:deleted` | `{ serverID, roleID }` | Role removed |
| `user:roleAdded` | `{ serverID, pubKey, roleID }` | Role assigned to user |
| `user:roleRemoved` | `{ serverID, pubKey, roleID }` | Role revoked from user |

---

### VoiceService

Voice channel participation and audio device management. Only one active voice session at a time.

```go
type VoiceParticipant struct {
    PubKey      string `json:"pubKey"`
    DisplayName string `json:"displayName"`
    IsMuted     bool   `json:"isMuted"`
    IsDeafened  bool   `json:"isDeafened"`
    IsSpeaking  bool   `json:"isSpeaking"`
}

type AudioDevice struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// Voice channel
func (v *VoiceService) JoinChannel(serverID int64, channelID string) ([]VoiceParticipant, error)
func (v *VoiceService) LeaveChannel() error
func (v *VoiceService) SetMuted(muted bool) error
func (v *VoiceService) SetDeafened(deafened bool) error

// Audio devices
func (v *VoiceService) GetInputDevices() ([]AudioDevice, error)
func (v *VoiceService) GetOutputDevices() ([]AudioDevice, error)
func (v *VoiceService) SetInputDevice(deviceID string) error
func (v *VoiceService) SetOutputDevice(deviceID string) error
func (v *VoiceService) GetVolume() int
func (v *VoiceService) SetVolume(level int) error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `voice:joined` | `{ serverID, channelID, participant }` | Someone joined voice |
| `voice:left` | `{ serverID, channelID, pubKey }` | Someone left voice |
| `voice:muted` | `{ pubKey, muted }` | Mute state changed |
| `voice:deafened` | `{ pubKey, deafened }` | Deafen state changed |
| `voice:speaking` | `{ pubKey, speaking }` | VAD speaking state |

**Notes:**
- Only one active voice session — `LeaveChannel()` needs no params.
- `JoinChannel()` returns current participants. WebRTC negotiation happens internally.
- Audio device management entirely in Go (spec section 5). Svelte renders dropdowns/sliders, Go does audio I/O.
- Ban/unban on `UserService` — keeps moderation co-located with user management.

#### DMService

```go
type DMConversationInfo struct {
    ID           string   `json:"id"`           // Server-side ULID
    IsGroup      bool     `json:"isGroup"`
    Label        string   `json:"label"`        // Group name or other user's display name
    Participants []string `json:"participants"` // Public keys
    CreatedAt    string   `json:"createdAt"`
    LastActivity string   `json:"lastActivity"`
}

type DMMessageOut struct {
    ID        string `json:"id"`
    ConvID    string `json:"convId"`
    SenderKey string `json:"senderKey"`
    Content   string `json:"content"`    // Decrypted plaintext
    Timestamp string `json:"timestamp"`
}

type DMMessagePage struct {
    Messages []DMMessageOut `json:"messages"`
    HasMore  bool           `json:"hasMore"`
}

// Conversations
func (d *DMService) CreateDM(serverID int64, participants []string) (DMConversationInfo, error)
func (d *DMService) GetConversations(serverID int64) ([]DMConversationInfo, error)

// Messages (content encrypted/decrypted transparently by Go)
func (d *DMService) Send(serverID int64, convID string, content string) (DMMessageOut, error)
func (d *DMService) GetHistory(serverID int64, convID string, beforeID string, limit int) (DMMessagePage, error)

// Group DM membership
func (d *DMService) AddMember(serverID int64, convID string, pubKey string) error
func (d *DMService) RemoveMember(serverID int64, convID string, pubKey string) error
func (d *DMService) LeaveConversation(serverID int64, convID string) error

// DM voice calls
func (d *DMService) StartCall(serverID int64, convID string) error
func (d *DMService) AcceptCall(serverID int64, convID string) error
func (d *DMService) RejectCall(serverID int64, convID string) error
func (d *DMService) LeaveCall() error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `dm:created` | `DMConversationInfo` | New DM conversation |
| `dm:message` | `DMMessageOut` | New DM message received |
| `dm:memberAdded` | `{ convId, pubKey }` | Member added to group DM |
| `dm:memberRemoved` | `{ convId, pubKey }` | Member removed from group DM |
| `dm:callIncoming` | `{ convId, callerKey }` | Incoming call |
| `dm:callStarted` | `{ convId, participants }` | Call connected |
| `dm:callEnded` | `{ convId }` | Call ended |

**Notes:**
- Encryption/decryption is fully transparent — Svelte sends/receives plaintext, Go handles X25519 + ChaCha20-Poly1305.
- `CreateDM` with one participant = 1:1, multiple = group DM. Same unified model.
- DM voice calls: one active call at a time (same as channel voice), `LeaveCall()` needs no params.
- Key distribution (`dm.key.distribute`) handled internally by Go when members change.

#### FileService

```go
type UploadProgress struct {
    FileID    string  `json:"fileId"`    // Local tracking ID
    FileName  string  `json:"fileName"`
    BytesSent int64   `json:"bytesSent"`
    TotalSize int64   `json:"totalSize"`
    Percent   float64 `json:"percent"`
    Done      bool    `json:"done"`
    Error     string  `json:"error,omitempty"`
}

type FileInfo struct {
    ID        string `json:"id"`        // Server ULID
    FileName  string `json:"fileName"`
    MimeType  string `json:"mimeType"`
    Size      int64  `json:"size"`
    URL       string `json:"url"`       // Tokenized download URL
    Thumbnail string `json:"thumbnail,omitempty"` // Tokenized thumbnail URL
}

func (f *FileService) PickFile() (string, error)                // Native file dialog, returns path
func (f *FileService) Upload(serverID int64, channelID string, filePath string) (FileInfo, error)
func (f *FileService) Download(serverID int64, fileID string, savePath string) error
func (f *FileService) PickAndDownload(serverID int64, fileID string) error  // Native save dialog + download
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `file:progress` | `UploadProgress` | Upload progress tick |
| `file:downloadProgress` | `{ fileId, percent, done }` | Download progress tick |

**Notes:**
- `Upload` requests a token via WS (`file.upload.request`), then POSTs to HTTP endpoint. Progress emitted via events.
- `Download` requests a token via WS, then GETs from HTTP endpoint with save-to-disk streaming.
- `PickFile` / `PickAndDownload` use Wails `runtime.OpenFileDialog` / `runtime.SaveFileDialog`.
- Svelte never touches HTTP directly — Go owns all network I/O.

#### SettingsService

```go
type PerServerSettings struct {
    ServerID         int64  `json:"serverId"`
    ShowAvatars      bool   `json:"showAvatars"`
    ShowDisplayNames bool   `json:"showDisplayNames"`
    ShowStatuses     bool   `json:"showStatuses"`
    ShowReadReceipts bool   `json:"showReadReceipts"`
    ShowTyping       bool   `json:"showTyping"`
    AcceptFiles      bool   `json:"acceptFiles"`
    MaxFileSize      int64  `json:"maxFileSize"`
}

type AppSettings struct {
    Theme           string `json:"theme"`    // "dark", "light", "system"
    NotifySound     bool   `json:"notifySound"`
    NotifyDesktop   bool   `json:"notifyDesktop"`
    MinimizeToTray  bool   `json:"minimizeToTray"`
}

func (s *SettingsService) GetAppSettings() (AppSettings, error)
func (s *SettingsService) UpdateAppSettings(settings AppSettings) error
func (s *SettingsService) GetServerSettings(serverID int64) (PerServerSettings, error)
func (s *SettingsService) UpdateServerSettings(serverID int64, settings PerServerSettings) error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `settings:appChanged` | `AppSettings` | App settings updated |
| `settings:serverChanged` | `PerServerSettings` | Per-server settings updated |

**Notes:**
- `PerServerSettings` maps to client-side Model 19 (PerServerConfig). Changes trigger `sync.subscribe` update to server.
- `AppSettings` stored in client DB as a singleton. Theme applies immediately via event.

#### AdminService

```go
type AuditEntry struct {
    ID        string `json:"id"`
    Action    string `json:"action"`
    ActorKey  string `json:"actorKey"`
    ActorName string `json:"actorName"`
    Details   string `json:"details"`   // JSON string
    Timestamp string `json:"timestamp"`
}

type AuditPage struct {
    Entries []AuditEntry `json:"entries"`
    Cursor  string       `json:"cursor"`
    HasMore bool         `json:"hasMore"`
}

type InviteInfo struct {
    Code      string `json:"code"`
    CreatedBy string `json:"createdBy"`
    UsesLeft  *int   `json:"usesLeft"` // null = unlimited
    ExpiresAt string `json:"expiresAt,omitempty"`
    CreatedAt string `json:"createdAt"`
}

func (a *AdminService) GetAuditLog(serverID int64, cursor string, limit int) (AuditPage, error)
func (a *AdminService) CreateInvite(serverID int64, usesLeft *int, expiresAt string) (InviteInfo, error)
func (a *AdminService) GetInvites(serverID int64) ([]InviteInfo, error)
func (a *AdminService) RevokeInvite(serverID int64, code string) error
func (a *AdminService) UpdateServer(serverID int64, name string, description string) error
func (a *AdminService) SetServerIcon(serverID int64, filePath string) error
```

**Events emitted:**
| Event | Payload | Description |
|-------|---------|-------------|
| `admin:auditNew` | `AuditEntry` | New audit log entry (live feed) |
| `admin:inviteCreated` | `InviteInfo` | Invite created |
| `admin:inviteRevoked` | `{ code }` | Invite revoked |

**Notes:**
- Admin operations require owner/admin permissions — Go checks before sending WS messages.
- `UpdateServer` sends `server.update`. `SetServerIcon` uploads via FileService internally, then updates server.
- Audit log is read-only with cursor pagination (same as `audit.list` WS message).
- `usesLeft: null` in `CreateInvite` = unlimited uses (matches Model 11).

---

### HTTP Endpoints

All file transfers use tokenized HTTP on the same port as WebSocket. Tokens are single-use, short-lived (60s), obtained via WS messages.

#### `POST /upload`

Upload a file using a token from `file.upload.request`.

**Auth:** Single-use upload token via either method:
- Header: `Authorization: Bearer <upload_token>`
- Query param: `POST /upload?token=<upload_token>`

**Request:**
```
POST /upload HTTP/1.1
Content-Type: multipart/form-data; boundary=...
Authorization: Bearer <upload_token>

--boundary
Content-Disposition: form-data; name="file"; filename="photo.png"
Content-Type: image/png

<binary data>
--boundary--
```

**Response (200):**
```json
{
  "id": "01JABCDEF...",
  "fileName": "photo.png",
  "mimeType": "image/png",
  "size": 204800,
  "thumbnail": "01JABCDEF..._thumb"
}
```

**Errors:**
| Status | Body | Cause |
|--------|------|-------|
| 401 | `{ "error": "invalid_token" }` | Token expired, already used, or malformed |
| 413 | `{ "error": "file_too_large", "maxSize": 52428800 }` | Exceeds server's `max_file_size` |
| 507 | `{ "error": "storage_full" }` | Server's `total_storage_limit` reached |
| 400 | `{ "error": "no_file" }` | Missing multipart file field |
| 500 | `{ "error": "internal" }` | Server-side failure |

**Server behavior:**
- Validates token against in-memory token store (maps token → channelID + uploaderPubKey + expiry).
- Streams to disk, generates thumbnail for images/videos.
- On success, creates `File` record (Model 6), sends `file.upload.complete` event to uploader via WS.
- Token consumed on first use regardless of outcome.

#### `GET /files/:fileID`

Download a file using a single-use download token (obtained via `file.download.request`).

**Auth:** Single-use download token via either method:
- Header: `Authorization: Bearer <download_token>`
- Query param: `GET /files/:fileID?token=<download_token>`

**Request:**
```
GET /files/01JABCDEF...?token=<download_token> HTTP/1.1
```

**Response (200):**
```
HTTP/1.1 200 OK
Content-Type: image/png
Content-Length: 204800
Content-Disposition: inline; filename="photo.png"
Cache-Control: no-store

<binary data>
```

**Errors:**
| Status | Body | Cause |
|--------|------|-------|
| 401 | `{ "error": "invalid_token" }` | Token expired, already used, or malformed |
| 404 | `{ "error": "not_found" }` | File ID doesn't exist or was deleted |
| 500 | `{ "error": "internal" }` | Server-side failure |

**Notes:**
- Server accepts token via `Authorization` header or `?token=` query param — either works.
- `Content-Disposition: inline` for images/video/audio, `attachment` for everything else.
- `Cache-Control: no-store` — tokens are single-use so caching would break re-requests.

#### `GET /files/:fileID/thumb`

Download a thumbnail. Same token mechanism and dual auth methods as `/files/:fileID`.

**Request:**
```
GET /files/01JABCDEF.../thumb?token=<download_token> HTTP/1.1
```

**Response (200):**
```
HTTP/1.1 200 OK
Content-Type: image/webp
Content-Length: 8192
Cache-Control: no-store

<binary data>
```

**Errors:** Same as `/files/:fileID`.

**Notes:**
- Thumbnails generated server-side on upload: 256×256 max, WebP format.
- Only generated for image and video files. Returns 404 for files without thumbnails.

#### Token Lifecycle Summary

```
Client                         Server
  │                              │
  ├─ file.upload.request ───────►│  Creates token (60s TTL, maps to channel + user)
  │◄─ file.upload.response ──────┤  Returns { token, uploadUrl }
  │                              │
  ├─ POST /upload ──────────────►│  Validates + consumes token, stores file
  │◄─ 200 { id, fileName, ... } ┤
  │                              │
  ├─ message.send (fileIDs) ────►│  Attaches files to message via MessageFile
  │                              │
  │  (Other clients receive message with file metadata)
  │                              │
  ├─ file.download.request ────►│  Creates download token (60s TTL)
  │◄─ file.download.request.ok ─┤  Returns { token, url }
  │                              │
  ├─ GET /files/:id?token= ────►│  Validates + consumes token, streams file
  │◄─ 200 <binary> ─────────────┤
```

---

## Implementation Architecture

Internal architecture decisions for structuring the codebase. These are not wire-visible but ensure implementation agents produce consistent, compatible code.

---

### CLI Arguments

#### Server (`haven-server`)

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `haven-server.toml` | Path to config file |
| `-data` | `data/` | Data directory (DB, files, keys) |
| `-verbose` | false | Set log level to Debug (default is Info) |
| `-log-format` | `text` | `text` (human-readable) or `json` (structured, for log ingestion) |
| `-generate-key` | *(action)* | Generate a new Ed25519 keypair, print pubkey hex, exit |
| `-version` | *(action)* | Print version and exit |

**Notes:**
- `-data` is the base for relative paths in the config (e.g. `private_key_path = "data/server.key"` resolves relative to `-data`).
- `-generate-key` is essential for first-time setup — the admin needs their own pubkey to put in `owners.public_keys` before the server becomes usable.

#### Client (`haven`)

| Flag | Default | Description |
|------|---------|-------------|
| `-data` | OS-appropriate app data dir | Data directory (client DB, cached files) |
| `-verbose` | false | Set log level to Debug (default is Info) |
| `-log-format` | `text` | `text` (human-readable) or `json` (structured) |
| `-reset` | *(action)* | Delete local database and cached data, keep identity key, exit |
| `-export-key` | *(action)* | Print public key hex to stdout, exit |
| `-version` | *(action)* | Print version and exit |

**Default data directories (client):**

| OS | Path |
|----|------|
| Windows | `%APPDATA%\Haven\` |
| macOS | `~/Library/Application Support/Haven/` |
| Linux | `~/.local/share/haven/` |

**Notes:**
- `-reset` is a nuclear option for corrupted DB. Identity key stays in OS keychain — user doesn't lose their identity.
- `-export-key` is useful for admins adding themselves to `owners.public_keys`.

### Logging

**Library:** Go standard library `log/slog` (Go 1.21+). Zero external dependencies.

**Log levels:**

| Level | Usage |
|-------|-------|
| `Debug` | Internal state, WS message payloads, crypto operations, SQL queries. Only with `-verbose`. |
| `Info` | Startup, shutdown, connections, auth success, file uploads, voice joins. |
| `Warn` | Auth failures, rate limit hits, expired tokens, config issues. |
| `Error` | DB errors, WS write failures, file I/O errors, panic recovery. |

**Output formats:**
- `-log-format text` (default): Human-readable for terminal use. `time=2026-02-22T14:30:00Z level=INFO msg="client connected" pubkey=a1b2c3...`
- `-log-format json`: Structured JSON for log ingestion. `{"time":"2026-02-22T14:30:00Z","level":"INFO","msg":"client connected","pubkey":"a1b2c3..."}`

**Log ingestion:** JSON output to stdout follows the 12-factor / container-native pattern. Admins pipe to their preferred aggregator:
```
haven-server -log-format json | fluentd    → Graylog
haven-server -log-format json | filebeat   → ELK
haven-server -log-format json | vector     → Splunk
```

No Haven-specific integrations needed — every log aggregator can ingest structured JSON from stdout.

**Implementation:**
```go
func initLogger(verbose bool, format string) {
    level := slog.LevelInfo
    if verbose {
        level = slog.LevelDebug
    }
    opts := &slog.HandlerOptions{Level: level}

    var handler slog.Handler
    if format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }
    slog.SetDefault(slog.New(handler))
}
```

**Conventions:**
- Always use structured fields, never string formatting: `slog.Info("client connected", "pubkey", hexKey)` not `slog.Info(fmt.Sprintf("client %s connected", hexKey))`.
- Include context identifiers: `pubkey` for user actions, `channel_id` for channel ops, `server_id` for client-side multi-server logs.
- Never log sensitive data: private keys, passwords, session tokens, decrypted DM content, encryption keys.

---

### Project Structure

```
haven/
├── server/
│   ├── main.go                  # Entry point, config load, server start
│   ├── config/                  # TOML parsing, hot-reload watcher
│   ├── models/                  # GORM structs (1:1 with server-side data models)
│   ├── handlers/                # WS message handlers, one file per namespace
│   ├── middleware/              # Permission checks, rate limiting
│   ├── ws/                     # WebSocket hub, connection management, router
│   ├── http/                   # HTTP upload/download handlers
│   ├── sfu/                    # Pion SFU, room management
│   ├── auth/                   # Challenge-response, session management
│   ├── crypto/                 # Ed25519/X25519 helpers, app-layer encryption
│   └── storage/                # File storage (disk I/O, thumbnails)
├── client/
│   ├── main.go                  # Wails entry point, service binding
│   ├── services/                # Wails-bound services (1:1 with Wails bindings section)
│   ├── connection/              # Connection manager, per-server state
│   ├── crypto/                  # Client-side crypto (signing, E2EE, key derivation)
│   ├── models/                  # SQLCipher GORM structs (client-side data models)
│   ├── audio/                   # PortAudio capture/playback, device management
│   ├── keystore/                # OS credential store abstraction
│   └── frontend/                # Svelte app
│       ├── src/
│       │   ├── lib/
│       │   │   ├── components/  # UI components (maps to design system)
│       │   │   ├── stores/      # Svelte 5 rune-based state ($state)
│       │   │   ├── wails.ts     # Wails binding wrappers + event listeners
│       │   │   └── types.ts     # TypeScript types (auto-generated + manual)
│       │   ├── routes/          # Top-level views (loading, setup, main, settings)
│       │   └── App.svelte       # Root, state-machine routing
│       └── ...
├── shared/                      # Shared constants (permission bits, error codes, message types)
├── docs/
│   ├── SPEC.md
│   └── IMPLEMENTATION.md
└── go.mod
```

**Notes:**
- `shared/` contains Go constants imported by both `server/` and `client/`. Includes permission bit definitions, WS message type strings, and error codes.
- `server/handlers/` has one file per namespace: `server.go`, `category.go`, `channel.go`, `message.go`, `user.go`, `role.go`, `voice.go`, `dm.go`, `file.go`, `sync.go`, `ban.go`, `invite.go`, `audit.go`.
- `client/services/` has one file per Wails service: `app.go`, `profile.go`, `server.go`, `channel.go`, `message.go`, `user.go`, `role.go`, `voice.go`, `dm.go`, `file.go`, `settings.go`, `admin.go`.

---

### WS Error Code Catalog

Standardized error codes used across all WS message namespaces. Returned in the error payload: `{ "code": "...", "message": "human-readable detail" }`.

**General errors (any namespace):**

| Code | Description |
|------|-------------|
| `BAD_REQUEST` | Malformed payload, missing required fields, invalid values |
| `NOT_FOUND` | Referenced entity doesn't exist |
| `PERMISSION_DENIED` | Missing required permission bit |
| `FORBIDDEN` | Not the owner/author, or role position violation |
| `RATE_LIMITED` | Rate limit exceeded |
| `PAYLOAD_TOO_LARGE` | Exceeds 64KB WS message limit |
| `CONFLICT` | Entity already exists (e.g. duplicate 1:1 DM) |
| `INTERNAL` | Server-side error |

**Auth-specific errors (auth namespace only):**

| Code | Description |
|------|-------------|
| `INVALID_SIGNATURE` | Ed25519 signature verification failed |
| `BANNED` | User is banned from the server |
| `INVALID_INVITE` | Invite code is invalid, expired, or exhausted |
| `INVALID_PASSWORD` | Wrong access password |
| `NOT_ALLOWLISTED` | Public key not on server allowlist |
| `SESSION_EXPIRED` | Session token no longer valid |

**File-specific errors (file namespace only):**

| Code | Description |
|------|-------------|
| `FILE_TOO_LARGE` | File exceeds server's `max_file_size` |
| `STORAGE_FULL` | Server's `total_storage_limit` reached |

**Usage:** `PERMISSION_DENIED` is for missing permission bits (e.g. `ManageChannels`). `FORBIDDEN` is for ownership/hierarchy violations (e.g. editing someone else's message, managing a role above your position). Both result in the action being rejected, but the distinction helps client-side error messages.

---

### Client Connection Manager

The Go client manages multiple simultaneous server connections through a centralized `Manager`.

```go
// connection/manager.go
type Manager struct {
    mu          sync.RWMutex
    connections map[int64]*ServerConnection  // keyed by TrustedServer.ID (client DB)
    wailsCtx    context.Context              // for runtime.EventsEmit
}

func (m *Manager) Get(serverID int64) (*ServerConnection, error)
func (m *Manager) Connect(address string) (*ServerConnection, error)
func (m *Manager) Disconnect(serverID int64) error
func (m *Manager) DisconnectAll()
```

```go
// connection/server_connection.go
type ServerConnection struct {
    ServerID     int64
    Address      string
    Conn         *websocket.Conn
    Session      SessionState
    SendCh       chan []byte                // outbound message queue
    PendingReqs  map[string]chan RawMessage // msg_id → response channel
    VoiceRoom    *VoiceState               // nil if not in voice
    Connected    bool
    mu           sync.Mutex
}

type SessionState struct {
    Token         string
    UserID        string
    EncryptionKey []byte  // nil if wss://, set if ws://
    NonceCounter  uint64  // for app-layer encryption frame counter
}

// Send a request and wait for response (blocking, with timeout)
func (sc *ServerConnection) Request(msgType string, payload interface{}) (RawMessage, error)

// Send fire-and-forget (typing, voice state)
func (sc *ServerConnection) Send(msgType string, payload interface{}) error
```

**Flow:**
1. All Wails services hold a `*Manager` reference (injected at startup).
2. Service methods receive `serverID int64` → call `manager.Get(serverID)` → get `*ServerConnection`.
3. `Request()` generates a unique `msg_id`, registers a channel in `PendingReqs`, serializes and sends via `SendCh`, blocks on the response channel (with timeout).
4. A per-connection **read goroutine** reads from the WebSocket:
   - Response messages (has `id` matching a pending request) → dispatched to `PendingReqs[id]`.
   - Event messages (`event.*`) → forwarded to Wails via `runtime.EventsEmit()`.
5. A per-connection **write goroutine** reads from `SendCh` and writes to the WebSocket (serializes access to the writer).
6. On disconnect, all pending requests receive an error, `Connected` is set to false, and `server:disconnected` event is emitted.

---

### Audio Library — PortAudio

Audio capture and playback use **PortAudio** via the `gordonklaus/portaudio` Go bindings.

**Dependency:** `github.com/gordonklaus/portaudio`

**Responsibilities (`client/audio/`):**
- **Device enumeration**: `portaudio.Devices()` → list of input/output devices, exposed via `VoiceService.GetInputDevices()` / `GetOutputDevices()`.
- **Capture**: Open an input stream on the selected microphone. Raw PCM samples are fed to the Opus encoder, then to the Pion WebRTC track.
- **Playback**: Received Opus packets from Pion are decoded to PCM, mixed, and written to an output stream on the selected speaker/headphone device.
- **Volume**: Master volume is a gain multiplier applied to the playback PCM buffer before writing to the output stream.

**Stream parameters:**
- Sample rate: 48000 Hz (Opus native)
- Frame size: 960 samples (20ms at 48kHz — Opus standard frame)
- Channels: 1 (mono — voice only)
- Format: Float32

**Notes:**
- PortAudio requires CGo — already a requirement for SQLCipher, so no additional build complexity.
- Platform libraries: ALSA (Linux), CoreAudio (macOS), WASAPI (Windows) — all handled by PortAudio internally.
- Device hot-plug: PortAudio detects new devices. On device change, the audio module re-enumerates and emits a Wails event so the settings UI can update the dropdown.

---

### SQLCipher Key Derivation

The client database (SQLCipher) is encrypted with a key derived from the user's Ed25519 private key.

```
HKDF-SHA256(
    ikm  = Ed25519_private_key_seed (32 bytes),
    salt = "haven-sqlcipher-v1",
    info = "database-encryption-key"
) → 32-byte raw key
```

**Passed to SQLCipher as:** `PRAGMA key = "x'<hex-encoded 32 bytes>'";`

**Why HKDF (not Argon2/bcrypt):**
- The input is a 32-byte cryptographic key — already high-entropy.
- Argon2/bcrypt are designed to slow down brute-force of low-entropy passwords. Applying them to a 256-bit key is unnecessary and adds startup latency for zero security benefit.
- HKDF is already used elsewhere in Haven (WS app-layer encryption), keeping the crypto primitives consistent.

**Key lifecycle:**
1. On app launch, retrieve Ed25519 private key from OS credential store (`client/keystore/`).
2. Derive SQLCipher key via HKDF.
3. Open database with derived key.
4. Hold derived key in memory only — never written to disk.
5. On app shutdown, key is garbage collected.

---

### Svelte Frontend Architecture

The Svelte frontend is a pure UI layer. All state originates from Go via Wails bindings and events.

**State management — Svelte 5 runes (`$state`):**

```
frontend/src/lib/stores/
├── app.svelte.ts       # AppState (phase, loading progress)
├── servers.svelte.ts   # Server list, active server, server info
├── channels.svelte.ts  # Categories, channels, active channel
├── messages.svelte.ts  # Message list, typing indicators, read state
├── users.svelte.ts     # User list, presence, member cards
├── voice.svelte.ts     # Voice state, participants, audio devices
├── dms.svelte.ts       # DM conversations, DM messages, call state
├── roles.svelte.ts     # Roles for active server
└── settings.svelte.ts  # App settings, per-server settings
```

Each store module:
1. Exports reactive `$state` variables.
2. Registers Wails event listeners on import (e.g. `EventsOn("message:new", ...)` → updates `$state`).
3. Exports action functions that call Wails bindings (e.g. `sendMessage()` → `MessageService.Send()`).

**Routing — State machine (no router library):**

`App.svelte` switches on `appState.phase`:

| Phase | View | Description |
|-------|------|-------------|
| `"loading"` | `LoadingScreen` | Progress bar, loading messages |
| `"setup"` | `ProfileSetup` | First-launch identity creation |
| `"ready"` (no servers) | `NoServerState` | Prompt to join a server |
| `"ready"` (has servers) | `MainLayout` | Sidebar + content area |

Within `MainLayout`, content is driven by selection state (active server → active channel / DM / settings panel), not URL routes. This is a desktop app — there are no URLs.

**Component structure:**

```
frontend/src/lib/components/
├── common/             # Button, Input, Avatar, Badge, Modal, Tooltip, Dropdown
├── layout/             # Sidebar, ContentArea, Header, StatusBar
├── server/             # ServerList, ServerIcon, JoinServerModal
├── channel/            # ChannelList, CategoryHeader, CreateChannelModal
├── message/            # MessageBubble, MessageInput, MessageList, SearchBar
├── user/               # UserCard, UserList, ProfilePanel, MemberSidebar
├── voice/              # VoicePanel, VoiceControls, ParticipantList
├── dm/                 # DMList, DMConversation, DMCallOverlay
├── admin/              # AuditLog, InviteManager, RoleEditor, ServerSettings
└── settings/           # AppSettings, PerServerSettings, RelayServers
```

Components map to the design system in `haven.pen` (90 reusable components). Atomic components in `common/` compose into feature components in the domain folders.

---

### SFU Room Management

The Pion SFU manages voice rooms for both channel voice and DM calls.

```go
// sfu/sfu.go
type SFU struct {
    mu    sync.RWMutex
    rooms map[string]*Room   // room ID → Room
    api   *webrtc.API        // shared Pion API, Opus-only codec
}

func NewSFU() *SFU                                          // configures webrtc.API with Opus
func (s *SFU) GetOrCreateRoom(roomID string) *Room
func (s *SFU) RemoveRoom(roomID string)
```

```go
// sfu/room.go
type Room struct {
    ID           string                         // channel ULID or DM conversation ULID
    mu           sync.RWMutex
    Participants map[string]*Participant         // pubkey hex → participant
    VoiceKey     []byte                          // E2EE shared key
}

type Participant struct {
    PubKey      string
    PeerConn    *webrtc.PeerConnection
    AudioTrack  *webrtc.TrackLocalStaticRTP     // outbound track (receives this participant's audio)
    IsMuted     bool
    IsDeafened  bool
}

func (r *Room) AddParticipant(pubKey string, peerConn *webrtc.PeerConnection) error
func (r *Room) RemoveParticipant(pubKey string) error
func (r *Room) IsEmpty() bool
```

**Join flow:**
1. `SFU.GetOrCreateRoom(channelID)` — creates `Room` if first joiner.
2. First joiner generates `VoiceKey` (32 random bytes). Subsequent joiners receive it from a current participant via the server relay.
3. Create `webrtc.PeerConnection` via shared `SFU.api` (Opus only, no video codecs registered).
4. Create a `TrackLocalStaticRTP` for the new participant's outbound audio.
5. Add existing participants' audio tracks to the new peer's connection (they hear everyone).
6. Add the new peer's audio track to all existing participants' connections (everyone hears them).
7. Generate SDP offer → return to client via `voice.join` response.
8. Client sends SDP answer via `voice.signal` → applied to the `PeerConnection`.
9. ICE candidates exchanged via `voice.signal` until connected.

**Leave flow:**
1. Remove the participant's audio track from all other participants' connections.
2. Close the `PeerConnection`.
3. Remove from `Room.Participants`.
4. If `Room.IsEmpty()`, call `SFU.RemoveRoom(roomID)`.
5. Broadcast `event.voice.left`.

**VAD (Voice Activity Detection):**
- The SFU cannot decrypt audio (E2EE), but can detect activity from RTP packet patterns.
- When a participant is sending audio: RTP packets arrive steadily (~50 packets/sec for 20ms frames).
- When silent: Opus DTX (Discontinuous Transmission) reduces packet rate dramatically.
- The SFU tracks packet rate per participant. Crossing a threshold (e.g., >10 packets in 200ms) = speaking. Falling below = stopped speaking.
- State changes emit `event.voice.speaking` to all room participants.

**DM voice calls** use the same `SFU` and `Room` infrastructure. The room ID is the `DMConversation.ID` instead of a channel ULID. Signaling goes through `dm.voice.*` messages instead of `voice.*`, but the underlying Pion mechanics are identical.

