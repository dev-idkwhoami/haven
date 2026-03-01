package shared

// Version is the current Haven protocol/application version.
const Version = "0.1.0"

// Access modes for server join control.
const (
	AccessModeOpen      = "open"
	AccessModeInvite    = "invite"
	AccessModePassword  = "password"
	AccessModeAllowlist = "allowlist"
)

// User status values.
const (
	StatusOnline  = "online"
	StatusIdle    = "idle"
	StatusDND     = "dnd"
	StatusOffline = "offline"
)

// Channel/category types.
const (
	ChannelTypeText  = "text"
	ChannelTypeVoice = "voice"
)

// Erasure modes.
const (
	ErasureModeGhost  = "ghost"
	ErasureModeForget = "forget"
)

// SentinelUserID is the ULID for the Ghost Mode placeholder user.
const SentinelUserID = "00000000000000000000000000"

// Audit log action types.
const (
	AuditUserKick       = "user.kick"
	AuditUserBan        = "user.ban"
	AuditUserUnban      = "user.unban"
	AuditUserRoleAdd    = "user.role.add"
	AuditUserRoleRemove = "user.role.remove"

	AuditChannelCreate = "channel.create"
	AuditChannelUpdate = "channel.update"
	AuditChannelDelete = "channel.delete"

	AuditCategoryCreate = "category.create"
	AuditCategoryUpdate = "category.update"
	AuditCategoryDelete = "category.delete"

	AuditRoleCreate = "role.create"
	AuditRoleUpdate = "role.update"
	AuditRoleDelete = "role.delete"

	AuditMessageDelete = "message.delete"

	AuditServerUpdate = "server.update"

	AuditInviteCreate = "invite.create"
	AuditInviteRevoke = "invite.revoke"

	AuditAccessRequestApprove = "access_request.approve"
	AuditAccessRequestReject  = "access_request.reject"
)

// Audit log target types.
const (
	TargetTypeUser     = "user"
	TargetTypeChannel  = "channel"
	TargetTypeCategory = "category"
	TargetTypeRole     = "role"
	TargetTypeMessage  = "message"
	TargetTypeServer        = "server"
	TargetTypeInvite        = "invite"
	TargetTypeAccessRequest = "access_request"
)

// WebSocket protocol constants.
const (
	MaxWSMessageSize = 64 * 1024 // 64KB
)

// App-layer encryption info string for HKDF.
const HKDFInfo = "haven-ws-encryption"
