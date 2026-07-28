package utils

import (
	"math"

	"github.com/gen2brain/malgo"
)

type Output struct {
	ctx *malgo.AllocatedContext
	device *malgo.Device
	Format Format
	Buffer *PlaybackBuffer
}

func NewOutput(buffer *PlaybackBuffer) (*Output, error) {

    ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
    if err != nil {
        return nil, err
    }

    return &Output{
        ctx: ctx,
        Format: Format{
            SampleRate: 48000,
            Channels:   1,
        },
        Buffer: buffer,
    }, nil
}

func (o *Output) Start() error {
	config := malgo.DefaultDeviceConfig(malgo.Playback)
	config.Playback.Format = malgo.FormatF32
	config.Playback.Channels = uint32(o.Format.Channels)
	config.SampleRate = uint32(o.Format.SampleRate)

	deviceCallback := malgo.DeviceCallbacks{
		Data: func(output, input []byte, frameCount uint32) {
			samples := make([]float32, frameCount)
			o.Buffer.Read(samples)

			for i, sample := range samples {
				bits := math.Float32bits(sample)
				offset := i*4

				output[offset] = byte(bits)
				output[offset+1] = byte(bits >> 8)
				output[offset+2] = byte(bits >> 16)
				output[offset+3] = byte(bits >> 24)
			}

		},
	}

	device, err := malgo.InitDevice(
		o.ctx.Context,
		config,
		deviceCallback,
	)

	if err != nil {
		return err
	}

	o.device = device

	return device.Start()
}

func (o *Output) Stop() {
	if o.device != nil {
		o.device.Uninit()
	}

	o.ctx.Uninit()
	o.ctx.Free()
}