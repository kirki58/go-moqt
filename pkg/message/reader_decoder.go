package message

import (
	"bufio"
	"fmt"
	"go-moq/pkg/model"
	"io"

	"github.com/LukaGiorgadze/gonull/v2"
	"github.com/quic-go/quic-go/quicvarint"
)

// Object Extension Headers are serialized as Key-Value-Pairs (see Figure 2), prefixed by the length of the serialized Key-Value-Pairs, in bytes

//		Extensions {
//	  Extension Headers Length (i),
//	  Extension headers (..),
//	}

// This is a version of message.DecodeExtensions that decodes objects from a buffered reader
func DecodeExtensionsFromReader(br *bufio.Reader) ([]model.MoqtKeyValuePair, error) {
	extensionsLen, err := quicvarint.Read(br)
	if err != nil {
		if err == io.EOF{
			return nil, err
		}
		return nil, fmt.Errorf("DecodeExtensions: failed to read Extension Headers Length: %w", err)
	}

	b := make([]byte, extensionsLen)
	if n, err := io.ReadFull(br, b); err != nil{
		if err == io.EOF{
			return nil, err
		}
		return nil, fmt.Errorf("DecodeExtensions: failed to read Extension Headers, expected %d bytes, got %d: %w", extensionsLen, n, err)
	} 

	var kvPairs []model.MoqtKeyValuePair
	var extBytesParsed uint64
	for extBytesParsed < extensionsLen {
		kvp, n, err := DecodeMoqtKeyValuePair(b)
		if err != nil {
			return nil, fmt.Errorf("DecodeExtensions: failed to parse Key-Value-Pair: %w", err)
		}
		kvPairs = append(kvPairs, kvp)
		b = b[n:]
		extBytesParsed += uint64(n)
	}

	if extBytesParsed != extensionsLen {
		return nil, fmt.Errorf("DecodeExtensions: parsed bytes %d does not match Extension Headers Length %d", extBytesParsed, extensionsLen)
	}
	return kvPairs, nil
}

// Subgroup Object{
//   Object ID Delta (i),
//   [Extensions (..),]
//   Object Payload Length (i),
//   [Object Status (i),]
//   [Object Payload (..),]
// }

// This is a version of message.DecodeSubgroupObject that decodes objects from a buffered reader (A QUIC stream)
func DecodeSubgroupObjectFromReader(br *bufio.Reader, subgroupHeaderType *model.SubGroupHeaderType) (*model.SubgroupObject, error) {
	objectIdDelta, err := quicvarint.Read(br) // First varint is objectid delta
	if err != nil {
		if err == io.EOF{
			return nil, err
		}
		return nil, fmt.Errorf("DecodeSubgroupObject: failed to read Object ID Delta: %w", err)
	}

	sgo := &model.SubgroupObject{
		ObjectIdDelta: objectIdDelta,
	}

	if subgroupHeaderType.ExtensionsPresent {
		ext, err := DecodeExtensionsFromReader(br)
		if err != nil {
			if err == io.EOF{
				return nil, err
			}
			return nil, fmt.Errorf("DecodeSubgroupObject: failed to read Extensions: %w", err)
		}

		sgo.Extensions = gonull.NewNullable(ext)

		// Note on why we considered 0-length extensions PROTOCOL_VIOLATION with datagrams and yet we don't do that for subgroup objects, The document states that:
		// For Type values where Extensions Present is No, the Extensions field is never present and all Objects have no extensions.
		// When Extensions Present is Yes, the Extensions structure defined in Section 10.2.1.2 is present in all Objects in this subgroup.
		// Objects with no extensions set Extension Headers Length to 0.
	}

	objPayloadLen, err := quicvarint.Read(br)
	if err != nil {
		if err == io.EOF{
			return nil, err
		}
		return nil, fmt.Errorf("DecodeSubgroupObject: failed to read Object Payload Length: %w", err)
	}

	if objPayloadLen == 0 { // we are looking at a status object
		statusRead, err := quicvarint.Read(br)
		if err != nil {
			if err == io.EOF{
				return nil, err
			}
			return nil, fmt.Errorf("DecodeSubgroupObject: failed to read Object Status: %w", err)
		}

		status := model.MoqtObjectStatus(statusRead)
		if !status.IsValid() {
			return nil, &model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Invalid Object Status: %d", statusRead)),
			}
		}
		if status != model.Normal && subgroupHeaderType.ExtensionsPresent && len(sgo.Extensions.Val) > 0 {
			return nil, &model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: model.NewReasonPhrase("Subgroup Object with status other than Normal MUST NOT have extension headers"),
			}
		}
		sgo.Status = gonull.NewNullable(status)
		return sgo, nil
	} else { // we are looking at a payload object
		payload := make([]byte, objPayloadLen)
		if n, err := io.ReadFull(br, payload); err != nil{
			if err == io.EOF{
				return nil, err
			}
			return nil, fmt.Errorf("DecodeSubgroupObject: failed to read Object Payload, expected %d bytes, got %d: %w", objPayloadLen, n, err)
		}

		sgo.Payload = gonull.NewNullable(payload)
		return sgo, nil
	}
}

// SUBGROUP_HEADER {
//   Type (i) = 0x10..0x1D,
//   Track Alias (i),
//   Group ID (i),
//   [Subgroup ID (i),]
//   [Publisher Priority (8),]
// }

func DecodeSubgroupHeaderFromReader(br *bufio.Reader) (*model.SubGroupHeader, error) {
	typId, err := quicvarint.Read(br)
	if err != nil {
		if err == io.EOF{
			return nil, err
		}
		return nil, fmt.Errorf("DecodeSubgroupHeader: failed to read Type ID: %w", err)
	}

	sgType, err := model.NewSubGroupHeaderType(typId)
	if err != nil {
		return nil, fmt.Errorf("DecodeSubgroupHeader: invalid Type ID %d: %w", typId, err)
	}

	// Decode Track Alias
	trackAlias, err := quicvarint.Read(br)
	if err != nil {
		return nil, fmt.Errorf("DecodeSubgroupHeader: failed to read Track Alias: %w", err)
	}

	// Decode Group ID
	groupId, err := quicvarint.Read(br)
	if err != nil {
		return nil, fmt.Errorf("DecodeSubgroupHeader: failed to read Group ID: %w", err)
	}

	sgh := &model.SubGroupHeader{
		SGType:     *sgType,
		TrackAlias: trackAlias,
		GroupId:    groupId,
	}

	// Decode Subgroup ID if present
	if sgType.SGIDMode == model.SubgroupIdModePresent {
		subgroupId, err := quicvarint.Read(br)
		if err != nil {
			return nil, fmt.Errorf("DecodeSubgroupHeader: failed to read Subgroup ID: %w", err)
		}
		sgh.SubgroupId = gonull.NewNullable(subgroupId)
	}

	// Decode Publisher Priority if present
	if sgType.PriorityPresent {
		var priority uint8
		b, err := br.ReadByte()
		if err != nil{
			return nil, fmt.Errorf("DecodeSubgroupHeader: failed to read Publisher Priority: %w", err)
		}
		priority = b
		sgh.PublisherPriority = gonull.NewNullable(priority)
	}

	return sgh, nil
}