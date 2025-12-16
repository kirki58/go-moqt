package message

import (
	"go-moq/internal"
	"go-moq/pkg/model"

	"reflect"
	"testing"
)

func TestDecodeMoqtLocation(t *testing.T){
	tests := []struct {
		name string
		buf []byte
		expectedLoc model.MoqtLocation
		expectedN int
		expectErr bool
	}{
		{
			name: "Simple Location",
			buf: []byte{0x01, 0x02},
			expectedLoc: model.MoqtLocation{GroupId: 1, ObjectId: 2},
			expectedN: 2,
			expectErr: false,
		},
		{
			name: "Larger GroupId and ObjectId",
			buf: []byte{0x80, 0x12, 0xD6, 0x87, 0x85, 0xF4, 0xCF, 0x88},
			expectedLoc: model.MoqtLocation{GroupId: 1234567, ObjectId: 99929992},
			expectedN: 8,
			expectErr: false,
		},
		{
			name: "Zero GroupId and ObjectId",
			buf: []byte{0x00, 0x00},
			expectedLoc: model.MoqtLocation{GroupId: 0, ObjectId: 0},
			expectedN: 2,
			expectErr: false,
		},
		{
			name: "Incomplete Location, (1 byte)",
			buf: []byte{0x80},
			expectedLoc: model.MoqtLocation{},
			expectedN: 1,
			expectErr: true,
		},
		{
			// 0x80 starts with 10 so the varint parser expects 3 more bytes, but found none
			name: "Incomplete ObjectId, parser expects 3 more bytes",
			buf: []byte{0x01, 0x80},
			expectedLoc: model.MoqtLocation{},
			expectedN: 2,
			expectErr: true,
		},
		{
			name: "Empty Slice passed",
			buf: []byte{},
			expectedLoc: model.MoqtLocation{},
			expectedN: 0,
			expectErr: true,	
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, n, err := DecodeMoqtLocation(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeMoqtLocation() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeMoqtLocation() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(loc, tt.expectedLoc) {
					t.Errorf("DecodeMoqtLocation() got location = %v, want %v", loc, tt.expectedLoc)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeMoqtLocation() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeMoqtKeyValuePair(t *testing.T){
	tests := []struct {
		name string
		buf []byte
		expectedKVP model.MoqtKeyValuePair
		expectedN int
		expectErr bool
	}{
		{
			name: "Even Type with 0 uint64 Value",
			buf: []byte{0x00, 0x00},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(0, uint64(0))),
			expectedN: 2,
			expectErr: false,
		},
		{
			name: "Even Type with simple uint64 Value",
			buf: []byte{0x02, 0x05},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(2, uint64(5))),
			expectedN: 2,
			expectErr: false,
		},
		{
			name: "Even Type with Large uint64 Value",
			buf: []byte{0x04, 0xc0, 0x00, 0x02, 0x05, 0xa9, 0x0f, 0x51, 0xf3},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(4, uint64(2223334445555))),
			expectedN: 9,
			expectErr: false,
		},
		{
			name: "Odd Type with empty []byte Value",
			buf: []byte{0x01, 0x00},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(1, []byte{})),
			expectedN: 2,
			expectErr: false,
		},
		{
			name: "Odd Type with simple []byte Value",
			buf: []byte{0x03, 0x03, 0x01, 0x02, 0x03},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(3, []byte{0x01, 0x02, 0x03})),
			expectedN: 5,
			expectErr: false,
		},
		{
			name: "Odd Type with longer []byte Value",
			buf: []byte{0x05, 0x0b, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(5, []byte("hello world"))),
			expectedN: 13,
			expectErr: false,
		},
		{
			name: "Even multi-byte long type value",
			buf: []byte{0x8d, 0x3e, 0xd7, 0x8e, 0x01},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(222222222, uint64(1))),
			expectedN: 5,
			expectErr: false,
		},
		{
			name: "Odd multi-byte long type value",
			buf: []byte{0x93, 0xde, 0x43, 0x55, 0x01, 0x01},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(333333333, []byte{0x01})),
			expectedN: 6,
			expectErr: false,
		},
		{
			name: "Incomplete Even Type - Missing Value",
			buf: []byte{0x02},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN: 1,
			expectErr: true,
		},
		{
			name: "Incomplete Odd Type - Missing Length",
			buf: []byte{0x03},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN: 1,
			expectErr: true,
		},
		{
			name: "Incomplete Odd Type - Missing Value Bytes",
			buf: []byte{0x03, 0x03, 0x01, 0x02}, // Length is 3, but only 2 bytes provided
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN: 4,
			expectErr: true,
		},
		{
			name: "Empty Slice passed",
			buf: []byte{},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN: 0,
			expectErr: true,
		},
		{
			name: "Invalid Type (odd type with uint64 value)",
			buf: []byte{0x01, 0x05}, // Type 1 is odd, so it expects length and then bytes, not a uint64 value
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN: 2,
			expectErr: true,
		},
		// {
		// 	name: "Invalid Type (even type with byte slice value)",
		// 	buf: []byte{0x00, 0x01, 0x01}, // Type 0 is even, so it expects a uint64 value, not length and then bytes
		// 	expectedKVP: model.MoqtKeyValuePair{},
		// 	expectedN: 1, // Only the type is parsed successfully
		// 	expectErr: true,
		// }, This is actually expected behaviour, the parser parses a kvpair as : (0, 1) and leaves the last element (0x01) unparsed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvp, n, err := DecodeMoqtKeyValuePair(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeMoqtKeyValuePair() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeMoqtKeyValuePair() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(kvp, tt.expectedKVP) {
					t.Errorf("DecodeMoqtKeyValuePair() got kvp = %v, want %v", kvp, tt.expectedKVP)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeMoqtKeyValuePair() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeExtensions(t *testing.T){
	tests := []struct {
		name string
		buf []byte
		expectedKVPs []model.MoqtKeyValuePair
		expectedN int
		expectErr bool
	}{
		{
			name: "0 Extensions (empty slice)",
			buf: []byte{0x00}, // Only length given
			expectedKVPs: []model.MoqtKeyValuePair{},
			expectedN: 1,
			expectErr: false,
		},
		{
			name: "1 Extension",
			buf: []byte{0x01, 0x02, 0x0A},
			expectedKVPs: []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(2, uint64(10))),
			},
			expectedN: 3,
			expectErr: false,
		},
		{
			name: "5 Extensions",
			buf: []byte{
				0x05,       // Length of extensions (5)
				0x00, 0x01, // KV1: Type 0, Value 1
				0x01, 0x02, 0x01, 0x02, // KV2: Type 1, Length 2, Value [0x01, 0x02]
				0x02, 0x03, // KV3: Type 2, Value 3
				0x03, 0x03, 0x04, 0x05, 0x06, // KV4: Type 3, Length 3, Value [0x04, 0x05, 0x06]
				0x04, 0x05, // KV5: Type 4, Value 5
			},
			expectedKVPs: []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(0, uint64(1))),
				internal.Must(model.NewMoqtKeyValuePair(1, []byte{0x01, 0x02})),
				internal.Must(model.NewMoqtKeyValuePair(2, uint64(3))),
				internal.Must(model.NewMoqtKeyValuePair(3, []byte{0x04, 0x05, 0x06})),
				internal.Must(model.NewMoqtKeyValuePair(4, uint64(5))),			},
			expectedN: 16,
			expectErr: false,
		},
		{
			name: "Incomplete Extensions - Missing KVP",
			buf: []byte{0x01, 0x02}, // Length is 1, but only type 2 is provided, missing value
			expectedKVPs: nil,
			expectedN: 2,
			expectErr: true,
		},
		{
			name: "Incomplete Extensions - Missing KVP entirely",
			buf: []byte{0x01}, // Length is 1, but no KVP is provided
			expectedKVPs: nil,
			expectedN: 1,
			expectErr: true,
		},
		{
			name: "Empty Slice passed",
			buf: []byte{},
			expectedKVPs: nil,
			expectedN: 0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvps, n, err := DecodeExtensions(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeExtensions() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeExtensions() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(kvps, tt.expectedKVPs) {
					t.Errorf("DecodeExtensions() got kvps = %v, want %v", kvps, tt.expectedKVPs)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeExtensions() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}