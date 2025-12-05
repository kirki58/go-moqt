package message

import (
	"fmt"
	"go-moq/pkg/model"

	"github.com/LukaGiorgadze/gonull/v2"
)

/*
	Important Note: As of writing this note, the type values 0x2C and 0x2D are obviosly wrongly written in Table 5. They are duplicates of 0x28 and 0x29 in their respectible order and yet have different
	typeid values, they also do not follow the bitmask logic implemented in here so they are considered outliers.

	ref: [IETF MOQT draft 15](https://www.ietf.org/archive/id/draft-ietf-moq-transport-15.html#name-object-extension-header)
*/

// Used when Forwarding Preference = Datagram. The entire object (header + payload) fits inside one QUIC Datagram frame.

// Wire Format:

//     Type (varint): One of the specific IDs (e.g., 0x00, 0x01, 0x20) from Table 5. This ID acts as a bitmask indicating which optional fields follow.

//     Track Alias (varint): The alias assigned in SUBSCRIBE_OK.

//     Group ID (varint).

//     [Optional] Object ID (varint): Present if the Type indicates it. If missing, it implies Object ID 0.

//     [Optional] Publisher Priority (8 bits): Present if the Type indicates it.

//     [Optional] Extensions (varint length + bytes): Present if the Type indicates it.

//     Payload or Status:

//         Payload: If the Type indicates a payload (e.g., 0x00), the rest of the datagram is the payload. There is no length field.

//         Status (varint): If the Type indicates status (e.g., 0x20), this field follows. Used for empty objects (e.g., EndOfGroup).

// OBJECT_DATAGRAM {
//   Type (i) = 0x0-0x7,0x20-21,0x24-25
//   Track Alias (i),
//   Group ID (i),
//   [Object ID (i),]
//   [Publisher Priority (8),]
//   [Extensions (..),]
//   [Object Status (i),]
//   [Object Payload (..),]
// }

// The Type value determines which fields are present in the OBJECT_DATAGRAM. There are 10 defined Type values for OBJECT_DATAGRAM
// Type acts as a bitmask regarding which values are present in the datagram.

// Following is the definition:

// Bit Mask		Hex		Logic Type		0 (Unset)			1 (Set)
// Bit 0		0x01	Normal			No Extensions		Extensions Present
// Bit 1		0x02	Normal			Not End of Group	End of Group
// Bit 2		0x04	Inverted		Object ID Present	Object ID Omitted (0)
// Bit 3		0x08	Inverted		Priority Present	Priority Omitted
// Bit 5		0x20	Toggle			Payload				Status

// Define the bitmasks based on MOQT Draft-15 Table 5
const (
	flagExtensionsPresent = 0x01 // Bit 0: 0000 0001
	flagEndOfGroup        = 0x02 // Bit 1: 0000 0010
	flagObjectIdOmitted   = 0x04 // Bit 2: 0000 0100
	flagPriorityOmitted   = 0x08 // Bit 3: 0000 1000
	flagStatusPresent     = 0x20 // Bit 5: 0010 0000
)

type ObjectDatagramType struct {
	TypeID            uint64
	EndOfGroup        bool // Is end of group? 1 = yes, 0 = no
	ExtensionsPresent bool // Are extension present in datagram? 1 = yes, 0 = no
	ObjectIdPresent   bool // Is Object ID present? 1 = yes, 0 = no, if no, ObjectId should be encoded as 0 by default
	PriorityPresent   bool // Is Publisher Priority present? 1 = yes, 0 = no
	StatusOrPayload   bool // Is it Status (true) or Payload (false)?
}

// Here we apply the bitmask
// This function comes in handy when deserializing objects from the wire
func NewObjectDatagramType(typeId uint64) (*ObjectDatagramType, error) {
	// Verify that the typeId doesn't use bits that aren't defined for Datagrams
	// (like the reserved 0x10 bit or anything > 0x29).
	// Note: While the spec defines specific IDs in Table 5, parsing bitwise is robust.
	// Note: Bit 4 is reserved for Streams and must be 0 for Datagrams.
	if typeId > 0x29 || (typeId&0x10) != 0 {
		return &ObjectDatagramType{}, fmt.Errorf("invalid datagram type ID: 0x%x", typeId)
	}

	dt := ObjectDatagramType{
		TypeID:            typeId,
		ExtensionsPresent: (typeId & flagExtensionsPresent) != 0,
		EndOfGroup:        (typeId & flagEndOfGroup) != 0, // Bit 1: EndOfGroup

		// Bit 2 and 3: Logic Inversion!
		// Flag Set (1) == Omitted == Present(False)
		// Flag Unset (0) == Not Omitted == Present(True)
		ObjectIdPresent: (typeId & flagObjectIdOmitted) == 0,
		PriorityPresent: (typeId & flagPriorityOmitted) == 0,

		StatusOrPayload: (typeId & flagStatusPresent) != 0,
	}

	if dt.StatusOrPayload && dt.EndOfGroup {
		// Depending on strictness, you might return an error or simply force false.
		// For strict correctness based on Table 5:
		return &ObjectDatagramType{}, fmt.Errorf("invalid type 0x%x: EndOfGroup cannot be set on Status objects", typeId)
	}

	// Note:
	// 		The only "invalid" values that pass the first validity check are the ones like
	//  	0x22 or 0x26 (where Status and EndOfGroup are both set). However, the second check (if dt.StatusOrPayload && dt.EndOfGroup) correctly catches those specific cases.

	return &dt, nil
}

