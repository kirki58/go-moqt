package control

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"

	"github.com/quic-go/quic-go/quicvarint"
)

// -- Subscribe message section 9.9 -- //

// SUBSCRIBE Message {
//   Type (i) = 0x3,
//   Length (16),
//   Request ID (i),
//   Track Namespace (..),
//   Track Name Length (i),
//   Track Name (..),
//   Number of Parameters (i),
//   Parameters (..) ...
// }

// Request ID increments by 2 with each FETCH, SUBSCRIBE, REQUEST_UPDATE, SUBSCRIBE_NAMESPACE, PUBLISH, PUBLISH_NAMESPACE or TRACK_STATUS request.
// Other messages with a Request ID field reference the Request ID of another message for correlation

type SubscribeMessage struct{
	RequestId uint64
	FullTrackName *model.MoqtFullTrackName
	Parameters []model.MoqtKeyValuePair
}

func (sm *SubscribeMessage) Type() ControlMessageType {
	return SUBSCRIBE
}

func (sm *SubscribeMessage) Encode() ([]byte, error) {
	payloadBuf := make([]byte, 0)
	// 1. Encode the request id
	payloadBuf = quicvarint.Append(payloadBuf, sm.RequestId)
	// 2. Encode the Full Track Name
	// Track Namespace (..),
	// Track Name Length (i),
	// Track Name (..)
	message.EncodeMoqtFullTrackName(&payloadBuf, *sm.FullTrackName)
	// 3. Encode the parameters
	// Number of Parameters (i),
  	// Parameters (..) ...
	message.EncodeExtensions(&payloadBuf, sm.Parameters)

	return payloadBuf, nil
}

func (sm *SubscribeMessage) Decode(payload []byte) (int, error) {
	parsed := 0
	// Decode Request id
	reqId, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	sm.RequestId = reqId

	// Decode Full Track Name
	ftn, n, err := message.DecodeMoqtFullTrackName(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	sm.FullTrackName = ftn

	// Decode Parameters
	params, n, err := message.DecodeExtensions(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	sm.Parameters = params

	return parsed, nil	
}