package handlers

import (
	"encoding/hex"
	"encoding/json"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"

	"github.com/pion/webrtc/v4"
)

func RegisterVoiceHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeVoiceJoin, handleVoiceJoin(d))
	router.Register(shared.TypeVoiceLeave, handleVoiceLeave(d))
	router.Register(shared.TypeVoiceSignal, handleVoiceSignal(d))
	router.Register(shared.TypeVoiceMute, handleVoiceMute(d))
	router.Register(shared.TypeVoiceDeafen, handleVoiceDeafen(d))
}

func handleVoiceJoin(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermJoinVoice) {
			return
		}

		var req struct {
			ChannelID string `json:"channel_id"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "channel_id is required")
			return
		}

		// Verify channel access
		if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, req.ChannelID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to channel")
			return
		}

		// Verify it's a voice channel
		var channel models.Channel
		if err := d.DB.First(&channel, "id = ?", req.ChannelID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "channel not found")
			return
		}
		if channel.Type != shared.ChannelTypeVoice {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "not a voice channel")
			return
		}

		room := d.SFU.GetOrCreateRoom(req.ChannelID)

		// Set up speaking event callback
		room.OnSpeakingChange = func(roomID, pubKeyHex string, speaking bool) {
			eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceSpeaking, map[string]any{
				"channel_id": roomID,
				"pubkey":     pubKeyHex,
				"speaking":   speaking,
			})
			d.Hub.BroadcastToChannel(roomID, eventBytes)
		}

		// Create PeerConnection
		peerConn, err := d.SFU.API().NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create peer connection")
			return
		}

		// Handle ICE candidates — send to client
		peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}
			candidateJSON := c.ToJSON()
			eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceSignal, map[string]any{
				"channel_id":    req.ChannelID,
				"ice_candidate": candidateJSON,
			})
			client.Send(eventBytes)
		})

		// Get current participants before adding new one
		existingParticipants := room.GetParticipantList()

		// Add participant
		_, err = room.AddParticipant(client.PubKeyHex, peerConn)
		if err != nil {
			peerConn.Close()
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to join voice room")
			return
		}

		// Create SDP offer
		offer, err := peerConn.CreateOffer(nil)
		if err != nil {
			room.RemoveParticipant(client.PubKeyHex)
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create SDP offer")
			return
		}
		if err := peerConn.SetLocalDescription(offer); err != nil {
			room.RemoveParticipant(client.PubKeyHex)
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to set local description")
			return
		}

		// Build participant list for response
		participantList := make([]map[string]any, 0, len(existingParticipants))
		for _, p := range existingParticipants {
			var user models.User
			if err := d.DB.Where("public_key = ?", hexToBytes(p.PubKey)).First(&user).Error; err != nil {
				continue
			}
			participantList = append(participantList, map[string]any{
				"pubkey":       p.PubKey,
				"display_name": user.DisplayName,
				"is_muted":     p.IsMuted,
				"is_deafened":  p.IsDeafened,
			})
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"participants": participantList,
			"voice_key":    hex.EncodeToString(room.GetVoiceKey()),
			"sdp_offer":    offer.SDP,
		})

		// Get display name for broadcast
		var user models.User
		displayName := client.PubKeyHex[:8]
		if err := d.DB.First(&user, "id = ?", client.UserID).Error; err == nil {
			displayName = user.DisplayName
		}

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceJoined, map[string]any{
			"channel_id":   req.ChannelID,
			"pubkey":       client.PubKeyHex,
			"display_name": displayName,
		})
		d.Hub.BroadcastToChannel(req.ChannelID, eventBytes)
	}
}

func handleVoiceLeave(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ChannelID string `json:"channel_id"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "channel_id is required")
			return
		}

		room := d.SFU.GetRoom(req.ChannelID)
		if room == nil {
			ws.SendOK(client, msg.Type, msg.ID, nil)
			return
		}

		room.RemoveParticipant(client.PubKeyHex)
		if room.IsEmpty() {
			d.SFU.RemoveRoom(req.ChannelID)
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceLeft, map[string]any{
			"channel_id": req.ChannelID,
			"pubkey":     client.PubKeyHex,
		})
		d.Hub.BroadcastToChannel(req.ChannelID, eventBytes)
	}
}

func handleVoiceSignal(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ChannelID    string           `json:"channel_id"`
			SDPAnswer    *string          `json:"sdp_answer"`
			ICECandidate *json.RawMessage `json:"ice_candidate"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "channel_id is required")
			return
		}

		room := d.SFU.GetRoom(req.ChannelID)
		if room == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "voice room not found")
			return
		}

		p := room.GetParticipant(client.PubKeyHex)
		if p == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "not in voice room")
			return
		}

		// Apply SDP answer
		if req.SDPAnswer != nil {
			answer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  *req.SDPAnswer,
			}
			if err := p.PeerConn.SetRemoteDescription(answer); err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to apply SDP answer")
				return
			}
		}

		// Apply ICE candidate
		if req.ICECandidate != nil {
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal(*req.ICECandidate, &candidate); err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid ICE candidate")
				return
			}
			if err := p.PeerConn.AddICECandidate(candidate); err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to add ICE candidate")
				return
			}
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)
	}
}

func handleVoiceMute(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Muted bool `json:"muted"`
		}
		if !parsePayload(msg, &req) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid payload")
			return
		}

		roomID, room := d.SFU.FindRoomByParticipant(client.PubKeyHex)
		if room == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "not in a voice room")
			return
		}

		p := room.GetParticipant(client.PubKeyHex)
		if p != nil {
			p.IsMuted = req.Muted
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceMute, map[string]any{
			"channel_id": roomID,
			"pubkey":     client.PubKeyHex,
			"muted":      req.Muted,
		})
		d.Hub.BroadcastToChannel(roomID, eventBytes)
	}
}

func handleVoiceDeafen(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Deafened bool `json:"deafened"`
		}
		if !parsePayload(msg, &req) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid payload")
			return
		}

		roomID, room := d.SFU.FindRoomByParticipant(client.PubKeyHex)
		if room == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "not in a voice room")
			return
		}

		p := room.GetParticipant(client.PubKeyHex)
		if p != nil {
			p.IsDeafened = req.Deafened
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceDeafen, map[string]any{
			"channel_id": roomID,
			"pubkey":     client.PubKeyHex,
			"deafened":   req.Deafened,
		})
		d.Hub.BroadcastToChannel(roomID, eventBytes)
	}
}

func hexToBytes(h string) []byte {
	b, _ := hex.DecodeString(h)
	return b
}
