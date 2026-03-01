package handlers

import (
	"haven/server/ws"
)

// RegisterAll registers all message handlers on the router.
func RegisterAll(d *Deps, router *ws.Router) {
	RegisterServerHandlers(d, router)
	RegisterCategoryHandlers(d, router)
	RegisterChannelHandlers(d, router)
	RegisterMessageHandlers(d, router)
	RegisterUserHandlers(d, router)
	RegisterRoleHandlers(d, router)
	RegisterBanHandlers(d, router)
	RegisterInviteHandlers(d, router)
	RegisterAuditHandlers(d, router)
	RegisterSyncHandlers(d, router)
	RegisterFileHandlers(d, router)
	RegisterDMHandlers(d, router)
	RegisterVoiceHandlers(d, router)
	RegisterDMVoiceHandlers(d, router)
	RegisterAccessRequestHandlers(d, router)
}
