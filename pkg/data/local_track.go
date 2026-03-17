package data

import (
	"go-moq/internal"
	"go-moq/pkg/model"
)

type LocalTrack struct{
	fullTrackName model.MoqtFullTrackName
	forwardingPreference model.MoqtObjectForwardingPreference
	
	datagramRingBuf *internal.RingBuffer[*ObjectDatagram] // RingBuffer contains a mutex, NEVER copy a mutex use a pointer.
	// subgroupRingBuf *internal.RingBuffer[*SubgroupObject]
}