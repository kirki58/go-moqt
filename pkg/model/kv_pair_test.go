package model

import (
	"reflect"
	"testing"
)

func TestNewMoqtKeyValuePair(t *testing.T) {
	tests := []struct {
		name      string
		typeID    uint64
		value     any
		expected  MoqtKeyValuePair
		expectErr bool
		errCode   MOQT_SESSION_TERMINATION_ERROR_CODE
	}{
		{
			name:      "Even TypeID with int64 value",
			typeID:    0x02,
			value:     int64(123),
			expected:  MoqtKeyValuePair{Type: 0x02, Value: int64(123)},
			expectErr: false,
		},
		{
			name:      "Odd TypeID with []byte value",
			typeID:    0x01,
			value:     []byte("hello"),
			expected:  MoqtKeyValuePair{Type: 0x01, Length: 5, Value: []byte("hello")},
			expectErr: false,
		},
		{
			name:      "Even TypeID with []byte value - error",
			typeID:    0x02,
			value:     []byte("world"),
			expectErr: true,
			errCode:   MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
		},
		{
			name:      "Odd TypeID with int64 value - error",
			typeID:    0x01,
			value:     int64(456),
			expectErr: true,
			errCode:   MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
		},
		{
			name:      "[]byte value exceeding max length - error",
			typeID:    0x01,
			value:     make([]byte, maxMoqtKeyValuePairValueLength+1),
			expectErr: true,
			errCode:   MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
		},
		{
			name:      "Unsupported value type - error",
			typeID:    0x02,
			value:     "string",
			expectErr: true,
			errCode:   MOQT_SESSION_TERMINATION_ERROR_CODE_KEY_VALUE_FORMATTING_ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMoqtKeyValuePair(tt.typeID, tt.value)

			if tt.expectErr {
				if err == nil {
					t.Errorf("NewMoqtKeyValuePair() expected error, got nil")
				}
				moqtErr, ok := err.(MOQT_SESSION_TERMINATION_ERROR)
				if !ok {
					t.Errorf("NewMoqtKeyValuePair() expected MOQT_SESSION_TERMINATION_ERROR, got %T", err)
				} else if moqtErr.ErrorCode != tt.errCode {
					t.Errorf("NewMoqtKeyValuePair() expected error code %v, got %v", tt.errCode, moqtErr.ErrorCode)
				}
			} else {
				if err != nil {
					t.Errorf("NewMoqtKeyValuePair() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("NewMoqtKeyValuePair() got = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}