package core

import "sync"

type LogHub struct {
	mu   sync.Mutex
	subs map[chan LogEntry]struct{}
	ring []LogEntry
	cap  int
	head int
	size int
}

func NewLogHub(capacity int) *LogHub {
	return &LogHub{
		subs: make(map[chan LogEntry]struct{}),
		ring: make([]LogEntry, capacity),
		cap:  capacity,
	}
}

func (h *LogHub) Append(e LogEntry) {
	if h == nil || h.cap == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ring[h.head] = e
	h.head = (h.head + 1) % h.cap
	if h.size < h.cap {
		h.size++
	}

	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *LogHub) Snapshot() []LogEntry {
	if h == nil || h.size == 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]LogEntry, h.size)
	start := (h.head - h.size + h.cap) % h.cap
	for i := 0; i < h.size; i++ {
		out[i] = h.ring[(start+i)%h.cap]
	}
	return out
}

func (h *LogHub) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 64)
	h.mu.Lock()
	if h.subs == nil {
		h.subs = make(map[chan LogEntry]struct{})
	}
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *LogHub) Unsubscribe(ch chan LogEntry) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}
