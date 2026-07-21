package utils

import (
	"sync"
)

type RingBuffer struct {
	mu sync.RWMutex
	buffer []float32
	writePos int
	count int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]float32, size),

	}
}

func (r *RingBuffer) Write(samples []float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sample := range samples {
		r.buffer[r.writePos] = sample
		r.writePos = (r.writePos+1) % len(r.buffer)

		if r.count < len(r.buffer) {
			r.count++
		}
	}
	
}

func (r *RingBuffer) ReadLatest(n int) []float32 {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if n > r.count {
        n = r.count
    }

    samples := make([]float32, 0, n)

    capacity := len(r.buffer)
    pointerIndex := (r.writePos - n + capacity) % capacity

    for i := 0; i < n; i++ {
        samples = append(samples, r.buffer[pointerIndex])

        pointerIndex++
        if pointerIndex == capacity {
            pointerIndex = 0
        }
    }

    return samples
}