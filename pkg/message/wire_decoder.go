package message

// This file will provide all the decoding logic implemented in `wire_encoder.go` for the structs defined in the "model" package

import (
	"fmt"
	"go-moq/pkg/data"
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
	extensionsLen, n, err := quicvarint.Parse(b)
	parsed += n

	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeExtensions: failed to parse Extension Headers Length: %w", err)
	}
	b = b[n:]

	if uint64(len(b)) < extensionsLen {
		return nil, parsed, fmt.Errorf("DecodeExtensions: insufficient bytes for extensions, expected %d, got %d", extensionsLen, len(b))
	}

	var kvPairs []model.MoqtKeyValuePair
	var extBytesParsed uint64
	for extBytesParsed < extensionsLen {
		kvp, n, err := DecodeMoqtKeyValuePair(b)
		if err != nil {
			return nil, parsed + int(extBytesParsed), fmt.Errorf("DecodeExtensions: failed to parse Key-Value-Pair: %w", err)
		}
		kvPairs = append(kvPairs, kvp)
		b = b[n:]
		extBytesParsed += uint64(n)
	}

	if extBytesParsed != extensionsLen {
		return nil, parsed + int(extBytesParsed), fmt.Errorf("DecodeExtensions: parsed bytes %d does not match Extension Headers Length %d", extBytesParsed, extensionsLen)
	}

	parsed += int(extBytesParsed)
	return kvPairs, parsed, nil
}

func DecodeObjectDatagram(b []byte) (*data.ObjectDatagram, int, error){
	parsed := 0
	typId, n, err := quicvarint.Parse(b)
	parsed += n

	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Type ID: %w", err)
	}
	b = b[n:]

	// Create ObjectDatagramType from the parsed TypeID
	dtype, err := data.NewObjectDatagramType(typId)
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

	// Construct the ObjectDatagram
	dg := &data.ObjectDatagram{
		Dtype:      *dtype,
		TrackAlias: trackAlias,
		Location:   location,
	}

	// Decode Publisher Priority if present
	var publisherPriority uint8
	if dtype.PriorityPresent {
		if len(b) < 1 {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: insufficient bytes for Publisher Priority")
		}
		publisherPriority = b[0]
		parsed += 1
		b = b[1:]

		dg.PublisherPriority = gonull.NewNullable(publisherPriority)
	}

	// Decode Extensions if present
	var extensions []model.MoqtKeyValuePair
	if dtype.ExtensionsPresent {
		// Protocol Violation check:
		// "If an endpoint receives a datagram with Extensions Present as "Yes" and a
		// Extension Headers Length of 0, it MUST close the session with a PROTOCOL_VIOLATION."
		extLen, _, err := quicvarint.Parse(b)
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to peek Extension Headers Length: %w", err)
		}
		if extLen == 0 {
			return nil, parsed, model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: model.NewReasonPhrase("OBJECT_DATAGRAM with Extensions Present MUST NOT have 0 Extension Headers Length"),
			}
		}

		ext, n, err := DecodeExtensions(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeObjectDatagram: failed to parse Extensions: %w", err)
		}
		extensions = ext
		b = b[n:]
		
		dg.Extensions = gonull.NewNullable(extensions)
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

		if status != model.Normal && dtype.ExtensionsPresent {
			return nil, parsed, model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: model.NewReasonPhrase("OBJECT_DATAGRAM with status other than Normal MUST NOT have extension headers"),
			}
		}

		dg.Status = gonull.NewNullable(status)
	} else {
		// The rest of the buffer is the payload
		payload = make([]byte, len(b))
		copy(payload, b)
		parsed += len(b)
		b = b[len(b):] // Consume all remaining bytes

		dg.Payload = gonull.NewNullable(payload)
	}

	return dg, parsed, nil	
}

// SUBGROUP_HEADER {
//   Type (i) = 0x10..0x1D,
//   Track Alias (i),
//   Group ID (i),
//   [Subgroup ID (i),]
//   [Publisher Priority (8),]
// }

