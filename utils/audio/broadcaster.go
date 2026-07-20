package utils

import "sync"

type Broadcaster struct {
	mu sync.RWMutex
	subs []chan Frame
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{}
}

func (b *Broadcaster) Subscribe() <-chan Frame {
	ch := make(chan Frame, 8)

	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()

	return ch
}

func (b *Broadcaster) Publish(frame Frame) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}