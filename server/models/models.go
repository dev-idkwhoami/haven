package models

// AllModels returns all server-side GORM models for auto-migration.
func AllModels() []any {
	return []any{
		&User{},
		&Server{},
		&Category{},
		&Channel{},
		&ChannelRoleAccess{},
		&Message{},
		&MessageFile{},
		&File{},
		&Role{},
		&UserRole{},
		&Ban{},
		&AuditLogEntry{},
		&InviteCode{},
		&DMConversation{},
		&DMParticipant{},
		&DMMessage{},
		&ErasureRecord{},
		&Session{},
		&AccessRequest{},
	}
}
