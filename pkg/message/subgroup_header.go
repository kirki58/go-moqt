package message

import (
	"fmt"
	"go-moq/pkg/model"

	"github.com/LukaGiorgadze/gonull/v2"
)

// Used when Object Forwarding Preference = Subgroup

// Defines common values for subgroup objects sent on a single unidirectional stream

// SUBGROUP_HEADER {
//   Type (i) = 0x10..0x1D,
//   Track Alias (i),
//   Group ID (i),
//   [Subgroup ID (i),]
//   [Publisher Priority (8),]
// }

// https://github.com/moq-wg/moq-transport/pull/1400/commits/52e8f403d62ed8697d5b932dae4c00ee23988a4e

// Subgroup Header type determines the presence properties of objects and the stream itself

// The Type field in the SUBGROUP_HEADER takes the form 0b00X1XXXX (or the set of values from 0x10 to 0x1F, 0x30 to 0x3F),
// where bit 4 is always set to 1. The
// four low-order bits and bit 5 determine which fields are present in the header:

// The **EXTENSIONS** bit (0x01) in the Type is set to indicate that an Extensions field is present in all Objects in this Subgroup.
// When set to 0, the Extensions field is never present in any object in the subgroup stream.
// Objects with no extensions set Extension Headers Length to 0 even though they are in a subgroup stream where this is set.

// The **SUBGROUP_ID_MODE** field (bits 1-2, mask 0x06) is a two-bit field that determines the encoding of the Subgroup ID.
// To extract this value, perform a bitwise AND with mask 0x06 and right-shift by 1 bit:

// * 0b00: The Subgroup ID field is absent and the Subgroup ID is 0.
// * 0b01: The Subgroup ID field is absent and the Subgroup ID is the Object ID of the first Object transmitted in this Subgroup.
// * 0b10: The Subgroup ID field is present in the header.
// * 0b11: Reserved. If an endpoint receives this value, it MUST close the session with a `PROTOCOL_VIOLATION`.

// The **END_OF_GROUP** bit (0x08) when set, means that the stream's end is the group's end

// The **NO_PRIORITY** bit (0x20) in the Type is set to indicate that the Priority
// field is absent. When set to 1, the Priority field is omitted and this Subgroup
// inherits the Publisher Priority specified in the control message that established
// the subscription. When set to 0, the Priority field is present in the Subgroup
// header.

// Define bitmasks as defined

const (
	maskExtensions     = 0x01
	maskSubgroupIdMode = 0x06
	maskEndOfGroup     = 0x08
	maskNoPriority     = 0x20
)

// Subgroup Id Mode enum
type SubgroupIdMode int

const (
	SubgroupIdModeAbsentZero SubgroupIdMode = iota
	SubgroupIdModeAbsentFirstObject
	SubgroupIdModePresent
	SubgroupIdModeReserved
)

type SubGroupHeaderType struct {
	TypeID            uint64
	ExtensionsPresent bool
	SGIDMode          SubgroupIdMode
	EndOfGroup        bool
	PriorityPresent   bool
}

func NewSubGroupHeaderType(typeId uint64) (*SubGroupHeaderType, error) {
	// Validate the typeId range for Subgroup Headers
	// The Type field in the SUBGROUP_HEADER takes the form 0b00X1XXXX (or the set of values from 0x10 to 0x1F, 0x30 to 0x3F),
	// where bit 4 is always set to 1.
	if (typeId&0x10) == 0 || (typeId > 0x1F && typeId < 0x30) || typeId > 0x3F {
		return nil, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("tried to resolve invalid subgroup header type: 0x%x", typeId)),
		}
	}

	sgt := SubGroupHeaderType{
		TypeID:            typeId,
		ExtensionsPresent: (typeId & maskExtensions) != 0,
		EndOfGroup:        (typeId & maskEndOfGroup) != 0,
		PriorityPresent:   (typeId & maskNoPriority) == 0, // Inverted logic: if maskNoPriority is set, Priority is NOT present
	}

	// Extract Subgroup ID Mode
	modeVal := (typeId & maskSubgroupIdMode) >> 1
	switch modeVal {
	case 0b00:
		sgt.SGIDMode = SubgroupIdModeAbsentZero
	case 0b01:
		sgt.SGIDMode = SubgroupIdModeAbsentFirstObject
	case 0b10:
		sgt.SGIDMode = SubgroupIdModePresent
	case 0b11:
		return nil, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Reserved subgroup ID mode (0b11) in type: 0x%x", typeId)),
		}
	}

	return &sgt, nil
}

