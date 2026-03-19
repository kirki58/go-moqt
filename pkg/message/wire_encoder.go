package message

import (
	"go-moq/pkg/data"
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

// OBJECT_DATAGRAM {
//   Type (i) = 0x00-0x1F,0x20-21,0x24-25,0x28-29,0x2C-2D
//   Track Alias (i),
//   Group ID (i),
//   [Object ID (i),]
//   [Publisher Priority (8),]
//   [Extensions (..),]
//   [Object Status (i),]
//   [Object Payload (..),]
// }

func EncodeObjectDatagram(b *[]byte, dg *data.ObjectDatagram){
	*b = quicvarint.Append(*b, dg.Dtype.TypeID)
	*b = quicvarint.Append(*b, dg.TrackAlias)
	EncodeMoqtLocation(b, dg.Location) // Note that if object id is ommited than it defaults to 0

	if dg.PublisherPriority.Valid {
		*b = append(*b, dg.PublisherPriority.Val) // Publisher Priority is a single byte, So no need to use quicvarint we manually append it to the byte slice.
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

// OBJECT_DATAGRAM {
//   Type (i) = 0x00-0x1F,0x20-21,0x24-25,0x28-29,0x2C-2D
//   Track Alias (i),
//   Group ID (i),
//   [Object ID (i),]
//   [Publisher Priority (8),]
//   [Extensions (..),]
//   [Object Status (i),]
// }
// In this function we do not encode the payload into the slice
// In video streams payloads can be big, and Instead of loading them into the memory we might want to directly write them to the network transport stream.
// After encoding of "only" the header, than writing it to the stream, we can write "payload" to the stream separetly in session implementation
// The testing for this function is not necessary as long as the common parts of the code are not changed
func EncodeObjectDatagramHeader(b *[]byte, dg *data.ObjectDatagram){
	*b = quicvarint.Append(*b, dg.Dtype.TypeID)
	*b = quicvarint.Append(*b, dg.TrackAlias)
	EncodeMoqtLocation(b, dg.Location) // Note that if object id is ommited than it defaults to 0

	if dg.PublisherPriority.Valid {
		*b = append(*b, dg.PublisherPriority.Val) // Publisher Priority is a single byte, So no need to use quicvarint we manually append it to the byte slice.
	}
	if dg.Extensions.Valid {
		EncodeExtensions(b, dg.Extensions.Val)
	}
	if dg.Status.Valid {
		*b = quicvarint.Append(*b, uint64(dg.Status.Val))
	}
}

// SUBGROUP_HEADER {
//   Type (i) = 0x10..0x1D,
//   Track Alias (i),
//   Group ID (i),
//   [Subgroup ID (i),]
//   [Publisher Priority (8),]
// }
func EncodeSubgroupHeader(b *[]byte, sgh *data.SubGroupHeader){
	*b = quicvarint.Append(*b, sgh.SGType.TypeID)
	*b = quicvarint.Append(*b, sgh.TrackAlias)
	*b = quicvarint.Append(*b, sgh.GroupId)

	if sgh.SGType.SGIDMode == data.SubgroupIdModePresent {
		*b = quicvarint.Append(*b, sgh.SubgroupId.Val)
	}
	if sgh.SGType.PriorityPresent {
		*b = append(*b, sgh.PublisherPriority.Val)
	}
}

// Subgroup Object{
//   Object ID Delta (i),
//   [Extensions (..),]
//   Object Payload Length (i),
//   [Object Status (i),]
//   [Object Payload (..),]
// }

func EncodeSubgroupObject(b* []byte, sgo *data.SubgroupObject){
	*b = quicvarint.Append(*b, sgo.ObjectIdDelta) // Encode ObjectIdDelta as varint
	// Encode extensions if they exist
	if sgo.Extensions.Valid {
		EncodeExtensions(b, sgo.Extensions.Val)
	}
	// For a status object payload length is encoded as 0 and then the status is encoded
	if sgo.Status.Valid {
		*b = quicvarint.Append(*b, uint64(0)) // Encode payload length (which is 0 for a status object)
		*b = quicvarint.Append(*b, uint64(sgo.Status.Val)) // Encode status value
	}else if sgo.Payload.Valid {
		*b = quicvarint.Append(*b, uint64(len(sgo.Payload.Val))) // Encode payload length
		*b = append(*b, sgo.Payload.Val...) // Encode the buffer in
	}
}

// Track Namespace {
//   Number of Track Namespace Fields (i),
//   Track Namespace Field (..) ...
// }

// Each Track Namespace Field is encoded as follows:

// Track Namespace Field {
//   Track Namespace Field Length (i),
//   Track Namespace Field Value (..)
// }

func EncodeTrackNamespace(b* []byte, trackNamespace model.MoqtTrackNamespace){
	*b = quicvarint.Append(*b, uint64(len(trackNamespace)))

	for _, field := range trackNamespace {
		*b = quicvarint.Append(*b, uint64(len(field)))
		*b = append(*b, field...)
	}
}

// FullTrackName {
//  Track Namespace (..),
//  Track Name Length (i),
//  Track Name (..)
//}
func EncodeMoqtFullTrackName(b* []byte, fullTrackName model.MoqtFullTrackName){
	EncodeTrackNamespace(b, fullTrackName.Namespace)
	*b = quicvarint.Append(*b, uint64(len(fullTrackName.Name)))
	*b = append(*b, fullTrackName.Name...)
}

// Reason Phrase {
//   Reason Phrase Length (i),
//   Reason Phrase Value (..)
// }

func EncodeMoqtReasonPhrase(b* []byte, reasonPhrase model.MoqtReasonPhrase){
	*b = quicvarint.Append(*b, uint64(len(reasonPhrase)))
	*b = append(*b, string(reasonPhrase)...)
}