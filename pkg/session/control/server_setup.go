package control

import (
	"go-moq/pkg/message"
	"go-moq/pkg/model"
)

// --- Server Setup Message Section 9.3 -- //

// SERVER_SETUP Message {
//   Type (i) = 0x21,
//   Length (16),
//   Number of Parameters (i),
//   Setup Parameters (..) ...,
// }

type ServerSetupMessage struct {
	Parameters []model.MoqtKeyValuePair
}

func (ssm *ServerSetupMessage) Type() ControlMessageType {
	return SERVER_SETUP // Type value of Client setup message in Section 9 table 1
}

func (ssm *ServerSetupMessage) Encode() ([]byte, error) {
	payloadBuf := make([]byte, 0)
	message.EncodeExtensions(&payloadBuf, ssm.Parameters) // EncodeExtensions function already encodes "Number of parameters" or "extensions len" as varint into the buffer

	return payloadBuf, nil
}

func (ssm *ServerSetupMessage) Decode(payload []byte) (int, error) {
	params, readN, err := message.DecodeExtensions(payload)
	ssm.Parameters = params

	return readN, err
}