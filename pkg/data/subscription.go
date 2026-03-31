package data

import (
	"fmt"
	"go-moq/pkg/model"

	"github.com/LukaGiorgadze/gonull/v2"
)

// SubscriptionStatus represents the lifecycle states of a subscription as defined in Section 5.1.
type SubscriptionStatus int

const (
	SubscriptionStatusIdle        SubscriptionStatus = iota // Initial state.
	SubscriptionStatusPending                               // SUBSCRIBE/PUBLISH sent, waiting for OK/ERROR.
	SubscriptionStatusEstablished                           // Handshake successful, data can flow.
	SubscriptionStatusTerminated                            // Subscription ended or rejected.
)

type SubscriptionFilterType uint64

const (
	NextGroupStart SubscriptionFilterType = 0x1
	LargestObject  SubscriptionFilterType = 0x2
	AbsoluteStart  SubscriptionFilterType = 0x3
	AbsoluteRange  SubscriptionFilterType = 0x4
)

func IsValidSubscriptionFilterType(filterType SubscriptionFilterType) bool {
	switch filterType {
	case NextGroupStart, LargestObject, AbsoluteStart, AbsoluteRange:
		return true
	}
	return false
}

// The publisher has a valid start location field always whether it assigns it itself through NextGroup or LargestObject OR
// The filter explicitly determines it
// While the subscriber encodes a subscription filter for subscription filters, publisher-implicit StartLocation is omitted on the wire!
// in which case the publisher determines the StartLocation after receiving the filter
// the filter-instantiating functions are implemented so that they can be used both by the subscriber and the publisher (thus, NewNextGroupStart and NewLargestObject) has a
// "startLocation" parameter, this field can just be assigned 0, 0 by the subscriber before encoding on the wire, it wont be encoded anyways. since it's going to be determined by the publisher itself,
//  and upon receiving a SUBSCRIBE_OK it can be assigned on the subscriber side too

type SubscriptionFilter struct {
	FilterType    SubscriptionFilterType
	StartLocation model.MoqtLocation
	EndGroup      gonull.Nullable[uint64]
}

func NewNextGroupStartFilter(startLocation model.MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    NextGroupStart,
		StartLocation: startLocation,
	}
}

func NewLargestObjectFilter(startLocation model.MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    LargestObject,
		StartLocation: startLocation,
	}
}

func NewAbsoluteStartFilter(startLocation model.MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    AbsoluteStart,
		StartLocation: startLocation,
	}
}

// If the specified End Group is the same group specified in Start Location,
// the remainder of that Group passes the filter. End Group MUST specify the same or a larger Group than specified in Start Location.
func NewAbsoluteRangeFilter(startLocation model.MoqtLocation, endGroup uint64) (*SubscriptionFilter, error) {
	if endGroup < startLocation.GroupId {
		return nil, fmt.Errorf("INVALID_RANGE specified EndGroup cannot be smaller than the specified StartLocation's GroupID")
	}

	return &SubscriptionFilter{
		FilterType:    AbsoluteRange,
		StartLocation: startLocation,
		EndGroup:      gonull.NewNullable(endGroup),
	}, nil
}

// Subscription represents the state of a single subscription within the session.
type Subscription struct {
	ID uint64 // The Request ID used in the SUBSCRIBE/PUBLISH message.

	Alias uint64

	FullTrackName *model.MoqtFullTrackName

	Filter SubscriptionFilter

	Status SubscriptionStatus

	Parameters []model.MoqtKeyValuePair

	Publisher         *Publisher               // Publisher for this subscription
	Dispatcher        *Dispatcher              // the track dispatcher this subscription was issued for
	DispatcherChannel <-chan *model.MoqtObject // The channel publisher use to receive objects from subscribed track's object dispatcher It's important that this is a buffered channel
	// Note: closing this channel is the responsibility of the
}
