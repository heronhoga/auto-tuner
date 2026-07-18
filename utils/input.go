package utils

import (
	"fmt"
	"unsafe"

	"github.com/gen2brain/malgo"
)

type Input struct {
	ctx *malgo.AllocatedContext
	device *malgo.Device
	Frames chan Frame
}

type Frame struct {
	Samples []float32
	SampleRate int
	Channels int
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
		Frames: make(chan Frame, 16),
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
				
				samples[j] = *(*float32)(unsafe.Pointer(&bits))
			}

			frame := Frame{
				Samples: samples,
			}

			select {
			case i.Frames <- frame:
			default:
			}
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