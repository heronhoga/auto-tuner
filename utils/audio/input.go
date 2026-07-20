package utils

import (
	"fmt"
	"math"

	"github.com/gen2brain/malgo"
)

type Input struct {
	ctx *malgo.AllocatedContext
	device *malgo.Device
	Format Format
	Broadcaster *Broadcaster
}

func NewInput() (*Input, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		fmt.Println(message)
	})

	if err != nil {
		return nil, err
	}

	return &Input{
		ctx: ctx,
		Format: Format{
			SampleRate: 48000,
			Channels: 1,
		},
		Broadcaster: NewBroadcaster(),
	}, nil
	
}

func (i *Input) Start() error {
	config := malgo.DefaultDeviceConfig(malgo.Capture)

	config.Capture.Format = malgo.FormatF32
	config.Capture.Channels = 1
	config.SampleRate = 48000

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(output, input []byte, frameCount uint32) {
			samples := make([]float32, frameCount)

			for j := uint32(0); j < frameCount; j++ {
				offset := j * 4

				bits := uint32(input[offset]) | uint32(input[offset+1])<<8 | uint32(input[offset+2])<<16 | uint32(input[offset+3])<<24
				
				samples[j] = math.Float32frombits(bits)
			}

			frame := Frame{
				Samples: samples,
			}

			i.Broadcaster.Publish(frame)
		},
	}

	device, err := malgo.InitDevice(i.ctx.Context, config, deviceCallbacks)

	if err != nil {
		return err
	}

	i.device = device

	return device.Start()

}

func (i *Input) Stop() {
	if i.device != nil {
		i.device.Uninit()
	}

	i.ctx.Uninit()
	i.ctx.Free()
}