package message

// This file will provide all the decoding logic implemented in `wire_encoder.go` for the structs defined in the "model" package

import (
	"fmt"
	"go-moq/pkg/model"

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