func (dt *ObjectDatagramType) ToUInt64() uint64 {
	var typeId uint64 = 0

	// Bit 0: Extensions Present
	if dt.ExtensionsPresent {
		typeId |= 0x01
	}

	// Bit 1: End of Group
	// Note: If StatusOrPayload is true (Status), this bit should technically be 0
	// to produce a valid ID, but this function faithfully encodes the struct state.
	if dt.EndOfGroup {
		typeId |= 0x02
	}

	// Bit 2: Object ID Omitted (Logic Inversion)
	// If ObjectIdPresent is FALSE, we SET the "Omitted" bit (1).
	// If ObjectIdPresent is TRUE, we LEAVE the "Omitted" bit (0).
	if !dt.ObjectIdPresent {
		typeId |= 0x04
	}

	// Bit 3: Priority Omitted
	if !dt.PriorityPresent {
		typeId |= 0x08
	}

	// Bit 5: Status Present
	// False = Payload (0), True = Status (1)
	if dt.StatusOrPayload {
		typeId |= 0x20
	}

	return typeId
}

// Although we avoid creating any invalid datagram type values, this function creates a sense of safety for freeuse of the function ToUInt64()
func (dt *ObjectDatagramType) IsValid() bool{
	// Verify that the typeId doesn't use bits that aren't defined for Datagrams
	if dt.StatusOrPayload && dt.EndOfGroup {
		// Depending on strictness, you might return an error or simply force false.
		// For strict correctness based on Table 5:
		return false
	}
	// Check range and reserved value
	if dt.TypeID > 0x3F || (dt.TypeID&0x10) != 0 {
		return false
	}
	return true
}

type ObjectDatagram struct {
	Dtype             ObjectDatagramType
	TrackAlias        uint64
	Location          model.MoqtLocation // ObjectId is set to zero if it's not "present"

	// Optional fields are marked as "nullable"
	PublisherPriority gonull.Nullable[uint8]
	Extensions        gonull.Nullable[[]model.MoqtKeyValuePair]
	Status            gonull.Nullable[model.MoqtObjectStatus]
	Payload           gonull.Nullable[[]byte]
}

type ObjectDatagramOption func(*ObjectDatagram)

func WithObjectId(oid uint64) ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.ObjectIdPresent = true
		od.Location.ObjectId = oid
	}
}

func WithPublisherPriority(priority uint8) ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.PriorityPresent = true
		od.PublisherPriority = gonull.NewNullable(priority)
	}
}

func WithExtensions(ext []model.MoqtKeyValuePair) ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.ExtensionsPresent = true
		od.Extensions = gonull.NewNullable(ext)
	}
}

func WithEndOfGroup() ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.EndOfGroup = true
	}
}


func WithStatus(status model.MoqtObjectStatus) ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.StatusOrPayload = true
		od.Status = gonull.NewNullable(status)
	}
}

func WithPayload(payload []byte) ObjectDatagramOption {
	return func(od *ObjectDatagram) {
		od.Dtype.StatusOrPayload = false // It's probably already "false" but no harm comes from extra safety right?
		od.Payload = gonull.NewNullable(payload)
	}
}

// Rule 1 --> Either WithPayload or WithStatus options are present, they cant be present at once (they are mutually exclusive)
// Rule 2 --> WithExtensions and WithEndOfGroup options are mutually exclusive and they can be both non-existent

func NewObjectDatagram(trackAlias uint64, groupId uint64, opts ...ObjectDatagramOption) (*ObjectDatagram, error){
	// Define default configuration
	dg := &ObjectDatagram{
		TrackAlias: trackAlias,
		Location: model.MoqtLocation{GroupId: groupId, ObjectId: 0},
	}
	// Apply all options
	for _, opt := range opts {
		opt(dg)
	}

	// Validate Rule 1
	if dg.Status.Valid && dg.Payload.Valid {
		return nil, fmt.Errorf("cannot have both status and payload")
	}
	if !dg.Status.Valid && !dg.Payload.Valid {
		return nil, fmt.Errorf("either status or payload must be present")
	}

	// Validate Rule 2
	if dg.Dtype.ExtensionsPresent && dg.Dtype.EndOfGroup {
		return nil, fmt.Errorf("extensions cannot be present when status is present")
	}

	// Determine the TypeId
	typeId := dg.Dtype.ToUInt64()
	dg.Dtype.TypeID = typeId
	
	// if !dg.Dtype.IsValid() { // It's certain that typeId is valid now, no need for an extra validity check
	// 	return nil, fmt.Errorf("invalid object datagram type constructed: 0x%x", dg.Dtype.TypeID)
	// }

	return dg, nil
}
