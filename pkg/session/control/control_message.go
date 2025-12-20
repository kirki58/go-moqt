package control

import (
	"bufio"
	"sync"

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

// 1. Why you need bufio
// For Reading (Critical)

// You are using quicvarint.Read().

//     The quicvarint library relies on the io.ByteReader interface (specifically the ReadByte() method) to efficiently parse variable-length integers one byte at a time.

//     A raw quic.Stream or net.Conn usually does not implement ReadByte.

//     If you don't wrap it in bufio.NewReader, the parser has to do a full generic Read() system call for every single byte of the integer. This is incredibly slow.

//     Verdict: bufio.Reader is effectively mandatory here for performance.

// For Writing (Optimization)

// Your WriteControlMessage function makes three distinct write calls:

//     Message Type (Varint) — tiny (1-8 bytes)

//     Message Length (Varint) — tiny (1-8 bytes)

//     Payload — variable

// Without bufio.Writer, these might trigger three separate write operations to the underlying QUIC stream. While QUIC stacks are smart, buffering these into a single chunk before handing them to the QUIC layer is much more efficient and ensures the packet is packed tightly.

//     Verdict: Highly recommended, but you must remember to call Flush().

type ControlMessageFactory struct {
	r *bufio.Reader // Stream reader
	w *bufio.Writer // Stream writer

	writeLock sync.Mutex
}

func NewControlMessageFactory(rw io.ReadWriter) *ControlMessageFactory {
	return &ControlMessageFactory{
		r: bufio.NewReader(rw),
		w: bufio.NewWriter(rw),
	}
}

func (cmf *ControlMessageFactory) ReadControlMessage() (ControlMessage, error) {
	// Read the control message header first
	msgType, err := quicvarint.Read(cmf.r)
	if err != nil { // Read message type value
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Read stream failed while reading control message type:\n\t %w", err)
	}

	// Read message length, Varint
	msgLength, err := quicvarint.Read(cmf.r)
	if err != nil {
		return nil, fmt.Errorf("ControlMessageFactory.ReadControlMessage():\n\t Read stream failed while reading control message length:\n\t %w", err)
	}
	// NOTE: Message length is a Varint in Draft-15.

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

	case uint64(SERVER_SETUP):
		msg = &ServerSetupMessage{}
	
	// TODO: More control messages as we go
	default:
		return nil, model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Unsupported Control Message Type: %#X", msgType)),
		}
	}

	// Decode the message payload and populate the message struct (Decode implementation should automatically populate)
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

// WriteControlMessage encodes and writes a message.
// It is safe for concurrent use.
func (cmf *ControlMessageFactory) WriteControlMessage(msg ControlMessage) error {
    cmf.writeLock.Lock()
    defer cmf.writeLock.Unlock()

    // 1. Encode Payload
    payload, err := msg.Encode()
    if err != nil {
        return fmt.Errorf("encode failed: %w", err)
    }

    // 2. Write Header (Type + Length) directly to buffer
    // Note: quicvarint.Append is great, but we can also use quicvarint.Write 
    // if we want to write directly to our bufio.Writer piece by piece.
    
    // We use a small intermediate buffer for varints to ensure we don't fail halfway through writing a header
    header := make([]byte, 0, 16)
    header = quicvarint.Append(header, uint64(msg.Type()))
    header = quicvarint.Append(header, uint64(len(payload)))

    if _, err := cmf.w.Write(header); err != nil {
        return fmt.Errorf("write header failed: %w", err)
    }

    // 3. Write Payload
    if len(payload) > 0 {
        if _, err := cmf.w.Write(payload); err != nil {
            return fmt.Errorf("write payload failed: %w", err)
        }
    }

    // 4. FLUSH is mandatory when using bufio.Writer
    if err := cmf.w.Flush(); err != nil {
        return fmt.Errorf("flush failed: %w", err)
    }

    return nil
}
