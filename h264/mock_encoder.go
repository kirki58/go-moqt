package h264

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"iter"
	"time"
)

// This is a mock for an encoder, it will act like it is actively encoding and "dispatching" network objects to the publisher via the dispatcher.
// Though there is no real encoding, it will dispatch a fake track to the publisher.
// A 1:1 mapping exists between an encoder and a dispatcher

type MockEncoder struct {
	dispatcher *session.Dispatcher
	trackSize  uint16
	groupSize  uint16
	ftn        *model.MoqtFullTrackName
}

func NewMockEncoder(disp *session.Dispatcher, trackSize uint16, groupSize uint16, ftn *model.MoqtFullTrackName) *MockEncoder {
	return &MockEncoder{
		dispatcher: disp,
		trackSize: trackSize,
		groupSize: groupSize,
		ftn: ftn,
	}
}

// will create and dispatch track_size amount of objects, will increment object group id for each group_size objects
// object payloads are psuedo-randomly generated with the given 2 seeds.
func (en *MockEncoder) Encode() {
	if en.groupSize > en.trackSize {
		return
	}

	// Group boundaries handling
	latestGroup := uint16(0)
	latestObj := uint16(0)

	for i := uint16(0); i < en.trackSize; i++ {
		// in-stack slice to prevent unnecessary heap allocation to store randomly-generated payload
		var payloadBytesArr [128]byte
		payloadBytes := payloadBytesArr[:0]

		loc := model.MoqtLocation{GroupId: uint64(latestGroup), ObjectId: uint64(latestObj)}
		message.EncodeMoqtLocation(&payloadBytes, loc)

		obj := &model.MoqtObject{
			Location:                   loc,
			SubgroupID:                 0, // Not important right now
			FullTrackName:              *en.ftn,
			PublisherPriority:          128, // Not important
			ObjectForwardingPreference: model.Subgroup,
			ObjectStatus:               model.Normal,
			Payload:                    payloadBytes,
		}
		en.dispatcher.Dispatch(obj)

		if latestObj == en.groupSize-1 {
			// End of group reached
			latestGroup++
			latestObj = 0
		}else{
			latestObj++
		}

		time.Sleep(33 * time.Millisecond)
	}
	en.dispatcher.CloseAll()
}

func (en *MockEncoder) ExpectedLocationsAfter(loc model.MoqtLocation) iter.Seq[model.MoqtLocation] {
	return func(yield func(model.MoqtLocation) bool) {
		latestGroup := uint16(loc.GroupId)
		latestObj := uint16(loc.ObjectId)
		currentIndex := (latestGroup * en.groupSize) + latestObj

		for i := currentIndex + 1; i < en.trackSize; i++ {
			if latestObj == en.groupSize -1 {
				latestGroup++
				latestObj = 0
			}else{
				latestObj++
			}

			loc := model.MoqtLocation{GroupId: uint64(latestGroup), ObjectId: uint64(latestObj)}
			if !yield(loc){
				return
			}
		}
	}
}

func (en *MockEncoder) ExpectedObjectCountAfter(loc model.MoqtLocation) uint16{
	latestGroup := uint16(loc.GroupId)
	latestObj := uint16(loc.ObjectId)
	currentCount := (latestGroup * en.groupSize) + latestObj + 1
	return en.trackSize - currentCount
}
