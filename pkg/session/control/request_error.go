package control

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"

	"github.com/quic-go/quic-go/quicvarint"
)

// REQUEST_ERROR Message {
//   Type (i) = 0x5,
//   Length (16),
//   Request ID (i),
//   Error Code (i),
//   Error Reason (Reason Phrase),
// }

type RequestErrorMessage struct{
	RequestId uint64
	ErrorCode model.MOQT_REQUEST_ERROR_CODE
	ErrorReason model.MoqtReasonPhrase
}

func (m *RequestErrorMessage) Type() ControlMessageType {
	return REQUEST_ERROR
}

func (m *RequestErrorMessage) Encode() ([]byte, error) {
	payloadBuf := make([]byte, 0)
	payloadBuf = quicvarint.Append(payloadBuf, m.RequestId)
	payloadBuf = quicvarint.Append(payloadBuf, uint64(m.ErrorCode))
	message.EncodeMoqtReasonPhrase(&payloadBuf, m.ErrorReason)
	return payloadBuf, nil
}

func (m *RequestErrorMessage) Decode(payload []byte) (int, error) {
	parsed := 0
	
	// Decode Request ID
	reqId, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.RequestId = reqId

	// Decode Error Code
	errCode, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.ErrorCode = model.MOQT_REQUEST_ERROR_CODE(errCode)

	// Decode Error Reason
	reason, n, err := message.DecodeMoqtReasonPhrase(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	m.ErrorReason = reason

	return parsed, nil
}