func DecodeSubgroupHeader(b []byte) (*data.SubGroupHeader, int, error){
	parsed := 0
	typId, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: failed to parse Type ID: %w", err)
	}
	b = b[n:]

	sgType, err := data.NewSubGroupHeaderType(typId)
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: invalid Type ID %d: %w", typId, err)
	}

	// Decode Track Alias
	trackAlias, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: failed to parse Track Alias: %w", err)
	}
	b = b[n:]

	// Decode Group ID
	groupId, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: failed to parse Group ID: %w", err)
	}
	b = b[n:]

	sgh := &data.SubGroupHeader{
		SGType:     *sgType,
		TrackAlias: trackAlias,
		GroupId:    groupId,
	}

	// Decode Subgroup ID if present
	if sgType.SGIDMode == data.SubgroupIdModePresent {
		subgroupId, n, err := quicvarint.Parse(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: failed to parse Subgroup ID: %w", err)
		}
		b = b[n:]
		sgh.SubgroupId = gonull.NewNullable(subgroupId)
	}

	// Decode Publisher Priority if present
	if sgType.PriorityPresent {
		if len(b) < 1 {
			return nil, parsed, fmt.Errorf("DecodeSubgroupHeader: insufficient bytes for Publisher Priority")
		}
		publisherPriority := b[0]
		parsed += 1
		b = b[1:]
		sgh.PublisherPriority = gonull.NewNullable(publisherPriority)
	}

	return sgh, parsed, nil
}

// Subgroup Object{
//   Object ID Delta (i),
//   [Extensions (..),]
//   Object Payload Length (i),
//   [Object Status (i),]
//   [Object Payload (..),]
// }

func DecodeSubgroupObject(b []byte, extensionsPresent bool) (*data.SubgroupObject, int, error) {
	parsed := 0
	objectIdDelta, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupObject: failed to parse Object ID Delta: %w", err)
	}
	b = b[n:]

	var extensions []model.MoqtKeyValuePair
	if extensionsPresent {
		ext, n, err := DecodeExtensions(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeSubgroupObject: failed to parse Extensions: %w", err)
		}
		extensions = ext
		b = b[n:]
	}

	payloadLen, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupObject: failed to parse Object Payload Length: %w", err)
	}
	b = b[n:]

	var status model.MoqtObjectStatus
	var payload []byte
	if payloadLen == 0 {
		// If payload length is zero, Object Status is present
		s, n, err := quicvarint.Parse(b)
		parsed += n
		if err != nil {
			return nil, parsed, fmt.Errorf("DecodeSubgroupObject: failed to parse Object Status: %w", err)
		}
		status = model.MoqtObjectStatus(s)
		// b = b[n:] // Not strictly needed as we return
	} else {
		if uint64(len(b)) < payloadLen {
			return nil, parsed, fmt.Errorf("DecodeSubgroupObject: insufficient bytes for Payload, expected %d, got %d", payloadLen, len(b))
		}
		payload = make([]byte, payloadLen)
		copy(payload, b[:payloadLen])
		parsed += int(payloadLen)
		// b = b[payloadLen:]
	}

	// Construct SubgroupObject using options for consistency
	var opts []data.SubgroupObjectOption
	if extensionsPresent {
		opts = append(opts, data.SOWithExtensions(extensions))
	}
	if payloadLen == 0 {
		opts = append(opts, data.SOWithStatus(status))
	} else {
		opts = append(opts, data.SOWithPayload(payload))
	}

	sgo, err := data.NewSubgroupObject(objectIdDelta, opts...)
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeSubgroupObject: failed to create SubgroupObject: %w", err)
	}

	return sgo, parsed, nil
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

