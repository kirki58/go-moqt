package model

import (
	"fmt"

	"github.com/LukaGiorgadze/gonull/v2"
)

// A subgroup object has the following wire representation format:

// {
//   Object ID Delta (i),
//   [Extensions (..),]
//   Object Payload Length (i),
//   [Object Status (i),]
//   [Object Payload (..),]
// }
// Refer to section 10.4.2

type SubgroupObject struct {
	ObjectIdDelta uint64
	Extensions    gonull.Nullable[[]MoqtKeyValuePair]
	Status        gonull.Nullable[MoqtObjectStatus]
	Payload       gonull.Nullable[[]byte]
}

type SubgroupObjectOption func(*SubgroupObject)

func SOWithExtensions(exts []MoqtKeyValuePair) SubgroupObjectOption { // Set extensions inside the subgroup object
	return func(so *SubgroupObject) {
		so.Extensions = gonull.NewNullable(exts)
	}
}

func SOWithStatus(status MoqtObjectStatus) SubgroupObjectOption { // Set the status field for the subgroup object
	return func(so *SubgroupObject) {
		so.Status = gonull.NewNullable(status)
	}
}

func SOWithPayload(payload []byte) SubgroupObjectOption { // Set the payload field (and the payload length field in wire representation) for the subgroup object
	return func(so *SubgroupObject) {
		so.Payload = gonull.NewNullable(payload)
	}
}

// Rule 1 --> Status and Payload are mutually exclusive, they can't exist together, but the object needs to be one of them
// Rule 2 --> Any status object with status value other than "Normal (0x0)" are not allowed to have extensions
func NewSubgroupObject(objectIdDelta uint64, opts ...SubgroupObjectOption) (*SubgroupObject, error) {
	subgroupObject := &SubgroupObject{
		ObjectIdDelta: objectIdDelta,
	}

	// Apply all options
	for _, opt := range opts {
		opt(subgroupObject)
	}

	// Validate the rules
	if subgroupObject.Status.Valid && subgroupObject.Status.Val != Normal && subgroupObject.Payload.Valid {
		return nil, fmt.Errorf("non-normal status and payload cannot both be present")
	}

	if !subgroupObject.Status.Valid && !subgroupObject.Payload.Valid {
		return nil, fmt.Errorf("either status or payload must be present")
	}

	if subgroupObject.Status.Valid && subgroupObject.Status.Val != Normal && subgroupObject.Extensions.Valid && len(subgroupObject.Extensions.Val) != 0 {
		return nil, fmt.Errorf("non-normal status objects cannot have extensions (except when they are empty)")
	}

	return subgroupObject, nil
}
