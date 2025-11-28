package model

import "fmt"

const maxMoqtKeyValuePairValueLength = 65535

// MoqtKeyValuePair represents the flexible Key-Value-Pair structure.
// The interpretation of Length and Value depends on the Type.
type MoqtKeyValuePair struct {
	// Type (i): An unsigned integer (varint), identifying the type/key
	// and determining the subsequent serialization (even/odd).
	Type uint64

	// Length (i): Only present on the wire if Type is odd.
	// It specifies the length of the Value field in bytes.
	// The maximum allowed length is 2^16 - 1 bytes (65535).
	Length uint16

	// Value (..): The actual data.
	// If Type is even, Value is a single varint (uint64).
	// If Type is odd, Value is a sequence of Length bytes ([]byte).

	// Value is either []byte or int64
	Value any
}

func NewMoqtKeyValuePair(typeID uint64, value any) (MoqtKeyValuePair, error) {

	switch t := value.(type) {
	case []byte:
		if typeID%2 == 0 {
			return MoqtKeyValuePair{}, MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
				ReasonPhrase: NewReasonPhrase("For even Type IDs, Value must be of type int64"),
			}
		}

		len := len(t)
		if len > maxMoqtKeyValuePairValueLength {
			return MoqtKeyValuePair{}, MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: NewReasonPhrase("Value length must not exceed 65535 bytes when it is []byte"),
			}
		}

		return MoqtKeyValuePair{
			Type:   typeID,
			Length: uint16(len),
			Value:  t,
		}, nil

	case int64:
		if typeID%2 == 1 {
			return MoqtKeyValuePair{}, MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode: MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
				ReasonPhrase: NewReasonPhrase("For odd Type IDs, Value must be of type []byte"),
			}
		}

		return MoqtKeyValuePair{
			Type:  typeID,
			Value: t,
		}, nil

	default:
		return MoqtKeyValuePair{}, MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode: MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
			ReasonPhrase: NewReasonPhrase(fmt.Sprintf("Value must be of type []byte or int64, got %T", t)),
		}
	}
}

// NOTE on implementation:
// The parser functions would use the 'Type' field to decide
// whether to read the 'Length' field next, and how to interpret the 'Value' bytes.
// - If Type is even (e.g., 0x02), Value is interpreted as a single uint64.
// - If Type is odd (e.g., 0x01), Length is read, and Value is read as 'Length' bytes.
