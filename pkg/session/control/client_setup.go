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

// Note, [Cite: Section 9.3]: To ensure future extensibility of MOQT, endpoints MUST ignore unknown setup parameters.
// Define known setup parameters as enum

type SetupParameterType uint64

const (
	// Setup Paramater Type values as defined in Section 9.3.1
	PATH_SETUP_PARAM_TYPE                      SetupParameterType = 0x01 // 9.3.1.2
	MAX_REQUEST_ID_SETUP_PARAM_TYPE            SetupParameterType = 0x02 // 9.3.1.3
	AUTHORIZATION_TOKEN_SETUP_PARAM_TYPE       SetupParameterType = 0x03 // 9.3.1.5 and 9.2.1.1
	MAX_AUTH_TOKEN_CACHE_SIZE_SETUP_PARAM_TYPE SetupParameterType = 0x04 // 9.3.1.4
	AUTHORITY_SETUP_PARAM_TYPE                 SetupParameterType = 0x05 // 9.3.1.1
	MOQT_IMPLEMENTATION_PARAM_TYPE             SetupParameterType = 0x07 // 9.3.1.6
)
