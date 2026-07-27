package utils

import "github.com/gen2brain/malgo"

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