package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestStringToMoqtFullTrackName(t *testing.T) {
	tests := []struct {
		testName string
		input    string
		expected MoqtFullTrackName
		err      bool
	}{
		{
			"Valid string with 2 fields", // TestName
			"valid/track/name",           // Input FTN
			MoqtFullTrackName{ // Expected result
				Namespace: MoqtTrackNamespace{[]byte("valid"), []byte("track")},
				Name:      []byte("name"),
			},
			false, // Should there be error?
		},
		{
			"Invalid string with 0 fields",
			"name",
			MoqtFullTrackName{},
			true,
		},
		{
			"Invalid string with 33 fields",
			"f1/f2/f3/f4/f5/f6/f7/f8/f9/f10/f11/f12/f13/f14/f15/f16/f17/f18/f19/f20/f21/f22/f23/f24/f25/f26/f27/f28/f29/f30/f31/f32/f33/name",
			MoqtFullTrackName{},
			true,
		},
		{
			"String that produces >4096 bytes Full Track Name",
			strings.Repeat("a", 4096) + "/name",
			MoqtFullTrackName{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			res, err := StringToMoqtFullTrackName(tt.input)

			if tt.err && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.err && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expected) {
				t.Errorf("Expected result %+v but got %+v", tt.expected, res)
			}
		})
	}
}
