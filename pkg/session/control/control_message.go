package control

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"go-moq/pkg/model"
	"io"

	"github.com/quic-go/quic-go/quicvarint"
)

type ControlMessageType uint64

const (
	// Control Message Type values as defined in Section 9 Table 1
	// Reserved messages (0x01, 0x40, 0x41) are treated as unsupported as of this version.

	CLIENT_SETUP             ControlMessageType = 0x20
	SERVER_SETUP             ControlMessageType = 0x21
	GOAWAY                   ControlMessageType = 0x10
	MAX_REQUEST_ID           ControlMessageType = 0x15
	REQUESTS_BLOCKED         ControlMessageType = 0x1A
	REQUEST_OK               ControlMessageType = 0x7
	REQUEST_ERROR            ControlMessageType = 0x5
	SUBSCRIBE                ControlMessageType = 0x3
	SUBSCRIBE_OK             ControlMessageType = 0x4
	REQUEST_UPDATE           ControlMessageType = 0x2
	UNSUBSCRIBE              ControlMessageType = 0xA
	PUBLISH                  ControlMessageType = 0x1D
	PUBLISH_OK               ControlMessageType = 0x1E
	PUBLISH_DONE             ControlMessageType = 0xB
	FETCH                    ControlMessageType = 0x16
	FETCH_OK                 ControlMessageType = 0x18
	FETCH_CANCEL             ControlMessageType = 0x17
	TRACK_STATUS             ControlMessageType = 0xD
	PUBLISH_NAMESPACE        ControlMessageType = 0x6
	NAMESPACE                ControlMessageType = 0x8
	PUBLISH_NAMESPACE_DONE   ControlMessageType = 0x9
	NAMESPACE_DONE           ControlMessageType = 0xE
	PUBLISH_NAMESPACE_CANCEL ControlMessageType = 0xC
	SUBSCRIBE_NAMESPACE      ControlMessageType = 0x11
)

type ControlMessage interface {
	Type() ControlMessageType // Returns the type value of the specific control message struct implementing this interface

	Encode() ([]byte, error) // Serializes the control message PAYLOAD into the wire, since header type is the same for all control messages, a wrapper should handle it's encoding (for the sake of clean code)

	Decode(payload []byte) (int, error) // Deserializes the control message PAYLOAD from the wire, Populates the ControlMessage with payload, again header deserialization should be handled by a wrapper
}

type ControlMessageFactory struct {
	r *bufio.Reader // Stream reader
}

func (cmf *ControlMessageFactory) ReadControlMessage() (ControlMessage, error) {
	// Read the control message header first
	msgType, err := quicvarint.Read(cmf.r)
	if err != nil { // Read message type value
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Read stream failed while reading control message type:\n\t %w", err)
	}

	// Read message length, 16-bit fixed
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(cmf.r, lenBuf); err != nil {
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Read stream failed while reading control message length:\n\t %w", err)
	}
	var msgLength uint16 = binary.BigEndian.Uint16(lenBuf) // Convert byte slice len to uint16 len
	// NOTE: Message length for every control message is globally defined as a 16-bit unsigned integer, making the maximum byte-length of a control message 65,535.

	

	// Read the payload, differently depending on the type of the control message
	payload := make([]byte, msgLength)
	if _, err := io.ReadFull(cmf.r, payload); err != nil {
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Read stream failed while reading control message payload:\n\t %w", err)
	}

	// Finally switch message type and factorize the final client message struct
	var msg ControlMessage
	switch msgType {
	case uint64(CLIENT_SETUP):
		msg = &ClientSetupMessage{}

	// ... TODO: Implement other control messages
	default:
		return nil, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Unsupported Control Message Type: %#X", msgType)),
		}
	}

	// Decode the message payload and populate the message struct
	decodedBytes, err := msg.Decode(payload)

	// Cite Section 9:
	// If the length does not match the length of the Message Payload, the receiver MUST close the session with a PROTOCOL_VIOLATION.
	if err != nil {
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Decoding control message payload failed:\n\t %w", err)
	}
	if decodedBytes != int(msgLength) {
		return nil, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Control Message payload length mismatch. Expected: %d, Got: %d", msgLength, decodedBytes)),
		}
	}

	return msg, nil
}
