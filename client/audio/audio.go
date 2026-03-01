package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	SampleRate = 48000
	FrameSize  = 960 // 20ms at 48kHz
	Channels   = 1
)

// AudioDevice represents an audio input or output device.
type AudioDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	initOnce sync.Once
	initErr  error
)

// Init initializes the PortAudio library. Safe to call multiple times.
func Init() error {
	initOnce.Do(func() {
		initErr = portaudio.Initialize()
	})
	return initErr
}

// Terminate shuts down the PortAudio library.
func Terminate() error {
	return portaudio.Terminate()
}

// ListInputDevices returns all available audio input (microphone) devices.
func ListInputDevices() ([]AudioDevice, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}

	var inputs []AudioDevice
	for _, d := range devices {
		if d.MaxInputChannels > 0 {
			inputs = append(inputs, AudioDevice{
				ID:   d.Name,
				Name: d.Name,
			})
		}
	}
	return inputs, nil
}

// ListOutputDevices returns all available audio output (speaker/headphone) devices.
func ListOutputDevices() ([]AudioDevice, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}

	var outputs []AudioDevice
	for _, d := range devices {
		if d.MaxOutputChannels > 0 {
			outputs = append(outputs, AudioDevice{
				ID:   d.Name,
				Name: d.Name,
			})
		}
	}
	return outputs, nil
}

// findDevice finds a PortAudio device by name. Returns nil if not found.
func findDevice(name string) (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}

	for _, d := range devices {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, fmt.Errorf("device not found: %s", name)
}
