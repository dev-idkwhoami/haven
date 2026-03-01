package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"gorm.io/gorm"

	"haven/client/audio"
	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/shared"
)

// VoiceParticipant represents a participant in a voice channel.
type VoiceParticipant struct {
	PubKey      string `json:"pubKey"`
	DisplayName string `json:"displayName"`
	IsMuted     bool   `json:"isMuted"`
	IsDeafened  bool   `json:"isDeafened"`
	IsSpeaking  bool   `json:"isSpeaking"`
}

// VoiceService manages voice channel participation and audio devices.
// Only one active voice session at a time.
type VoiceService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey

	mu       sync.Mutex
	session  *voiceSession
	capturer *audio.Capturer
	player   *audio.Player
	encoder  *audio.Encoder
	decoder  *audio.Decoder

	inputDeviceID  string
	outputDeviceID string
	muted          bool
	deafened       bool
}

// voiceSession tracks the active voice session state.
type voiceSession struct {
	serverID   int64
	channelID  string
	voiceKey   []byte
	peerConn   *webrtc.PeerConnection
	audioTrack *webrtc.TrackLocalStaticRTP
}

// NewVoiceService creates a new VoiceService.
func NewVoiceService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *VoiceService {
	return &VoiceService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (v *VoiceService) SetContext(ctx context.Context) {
	v.ctx = ctx
}

// JoinChannel joins a voice channel and sets up WebRTC audio.
func (v *VoiceService) JoinChannel(serverID int64, channelID string) ([]VoiceParticipant, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.session != nil {
		return nil, fmt.Errorf("already in a voice channel — leave first")
	}

	conn, err := v.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	// Request to join voice channel via WS.
	resp, err := conn.Request(shared.TypeVoiceJoin, map[string]interface{}{
		"channel_id": channelID,
	})
	if err != nil {
		return nil, fmt.Errorf("voice.join: %w", err)
	}

	var joinResp struct {
		Participants []voiceParticipantWS `json:"participants"`
		VoiceKey     string               `json:"voice_key"`
		SDPOffer     string               `json:"sdp_offer"`
	}
	if err := json.Unmarshal(resp.Payload, &joinResp); err != nil {
		return nil, fmt.Errorf("unmarshal voice.join: %w", err)
	}

	voiceKey, err := havenCrypto.HexDecode(joinResp.VoiceKey)
	if err != nil {
		return nil, fmt.Errorf("decode voice key: %w", err)
	}

	// Set up Pion PeerConnection.
	peerConn, audioTrack, err := v.setupPeerConnection(conn, channelID, voiceKey, joinResp.SDPOffer)
	if err != nil {
		// Tell server we're leaving since setup failed.
		conn.Send(shared.TypeVoiceLeave, map[string]interface{}{
			"channel_id": channelID,
		})
		return nil, fmt.Errorf("setup peer connection: %w", err)
	}

	v.session = &voiceSession{
		serverID:   serverID,
		channelID:  channelID,
		voiceKey:   voiceKey,
		peerConn:   peerConn,
		audioTrack: audioTrack,
	}

	// Update connection voice state.
	conn.SetVoiceRoom(&connection.VoiceState{
		ChannelID: channelID,
		RoomID:    channelID,
	})

	// Start audio capture if not muted.
	if !v.muted {
		v.startCapture()
	}

	// Start audio playback if not deafened.
	if !v.deafened {
		v.startPlayback()
	}

	// Convert participants.
	participants := make([]VoiceParticipant, len(joinResp.Participants))
	for i, p := range joinResp.Participants {
		participants[i] = VoiceParticipant{
			PubKey:      p.PubKey,
			DisplayName: p.DisplayName,
			IsMuted:     p.IsMuted,
			IsDeafened:  p.IsDeafened,
		}
	}

	slog.Info("joined voice channel", "server", serverID, "channel", channelID, "participants", len(participants))
	return participants, nil
}

// LeaveChannel leaves the current voice channel.
func (v *VoiceService) LeaveChannel() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.session == nil {
		return nil
	}

	// Stop audio.
	v.stopCapture()
	v.stopPlayback()

	// Close WebRTC peer connection.
	if v.session.peerConn != nil {
		v.session.peerConn.Close()
	}

	// Notify server.
	conn, err := v.manager.Get(v.session.serverID)
	if err == nil {
		conn.Send(shared.TypeVoiceLeave, map[string]interface{}{
			"channel_id": v.session.channelID,
		})
		conn.SetVoiceRoom(nil)
	}

	slog.Info("left voice channel", "server", v.session.serverID, "channel", v.session.channelID)
	v.session = nil
	return nil
}

