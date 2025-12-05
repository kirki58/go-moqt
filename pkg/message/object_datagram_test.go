package message

import (
	"reflect"
	"testing"
)

func TestNewObjectDatagramType(t *testing.T) {
	tests := []struct {
		name      string
		typeID    uint64
		expected  *ObjectDatagramType
		expectErr bool
	}{
		// Valid type range (inclusive):
		// [0x00 - 0x0F] + [0x20, 0x21, 0x24, 0x25, 0x28, 0x29]
		{
			name: "Out of range - TypeId uses reserved bit 0x10",
			typeID: 0x10,
			expected: nil,
			expectErr: true,
		},
		{
			name: "Out of range - TypeId uses reserved bit, value 0x1F",
			typeID: 0x1F,
			expected: nil,
			expectErr: true,
		},
		{
			name: "Out of range - Status-EndOfGroup Conflict 0x22",
			typeID: 0x22,
			expected: nil,
			expectErr: true,
		},
		{
			name: "Out of range - Status-EndOfGroup Conflict 0x26",
			typeID: 0x26,
			expected: nil,
			expectErr: true,
		},
		{
			name: "Out of range - Exceeds defined range by value 0x30",
			typeID: 0x30,
			expected: nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObjectDatagramType(tt.typeID)

			if tt.expectErr {
				if err == nil {
					t.Errorf("NewObjectDatagramType() expected error, got err = nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewObjectDatagramType() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("NewObjectDatagramType() got = %+v, want %+v", got, tt.expected)
				}
			}
		})
	}
}

func TestToUInt64(t *testing.T) {
	tests := []struct {
		name     string
		dt       ObjectDatagramType
		expected uint64
	}{
		{
			name: "0x00 --> ObjectId present, Priority Present, With Payload",
			dt: ObjectDatagramType{
				ObjectIdPresent: true,
				PriorityPresent: true,
			},
			expected: 0x00,
		},
		{
			name: "0x07 --> ObjectId Omitted, Rest present, With Payload",
			dt: ObjectDatagramType{
				EndOfGroup:        true,
				ExtensionsPresent: true,
				PriorityPresent:   true,
			},
			expected: 0x07,
		},
		{
			name: "0x20 --> ObjectId present, Priority Present, With Status",
			dt: ObjectDatagramType{
				ObjectIdPresent: true,
				PriorityPresent: true,
				StatusOrPayload: true,
			},
			expected: 0x20,
		},
		{
			name: "0x25 --> Priority present, Extensions present, With Status",
			dt: ObjectDatagramType{
				PriorityPresent:   true,
				ExtensionsPresent: true,
				StatusOrPayload:   true,
			},
			expected: 0x25,
		},
		{
			name: "0x08 --> ObjectId present, With Payload",
			dt: ObjectDatagramType{
				ObjectIdPresent: true,
			},
			expected: 0x08,
		},
		{
			name: "0x0F --> EndOfGroup, Extensions Present, With Payload",
			dt: ObjectDatagramType{
				EndOfGroup:        true,
				ExtensionsPresent: true,
			},
			expected: 0x0F,
		},
		{
			name: "0x28 --> ObjectId present, With Status",
			dt: ObjectDatagramType{
				ObjectIdPresent: true,
				StatusOrPayload: true,
			},
			expected: 0x28,
		},
		{
			name: "0x29 --> ObjectId present, Extension present, With status",
			dt: ObjectDatagramType{
				ObjectIdPresent:   true,
				ExtensionsPresent: true,
				StatusOrPayload:   true,
			},
			expected: 0x29,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dt.ToUInt64()
			if got != tt.expected {
				t.Errorf("ToUInt64() got = 0x%x, want 0x%x", got, tt.expected)
			}
		})
	}
}
