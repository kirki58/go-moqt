package model

import (
	"go-moq/internal"
	"reflect"
	"testing"

	"github.com/LukaGiorgadze/gonull/v2"
)

func TestNewSubgroupObject(t *testing.T){
	tests := []struct{
		name string
		objectIdDelta uint64
		opts []SubgroupObjectOption
		expected *SubgroupObject
		expectErr bool
	}{
		{
			name: "Neither status nor payload option given",
			objectIdDelta: 0,
			opts: []SubgroupObjectOption{},
			expected: nil,
			expectErr: true,
		},
		{
			name: "Payload object with extensions",
			objectIdDelta: 0,
			opts: []SubgroupObjectOption{
				SOWithPayload([]byte("testing")),
				SOWithExtensions(
					[]MoqtKeyValuePair{
						internal.Must(NewMoqtKeyValuePair(2, uint64(1))),
						internal.Must(NewMoqtKeyValuePair(3, []byte("value2"))),
					},
				),
			},
			expected: &SubgroupObject{
				ObjectIdDelta: 0,
				Payload:       gonull.NewNullable([]byte("testing")),
				Extensions: gonull.NewNullable([]MoqtKeyValuePair{
					internal.Must(NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(NewMoqtKeyValuePair(3, []byte("value2"))),
				}),
			},
			expectErr: false,
		},
		{
			name: "Payload object classic",
			objectIdDelta: 0,
			opts: []SubgroupObjectOption{
				SOWithPayload([]byte("testing")),
			},
			expected: &SubgroupObject{
				ObjectIdDelta: 0,
				Payload:       gonull.NewNullable([]byte("testing")),
			},
			expectErr: false,
		},
		{
			name:          "Status object (Normal) with extensions",
			objectIdDelta: 1,
			opts: []SubgroupObjectOption{
				SOWithStatus(Normal),
				SOWithExtensions([]MoqtKeyValuePair{
					internal.Must(NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(NewMoqtKeyValuePair(3, []byte("value2"))),
				}),
			},
			expected: &SubgroupObject{
				ObjectIdDelta: 1,
				Status:        gonull.NewNullable(Normal),
				Extensions: gonull.NewNullable([]MoqtKeyValuePair{
					internal.Must(NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(NewMoqtKeyValuePair(3, []byte("value2"))),
				}),
			},
			expectErr: false,
		},
		{
			name:          "Status object (Non-Normal) with extensions - Error",
			objectIdDelta: 1,
			opts: []SubgroupObjectOption{
				SOWithStatus(EndOfGroup),
				SOWithExtensions([]MoqtKeyValuePair{
					internal.Must(NewMoqtKeyValuePair(2, uint64(1))),
					internal.Must(NewMoqtKeyValuePair(3, []byte("value2"))),
				}),
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name:          "Both status and payload given - Error",
			objectIdDelta: 5,
			opts: []SubgroupObjectOption{
				SOWithStatus(Normal),
				SOWithPayload([]byte("data")),
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name:          "Status object (Non-Normal) without extensions",
			objectIdDelta: 10,
			opts: []SubgroupObjectOption{
				SOWithStatus(EndOfTrack),
			},
			expected: &SubgroupObject{
				ObjectIdDelta: 10,
				Status:        gonull.NewNullable(EndOfTrack),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSubgroupObject(tt.objectIdDelta, tt.opts...)

			if tt.expectErr {
				if err == nil {
					t.Errorf("NewSubgroupObject() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewSubgroupObject() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("NewSubgroupObject() got = %+v, want %+v", got, tt.expected)
				}
			}
		})
	}
}