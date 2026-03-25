package control

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"

	"github.com/quic-go/quic-go/quicvarint"
)

// -- Subscribe OK message section 9.10 -- //

// SUBSCRIBE_OK Message {
//   Type (i) = 0x4,
//   Length (16),
//   Request ID (i),
//   Track Alias (i),
//   Number of Parameters (i),
//   Parameters (..) ...
// }

type SubscribeOkMessage struct {
	RequestId  uint64 // The Request ID of the SUBSCRIBE this message is replying to
	TrackAlias uint64
	Parameters []model.MoqtKeyValuePair
}

func (m *SubscribeOkMessage) Type() ControlMessageType {
	return SUBSCRIBE_OK
}

func (m *SubscribeOkMessage) RequestID() (uint64, bool) {
	return 0, false
}

func (m *SubscribeOkMessage) Encode() ([]byte, error) {
	payloadBuf := make([]byte, 0)
	payloadBuf = quicvarint.Append(payloadBuf, m.RequestId)
	payloadBuf = quicvarint.Append(payloadBuf, m.TrackAlias)
	message.EncodeExtensions(&payloadBuf, m.Parameters)
	return payloadBuf, nil
}

func (m *SubscribeOkMessage) Decode(payload []byte) (int, error) {
	parsed := 0
	reqId, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.RequestId = reqId

	trackAlias, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.TrackAlias = trackAlias

	params, n, err := message.DecodeExtensions(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	m.Parameters = params

	return parsed, nil
}
