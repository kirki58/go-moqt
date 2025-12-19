package control

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"
)

// --- Client Setup Message Section 9.3 -- //

// CLIENT_SETUP Message {
//   Type (i) = 0x20,
//   Length (16),
//   Number of Parameters (i),
//   Setup Parameters (..) ...,
// }

type ClientSetupMessage struct {
	Parameters []model.MoqtKeyValuePair
}

func (csm *ClientSetupMessage) Type() ControlMessageType {
	return CLIENT_SETUP // Type value of Client setup message in Section 9 table 1
}

func (csm *ClientSetupMessage) Encode() ([]byte, error) {
	payloadBuf := make([]byte, 0)
	message.EncodeExtensions(&payloadBuf, csm.Parameters) // EncodeExtensions function already encodes "Number of parameters" or "extensions len" as varint into the buffer

	return payloadBuf, nil
}

func (csm *ClientSetupMessage) Decode(payload []byte) (int, error) {
	params, readN, err := message.DecodeExtensions(payload)
	csm.Parameters = params

	return readN, err
}