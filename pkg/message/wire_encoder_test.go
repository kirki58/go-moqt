package message

import (
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
			expected: []byte{0x04 , 0xc0, 0x00, 0x02, 0x05, 0xa9, 0x0f, 0x51, 0xf3},
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
			expected: []byte{0x05, 0x0b ,0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}, // type,len,utf-8 encoding for "hello world"
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
