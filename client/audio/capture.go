package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

// Capturer captures PCM audio from an input device.
type Capturer struct {
	mu      sync.Mutex
	stream  *portaudio.Stream
	running bool
	stopCh  chan struct{}
}

// NewCapturer creates a new audio capturer.
func NewCapturer() *Capturer {
	return &Capturer{}
}

// Start opens the input stream and begins capturing audio frames.
// onFrame is called with 960 float32 samples (20ms at 48kHz mono) per frame.
func (c *Capturer) Start(deviceID string, onFrame func([]float32)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("capturer already running")
	}

	dev, err := findDevice(deviceID)
	if err != nil {
		return fmt.Errorf("find input device: %w", err)
	}

	if dev.MaxInputChannels < Channels {
		return fmt.Errorf("device %s does not support %d input channels", deviceID, Channels)
	}

	buf := make([]float32, FrameSize)
	params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   dev,
			Channels: Channels,
			Latency:  dev.DefaultLowInputLatency,
		},
		SampleRate:      SampleRate,
		FramesPerBuffer: FrameSize,
	}

	stream, err := portaudio.OpenStream(params, buf)
	if err != nil {
		return fmt.Errorf("open input stream: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("start input stream: %w", err)
	}

	c.stream = stream
	c.running = true
	c.stopCh = make(chan struct{})

	go c.readLoop(buf, onFrame)

	return nil
}

// Stop stops the capture stream.
func (c *Capturer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	close(c.stopCh)
	c.running = false

	if err := c.stream.Stop(); err != nil {
		c.stream.Close()
		return fmt.Errorf("stop input stream: %w", err)
	}
	return c.stream.Close()
}

func (c *Capturer) readLoop(buf []float32, onFrame func([]float32)) {
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.stream.Read(); err != nil {
			select {
			case <-c.stopCh:
				return
			default:
				continue
			}
		}

		frame := make([]float32, len(buf))
		copy(frame, buf)
		onFrame(frame)
	}
}
