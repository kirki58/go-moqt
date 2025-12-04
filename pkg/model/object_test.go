package model

import (
	"reflect"
	"testing"
	"go-moq/internal"
)

func TestNewMoqtObject(t *testing.T) {
	tests := []struct {
		name                    string
		loc                     MoqtLocation
		subGroupId              uint64
		ftn                     MoqtFullTrackName
		publisherPriority       uint8
		objectForwardingPreference MoqtObjectForwardingPreference
		objectStatus            MoqtObjectStatus
		extensionHeaders        []MoqtKeyValuePair
		payload                 []byte
		expectedObject          *MoqtObject
		expectError             bool
		expectedErrorCode       MOQT_SESSION_TERMINATION_ERROR_CODE
	}{
		{
			name: "Valid Object - Normal status, no extensions, payload",
			loc: MoqtLocation{
				GroupId:  1,
				ObjectId: 2,
			},
			subGroupId:        3,
			ftn:               MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
			publisherPriority: 0,
			objectForwardingPreference: Subgroup,
			objectStatus:            Normal,
			extensionHeaders:        nil,
			payload:                 []byte("some payload"),
			expectedObject: &MoqtObject{
				Location:                   MoqtLocation{GroupId: 1, ObjectId: 2},
				SubgroupID:                 3,
				FullTrackName:              MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
				PublisherPriority:          0,
				ObjectForwardingPreference: Subgroup,
				ObjectStatus:               Normal,
				ExtensionHeaders:           nil,
				Payload:                    []byte("some payload"),
			},
			expectError: false,
		},
		{
			name: "Valid Object - Normal status, 2 extensions, payload",
			loc: MoqtLocation{
				GroupId:  1,
				ObjectId: 2,
			},
			subGroupId:        3,
			ftn:               MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
			publisherPriority: 0,
			objectForwardingPreference: Subgroup,
			objectStatus:            Normal,
			extensionHeaders:        []MoqtKeyValuePair{
				internal.Must(NewMoqtKeyValuePair(0x02, uint64(123))),
				internal.Must(NewMoqtKeyValuePair(0x01, []byte("extension value"))),
			},
			payload:                 []byte("some payload"),
			expectedObject: &MoqtObject{
				Location:                   MoqtLocation{GroupId: 1, ObjectId: 2},
				SubgroupID:                 3,
				FullTrackName:              MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
				PublisherPriority:          0,
				ObjectForwardingPreference: Subgroup,
				ObjectStatus:               Normal,
				ExtensionHeaders: []MoqtKeyValuePair{
					internal.Must(NewMoqtKeyValuePair(0x02, uint64(123))),
					internal.Must(NewMoqtKeyValuePair(0x01, []byte("extension value"))),
				},
				Payload: []byte("some payload"),
			},
			expectError: false,
		},
		{
			name: "Invalid Object - DoesNotExist status with payload",
			loc: MoqtLocation{
				GroupId:  1,
				ObjectId: 2,
			},
			subGroupId:        3,
			ftn:               MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
			publisherPriority: 0,
			objectForwardingPreference: Subgroup,
			objectStatus:            DoesNotExist,
			extensionHeaders:        nil,
			payload:                 []byte("some payload"),
			expectedObject:          nil,
			expectError:             true,
			expectedErrorCode:       MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
		},
		{
			name: "Invalid Object - EndOfGroup status with extension headers",
			loc: MoqtLocation{
				GroupId:  1,
				ObjectId: 2,
			},
			subGroupId:        3,
			ftn:               MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
			publisherPriority: 0,
			objectForwardingPreference: Subgroup,
			objectStatus:            EndOfGroup,
			extensionHeaders:        []MoqtKeyValuePair{internal.Must(NewMoqtKeyValuePair(0x02, uint64(123)))},
			payload:                 nil,
			expectedObject:          nil,
			expectError:             true,
			expectedErrorCode:       MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
		},
		{
			name: "Valid Object - EndOfTrack status, no extensions, no payload",
			loc: MoqtLocation{
				GroupId:  1,
				ObjectId: 2,
			},
			subGroupId:        3,
			ftn:               MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
			publisherPriority: 0,
			objectForwardingPreference: Subgroup,
			objectStatus:            EndOfTrack,
			extensionHeaders:        nil,
			payload:                 nil,
			expectedObject: &MoqtObject{
				Location:                   MoqtLocation{GroupId: 1, ObjectId: 2},
				SubgroupID:                 3,
				FullTrackName:              MoqtFullTrackName{Namespace: MoqtTrackNamespace{[]byte("test")}, Name: []byte("track")},
				PublisherPriority:          0,
				ObjectForwardingPreference: Subgroup,
				ObjectStatus:               EndOfTrack,
				ExtensionHeaders:           nil,
				Payload:                    nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := NewMoqtObject(
				tt.loc,
				tt.subGroupId,
				tt.ftn,
				tt.publisherPriority,
				tt.objectForwardingPreference,
				tt.objectStatus,
				tt.extensionHeaders,
				tt.payload,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none")
				}
				moqtErr, ok := err.(MOQT_SESSION_TERMINATION_ERROR)
				if !ok {
					t.Errorf("Expected MOQT_SESSION_TERMINATION_ERROR, got %T", err)
				} else if moqtErr.ErrorCode != tt.expectedErrorCode {
					t.Errorf("Expected error code %v, got %v", tt.expectedErrorCode, moqtErr.ErrorCode)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error but got: %v", err)
				}
				if !reflect.DeepEqual(obj, tt.expectedObject) {
					t.Errorf("Expected object %+v, got %+v", tt.expectedObject, obj)
				}
			}
		})	
	}
}