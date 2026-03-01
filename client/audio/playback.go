package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

// Player plays PCM audio to an output device.
type Player struct {
	mu      sync.Mutex
	stream  *portaudio.Stream
	running bool
	volume  float32 // gain multiplier, 0.0 to 1.0
	buf     []float32
}

// NewPlayer creates a new audio player with default volume.
func NewPlayer() *Player {
	return &Player{
		volume: 1.0,
	}
}

// Start opens the output stream on the specified device.
func (p *Player) Start(deviceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("player already running")
	}

	dev, err := findDevice(deviceID)
	if err != nil {
		return fmt.Errorf("find output device: %w", err)
	}

	if dev.MaxOutputChannels < Channels {
		return fmt.Errorf("device %s does not support %d output channels", deviceID, Channels)
	}

	p.buf = make([]float32, FrameSize)
	params := portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Device:   dev,
			Channels: Channels,
			Latency:  dev.DefaultLowOutputLatency,
		},
		SampleRate:      SampleRate,
		FramesPerBuffer: FrameSize,
	}

	stream, err := portaudio.OpenStream(params, p.buf)
	if err != nil {
		return fmt.Errorf("open output stream: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("start output stream: %w", err)
	}

	p.stream = stream
	p.running = true
	return nil
}

// Stop stops the playback stream.
func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.running = false
	if err := p.stream.Stop(); err != nil {
		p.stream.Close()
		return fmt.Errorf("stop output stream: %w", err)
	}
	return p.stream.Close()
}

// Play writes PCM samples to the output stream with volume gain applied.
func (p *Player) Play(pcmData []float32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	vol := p.volume
	for i := 0; i < len(pcmData) && i < len(p.buf); i++ {
		p.buf[i] = pcmData[i] * vol
	}

	// Pad with silence if pcmData is shorter than the buffer.
	for i := len(pcmData); i < len(p.buf); i++ {
		p.buf[i] = 0
	}

	p.stream.Write()
}

// SetVolume sets the playback volume. level is 0-100, mapped to a gain multiplier.
func (p *Player) SetVolume(level int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	p.volume = float32(level) / 100.0
}

// GetVolume returns the current volume level (0-100).
func (p *Player) GetVolume() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return int(p.volume * 100)
}
