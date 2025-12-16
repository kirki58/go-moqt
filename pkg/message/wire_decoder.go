package message

// This file will provide all the decoding logic implemented in `wire_encoder.go` for the structs defined in the "model" package

import (
	"fmt"
	"go-moq/pkg/model"

	"github.com/LukaGiorgadze/gonull/v2"
	"github.com/quic-go/quic-go/quicvarint"
)

// Location {
//   Group (i),
//   Object (i)
// }

/*
Passing a byte slice as a reference-type parameter is cheap,
Slices only contain 3 values pointer, length, capacity.
*/
func DecodeMoqtLocation(b []byte) (model.MoqtLocation, int, error) { // location, parsed bytes in total, error
	parsed := 0                            // In order to keep track of parsed bytes from the slice
	groupId, n, err := quicvarint.Parse(b) // Extract GroupId, read n bytes
	parsed += n
	if err != nil {
		return model.MoqtLocation{}, parsed, fmt.Errorf("DecodeMoqtLocation: failed to parse GroupId: %w", err)
	}
	b = b[n:]

	objectId, n, err := quicvarint.Parse(b) // Extract ObjectId, read n bytes
	parsed += n
	if err != nil {
		return model.MoqtLocation{}, parsed, fmt.Errorf("DecodeMoqtLocation: failed to parse ObjectId: %w", err)
	}

	return model.MoqtLocation{GroupId: groupId, ObjectId: objectId}, parsed, nil
}

// Key-Value-Pair {
//   Type (i),
//   [Length (i),]
//   Value (..)
// }

func DecodeMoqtKeyValuePair(b []byte) (model.MoqtKeyValuePair, int, error) {
	parsed := 0
	typ, n, err := quicvarint.Parse(b) // Extract type, read n bytes
	parsed += n
	if err != nil {
		return model.MoqtKeyValuePair{}, parsed, fmt.Errorf("DecodeMoqtKeyValuePair: failed to parse Type: %w", err)
	}
	b = b[n:]

	// Determine the type of the value the kvpair has, also determine if it has the optional "length" field
	if typ%2 == 0 { // kvpair with even type value, has a single varint
		valUint64, n, err := quicvarint.Parse(b) // Extract the varint value
		parsed += n
		if err != nil {
			return model.MoqtKeyValuePair{}, parsed, fmt.Errorf("DecodeMoqtKeyValuePair: failed to parse Value (uint64): %w", err)
		}

		kvPair, err := model.NewMoqtKeyValuePair(typ, valUint64)
		if err != nil {
			return model.MoqtKeyValuePair{}, parsed, err
		}
		return kvPair, parsed, nil

	}

	// Type is odd, there is length on the wire, value is a byte slice
	sliceLen, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return model.MoqtKeyValuePair{}, parsed, fmt.Errorf("DecodeMoqtKeyValuePair: failed to parse Length: %w", err)
	}
	b = b[n:]

	// At this point the buffer, may contain more bytes about different
	// components of an object, But it should be impossible that the buffer's length is smaller than sliceLen
	if len(b) < int(sliceLen) {
		return model.MoqtKeyValuePair{}, parsed, fmt.Errorf("DecodeMoqtKeyValuePair: insufficient bytes for Value, expected %d, got %d", sliceLen, len(b))
	}

	// Allocate a new slice
	valueBytes := make([]byte, sliceLen)
	copy(valueBytes, b[:sliceLen])

	kvPair, err := model.NewMoqtKeyValuePair(typ, valueBytes)
	if err != nil {
		return model.MoqtKeyValuePair{}, parsed, err
	}
	parsed += int(sliceLen)

	return kvPair, parsed, nil
}

// Object Extension Headers are serialized as Key-Value-Pairs (see Figure 2), prefixed by the length of the serialized Key-Value-Pairs, in bytes

//		Extensions {
//	  Extension Headers Length (i),
//	  Extension headers (..),
//	}
func DecodeExtensions(b []byte) ([]model.MoqtKeyValuePair, int, error) {
	parsed := 0
	kvPairsLen, n, err := quicvarint.Parse(b)
	parsed += n

	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeExtensions: failed to parse Extension Headers Length: %w", err)
	}
	b = b[n:]

	kvPairs := make([]model.MoqtKeyValuePair, kvPairsLen)
	for i := uint64(0); i < kvPairsLen; i++ {
		kvp, n, err := DecodeMoqtKeyValuePair(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeExtensions: failed to parse Key-Value-Pair %d: %w", i, err)
		}
		kvPairs[i] = kvp
		b = b[n:]
	}
	return kvPairs, parsed, nil

}

func DecodeObjectDatagram(b []byte) (*ObjectDatagram, int, error){
	parsed := 0
	typId, n, err := quicvarint.Parse(b)
	parsed += n

	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Type ID: %w", err)
	}
	b = b[n:]

	// Create ObjectDatagramType from the parsed TypeID
	dtype, err := NewObjectDatagramType(typId)
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeObjectDatagram: invalid Type ID %d: %w", typId, err)
	}

	// Decode Track Alias
	trackAlias, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Track Alias: %w", err)
	}
	b = b[n:]

	// Decode Location
	location, n, err := DecodeMoqtLocation(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Location: %w", err)
	}
	b = b[n:]

	// Decode Publisher Priority if present
	var publisherPriority uint8
	if dtype.PriorityPresent {
		if len(b) < 1 {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: insufficient bytes for Publisher Priority")
		}
		publisherPriority = b[0]
		parsed += 1
		b = b[1:]
	}

	// Decode Extensions if present
	var extensions []model.MoqtKeyValuePair
	if dtype.ExtensionsPresent {
		ext, n, err := DecodeExtensions(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Extensions: %w", err)
		}
		extensions = ext
		b = b[n:]
	}

	// Decode Status or Payload
	var status model.MoqtObjectStatus
	var payload []byte
	if dtype.StatusOrPayload {
		s, n, err := quicvarint.Parse(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Object Status: %w", err)
		}
		status = model.MoqtObjectStatus(s)
		b = b[n:]
	} else {
		// The rest of the buffer is the payload
		payload = make([]byte, len(b))
		copy(payload, b)
		parsed += len(b)
		b = b[len(b):] // Consume all remaining bytes
	}

	// Construct the ObjectDatagram
	dg := &ObjectDatagram{
		Dtype:      *dtype,
		TrackAlias: trackAlias,
		Location:   location,
	}

	if dtype.PriorityPresent {
		dg.PublisherPriority = gonull.NewNullable(publisherPriority)
	}
	if dtype.ExtensionsPresent {
		dg.Extensions = gonull.NewNullable(extensions)
	}
	if dtype.StatusOrPayload {
		dg.Status = gonull.NewNullable(status)
	} else {
		dg.Payload = gonull.NewNullable(payload)
	}

	return dg, parsed, nil	
}
