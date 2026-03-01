package sfu

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Participant represents a user in a voice room.
type Participant struct {
	PubKey     string
	PeerConn   *webrtc.PeerConnection
	AudioTrack *webrtc.TrackLocalStaticRTP
	IsMuted    bool
	IsDeafened bool

	// VAD state
	lastPacketTime time.Time
	packetCount    int
	speaking       bool
	vadMu          sync.Mutex
}

// Room represents a voice room (channel or DM).
type Room struct {
	ID           string
	mu           sync.RWMutex
	Participants map[string]*Participant
	VoiceKey     []byte

	// VAD callback: called when speaking state changes
	OnSpeakingChange func(roomID, pubKeyHex string, speaking bool)
}

// NewRoom creates a new voice room.
func NewRoom(id string) *Room {
	return &Room{
		ID:           id,
		Participants: make(map[string]*Participant),
	}
}

// AddParticipant adds a new participant to the room. It creates an audio track,
// adds existing tracks to the new peer, and adds the new track to all existing peers.
func (r *Room) AddParticipant(pubKey string, peerConn *webrtc.PeerConnection) (*Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate voice key if first participant
	if len(r.Participants) == 0 && len(r.VoiceKey) == 0 {
		r.VoiceKey = make([]byte, 32)
		rand.Read(r.VoiceKey)
	}

	// Create audio track for the new participant
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		fmt.Sprintf("audio-%s", pubKey),
		fmt.Sprintf("haven-%s", pubKey),
	)
	if err != nil {
		return nil, fmt.Errorf("create audio track: %w", err)
	}

	p := &Participant{
		PubKey:     pubKey,
		PeerConn:   peerConn,
		AudioTrack: track,
	}

	// Add existing participants' audio tracks to the new peer connection
	for _, existing := range r.Participants {
		if _, err := peerConn.AddTrack(existing.AudioTrack); err != nil {
			slog.Error("add existing track to new peer", "error", err, "existing", existing.PubKey)
		}
	}

	// Add the new participant's audio track to all existing peer connections
	for _, existing := range r.Participants {
		if _, err := existing.PeerConn.AddTrack(track); err != nil {
			slog.Error("add new track to existing peer", "error", err, "existing", existing.PubKey)
		}
	}

	// Handle incoming audio from this participant
	peerConn.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go r.forwardRTP(pubKey, remoteTrack, track)
	})

	r.Participants[pubKey] = p
	return p, nil
}

// RemoveParticipant removes a participant and cleans up their tracks.
func (r *Room) RemoveParticipant(pubKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.Participants[pubKey]
	if !ok {
		return
	}

	// Close peer connection (which removes all tracks)
	if p.PeerConn != nil {
		p.PeerConn.Close()
	}

	delete(r.Participants, pubKey)
}

// IsEmpty returns true if the room has no participants.
func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Participants) == 0
}

// HasParticipant checks if a participant is in the room.
func (r *Room) HasParticipant(pubKeyHex string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.Participants[pubKeyHex]
	return ok
}

// GetParticipant returns a participant by pubkey hex, or nil.
func (r *Room) GetParticipant(pubKeyHex string) *Participant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Participants[pubKeyHex]
}

// GetParticipantList returns a snapshot of all participants.
func (r *Room) GetParticipantList() []*Participant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Participant, 0, len(r.Participants))
	for _, p := range r.Participants {
		list = append(list, p)
	}
	return list
}

// GetVoiceKey returns the room's voice encryption key.
func (r *Room) GetVoiceKey() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := make([]byte, len(r.VoiceKey))
	copy(key, r.VoiceKey)
	return key
}

// forwardRTP reads RTP packets from a remote track and writes them to the local track,
// while tracking packet rate for VAD.
func (r *Room) forwardRTP(pubKey string, remote *webrtc.TrackRemote, local *webrtc.TrackLocalStaticRTP) {
	buf := make([]byte, 1500)
	for {
		n, _, err := remote.Read(buf)
		if err != nil {
			return
		}

		if _, err := local.Write(buf[:n]); err != nil {
			return
		}

		// VAD: track packet timing
		r.updateVAD(pubKey)
	}
}

// updateVAD updates Voice Activity Detection state for a participant.
func (r *Room) updateVAD(pubKey string) {
	r.mu.RLock()
	p, ok := r.Participants[pubKey]
	r.mu.RUnlock()
	if !ok {
		return
	}

	p.vadMu.Lock()
	defer p.vadMu.Unlock()

	now := time.Now()

	// Reset counter if window expired (200ms)
	if now.Sub(p.lastPacketTime) > 200*time.Millisecond {
		p.packetCount = 0
	}
	p.lastPacketTime = now
	p.packetCount++

	// Threshold: >10 packets in 200ms = speaking
	wasSpeaking := p.speaking
	p.speaking = p.packetCount > 10

	if p.speaking != wasSpeaking && r.OnSpeakingChange != nil {
		speaking := p.speaking
		go r.OnSpeakingChange(r.ID, pubKey, speaking)
	}
}
