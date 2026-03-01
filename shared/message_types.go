package shared

// Auth message types.
const (
	TypeAuthHello   = "auth.hello"
	TypeAuthRespond = "auth.respond"
	TypeAuthSuccess = "auth.success"
	TypeAuthError   = "auth.error"
)

// Server message types.
const (
	TypeServerInfo   = "server.info"
	TypeServerUpdate = "server.update"
)

// Category message types.
const (
	TypeCategoryList   = "category.list"
	TypeCategoryCreate = "category.create"
	TypeCategoryUpdate = "category.update"
	TypeCategoryDelete = "category.delete"
)

// Channel message types.
const (
	TypeChannelList   = "channel.list"
	TypeChannelCreate = "channel.create"
	TypeChannelUpdate = "channel.update"
	TypeChannelDelete = "channel.delete"
)

// Message message types.
const (
	TypeMessageSend    = "message.send"
	TypeMessageEdit    = "message.edit"
	TypeMessageDelete  = "message.delete"
	TypeMessageHistory = "message.history"
	TypeMessageSearch  = "message.search"
	TypeMessageTyping  = "message.typing"
	TypeMessageRead    = "message.read"
)

// User message types.
const (
	TypeUserProfile = "user.profile"
	TypeUserUpdate  = "user.update"
	TypeUserList    = "user.list"
	TypeUserKick    = "user.kick"
	TypeUserLeave   = "user.leave"
)

// Role message types.
const (
	TypeRoleList   = "role.list"
	TypeRoleCreate = "role.create"
	TypeRoleUpdate = "role.update"
	TypeRoleDelete = "role.delete"
	TypeRoleAssign = "role.assign"
	TypeRoleRevoke = "role.revoke"
)

// Voice message types.
const (
	TypeVoiceJoin   = "voice.join"
	TypeVoiceLeave  = "voice.leave"
	TypeVoiceSignal = "voice.signal"
	TypeVoiceMute   = "voice.mute"
	TypeVoiceDeafen = "voice.deafen"
)

// DM message types.
const (
	TypeDMCreate       = "dm.create"
	TypeDMList         = "dm.list"
	TypeDMSend         = "dm.send"
	TypeDMHistory      = "dm.history"
	TypeDMAddMember    = "dm.add_member"
	TypeDMRemoveMember = "dm.remove_member"
	TypeDMLeave        = "dm.leave"
	TypeDMKeyDistrib   = "dm.key.distribute"
	TypeDMVoiceStart   = "dm.voice.start"
	TypeDMVoiceAccept  = "dm.voice.accept"
	TypeDMVoiceReject  = "dm.voice.reject"
	TypeDMVoiceLeave   = "dm.voice.leave"
)

// File message types.
const (
	TypeFileUploadRequest   = "file.upload.request"
	TypeFileDownloadRequest = "file.download.request"
)

// Sync message types.
const (
	TypeSyncSubscribe = "sync.subscribe"
	TypeSyncRequest   = "sync.request"
)

// Ban message types.
const (
	TypeBanCreate = "ban.create"
	TypeBanRemove = "ban.remove"
	TypeBanList   = "ban.list"
)

// Invite message types.
const (
	TypeInviteCreate = "invite.create"
	TypeInviteList   = "invite.list"
	TypeInviteRevoke = "invite.revoke"
)

// Access request message types.
const (
	TypeAccessRequestSubmit  = "access_request.submit"
	TypeAccessRequestList    = "access_request.list"
	TypeAccessRequestApprove = "access_request.approve"
	TypeAccessRequestReject  = "access_request.reject"
)

// Auth waiting room message types.
const (
	TypeAuthWaitingRoom   = "auth.waiting_room"
	TypeAuthAccessGranted = "auth.access_granted"
	TypeAuthAccessDenied  = "auth.access_denied"
)

// Audit message types.
const (
	TypeAuditList = "audit.list"
)

// Response suffixes.
const (
	SuffixOK    = ".ok"
	SuffixError = ".error"
)

// Event message types (server -> client broadcasts).
const (
	TypeEventServerUpdated = "event.server.updated"

	TypeEventCategoryCreated = "event.category.created"
	TypeEventCategoryUpdated = "event.category.updated"
	TypeEventCategoryDeleted = "event.category.deleted"

	TypeEventChannelCreated = "event.channel.created"
	TypeEventChannelUpdated = "event.channel.updated"
	TypeEventChannelDeleted = "event.channel.deleted"

	TypeEventMessageNew     = "event.message.new"
	TypeEventMessageEdited  = "event.message.edited"
	TypeEventMessageDeleted = "event.message.deleted"
	TypeEventMessageTyping  = "event.message.typing"
	TypeEventMessageRead    = "event.message.read"

	TypeEventUserUpdated  = "event.user.updated"
	TypeEventUserKicked   = "event.user.kicked"
	TypeEventUserErased   = "event.user.erased"
	TypeEventUserBanned   = "event.user.banned"
	TypeEventUserUnbanned = "event.user.unbanned"

	TypeEventRoleCreated = "event.role.created"
	TypeEventRoleUpdated = "event.role.updated"
	TypeEventRoleDeleted = "event.role.deleted"

	TypeEventUserRoleAdded   = "event.user.role.added"
	TypeEventUserRoleRemoved = "event.user.role.removed"

	TypeEventVoiceJoined   = "event.voice.joined"
	TypeEventVoiceLeft     = "event.voice.left"
	TypeEventVoiceSignal   = "event.voice.signal"
	TypeEventVoiceMute     = "event.voice.mute"
	TypeEventVoiceDeafen   = "event.voice.deafen"
	TypeEventVoiceSpeaking = "event.voice.speaking"

	TypeEventDMNew           = "event.dm.new"
	TypeEventDMMemberAdded   = "event.dm.member.added"
	TypeEventDMMemberRemoved = "event.dm.member.removed"

	TypeEventDMVoiceRinging  = "event.dm.voice.ringing"
	TypeEventDMVoiceJoined   = "event.dm.voice.joined"
	TypeEventDMVoiceDeclined = "event.dm.voice.declined"
	TypeEventDMVoiceLeft     = "event.dm.voice.left"
	TypeEventDMVoiceEnded    = "event.dm.voice.ended"

	TypeEventFileUploadComplete = "event.file.upload.complete"

	TypeEventOwnerChanged = "event.owner.changed"

	TypeEventAccessRequestNew      = "event.access_request.new"
	TypeEventAccessRequestApproved = "event.access_request.approved"
	TypeEventAccessRequestRejected = "event.access_request.rejected"
)
