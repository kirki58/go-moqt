package model

type MoqtObject struct {
	// I. Identification
	Location      MoqtLocation
	SubgroupID    uint64
	FullTrackName MoqtFullTrackName

	// II. Control & Status Fields
	PublisherPriority          uint8 // 0-255
	ObjectForwardingPreference MoqtObjectForwardingPreference
	ObjectStatus               MoqtObjectStatus

	// III. Payload and Extensions
	// Note: ExtensionLength is the length of the ExtensionHeaders in bytes, since the varint wire represantation would differ from fixed-size ints in the application logic
	// and the application-level use case for this variable is seemingly none, this is only calculated when serializing the MoqtObject for the wire
	ExtensionHeaders []MoqtKeyValuePair
	Payload          []byte
}

func NewMoqtObject(loc MoqtLocation, subGroupId uint64, ftn MoqtFullTrackName, publisherPriority uint8, objectForwardingPreference MoqtObjectForwardingPreference, objectStatus MoqtObjectStatus, extensionHeaders []MoqtKeyValuePair, payload []byte) (*MoqtObject, error) {
	// Assumes that all parameters are valid according to their types, created with NewTypeX() functions for self-implemented types.
	
	// Any object with a status code other than zero MUST have an empty payload. (10.2.1.1.)
	if objectStatus != Normal && len(payload) > 0 {
		return nil, MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: NewReasonPhrase("Objects with non-normal status must have an empty payload"),
		}
	}

	// Any Object with status Normal can have extension headers.
	// If an endpoint receives extension headers on Objects with status that is not Normal, it MUST close the session with a PROTOCOL_VIOLATION. (10.2.1.2.)
	if objectStatus != Normal && len(extensionHeaders) > 0 {
		return nil, MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: NewReasonPhrase("Objects with non-normal status must not have extension headers"),
		}
	}

	return &MoqtObject{
		Location:                   loc,
		SubgroupID:                 subGroupId,
		FullTrackName:              ftn,
		PublisherPriority:          publisherPriority,
		ObjectForwardingPreference: objectForwardingPreference,
		ObjectStatus:               objectStatus,
		ExtensionHeaders:           extensionHeaders,
		Payload:                    payload,
	}, nil
}

type MoqtObjectForwardingPreference int

const (
	Datagram MoqtObjectForwardingPreference = 0
	Subgroup MoqtObjectForwardingPreference = 1
)

type MoqtObjectStatus int

const (
	Normal       MoqtObjectStatus = 0x0
	DoesNotExist MoqtObjectStatus = 0x1
	EndOfGroup   MoqtObjectStatus = 0x3
	EndOfTrack   MoqtObjectStatus = 0x4
)
