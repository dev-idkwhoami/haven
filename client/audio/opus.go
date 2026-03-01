package audio

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

// Encoder wraps an Opus encoder for voice encoding.
type Encoder struct {
	enc *opus.Encoder
}

// Decoder wraps an Opus decoder for voice decoding.
type Decoder struct {
	dec *opus.Decoder
}

// NewEncoder creates a new Opus encoder with the specified bitrate in kbps.
// Common values: 32, 64, 96, 128.
func NewEncoder(bitrateKbps int) (*Encoder, error) {
	enc, err := opus.NewEncoder(SampleRate, Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %w", err)
	}

	if err := enc.SetBitrate(bitrateKbps * 1000); err != nil {
		return nil, fmt.Errorf("set opus bitrate: %w", err)
	}

	// Enable DTX (Discontinuous Transmission) — reduces bandwidth during silence.
	if err := enc.SetDTX(true); err != nil {
		return nil, fmt.Errorf("set opus dtx: %w", err)
	}

	return &Encoder{enc: enc}, nil
}

// Encode encodes PCM float32 samples to an Opus packet.
// Input must be FrameSize (960) samples.
func (e *Encoder) Encode(pcm []float32) ([]byte, error) {
	// Convert float32 to int16 for the Opus encoder.
	pcm16 := make([]int16, len(pcm))
	for i, s := range pcm {
		if s > 1.0 {
			s = 1.0
		}
		if s < -1.0 {
			s = -1.0
		}
		pcm16[i] = int16(s * 32767)
	}

	// Max Opus packet size.
	buf := make([]byte, 4000)
	n, err := e.enc.Encode(pcm16, buf)
	if err != nil {
		return nil, fmt.Errorf("opus encode: %w", err)
	}
	return buf[:n], nil
}

// NewDecoder creates a new Opus decoder.
func NewDecoder() (*Decoder, error) {
	dec, err := opus.NewDecoder(SampleRate, Channels)
	if err != nil {
		return nil, fmt.Errorf("create opus decoder: %w", err)
	}
	return &Decoder{dec: dec}, nil
}

// Decode decodes an Opus packet to PCM float32 samples.
func (d *Decoder) Decode(data []byte) ([]float32, error) {
	pcm16 := make([]int16, FrameSize)
	n, err := d.dec.Decode(data, pcm16)
	if err != nil {
		return nil, fmt.Errorf("opus decode: %w", err)
	}

	pcm := make([]float32, n)
	for i := 0; i < n; i++ {
		pcm[i] = float32(pcm16[i]) / 32768.0
	}
	return pcm, nil
}
