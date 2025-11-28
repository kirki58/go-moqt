package model

type MoqtLocation struct {
	GroupId  uint64
	ObjectId uint64
}

// LessThan implements the official MOQT Location comparison. [cite: 269, 270]
func (locA MoqtLocation) LessThan(locB MoqtLocation) bool {
	// A < B if A.GroupId < B.GroupId
	if locA.GroupId < locB.GroupId {
		return true
	}
	// If GroupIds are equal, A < B if A.ObjectId < B.ObjectId
	if locA.GroupId == locB.GroupId && locA.ObjectId < locB.ObjectId {
		return true
	}
	return false
}

// Equal implements the official MOQT Location equality check. [cite: 269, 270]
func (locA MoqtLocation) Equal(locB MoqtLocation) bool {
	return locA.GroupId == locB.GroupId && locA.ObjectId == locB.ObjectId
}

// GreaterThan implements the official MOQT Location comparison. [cite: 269, 270]
func (locA MoqtLocation) GreaterThan(locB MoqtLocation) bool {
	// A > B if A.GroupId > B.GroupId
	if locA.GroupId > locB.GroupId {
		return true
	}
	// If GroupIds are equal, A > B if A.ObjectId > B.ObjectId
	if locA.GroupId == locB.GroupId && locA.ObjectId > locB.ObjectId {
		return true
	}
	return false
}