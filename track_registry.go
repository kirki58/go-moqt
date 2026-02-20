package moqt

import (
	"go-moq/pkg/model"
	"sync"
)

// TrackRegistry defines an interface for managing and querying available tracks.
type TrackRegistry interface {
	// AddTrack registers an available track identified by its full track name.
	AddTrack(ftn model.MoqtFullTrackName)

	// RemoveTrack unregisters a track identified by its full track name.
	RemoveTrack(ftn model.MoqtFullTrackName)

	// HasTrack checks if a track with the given full track name exists in the registry.
	HasTrack(ftn model.MoqtFullTrackName) bool
}

// SimpleTrackRegistry is a thread-safe in-memory implementation of TrackRegistry.
type SimpleTrackRegistry struct {
	mu     sync.RWMutex
	tracks map[string]struct{}
}

// NewSimpleTrackRegistry creates a new SimpleTrackRegistry.
func NewSimpleTrackRegistry() *SimpleTrackRegistry {
	return &SimpleTrackRegistry{
		tracks: make(map[string]struct{}),
	}
}

// AddTrack registers an available track.
func (r *SimpleTrackRegistry) AddTrack(ftn model.MoqtFullTrackName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracks[ftn.ToString()] = struct{}{}
}

// RemoveTrack unregisters a track.
func (r *SimpleTrackRegistry) RemoveTrack(ftn model.MoqtFullTrackName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tracks, ftn.ToString())
}

// HasTrack checks if a track exists.
func (r *SimpleTrackRegistry) HasTrack(ftn model.MoqtFullTrackName) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.tracks[ftn.ToString()]
	return exists
}