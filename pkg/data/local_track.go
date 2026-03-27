package data

import (
	"go-moq/internal"
	"go-moq/pkg/model"
)

type LocalTrack struct {
	fullTrackName        model.MoqtFullTrackName
	forwardingPreference model.MoqtObjectForwardingPreference

	datagramRingBuf *internal.RingBuffer[*ObjectDatagram] // RingBuffer contains a mutex, NEVER copy a mutex use a pointer.
	subgroupRingBuf *internal.RingBuffer[*SubgroupObject]
}

func NewLocalTrack(fullTrackName model.MoqtFullTrackName, forwardingPreference model.MoqtObjectForwardingPreference, size int) *LocalTrack {
	if forwardingPreference == model.Subgroup {
		return &LocalTrack{
			fullTrackName: fullTrackName,
			forwardingPreference: forwardingPreference,
			subgroupRingBuf: internal.NewRingBuffer[*SubgroupObject](size),
			datagramRingBuf: nil,
		}
	} else {
		return &LocalTrack{
			fullTrackName:        fullTrackName,
			forwardingPreference: forwardingPreference,
			subgroupRingBuf:      nil,
			datagramRingBuf:      internal.NewRingBuffer[*ObjectDatagram](size),
		}
	}
}
