package message

import (
	"go-moq/pkg/model"

	"github.com/quic-go/quic-go/quicvarint"
)

// This file will provide all the encoding logic for structs defined in the "model" package

/*
Since the first 2 bits of a varint specifies it's size, the maximum value it can hold is:
(2^62 - 1), the quicvarint package's implementation for the "Append" function panics when this value is exceeded.
when using any of these function, we should include recover() if we dont want any panic
*/

// Location {
//   Group (i),
//   Object (i)
// }

func EncodeMoqtLocation(b *[]byte, loc model.MoqtLocation) {
	*b = quicvarint.Append(*b, loc.GroupId)
	*b = quicvarint.Append(*b, loc.ObjectId)
}

// Key-Value-Pair {
//   Type (i),
//   [Length (i),]
//   Value (..)
// }

func EncodeMoqtKeyValuePair(b *[]byte, kvPair model.MoqtKeyValuePair){
	// If type is even
	// type(i) + value(i)

	// If type is odd
	// type(i) + length(i) + buf[len]

	if kvPair.Type % 2 == 0 {
		*b = quicvarint.Append(*b, kvPair.Type)
		*b = quicvarint.Append(*b, kvPair.ValueUInt64)
	} else {
		*b = quicvarint.Append(*b, kvPair.Type)
		*b = quicvarint.Append(*b, uint64(len(kvPair.ValueBytes)))
		*b = append(*b, kvPair.ValueBytes... )
	}
}

// Reason Phrase {
//   Reason Phrase Length (i),
//   Reason Phrase Value (..)
// }