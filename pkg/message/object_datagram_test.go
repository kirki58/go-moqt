package message

import (
	"reflect"
	"testing"

	"go-moq/internal"
	"go-moq/pkg/model"

	"github.com/LukaGiorgadze/gonull/v2"
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
			name:      "Out of range - TypeId uses reserved bit 0x10",
			typeID:    0x10,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Out of range - TypeId uses reserved bit, value 0x1F",
			typeID:    0x1F,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Out of range - Status-EndOfGroup Conflict 0x22",
			typeID:    0x22,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Out of range - Status-EndOfGroup Conflict 0x26",
			typeID:    0x26,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Out of range - Exceeds defined range by value 0x30",
			typeID:    0x30,
			expected:  nil,
			expectErr: true,
		},
		{
			name:   "0x00 - Yields object with ObjectId present, Priority present and with payload",
			typeID: 0x00,
			expected: &ObjectDatagramType{
				TypeID:          0x00,
				ObjectIdPresent: true,
				PriorityPresent: true,
			},
			expectErr: false,
		},
		{
			name:   "0x07 ---> ObjectId omitted, Rest present and with payload",
			typeID: 0x07,
			expected: &ObjectDatagramType{
				TypeID:            0x07,
				ExtensionsPresent: true,
				EndOfGroup:        true,
				PriorityPresent:   true, // Not Omitted
			},
			expectErr: false,
		},
		{
			name:   "0x20 --> ObjectId present, Priority Present, With Status",
			typeID: 0x20,
			expected: &ObjectDatagramType{
				TypeID:          0x20,
				ObjectIdPresent: true,
				PriorityPresent: true,
				StatusOrPayload: true,
			},
			expectErr: false,
		},
		{
			name:   "0x25 --> Priority present, Extensions present, With Statuss",
			typeID: 0x25,
			expected: &ObjectDatagramType{
				TypeID:            0x25,
				ExtensionsPresent: true,
				PriorityPresent:   true,
				StatusOrPayload:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x08 --> ObjectId present, With Payload",
			typeID: 0x08,
			expected: &ObjectDatagramType{
				TypeID:          0x08,
				ObjectIdPresent: true,
			},
			expectErr: false,
		},
		{
			name:   "0x0F --> EndOfGroup, Extensions Present, With Payload",
			typeID: 0x0F,
			expected: &ObjectDatagramType{
				TypeID:            0x0F,
				ExtensionsPresent: true,
				EndOfGroup:        true,
			},
			expectErr: false,
		},
		{
			name:   "0x28 --> ObjectId present, With Status",
			typeID: 0x28,
			expected: &ObjectDatagramType{
				TypeID:          0x28,
				ObjectIdPresent: true,
				StatusOrPayload: true,
			},
			expectErr: false,
		},
		{
			name:   "0x29 --> ObjectId present, Extension present, With status",
			typeID: 0x29,
			expected: &ObjectDatagramType{
				TypeID:            0x29,
				ExtensionsPresent: true,
				ObjectIdPresent:   true,
				StatusOrPayload:   true,
			},
			expectErr: false,
		},
		{
			name: "0x2D --> Extensions present, With status",
			typeID: 0x2D,
			expected: &ObjectDatagramType{
				TypeID: 0x2D,
				ExtensionsPresent: true,
				StatusOrPayload: true,
			},
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
				if !reflect.DeepEqual(*got, *(tt.expected)) {
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
		{
			name: "0x2D --> Extensions Present with Status",
			dt: ObjectDatagramType{
				ExtensionsPresent: true,
				StatusOrPayload: true,
			},
			expected: 0x2D,
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

func TestNewObjectDatagram(t *testing.T) {
	tests := []struct {
		name       string
		trackAlias uint64
		groupID    uint64
		opts       []ObjectDatagramOption
		expectErr  bool
		expected   *ObjectDatagram
	}{
		{
			name:       "Error condition - both WithStatus and WithPayload options are passed, which defies the toggle logic",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithStatus(model.DoesNotExist),
				WithPayload([]byte{0x01, 0x02}),
			},
			expectErr: true,
			expected:  nil,
		},
		{
			name:       "Error condition - Neither WithStatus or WithPayload options are passed",
			trackAlias: 0,
			groupID:    0,
			opts:       []ObjectDatagramOption{},
			expectErr:  true,
			expected:   nil,
		},
		{
			name:       "Error condition - Both WithStatus and WithEndOfGroup options are passed",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithStatus(model.EndOfGroup),
				WithEndOfGroup(),
			},
			expectErr: true,
			expected:  nil,
		},
		{
			name:       "Status object with ObjectId",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithStatus(model.EndOfGroup),
				WithObjectId(2),
			},
			expectErr: false,
			expected: &ObjectDatagram{
				Dtype: ObjectDatagramType{
					TypeID:          0x28,
					StatusOrPayload: true,
					ObjectIdPresent: true,
				},
				TrackAlias: 0,
				Location:   model.MoqtLocation{GroupId: 0, ObjectId: 2},
				Status: gonull.NewNullable(model.EndOfGroup),
			},
		},
		{
			name:       "Payload object with no options passed",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithPayload([]byte{0x01, 0x02}),
			},
			expectErr: false,
			expected: &ObjectDatagram{
				Dtype: ObjectDatagramType{
					TypeID: 0x0C,
				},
				TrackAlias: 0,
				Location:   model.MoqtLocation{GroupId: 0, ObjectId: 0},
				Payload:    gonull.NewNullable([]byte{0x01, 0x02}),
			},
		},
		{
			name:       "Status object with every possible option passed",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithExtensions(
					[]model.MoqtKeyValuePair{
						internal.Must(model.NewMoqtKeyValuePair(2, uint64(1))),
						internal.Must(model.NewMoqtKeyValuePair(4, uint64(123))),
					},
				),
				WithObjectId(2),
				WithPublisherPriority(12),
				WithStatus(model.EndOfGroup),
			},
			expectErr: false,
			expected: &ObjectDatagram{
				Dtype: ObjectDatagramType{
					TypeID:            0x21,
					ExtensionsPresent: true,
					ObjectIdPresent:   true,
					PriorityPresent:   true,
					StatusOrPayload:   true,
				},
				TrackAlias: 0,
				Location: model.MoqtLocation{
					GroupId:  0,
					ObjectId: 2,
				},
				PublisherPriority: gonull.NewNullable(uint8(12)),
				Extensions: gonull.NewNullable([]model.MoqtKeyValuePair{
					internal.Must(model.NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(model.NewMoqtKeyValuePair(4, uint64(123))),
				}),
				Status: gonull.NewNullable(model.EndOfGroup),
			},
		},
		{
			name:       "Payload object with every possible option passed",
			trackAlias: 0,
			groupID:    0,
			opts: []ObjectDatagramOption{
				WithExtensions(
					[]model.MoqtKeyValuePair{
						internal.Must(model.NewMoqtKeyValuePair(2, uint64(1))),
						internal.Must(model.NewMoqtKeyValuePair(4, uint64(123))),
					},
				),
				WithObjectId(2),
				WithPublisherPriority(12),
				WithPayload([]byte{0x01, 0x02}),
				WithEndOfGroup(),
			},
			expectErr: false,
			expected: &ObjectDatagram{
				Dtype: ObjectDatagramType{
					TypeID:            0x03,
					ExtensionsPresent: true,
					ObjectIdPresent:   true,
					PriorityPresent:   true,
					EndOfGroup:        true,
				},
				TrackAlias: 0,
				Location: model.MoqtLocation{
					GroupId:  0,
					ObjectId: 2,
				},
				PublisherPriority: gonull.NewNullable(uint8(12)),
				Extensions: gonull.NewNullable([]model.MoqtKeyValuePair{
					internal.Must(model.NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(model.NewMoqtKeyValuePair(4, uint64(123))),
				}),
				Payload: gonull.NewNullable([]byte{0x01, 0x02}),
			},
		},
		
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObjectDatagram(tt.trackAlias, tt.groupID, tt.opts...)

			if tt.expectErr {
				if err == nil {
					t.Errorf("NewObjectDatagram() expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("NewObjectDatagram() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(*got, *(tt.expected)) {
					t.Errorf("NewObjectDatagram() got = %+v, want %+v", *got, *(tt.expected))
				}
			}
		})
	}
}
