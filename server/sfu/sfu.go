package sfu

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

// SFU manages voice rooms for both channel voice and DM calls.
type SFU struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	api   *webrtc.API
}

// NewSFU creates a new SFU with Opus-only codec configuration.
func NewSFU() *SFU {
	m := &webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	return &SFU{
		rooms: make(map[string]*Room),
		api:   api,
	}
}

// GetOrCreateRoom returns the room for the given ID, creating it if needed.
func (s *SFU) GetOrCreateRoom(roomID string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[roomID]; ok {
		return room
	}

	room := NewRoom(roomID)
	s.rooms[roomID] = room
	return room
}

// GetRoom returns the room for the given ID, or nil if it doesn't exist.
func (s *SFU) GetRoom(roomID string) *Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rooms[roomID]
}

// RemoveRoom removes a room by ID.
func (s *SFU) RemoveRoom(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, roomID)
}

// API returns the shared webrtc.API for creating PeerConnections.
func (s *SFU) API() *webrtc.API {
	return s.api
}

// FindRoomByParticipant returns the room ID and room containing a participant, or empty string/nil.
func (s *SFU) FindRoomByParticipant(pubKeyHex string) (string, *Room) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, room := range s.rooms {
		if room.HasParticipant(pubKeyHex) {
			return id, room
		}
	}
	return "", nil
}
