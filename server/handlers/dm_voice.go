package handlers

import (
	"encoding/hex"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"

	"github.com/pion/webrtc/v4"
)

func RegisterDMVoiceHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeDMVoiceStart, handleDMVoiceStart(d))
	router.Register(shared.TypeDMVoiceAccept, handleDMVoiceAccept(d))
	router.Register(shared.TypeDMVoiceReject, handleDMVoiceReject(d))
	router.Register(shared.TypeDMVoiceLeave, handleDMVoiceLeave(d))
}

func handleDMVoiceStart(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		room := d.SFU.GetOrCreateRoom(req.ConversationID)

		// Set up speaking event callback for DM voice
		room.OnSpeakingChange = func(roomID, pubKeyHex string, speaking bool) {
			eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceSpeaking, map[string]any{
				"channel_id": roomID,
				"pubkey":     pubKeyHex,
				"speaking":   speaking,
			})
			broadcastToDMParticipants(d, roomID, eventBytes)
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
				"channel_id":    req.ConversationID,
				"ice_candidate": candidateJSON,
			})
			client.Send(eventBytes)
		})

		_, err = room.AddParticipant(client.PubKeyHex, peerConn)
		if err != nil {
			peerConn.Close()
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to start voice call")
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

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"voice_key": hex.EncodeToString(room.GetVoiceKey()),
			"sdp_offer": offer.SDP,
		})

		// Ring all participants
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMVoiceRinging, map[string]any{
			"conversation_id": req.ConversationID,
			"caller_pubkey":   client.PubKeyHex,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMVoiceAccept(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
			SDPAnswer      string `json:"sdp_answer"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		room := d.SFU.GetOrCreateRoom(req.ConversationID)

		// Create PeerConnection for the accepting participant
		peerConn, err := d.SFU.API().NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create peer connection")
			return
		}

		peerConn.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}
			candidateJSON := c.ToJSON()
			eventBytes, _ := ws.MarshalEvent(shared.TypeEventVoiceSignal, map[string]any{
				"channel_id":    req.ConversationID,
				"ice_candidate": candidateJSON,
			})
			client.Send(eventBytes)
		})

		existingParticipants := room.GetParticipantList()

		_, err = room.AddParticipant(client.PubKeyHex, peerConn)
		if err != nil {
			peerConn.Close()
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to join voice call")
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
			})
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"participants": participantList,
		})

		// Broadcast joined event
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMVoiceJoined, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          client.PubKeyHex,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMVoiceReject(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMVoiceDeclined, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          client.PubKeyHex,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMVoiceLeave(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		room := d.SFU.GetRoom(req.ConversationID)
		if room == nil {
			ws.SendOK(client, msg.Type, msg.ID, nil)
			return
		}

		room.RemoveParticipant(client.PubKeyHex)

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMVoiceLeft, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          client.PubKeyHex,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)

		// If room is empty, end the call
		if room.IsEmpty() {
			d.SFU.RemoveRoom(req.ConversationID)

			endBytes, _ := ws.MarshalEvent(shared.TypeEventDMVoiceEnded, map[string]any{
				"conversation_id": req.ConversationID,
			})
			broadcastToDMParticipants(d, req.ConversationID, endBytes)
		}
	}
}
