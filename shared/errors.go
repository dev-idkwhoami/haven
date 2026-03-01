package shared

// General WS error codes (any namespace).
const (
	ErrBadRequest       = "BAD_REQUEST"
	ErrNotFound         = "NOT_FOUND"
	ErrPermissionDenied = "PERMISSION_DENIED"
	ErrForbidden        = "FORBIDDEN"
	ErrRateLimited      = "RATE_LIMITED"
	ErrPayloadTooLarge  = "PAYLOAD_TOO_LARGE"
	ErrConflict         = "CONFLICT"
	ErrInternal         = "INTERNAL"
)

// Auth-specific error codes.
const (
	ErrInvalidSignature = "INVALID_SIGNATURE"
	ErrBanned           = "BANNED"
	ErrInvalidInvite    = "INVALID_INVITE"
	ErrInvalidPassword  = "INVALID_PASSWORD"
	ErrNotAllowlisted   = "NOT_ALLOWLISTED"
	ErrSessionExpired   = "SESSION_EXPIRED"
)

// Access request error codes.
const (
	ErrAccessRequestNotFound  = "ACCESS_REQUEST_NOT_FOUND"
	ErrAccessRequestDuplicate = "ACCESS_REQUEST_DUPLICATE"
)

// File-specific error codes.
const (
	ErrFileTooLarge = "FILE_TOO_LARGE"
	ErrStorageFull  = "STORAGE_FULL"
)
