package model

import (
	"testing"
	"reflect"

	"github.com/LukaGiorgadze/gonull/v2"
)

func TestNewSubGroupHeaderType(t *testing.T){
	tests := []struct {
		name      string
		typeID    uint64
		expected  *SubGroupHeaderType
		expectErr bool
	}{
		{
			name:   "0x10 - AbsentZero, No Extensions, No EndOfGroup, Priority Present",
			typeID: 0x10,
			expected: &SubGroupHeaderType{
				TypeID:            0x10,
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x11 - AbsentZero, Extensions, No EndOfGroup, Priority Present",
			typeID: 0x11,
			expected: &SubGroupHeaderType{
				TypeID:            0x11,
				ExtensionsPresent: true,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x12 - AbsentFirstObject, No Extensions, No EndOfGroup, Priority Present",
			typeID: 0x12,
			expected: &SubGroupHeaderType{
				TypeID:            0x12,
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentFirstObject,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x14 - Present, No Extensions, No EndOfGroup, Priority Present",
			typeID: 0x14,
			expected: &SubGroupHeaderType{
				TypeID:            0x14,
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModePresent,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x18 - AbsentZero, No Extensions, EndOfGroup, Priority Present",
			typeID: 0x18,
			expected: &SubGroupHeaderType{
				TypeID:            0x18,
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        true,
				PriorityPresent:   true,
			},
			expectErr: false,
		},
		{
			name:   "0x30 - AbsentZero, No Extensions, No EndOfGroup, Priority Omitted",
			typeID: 0x30,
			expected: &SubGroupHeaderType{
				TypeID:            0x30,
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   false,
			},
			expectErr: false,
		},
		{
			name:      "Invalid Type - Reserved Subgroup ID Mode (0b11)",
			typeID:    0x16, // 0b00010110, bits 1-2 are 0b11
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Invalid Type - Reserved Subgroup ID Mode (0b11) with other bits set",
			typeID:    0x37, // 0b00110111, bits 1-2 are 0b11
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Invalid Type - Bit 4 not set (e.g., 0x0F)",
			typeID:    0x0F,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Invalid Type - Bit 4 not set (e.g., 0x00)",
			typeID:    0x00,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Invalid Type - Out of range (e.g., 0x40)",
			typeID:    0x40,
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Invalid Type - In forbidden range (e.g., 0x20)",
			typeID:    0x20, // This is a valid ObjectDatagram type, but not SubgroupHeader
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSubGroupHeaderType(tt.typeID)

			if tt.expectErr {
				if err == nil {
					t.Errorf("NewSubGroupHeaderType() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewSubGroupHeaderType() unexpected error: %v", err)
				}
				if got.TypeID != tt.expected.TypeID ||
					got.ExtensionsPresent != tt.expected.ExtensionsPresent ||
					got.SGIDMode != tt.expected.SGIDMode ||
					got.EndOfGroup != tt.expected.EndOfGroup ||
					got.PriorityPresent != tt.expected.PriorityPresent {
					t.Errorf("NewSubGroupHeaderType() got = %+v, want %+v", got, tt.expected)
				}
			}
		})
	}
}

func TestSubGroupHeaderType_ToUint64(t *testing.T) {
	tests := []struct {
		name     string
		sgt      SubGroupHeaderType
		expected uint64
	}{
		{
			name: "0x10 - AbsentZero, No Extensions, No EndOfGroup, Priority Present",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expected: 0x10,
		},
		{
			name: "0x11 - AbsentZero, Extensions, No EndOfGroup, Priority Present",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: true,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expected: 0x11,
		},
		{
			name: "0x12 - AbsentFirstObject, No Extensions, No EndOfGroup, Priority Present",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentFirstObject,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expected: 0x12,
		},
		{
			name: "0x14 - Present, No Extensions, No EndOfGroup, Priority Present",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModePresent,
				EndOfGroup:        false,
				PriorityPresent:   true,
			},
			expected: 0x14,
		},
		{
			name: "0x18 - AbsentZero, No Extensions, EndOfGroup, Priority Present",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        true,
				PriorityPresent:   true,
			},
			expected: 0x18,
		},
		{
			name: "0x30 - AbsentZero, No Extensions, No EndOfGroup, Priority Omitted",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: false,
				SGIDMode:          SubgroupIdModeAbsentZero,
				EndOfGroup:        false,
				PriorityPresent:   false,
			},
			expected: 0x30,
		},
		{
			name: "0x3F - All flags set (except reserved SGID mode)",
			sgt: SubGroupHeaderType{
				ExtensionsPresent: true,
				SGIDMode:          SubgroupIdModePresent, // This will encode as 0b10
				EndOfGroup:        true,
				PriorityPresent:   false,
			},
			expected: 0x3D, // 0b00111101
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sgt.ToUint64()
			if got != tt.expected {
				t.Errorf("ToUint64() got = 0x%x, want 0x%x", got, tt.expected)
			}
		})
	}
}

func TestNewSubGroupHeader(t *testing.T){
	tests := []struct {
		name       string
		trackAlias uint64
		groupId    uint64
		opts       []SubgroupHeaderOption
		expected   *SubGroupHeader
	}{
		{
			name:       "Basic Subgroup Header (0x10)",
			trackAlias: 1,
			groupId:    100,
			opts:       []SubgroupHeaderOption{
				SHWithPublisherPriority(8),
			},
			expected: &SubGroupHeader{
				SGType: SubGroupHeaderType{
					TypeID:            0x10,
					ExtensionsPresent: false,
					SGIDMode:          SubgroupIdModeAbsentZero,
					EndOfGroup:        false,
					PriorityPresent:   true,
				},
				TrackAlias:        1,
				GroupId:           100,
				PublisherPriority: gonull.NewNullable(uint8(8)),
			},
		},
		{
			name:       "Subgroup Header with Extensions, EndOfGroup, and SubgroupId (0x1D)",
			trackAlias: 2,
			groupId:    200,
			opts: []SubgroupHeaderOption{
				SHWithExtensions(),
				SHWithSubgroupId(50),
				SHWithEndOfGroup(),
				SHWithPublisherPriority(10),
			},
			expected: &SubGroupHeader{
				SGType: SubGroupHeaderType{
					TypeID:            0x1D, // 0b00011101 (Extensions, SGIDPresent, EndOfGroup, PriorityPresent)
					ExtensionsPresent: true,
					SGIDMode:          SubgroupIdModePresent,
					EndOfGroup:        true,
					PriorityPresent:   true,
				},
				TrackAlias:        2,
				GroupId:           200,
				SubgroupId:        gonull.NewNullable(uint64(50)),
				PublisherPriority: gonull.NewNullable(uint8(10)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got:= NewSubGroupHeader(tt.trackAlias, tt.groupId, tt.opts...)

			if !reflect.DeepEqual(*got, *(tt.expected)) {
				t.Errorf("NewSubGroupHeader() got = %+v, want %+v", got, tt.expected)
			}
		})
	}
}