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

func EncodeExtensions(b *[]byte, kvPairs []model.MoqtKeyValuePair){
	// Object Extension Headers are serialized as Key-Value-Pairs (see Figure 2), prefixed by the length of the serialized Key-Value-Pairs, in bytes

	// 	Extensions {
	//   Extension Headers Length (i),
	//   Extension headers (..),
	//}

	*b = quicvarint.Append(*b, uint64(len(kvPairs)))
	for _, kv := range kvPairs {
		EncodeMoqtKeyValuePair(b, kv)
	}
}

func EncodeObjectDatagram(b *[]byte, dg *ObjectDatagram){
	*b = quicvarint.Append(*b, dg.Dtype.TypeID)
	*b = quicvarint.Append(*b, dg.TrackAlias)
	EncodeMoqtLocation(b, dg.Location) // Note that if object id is ommited than it defaults to 0

	if dg.PublisherPriority.Valid {
		*b = quicvarint.Append(*b ,uint64(dg.PublisherPriority.Val))
	}
	if dg.Extensions.Valid {
		EncodeExtensions(b, dg.Extensions.Val)
	}
	if dg.Status.Valid {
		*b = quicvarint.Append(*b, uint64(dg.Status.Val))
	} else if dg.Payload.Valid {
		*b = append(*b, dg.Payload.Val...)
	}
}

// Reason Phrase {
//   Reason Phrase Length (i),
//   Reason Phrase Value (..)
// }