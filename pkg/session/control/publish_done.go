package control

import (
	"fmt"
	"go-moq/pkg/message"
	"go-moq/pkg/model"

	"github.com/quic-go/quic-go/quicvarint"
)

const (
	PublishDoneInternalError     = 0x0
	PublishDoneUnauthorized      = 0x1
	PublishDoneTrackEnded        = 0x2
	PublishDoneSubscriptionEnded = 0x3
	PublishDoneGoingAway         = 0x4
	PublishDoneExpired           = 0x5
	PublishDoneTooFarBehind      = 0x6
	PublishDoneMalformedTrack    = 0x7
	PublishDoneUpdateFailed      = 0x8
)

func IsPublishDoneStatusCode(i uint64) bool {
	switch i {
	case PublishDoneInternalError, PublishDoneUnauthorized, PublishDoneTrackEnded,
		PublishDoneSubscriptionEnded, PublishDoneGoingAway, PublishDoneExpired, PublishDoneTooFarBehind,
		PublishDoneMalformedTrack, PublishDoneUpdateFailed:
		return true
	default:
		return false
	}

}

// TODO: Implement the message
type PublishDoneMessage struct {
	RequestId   uint64 // The Request ID of the SUBSCRIBE being terminated
	StatusCode  uint64
	StreamCount uint64
	ErrorReason model.MoqtReasonPhrase
}

func (m *PublishDoneMessage) Type() ControlMessageType {
	return PUBLISH_DONE
}

func (m *PublishDoneMessage) RequestID() (uint64, bool) {
	return 0, false
}

func (m *PublishDoneMessage) Encode() ([]byte, error) {
	if !IsPublishDoneStatusCode(m.StatusCode){
		return nil, fmt.Errorf("invalid status code: %d", m.StatusCode)
	}

	payloadBuf := make([]byte, 0)
	payloadBuf = quicvarint.Append(payloadBuf, m.RequestId)
	payloadBuf = quicvarint.Append(payloadBuf, m.StreamCount)
	message.EncodeMoqtReasonPhrase(&payloadBuf, m.ErrorReason)
	return payloadBuf, nil
}

func (m *PublishDoneMessage) Decode(payload []byte) (int, error) {
	parsed := 0
	reqId, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.RequestId = reqId

	streamCount, n, err := quicvarint.Parse(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	payload = payload[n:]
	m.StreamCount = streamCount

	reason, n, err := message.DecodeMoqtReasonPhrase(payload)
	parsed += n
	if err != nil {
		return parsed, err
	}
	m.ErrorReason = reason

	return parsed, nil
}