func (sgt *SubGroupHeaderType) ToUint64() uint64 {
	// 1. Start with the fixed bit 4 set (0x10)
	// The specification requires form 0b00X1XXXX, so bit 4 is always 1.
	var typeId uint64 = 0x10

	// 2. Set Extensions bit (Bit 0, Mask 0x01)
	if sgt.ExtensionsPresent {
		typeId |= 0x01
	}

	// 3. Set Subgroup ID Mode bits (Bits 1-2, Mask 0x06)
	switch sgt.SGIDMode {
	case SubgroupIdModeAbsentZero:
		// 0b00: Do nothing (bits remain 0)
	case SubgroupIdModeAbsentFirstObject:
		// 0b01: Set bit 1
		typeId |= 0x02
	case SubgroupIdModePresent:
		// 0b10: Set bit 2
		typeId |= 0x04

		// Note: 0b11 should never be the case
	}

	// 4. Set End Of Group bit (Bit 3, Mask 0x08)
	if sgt.EndOfGroup {
		typeId |= 0x08
	}

	// 5. Set No Priority bit (Bit 5, Mask 0x20)
	// IMPORTANT: The wire bit is "NO_PRIORITY".
	// If PriorityPresent is TRUE, the bit must be 0.
	// If PriorityPresent is FALSE, the bit must be 1.
	if !sgt.PriorityPresent {
		typeId |= 0x20
	}

	return typeId
}

type SubGroupHeader struct {
	SGType            SubGroupHeaderType
	TrackAlias        uint64
	GroupId           uint64
	SubgroupId        gonull.Nullable[uint64]
	PublisherPriority gonull.Nullable[uint8]
}

type SubgroupHeaderOption func(*SubGroupHeader)

func SHWithExtensions() SubgroupHeaderOption { // Objects inside this subgroup may have extensions
	return func(sh *SubGroupHeader) {
		sh.SGType.ExtensionsPresent = true
	}
}

func SHWithSubgroupId(sgid uint64) SubgroupHeaderOption {
	return func(sh *SubGroupHeader) {
		sh.SGType.SGIDMode = SubgroupIdModePresent
		sh.SubgroupId = gonull.NewNullable(sgid)
	}
}

func SHWithSubgroupIdAbsentFirstObject() SubgroupHeaderOption {
	return func(sh *SubGroupHeader) {
		sh.SGType.SGIDMode = SubgroupIdModeAbsentFirstObject
	}
}

// Note, when neither of these options are given, sgidmode will default to AbsentZero!!

func SHWithEndOfGroup() SubgroupHeaderOption {
	return func(sh *SubGroupHeader) {
		sh.SGType.EndOfGroup = true
	}
}

func SHWithPublisherPriority(priority uint8) SubgroupHeaderOption {
	return func(sh *SubGroupHeader) {
		sh.SGType.PriorityPresent = true
		sh.PublisherPriority = gonull.NewNullable(priority)
	}
}

func NewSubGroupHeader(trackAlias uint64, groupId uint64, opts ...SubgroupHeaderOption) (*SubGroupHeader, error) {
	subgroupHeader := &SubGroupHeader{
		TrackAlias: trackAlias,
		GroupId:    groupId,
	}

	// Apply all options
	for _, opt := range opts {
		opt(subgroupHeader)
	}

	typeId := subgroupHeader.SGType.ToUint64()
	subgroupHeader.SGType.TypeID = typeId

	return subgroupHeader, nil
}
