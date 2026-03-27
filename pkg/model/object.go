package model

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

func (status MoqtObjectStatus) IsValid() bool{
	switch status {
	case Normal, DoesNotExist, EndOfGroup, EndOfTrack:
		return true
	default:
		return false
	}
}
