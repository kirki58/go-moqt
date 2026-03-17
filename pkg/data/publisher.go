package data

import (
	moqt "go-moq"
	"sync"
)

// Per-session publisher, deals with the data plane
// Publisher streams tracks to their subscribers
type Publisher struct{
	TrackRegistry moqt.TrackRegistry

	// Active Incoming Subscription by their Track alias
	SubInAliasesMutex sync.Mutex
	ActiveIncomingSubscriptionAliases map[uint64]*Subscription

	// Active Incoming Subscription by their Full Track Name
	SubInNamesMutex sync.Mutex
	ActiveIncomingSubscriptionNames map[string]*Subscription
}