// SetMuted sets the mute state. When muted, audio capture stops.
func (v *VoiceService) SetMuted(muted bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.muted = muted

	if v.session != nil {
		conn, err := v.manager.Get(v.session.serverID)
		if err != nil {
			return fmt.Errorf("get connection: %w", err)
		}

		conn.Send(shared.TypeVoiceMute, map[string]interface{}{
			"muted": muted,
		})

		if muted {
			v.stopCapture()
		} else {
			v.startCapture()
		}
	}

	emitEvent(v.ctx, "voice:muted", map[string]interface{}{
		"pubKey": havenCrypto.HexEncode(v.privKey.Public().(ed25519.PublicKey)),
		"muted":  muted,
	})
	return nil
}

// SetDeafened sets the deafen state. When deafened, audio playback stops.
func (v *VoiceService) SetDeafened(deafened bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.deafened = deafened

	if v.session != nil {
		conn, err := v.manager.Get(v.session.serverID)
		if err != nil {
			return fmt.Errorf("get connection: %w", err)
		}

		conn.Send(shared.TypeVoiceDeafen, map[string]interface{}{
			"deafened": deafened,
		})

		if deafened {
			v.stopPlayback()
		} else {
			v.startPlayback()
		}
	}

	emitEvent(v.ctx, "voice:deafened", map[string]interface{}{
		"pubKey":   havenCrypto.HexEncode(v.privKey.Public().(ed25519.PublicKey)),
		"deafened": deafened,
	})
	return nil
}

// GetInputDevices returns available audio input devices.
func (v *VoiceService) GetInputDevices() ([]audio.AudioDevice, error) {
	if err := audio.Init(); err != nil {
		return nil, fmt.Errorf("init audio: %w", err)
	}
	return audio.ListInputDevices()
}

// GetOutputDevices returns available audio output devices.
func (v *VoiceService) GetOutputDevices() ([]audio.AudioDevice, error) {
	if err := audio.Init(); err != nil {
		return nil, fmt.Errorf("init audio: %w", err)
	}
	return audio.ListOutputDevices()
}

// SetInputDevice sets the input device for future voice sessions.
func (v *VoiceService) SetInputDevice(deviceID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.inputDeviceID = deviceID

	// If currently in a session, restart capture with new device.
	if v.session != nil && !v.muted {
		v.stopCapture()
		v.startCapture()
	}
	return nil
}

// SetOutputDevice sets the output device for future voice sessions.
func (v *VoiceService) SetOutputDevice(deviceID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.outputDeviceID = deviceID

	// If currently in a session, restart playback with new device.
	if v.session != nil && !v.deafened {
		v.stopPlayback()
		v.startPlayback()
	}
	return nil
}

// GetVolume returns the current playback volume (0-100).
func (v *VoiceService) GetVolume() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.player != nil {
		return v.player.GetVolume()
	}
	return 100
}

// SetVolume sets the playback volume (0-100).
func (v *VoiceService) SetVolume(level int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.player != nil {
		v.player.SetVolume(level)
	}
	return nil
}

// setupPeerConnection creates the Pion PeerConnection, audio track, and handles signaling.
func (v *VoiceService) setupPeerConnection(conn *connection.ServerConnection, channelID string, voiceKey []byte, sdpOffer string) (*webrtc.PeerConnection, *webrtc.TrackLocalStaticRTP, error) {
	// Configure media engine for Opus only.
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    1,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, nil, fmt.Errorf("register opus codec: %w", err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	peerConn, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, nil, fmt.Errorf("create peer connection: %w", err)
	}

	// Create outbound audio track for sending our audio.
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  1,
		},
		"audio",
		"haven-voice",
	)
	if err != nil {
		peerConn.Close()
		return nil, nil, fmt.Errorf("create audio track: %w", err)
	}

	if _, err := peerConn.AddTrack(audioTrack); err != nil {
		peerConn.Close()
		return nil, nil, fmt.Errorf("add audio track: %w", err)
	}

	// Handle incoming audio tracks from other participants.
	peerConn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go v.handleIncomingTrack(track, voiceKey)
	})

	// Handle ICE candidates — send to server.
	peerConn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		conn.Send(shared.TypeVoiceSignal, map[string]interface{}{
			"channel_id":    channelID,
			"ice_candidate": candidate.ToJSON(),
		})
	})

	// Set the remote SDP offer from the SFU.
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpOffer,
	}
	if err := peerConn.SetRemoteDescription(offer); err != nil {
		peerConn.Close()
		return nil, nil, fmt.Errorf("set remote description: %w", err)
	}

	// Create and set local SDP answer.
	answer, err := peerConn.CreateAnswer(nil)
	if err != nil {
		peerConn.Close()
		return nil, nil, fmt.Errorf("create answer: %w", err)
	}

	if err := peerConn.SetLocalDescription(answer); err != nil {
		peerConn.Close()
		return nil, nil, fmt.Errorf("set local description: %w", err)
	}

	// Send SDP answer to server.
	conn.Send(shared.TypeVoiceSignal, map[string]interface{}{
		"channel_id": channelID,
		"sdp_answer": answer.SDP,
	})

	return peerConn, audioTrack, nil
}

