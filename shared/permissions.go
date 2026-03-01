package shared

// Permission bitfield flags for roles.
const (
	PermManageServer   int64 = 1 << 0
	PermManageChannels int64 = 1 << 1
	PermManageRoles    int64 = 1 << 2
	PermManageMessages int64 = 1 << 3
	PermKickUsers      int64 = 1 << 4
	PermBanUsers       int64 = 1 << 5
	PermManageInvites  int64 = 1 << 6
	PermSendMessages   int64 = 1 << 7
	PermAttachFiles    int64 = 1 << 8
	PermJoinVoice      int64 = 1 << 9
	PermSpeak                 int64 = 1 << 10
	PermManageAccessRequests  int64 = 1 << 11

	// PermAllAdmin is a convenience mask for all admin permissions.
	PermAllAdmin int64 = PermManageServer | PermManageChannels | PermManageRoles |
		PermManageMessages | PermKickUsers | PermBanUsers | PermManageInvites |
		PermManageAccessRequests

	// PermDefaultMember is the default permission set for new users.
	PermDefaultMember int64 = PermSendMessages | PermAttachFiles | PermJoinVoice | PermSpeak
)

// HasPermission checks if a permission bitfield contains a specific flag.
func HasPermission(perms, flag int64) bool {
	return perms&flag != 0
}
