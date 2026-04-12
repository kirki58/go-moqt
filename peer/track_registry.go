package peer

import (
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"sync"
)

// TrackRegistry is a thread-safe in-memory implementation of TrackRegistry.
type TrackRegistry struct {
	mu     sync.RWMutex
	tracks map[string]*session.Dispatcher
}

// NewTrackRegistry creates a new TrackRegistry.
func NewTrackRegistry() *TrackRegistry {
	return &TrackRegistry{
		tracks: make(map[string]*session.Dispatcher),
	}
}

// AddTrack registers an available track.
func (r *TrackRegistry) AddTrack(ftn *model.MoqtFullTrackName, dispatcher *session.Dispatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracks[ftn.ToString()] = dispatcher
}

// RemoveTrack unregisters a track.
func (r *TrackRegistry) RemoveTrack(ftn *model.MoqtFullTrackName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tracks, ftn.ToString())
}

func (r *TrackRegistry) GetTrack(ftn *model.MoqtFullTrackName) (*session.Dispatcher, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.tracks[ftn.ToString()]
	return d, ok
}