// handleIncomingTrack reads RTP packets from a remote track, decrypts, decodes, and plays audio.
func (v *VoiceService) handleIncomingTrack(track *webrtc.TrackRemote, voiceKey []byte) {
	dec, err := audio.NewDecoder()
	if err != nil {
		slog.Error("failed to create opus decoder for incoming track", "error", err)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, _, readErr := track.Read(buf)
		if readErr != nil {
			return
		}

		// Parse the RTP packet.
		pkt := &rtp.Packet{}
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}

		// Decrypt the RTP payload using the voice key (E2EE).
		plaintext, err := havenCrypto.DecryptBlob(voiceKey, pkt.Payload)
		if err != nil {
			continue
		}

		// Decode Opus to PCM.
		pcm, err := dec.Decode(plaintext)
		if err != nil {
			continue
		}

		// Play the decoded audio.
		v.mu.Lock()
		player := v.player
		deafened := v.deafened
		v.mu.Unlock()

		if player != nil && !deafened {
			player.Play(pcm)
		}
	}
}

// startCapture begins audio capture and sends encrypted RTP to the SFU.
func (v *VoiceService) startCapture() {
	if v.session == nil || v.inputDeviceID == "" {
		return
	}

	enc, err := audio.NewEncoder(64)
	if err != nil {
		slog.Error("failed to create opus encoder", "error", err)
		return
	}
	v.encoder = enc

	capturer := audio.NewCapturer()
	v.capturer = capturer

	session := v.session
	if err := capturer.Start(v.inputDeviceID, func(pcm []float32) {
		// Encode PCM to Opus.
		opusData, err := enc.Encode(pcm)
		if err != nil {
			return
		}

		// Encrypt the Opus data with voice key (E2EE).
		encrypted, err := havenCrypto.EncryptBlob(session.voiceKey, opusData)
		if err != nil {
			return
		}

		// Build an RTP packet with encrypted payload.
		pkt := &rtp.Packet{
			Header: rtp.Header{
				Version:     2,
				PayloadType: 111,
			},
			Payload: encrypted,
		}

		rtpBytes, err := pkt.Marshal()
		if err != nil {
			return
		}

		// Write to the outbound audio track.
		if session.audioTrack != nil {
			session.audioTrack.Write(rtpBytes)
		}
	}); err != nil {
		slog.Error("failed to start audio capture", "error", err)
	}
}

// stopCapture stops audio capture.
func (v *VoiceService) stopCapture() {
	if v.capturer != nil {
		v.capturer.Stop()
		v.capturer = nil
	}
	v.encoder = nil
}

// startPlayback begins audio playback.
func (v *VoiceService) startPlayback() {
	if v.outputDeviceID == "" {
		return
	}

	player := audio.NewPlayer()
	if err := player.Start(v.outputDeviceID); err != nil {
		slog.Error("failed to start audio playback", "error", err)
		return
	}
	v.player = player

	dec, err := audio.NewDecoder()
	if err != nil {
		slog.Error("failed to create opus decoder", "error", err)
		return
	}
	v.decoder = dec
}

// stopPlayback stops audio playback.
func (v *VoiceService) stopPlayback() {
	if v.player != nil {
		v.player.Stop()
		v.player = nil
	}
	v.decoder = nil
}

// voiceParticipantWS is the wire format for a voice participant.
type voiceParticipantWS struct {
	PubKey      string `json:"pubkey"`
	DisplayName string `json:"display_name"`
	IsMuted     bool   `json:"is_muted"`
	IsDeafened  bool   `json:"is_deafened"`
}
