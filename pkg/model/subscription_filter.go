package model

import (
	"fmt"

	"github.com/LukaGiorgadze/gonull/v2"
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
	StartLocation MoqtLocation
	EndGroup      gonull.Nullable[uint64]
}

func NewNextGroupStartFilter(startLocation MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    NextGroupStart,
		StartLocation: startLocation,
	}
}

func NewLargestObjectFilter(startLocation MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    LargestObject,
		StartLocation: startLocation,
	}
}

func NewAbsoluteStartFilter(startLocation MoqtLocation) *SubscriptionFilter {
	return &SubscriptionFilter{
		FilterType:    AbsoluteStart,
		StartLocation: startLocation,
	}
}

// If the specified End Group is the same group specified in Start Location,
// the remainder of that Group passes the filter. End Group MUST specify the same or a larger Group than specified in Start Location.
func NewAbsoluteRangeFilter(startLocation MoqtLocation, endGroup uint64) (*SubscriptionFilter, error) {
	if endGroup < startLocation.GroupId {
		return nil, fmt.Errorf("INVALID_RANGE specified EndGroup cannot be smaller than the specified StartLocation's GroupID")
	}

	return &SubscriptionFilter{
		FilterType:    AbsoluteRange,
		StartLocation: startLocation,
		EndGroup:      gonull.NewNullable(endGroup),
	}, nil
}