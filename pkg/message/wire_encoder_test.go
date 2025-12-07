package message

import (
	"go-moq/internal"
	"go-moq/pkg/model"

	"reflect"
	"testing"
)

func TestEncodeMoqtLocation(t *testing.T) {
	tests := []struct {
		name     string
		location model.MoqtLocation
		expected []byte
	}{
		{
			name:     "Simple Location",
			location: model.MoqtLocation{GroupId: 1, ObjectId: 2},
			expected: []byte{0x01, 0x02},
		},
		{
			name:     "Larger GroupId and ObjectId",
			location: model.MoqtLocation{GroupId: 1234567, ObjectId: 99929992},
			expected: []byte{0x80, 0x12, 0xD6, 0x87, 0x85, 0xF4, 0xCF, 0x88},
		},
		{
			name:     "Zero GroupId and ObjectId",
			location: model.MoqtLocation{GroupId: 0, ObjectId: 0},
			expected: []byte{0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			EncodeMoqtLocation(&buf, tt.location)
			if !reflect.DeepEqual(buf, tt.expected) {
				t.Errorf("EncodeMoqtLocation() got = %v, want %v", buf, tt.expected)
			}
		})
	}
}

func TestEncodeMoqtKeyValuePair(t *testing.T) {
	// It's really important to create kvpairs with the appropriate NewMoqtKeyValuePair function.
	kvPair1, _ := model.NewMoqtKeyValuePair(0, 0)
	kvPair2, _ := model.NewMoqtKeyValuePair(2, uint64(5))
	kvPair3, _ := model.NewMoqtKeyValuePair(4, uint64(2223334445555))
	kvPair4, _ := model.NewMoqtKeyValuePair(1, []byte{})
	kvPair5, _ := model.NewMoqtKeyValuePair(3, []byte{0x01, 0x02, 0x03})
	kvPair6, _ := model.NewMoqtKeyValuePair(5, []byte("hello world"))
	kvPair7, _ := model.NewMoqtKeyValuePair(222222222, uint64(1))
	kvPair8, _ := model.NewMoqtKeyValuePair(333333333, []byte{0x01})

	tests := []struct {
		name     string
		kvPair   model.MoqtKeyValuePair
		expected []byte
	}{
		{
			name:     "Even Type with 0 uint64 Value",
			kvPair:   kvPair1,
			expected: []byte{0x00, 0x00}, // first byte correspondes to "Type" field
		},
		{
			name:     "Even Type with simple uint64 Value",
			kvPair:   kvPair2,
			expected: []byte{0x02, 0x05},
		},
		{
			name:     "Even Type with Large uint64 Value",
			kvPair:   kvPair3,
			expected: []byte{0x04, 0xc0, 0x00, 0x02, 0x05, 0xa9, 0x0f, 0x51, 0xf3},
		},
		{
			name:     "Odd Type with empty []byte Value",
			kvPair:   kvPair4,
			expected: []byte{0x01, 0x00}, // type, len (0)
		},
		{
			name:     "Odd Type with simple []byte Value",
			kvPair:   kvPair5,
			expected: []byte{0x03, 0x03, 0x01, 0x02, 0x03}, // type, len, 1, 2, 3
		},
		{
			name:     "Odd Type with longer []byte Value",
			kvPair:   kvPair6,
			expected: []byte{0x05, 0x0b, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}, // type,len,utf-8 encoding for "hello world"
		},
		{
			name:     "Even multi-byte long type value",
			kvPair:   kvPair7,
			expected: []byte{0x8d, 0x3e, 0xd7, 0x8e, 0x01},
		},
		{
			name:     "Odd multi-byte long type value",
			kvPair:   kvPair8,
			expected: []byte{0x93, 0xde, 0x43, 0x55, 0x01, 0x01}, // typeid, 0x01(len), 0x01(elements)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			EncodeMoqtKeyValuePair(&buf, tt.kvPair)
			if !reflect.DeepEqual(buf, tt.expected) {
				t.Errorf("EncodeMoqtKeyValuePair() got = %v, want %v", buf, tt.expected)
			}
		})
	}
}

func TestEncodeExtensions(t *testing.T) {
	tests := []struct {
		name     string
		kvPairs  []model.MoqtKeyValuePair
		expected []byte
	}{
		{
			name:     "0 Extensions (empty slice)",
			kvPairs:  []model.MoqtKeyValuePair{},
			expected: []byte{0x00}, // Only the length is encoded
		},
		{
			name: "1 Extension",
			kvPairs: []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(2, uint64(10))),
			},
			expected: []byte{0x01, 0x02, 0x0A}, // length (1), type(2), value(10)
		},
		{
			name: "5 Extensions",
			kvPairs: []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(0, uint64(1))),
				internal.Must(model.NewMoqtKeyValuePair(1, []byte{0x01, 0x02})),
				internal.Must(model.NewMoqtKeyValuePair(2, uint64(3))),
				internal.Must(model.NewMoqtKeyValuePair(3, []byte{0x04, 0x05, 0x06})),
				internal.Must(model.NewMoqtKeyValuePair(4, uint64(5))),
			},
			expected: []byte{
				0x05,       // Length of extensions (5)
				0x00, 0x01, // KV1: Type 0, Value 1
				0x01, 0x02, 0x01, 0x02, // KV2: Type 1, Length 2, Value [0x01, 0x02]
				0x02, 0x03, // KV3: Type 2, Value 3
				0x03, 0x03, 0x04, 0x05, 0x06, // KV4: Type 3, Length 3, Value [0x04, 0x05, 0x06]
				0x04, 0x05, // KV5: Type 4, Value 5
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			EncodeExtensions(&buf, tt.kvPairs)
			if !reflect.DeepEqual(buf, tt.expected) {
				t.Errorf("EncodeExtensions() got = %v, want %v", buf, tt.expected)
			}
		})
	}
}

func TestEncodeObjectDatagram(t *testing.T) {
	tests := []struct {
		name     string
		datagram *ObjectDatagram
		expected []byte
	}{
		{
			name: "0x03 ObjectDatagram",
			datagram: internal.Must(NewObjectDatagram(1, 2,
				WithObjectId(12),
				WithPublisherPriority(12),
				WithExtensions(
					[]model.MoqtKeyValuePair{
						internal.Must(model.NewMoqtKeyValuePair(10, uint64(11))),
						internal.Must(model.NewMoqtKeyValuePair(21, []byte{0x00, 0x01})),
					},
				),
				WithEndOfGroup(),
				WithPayload(
					[]byte{
						0x21, 0x22, 0x22, 0xFF,
					},
				),
			)),
			expected: []byte{
				0x03, // TypeId of the datagram
				0x01, // Track Alias

				// Location - Start
				0x02, // GroupId
				0x0C, // ObjectId
				// Location - End

				0x0C, // Publisher Priority

				// Extensions - Start
				0x02, // Length of extensions
				// Extension 1
				0x0A, 0x0B,
				// Extensions 2
				0x15, 0x02, 0x00, 0x01, // Type = 21, Len = 2, Value = [0, 1]
				// Extensions - End

				// Status field omitted entirely because of datagram's type (0x03)

				// Dump the rest with the paylaod as is
				0x21, 0x22, 0x22, 0xFF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			EncodeObjectDatagram(&buf, tt.datagram)
			if !reflect.DeepEqual(buf, tt.expected) {
				t.Errorf("EncodeObjectDatagram() got = %v, want %v", buf, tt.expected)
			}
		})
	}
}
