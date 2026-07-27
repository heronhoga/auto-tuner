package utils

import "sync"

type PlaybackBuffer struct {
	mu sync.Mutex
	buffer []float32
	readPos int
	writePos int
	count int	
}

func NewPlaybackBuffer(size int) *PlaybackBuffer {
	return &PlaybackBuffer{
		buffer: make([]float32, size),
	}
}

func (b *PlaybackBuffer) Write(samples []float32) {
    b.mu.Lock()
    defer b.mu.Unlock()

    for _, sample := range samples {

        if b.count == len(b.buffer) {
            break
        }

        b.buffer[b.writePos] = sample

        b.writePos = (b.writePos + 1) % len(b.buffer)

        b.count++
    }
}

func (b *PlaybackBuffer) Read(dst []float32) {

    b.mu.Lock()
    defer b.mu.Unlock()

	if len(b.buffer) == 0 {
    	for i := range dst {
        	dst[i] = 0
    	}
    	return
	}

    for i := range dst {

        if b.count == 0 {
            dst[i] = 0
            continue
        }

        dst[i] = b.buffer[b.readPos]
		b.buffer[b.writePos] = 0

        b.readPos = (b.readPos + 1) % len(b.buffer)
        b.count--
    }
}