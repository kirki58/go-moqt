package message

import (
	"go-moq/internal"
	"go-moq/pkg/data"
	"go-moq/pkg/model"

	"reflect"
	"testing"
)

func TestDecodeMoqtLocation(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		expectedLoc model.MoqtLocation
		expectedN   int
		expectErr   bool
	}{
		{
			name:        "Simple Location",
			buf:         []byte{0x01, 0x02},
			expectedLoc: model.MoqtLocation{GroupId: 1, ObjectId: 2},
			expectedN:   2,
			expectErr:   false,
		},
		{
			name:        "Larger GroupId and ObjectId",
			buf:         []byte{0x80, 0x12, 0xD6, 0x87, 0x85, 0xF4, 0xCF, 0x88},
			expectedLoc: model.MoqtLocation{GroupId: 1234567, ObjectId: 99929992},
			expectedN:   8,
			expectErr:   false,
		},
		{
			name:        "Zero GroupId and ObjectId",
			buf:         []byte{0x00, 0x00},
			expectedLoc: model.MoqtLocation{GroupId: 0, ObjectId: 0},
			expectedN:   2,
			expectErr:   false,
		},
		{
			name:        "Incomplete Location, (1 byte)",
			buf:         []byte{0x80},
			expectedLoc: model.MoqtLocation{},
			expectedN:   1,
			expectErr:   true,
		},
		{
			// 0x80 starts with 10 so the varint parser expects 3 more bytes, but found none
			name:        "Incomplete ObjectId, parser expects 3 more bytes",
			buf:         []byte{0x01, 0x80},
			expectedLoc: model.MoqtLocation{},
			expectedN:   2,
			expectErr:   true,
		},
		{
			name:        "Empty Slice passed",
			buf:         []byte{},
			expectedLoc: model.MoqtLocation{},
			expectedN:   0,
			expectErr:   true,
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

func TestDecodeMoqtKeyValuePair(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		expectedKVP model.MoqtKeyValuePair
		expectedN   int
		expectErr   bool
	}{
		{
			name:        "Even Type with 0 uint64 Value",
			buf:         []byte{0x00, 0x00},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(0, uint64(0))),
			expectedN:   2,
			expectErr:   false,
		},
		{
			name:        "Even Type with simple uint64 Value",
			buf:         []byte{0x02, 0x05},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(2, uint64(5))),
			expectedN:   2,
			expectErr:   false,
		},
		{
			name:        "Even Type with Large uint64 Value",
			buf:         []byte{0x04, 0xc0, 0x00, 0x02, 0x05, 0xa9, 0x0f, 0x51, 0xf3},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(4, uint64(2223334445555))),
			expectedN:   9,
			expectErr:   false,
		},
		{
			name:        "Odd Type with empty []byte Value",
			buf:         []byte{0x01, 0x00},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(1, []byte{})),
			expectedN:   2,
			expectErr:   false,
		},
		{
			name:        "Odd Type with simple []byte Value",
			buf:         []byte{0x03, 0x03, 0x01, 0x02, 0x03},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(3, []byte{0x01, 0x02, 0x03})),
			expectedN:   5,
			expectErr:   false,
		},
		{
			name:        "Odd Type with longer []byte Value",
			buf:         []byte{0x05, 0x0b, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(5, []byte("hello world"))),
			expectedN:   13,
			expectErr:   false,
		},
		{
			name:        "Even multi-byte long type value",
			buf:         []byte{0x8d, 0x3e, 0xd7, 0x8e, 0x01},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(222222222, uint64(1))),
			expectedN:   5,
			expectErr:   false,
		},
		{
			name:        "Odd multi-byte long type value",
			buf:         []byte{0x93, 0xde, 0x43, 0x55, 0x01, 0x01},
			expectedKVP: internal.Must(model.NewMoqtKeyValuePair(333333333, []byte{0x01})),
			expectedN:   6,
			expectErr:   false,
		},
		{
			name:        "Incomplete Even Type - Missing Value",
			buf:         []byte{0x02},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN:   1,
			expectErr:   true,
		},
		{
			name:        "Incomplete Odd Type - Missing Length",
			buf:         []byte{0x03},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN:   1,
			expectErr:   true,
		},
		{
			name:        "Incomplete Odd Type - Missing Value Bytes",
			buf:         []byte{0x03, 0x03, 0x01, 0x02}, // Length is 3, but only 2 bytes provided
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN:   4,
			expectErr:   true,
		},
		{
			name:        "Empty Slice passed",
			buf:         []byte{},
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN:   0,
			expectErr:   true,
		},
		{
			name:        "Invalid Type (odd type with uint64 value)",
			buf:         []byte{0x01, 0x05}, // Type 1 is odd, so it expects length and then bytes, not a uint64 value
			expectedKVP: model.MoqtKeyValuePair{},
			expectedN:   2,
			expectErr:   true,
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

func TestDecodeExtensions(t *testing.T) {
	tests := []struct {
		name         string
		buf          []byte
		expectedKVPs []model.MoqtKeyValuePair
		expectedN    int
		expectErr    bool
	}{
		{
			name:         "0 Extensions (empty slice)",
			buf:          []byte{0x00}, // Only length given
			expectedKVPs: nil,
			expectedN:    1,
			expectErr:    false,
		},
		{
			name: "1 Extension",
			buf:  []byte{0x02, 0x02, 0x0A},
			expectedKVPs: []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(2, uint64(10))),
			},
			expectedN: 3,
			expectErr: false,
		},
		{
			name: "5 Extensions",
			buf: []byte{
				0x0F,       // Length of extensions (15)
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
				internal.Must(model.NewMoqtKeyValuePair(4, uint64(5)))},
			expectedN: 16,
			expectErr: false,
		},
		{
			name:         "Incomplete Extensions - Missing KVP",
			buf:          []byte{0x02, 0x02}, // Length is 2, but only type 2 is provided, missing value
			expectedKVPs: nil,
			expectedN:    2,
			expectErr:    true,
		},
		{
			name:         "Incomplete Extensions - Missing KVP entirely",
			buf:          []byte{0x01}, // Length is 1, but no KVP is provided
			expectedKVPs: nil,
			expectedN:    1,
			expectErr:    true,
		},
		{
			name:         "Empty Slice passed",
			buf:          []byte{},
			expectedKVPs: nil,
			expectedN:    0,
			expectErr:    true,
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

func TestDecodeObjectDatagram(t *testing.T) {
	tests := []struct {
		name       string
		buf        []byte
		expectedDG *data.ObjectDatagram
		expectedN  int
		expectErr  bool
	}{
		{
			name: "0x03 ObjectDatagram",
			buf: []byte{
				0x03, // TypeId of the datagram
				0x01, // Track Alias

				// Location - Start
				0x02, // GroupId
				0x0C, // ObjectId
				// Location - End

				0x0C, // Publisher Priority

				// Extensions - Start
				0x06, // Length of extensions
				// Extension 1
				0x0A, 0x0B,
				// Extensions 2
				0x15, 0x02, 0x00, 0x01, // Type = 21, Len = 2, Value = [0, 1]
				// Extensions - End

				// Status field omitted entirely because of datagram's type (0x03)

				// Dump the rest with the paylaod as is
				0x21, 0x22, 0x22, 0xFF,
			},
			expectedDG: internal.Must(data.NewObjectDatagram(1, 2,
				data.WithObjectId(12),
				data.WithPublisherPriority(12),
				data.WithExtensions(
					[]model.MoqtKeyValuePair{
						internal.Must(model.NewMoqtKeyValuePair(10, uint64(11))),
						internal.Must(model.NewMoqtKeyValuePair(21, []byte{0x00, 0x01})),
					},
				),
				data.WithEndOfGroup(),
				data.WithPayload(
					[]byte{
						0x21, 0x22, 0x22, 0xFF,
					},
				),
			)),
			expectedN: 16,
			expectErr: false,
		},
		{
			name: "0x21 ObjectDatagram (Status, Extensions, Priority, ObjectId)",
			buf: []byte{
				0x21, // TypeId
				0x01, // Track Alias

				// Location - Start
				0x02, // GroupId
				0x0C, // ObjectId
				// Location - End

				0x0C, // Publisher Priority

				// Extensions - Start
				0x06, // Length of extensions
				// Extension 1
				0x0A, 0x0B,
				// Extensions 2
				0x15, 0x02, 0x00, 0x01, // Type = 21, Len = 2, Value = [0, 1]
				// Extensions - End

				0x00, // Status, Normal = 0x00

				// Payload field ommited entirely in status object
			},
			expectedDG: internal.Must(data.NewObjectDatagram(1, 2,
				data.WithObjectId(12),
				data.WithPublisherPriority(12),
				data.WithExtensions(
					[]model.MoqtKeyValuePair{
						internal.Must(model.NewMoqtKeyValuePair(10, uint64(11))),
						internal.Must(model.NewMoqtKeyValuePair(21, []byte{0x00, 0x01})),
					},
				),
				data.WithStatus(model.Normal),
			)),
			expectedN: 13,
		},
		{
			name: "Non-normal status with extensions, protocol violation",
			buf: []byte{
				0x21, // TypeId status, extensions, priority, object id
				0x01, // Track Alias
				0x02, 0x0C, // Location (Group 2, Object 12)
				0x0C,       // Publisher Priority
				0x02, 0x00, 0x01, // Extensions (Len 2, Type 0, Val 1)
				0x03, // Status (EndOfGroup)
			},
			expectedDG: nil,
			expectedN: 8,
			expectErr: true, // Protocol violation: non-Normal status with extensions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dg, n, err := DecodeObjectDatagram(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeObjectDatagram() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeObjectDatagram() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(dg, tt.expectedDG) {
					t.Errorf("DecodeObjectDatagram() got datagram = %v, want %v", *dg, *(tt.expectedDG))
				}
				if n != tt.expectedN {
					t.Errorf("DecodeObjectDatagram() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeSubgroupHeader(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		expectedSGH *data.SubGroupHeader
		expectedN   int
		expectErr   bool
	}{
		{
			name: "Basic Subgroup Header (0x10)",
			buf: []byte{
				0x10, // Type
				0x01, // Track Alias
				0x05, // Group ID
				0x08, // Publisher Priority
			},
			expectedSGH: data.NewSubGroupHeader(1, 5,
				data.SHWithPublisherPriority(8),
			),
			expectedN: 4,
			expectErr: false,
		},
		{
			name: "0x1D subgroup header, all fields present",
			buf: []byte{
				0x1D, // Type
				0x01, // Track Alias
				0x05, // Group ID
				0x0A, // Subgroup ID
				0x08, // Publisher Priority
			},
			expectedSGH: data.NewSubGroupHeader(1, 5,
				data.SHWithSubgroupId(10),
				data.SHWithPublisherPriority(8),
				data.SHWithExtensions(),
				data.SHWithEndOfGroup(),
			),
			expectedN: 5,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgh, n, err := DecodeSubgroupHeader(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeSubgroupHeader() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeSubgroupHeader() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(sgh, tt.expectedSGH) {
					t.Errorf("DecodeSubgroupHeader() got = %+v, want %+v", *sgh, *(tt.expectedSGH))
				}
				if n != tt.expectedN {
					t.Errorf("DecodeSubgroupHeader() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeTrackNamespace(t *testing.T) {
	tests := []struct {
		name                   string
		buf                    []byte
		expectedTrackNamespace model.MoqtTrackNamespace
		expectedN              int
		expectErr              bool
	}{
		{
			name: "Regular track namespace with 2 fields",
			buf: []byte{
				0x02,       // Number of Track Namespace Fields (2)
				0x05,       // Length of first field (5)
				'h', 'e', 'l', 'l', 'o', // First field value ("hello")
				0x05,       // Length of second field (5)
				'w', 'o', 'r', 'l', 'd', // Second field value ("world")
			},
			expectedTrackNamespace: model.MoqtTrackNamespace([][]byte{
				[]byte("hello"),
				[]byte("world"),
			}),
			expectedN: 13,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn, n, err := DecodeTrackNamespace(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeTrackNamespace() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeTrackNamespace() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(tn, tt.expectedTrackNamespace) {
					t.Errorf("DecodeTrackNamespace() got track namespace = %v, want %v", tn, tt.expectedTrackNamespace)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeTrackNamespace() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeMoqtFullTrackName(t *testing.T) {
	tests := []struct {
		name                 string
		buf                  []byte
		expectedFullTrackName *model.MoqtFullTrackName
		expectedN            int
		expectErr            bool
	}{
		{
			name: "Regular full track name with 1 track name field and track name",
			buf: []byte{
				0x01,       // Number of Track Namespace Fields (1)
				0x05,       // Length of first field (5)
				'h', 'e', 'l', 'l', 'o', // First field value ("hello")
				0x05,       // Length of Track Name (5)
				'w', 'o', 'r', 'l', 'd', // Track Name value ("world")
			},
			expectedFullTrackName: &model.MoqtFullTrackName{
				Namespace: model.MoqtTrackNamespace([][]byte{
					[]byte("hello"),
				}),
				Name: []byte("world"),
			},
			expectedN: 13,
			expectErr: false, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ftn, n, err := DecodeMoqtFullTrackName(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeMoqtFullTrackName() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeMoqtFullTrackName() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(ftn, tt.expectedFullTrackName) {
					t.Errorf("DecodeMoqtFullTrackName() got full track name = %v, want %v", ftn, tt.expectedFullTrackName)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeMoqtFullTrackName() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeMoqtReasonPhrase(t *testing.T) {
	tests := []struct {
		name               string
		buf                []byte
		expectedReason     model.MoqtReasonPhrase
		expectedN          int
		expectErr          bool
	}{
		{
			name:           "Simple Reason Phrase",
			buf:            []byte{0x05, 'e', 'r', 'r', 'o', 'r'},
			expectedReason: "error",
			expectedN:      6,
			expectErr:      false,
		},
		{
			name:           "Empty Reason Phrase",
			buf:            []byte{0x00},
			expectedReason: "",
			expectedN:      1,
			expectErr:      false,
		},
		{
			name:           "Incomplete Reason Phrase - Missing Value",
			buf:            []byte{0x05, 'e', 'r', 'r'},
			expectedReason: "",
			expectedN:      1,
			expectErr:      true,
		},
		{
			name:           "Empty Slice",
			buf:            []byte{},
			expectedReason: "",
			expectedN:      0,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, n, err := DecodeMoqtReasonPhrase(tt.buf)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeMoqtReasonPhrase() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeMoqtReasonPhrase() unexpected error: %v", err)
				}
				if reason != tt.expectedReason {
					t.Errorf("DecodeMoqtReasonPhrase() got reason = %v, want %v", reason, tt.expectedReason)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeMoqtReasonPhrase() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeSubgroupObject(t *testing.T) {
	tests := []struct {
		name               string
		buf                []byte
		subgroupHeaderType *data.SubGroupHeaderType
		expectedSGO        *data.SubgroupObject
		expectedN          int
		expectErr          bool
	}{
		{
			name: "Basic object with payload",
			buf: []byte{
				0x01, // Object ID Delta
				0x04, // Payload Length
				0x0a, 0x0b, 0x0c, 0x0d, // Payload
			},
			subgroupHeaderType: internal.Must(data.NewSubGroupHeaderType(uint64(0x10))),
			expectedSGO: internal.Must(data.NewSubgroupObject(0x01,
				data.SOWithPayload([]byte{0x0a, 0x0b, 0x0c, 0x0d}),
			)),
			expectedN: 6,
			expectErr: false,
		},
		{
			name: "Object with status (Normal) and extensions",
			buf: []byte{
				0x02,             // Object ID Delta
				0x02, 0x00, 0x01, // Extensions (Len 2, Type 0, Val 1)
				0x00, // Payload Length (0)
				0x00, // Status (Normal)
			},
			subgroupHeaderType: internal.Must(data.NewSubGroupHeaderType(uint64(0x11))), // ExtensionsPresent = true
			expectedSGO: internal.Must(data.NewSubgroupObject(0x02,
				data.SOWithStatus(model.Normal),
				data.SOWithExtensions([]model.MoqtKeyValuePair{
					internal.Must(model.NewMoqtKeyValuePair(0, uint64(1))),
				}),
			)),
			expectedN: 6,
			expectErr: false,
		},
		{
			name: "Non-normal status with extensions, protocol violation",
			buf: []byte{
				0x03,             // Object ID Delta
				0x02, 0x00, 0x01, // Extensions (Len 2, Type 0, Val 1)
				0x00, // Payload Length (0)
				0x03, // Status (EndOfGroup)
			},
			subgroupHeaderType: internal.Must(data.NewSubGroupHeaderType(uint64(0x11))), // ExtensionsPresent = true
			expectedSGO:        nil,
			expectedN:          6,
			expectErr:          true,
		},
		{
			name: "Object with status (Normal) and no extensions",
			buf: []byte{
				0x04, // Object ID Delta
				0x00, // Payload Length (0)
				0x00, // Status (Normal)
			},
			subgroupHeaderType: internal.Must(data.NewSubGroupHeaderType(uint64(0x10))), // ExtensionsPresent = false
			expectedSGO: internal.Must(data.NewSubgroupObject(0x04,
				data.SOWithStatus(model.Normal),
			)),
			expectedN: 3,
			expectErr: false,	
		},
		{
			name: "Non-normal status object in extensions-present subgroup header (ext length must be 0)",
			buf: []byte{
				0x05, // Object ID Delta
				0x00, // Extensions Length (0)
				0x00, // Payload Length (0)
				0x03, // Status (EndOfGroup)
			},
			subgroupHeaderType: internal.Must(data.NewSubGroupHeaderType(uint64(0x11))), // ExtensionsPresent = true
			expectedSGO: internal.Must(data.NewSubgroupObject(0x05,
				data.SOWithStatus(model.EndOfGroup),
			)),
			expectedN: 4,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgo, n, err := DecodeSubgroupObject(tt.buf, tt.subgroupHeaderType)

			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeSubgroupObject() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeSubgroupObject() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(sgo, tt.expectedSGO) {
					t.Errorf("DecodeSubgroupObject() got = %+v, want %+v", sgo, tt.expectedSGO)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeSubgroupObject() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
}

func TestDecodeSubscriptionFilter(t *testing.T){
	// TODO: add decode tests for AbsoluteStart and AbsoluteRange too later
	tests := []struct {
		name           string
		buf            []byte
		expectedFilter *data.SubscriptionFilter
		expectedN      int
		expectErr      bool
	}{
		{
			name: "NextGroupStart Filter",
			buf:  []byte{0x01},
			expectedFilter: &data.SubscriptionFilter{
				FilterType: data.NextGroupStart,
			},
			expectedN: 1,
			expectErr: false,
		},
		{
			name: "LargestObject Filter",
			buf:  []byte{0x02},
			expectedFilter: &data.SubscriptionFilter{
				FilterType: data.LargestObject,
			},
			expectedN: 1,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, n, err := DecodeSubscriptionFilter(tt.buf)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("DecodeSubscriptionFilter() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("DecodeSubscriptionFilter() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(filter, tt.expectedFilter) {
					t.Errorf("DecodeSubscriptionFilter() got filter = %v, want %v", filter, tt.expectedFilter)
				}
				if n != tt.expectedN {
					t.Errorf("DecodeSubscriptionFilter() got parsed bytes = %v, want %v", n, tt.expectedN)
				}
			}
		})
	}
	
}