package h264

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
)

// This is a mock for an encoder, it will act like it is actively encoding and "dispatching" network objects to the publisher via the dispatcher.
// Though there is no real encoding, it will dispatch a fake track to the publisher.
// A 1:1 mapping exists between an encoder and a dispatcher

type MockEncoder struct {
	dispatcher *session.Dispatcher
}

func NewMockEncoder(disp *session.Dispatcher) *MockEncoder {
	return &MockEncoder{
		dispatcher: disp,
	}
}

// will create and dispatch track_size amount of objects, will increment object group id for each group_size objects
// object payloads are psuedo-randomly generated with the given 2 seeds.
func (en *MockEncoder) Encode(seed1, seed2 uint64, trackSize, groupSize uint16, ftn *model.MoqtFullTrackName) {
	if groupSize > trackSize {
		return
	}

	// Group boundaries handling
	latestGroup := uint16(0)
	latestObj := uint16(0)

	for i := uint16(0); i < trackSize; i++ {
		// in-stack slice to prevent unnecessary heap allocation to store randomly-generated payload
		var payloadBytesArr [128]byte
		payloadBytes := payloadBytesArr[:0]

		loc := model.MoqtLocation{GroupId: uint64(latestGroup), ObjectId: uint64(latestObj)}
		message.EncodeMoqtLocation(&payloadBytes, loc)

		obj := &model.MoqtObject{
			Location:                   loc,
			SubgroupID:                 0, // Not important right now
			FullTrackName:              *ftn,
			PublisherPriority:          128, // Not important
			ObjectForwardingPreference: model.Subgroup,
			ObjectStatus:               model.Normal,
			Payload:                    payloadBytes,
		}
		en.dispatcher.Dispatch(obj)

		if latestObj == groupSize-1 {
			// End of group reached
			latestGroup++
		}
		latestObj++
	}
}
