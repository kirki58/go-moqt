package h264

import (
	"encoding/binary"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"math/rand/v2"
)

// This is a mock for an encoder, it will act like it is actively encoding and "dispatching" network objects to the publisher via the dispatcher.
// Though there is no real encoding, it will dispatch a fake track to the publisher.
// A 1:1 mapping exists between an encoder and a dispatcher

// const TRACK_SIZE = 1800 // 60 objects per 30 groups
const PAYLOAD_SIZE = 8 // uint64s, x 8 = 64 bytes

type MockEncoder struct {
	dispatcher *session.Dispatcher
}

func NewMockEncoder(disp *session.Dispatcher) *MockEncoder{
	return &MockEncoder{
		dispatcher: disp,
	}
}

// will create and dispatch track_size amount of objects, will increment object group id for each group_size objects
// object payloads are psuedo-randomly generated with the given 2 seeds.
func (en *MockEncoder) Encode(seed1, seed2 uint64, trackSize, groupSize uint16, ftn *model.MoqtFullTrackName) {
	if groupSize > trackSize{
		return
	}

	// Group boundaries handling
	latestGroup := uint16(0)
	latestObj := uint16(0)

	// Psuedo-random payload generation
	pcg := rand.NewPCG(seed1, seed2) // psuedo-random number generator to generate payloads for us

	for i := uint16(0); i < trackSize; i++ {
		// in-stack slice to prevent unnecessary heap allocation to store randomly-generated payload
		var payloadBytesArr [PAYLOAD_SIZE * 8]byte
		payloadBytes := payloadBytesArr[:]

		// Put a single uint64 as 8 bytes into the slice in little-endian fashion (to provide conventionality across test platforms)
		for i := 0; i < PAYLOAD_SIZE; i++ {
			n := pcg.Uint64()
			binary.LittleEndian.PutUint64(payloadBytes[i*8:], n)
		}

		obj := &model.MoqtObject{
			Location: model.MoqtLocation{GroupId: uint64(latestGroup), ObjectId: uint64(latestObj)},
			SubgroupID: 0, // Not important right now
			FullTrackName: *ftn,
			PublisherPriority: 128, // Not important
			ObjectForwardingPreference: model.Subgroup,
			ObjectStatus: model.Normal,
			Payload: payloadBytes,
		}
		en.dispatcher.Dispatch(obj)

		if latestObj == groupSize - 1 {
			// End of group reached
			latestGroup++
		}
		latestObj++
	}
}