func DecodeTrackNamespace(b []byte) (model.MoqtTrackNamespace, int, error) {
	// Decode length
	parsed := 0
	noFields, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return model.MoqtTrackNamespace{}, parsed, fmt.Errorf("DecodeTrackNamespace: failed to parse Number of Track Namespace Fields: %w", err)
	}
	b = b[n:]

	// Decode each Track Namespace Field
	trackNamespaceFields := make([]string, noFields)
	for i := uint64(0); i < noFields; i++ {
		fieldLen, n, err := quicvarint.Parse(b)
		parsed += n
		if err != nil {
			return model.MoqtTrackNamespace{}, parsed, fmt.Errorf("DecodeTrackNamespace: failed to parse Track Namespace Field Length for field %d: %w", i, err)
		}
		b = b[n:]

		if len(b) < int(fieldLen) {
			return model.MoqtTrackNamespace{}, parsed, fmt.Errorf("DecodeTrackNamespace: insufficient bytes for Track Namespace Field Value, expected %d, got %d", fieldLen, len(b))
		}

		fieldValue := string(b[:fieldLen])
		trackNamespaceFields[i] = fieldValue
		parsed += int(fieldLen)
		b = b[fieldLen:]
	}
	
	trackNamespace := make([][]byte, noFields)
	for i, field := range trackNamespaceFields {
		trackNamespace[i] = []byte(field)
	}
	
	if tn := model.MoqtTrackNamespace(trackNamespace); tn.IsValid(){
		return tn, parsed, nil
	} else {
		return model.MoqtTrackNamespace{}, parsed, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase("Decoded Track Namespace is invalid, due to number of fields"),
		}
	}
}

// FullTrackName {
//  Track Namespace (..),
//  Track Name Length (i),
//  Track Name (..)
//}

func DecodeMoqtFullTrackName(b []byte) (*model.MoqtFullTrackName, int, error) {
	parsed := 0

	// Decode Track Namespace
	trackNamespace, n, err := DecodeTrackNamespace(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeMoqtFullTrackName: failed to parse Track Namespace: %w", err)
	}
	b = b[n:]

	// Decode Track Name Length
	trackNameLen, n, err := quicvarint.Parse(b)
	parsed += n
	if err != nil {
		return nil, parsed, fmt.Errorf("DecodeMoqtFullTrackName: failed to parse Track Name Length: %w", err)
	}
	b = b[n:]

	// Decode Track Name
	if len(b) < int(trackNameLen) {
		return nil, parsed, fmt.Errorf("DecodeMoqtFullTrackName: insufficient bytes for Track Name, expected %d, got %d", trackNameLen, len(b))
	}
	trackName := make([]byte, trackNameLen)
	copy(trackName, b[:trackNameLen])
	parsed += int(trackNameLen)
	b = b[trackNameLen:]

	ftn := model.MoqtFullTrackName{
		Namespace: trackNamespace,
		Name: trackName,
	}

	if ftn.IsValid() {
		return &ftn, parsed, nil
	} else {
		return nil, parsed, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase("Decoded Full Track Name is invalid"),
		}
	}
}

// Reason Phrase {
//   Reason Phrase Length (i),
//   Reason Phrase Value (..)
// }

func DecodeMoqtReasonPhrase(b []byte) (model.MoqtReasonPhrase, int, error) {
	parsed := 0
	reasonPhraseLen, n, err := quicvarint.Parse(b)

	parsed += n
	if err != nil {
		return "", parsed, fmt.Errorf("DecodeMoqtReasonPhrase: failed to parse Reason Phrase Length: %w", err)
	}
	b = b[n:]

	if len(b) < int(reasonPhraseLen) {
		return "", parsed, fmt.Errorf("DecodeMoqtReasonPhrase: insufficient bytes for Reason Phrase Value, expected %d, got %d", reasonPhraseLen, len(b))
	}

	reasonPhrase := string(b[:reasonPhraseLen])
	parsed += int(reasonPhraseLen)

	return model.MoqtReasonPhrase(reasonPhrase), parsed, nil
}