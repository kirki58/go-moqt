package session

import (
	"go-moq/pkg/model"
)

// SubscriptionStatus represents the lifecycle states of a subscription as defined in Section 5.1.
type SubscriptionStatus int

const (
	SubscriptionStatusIdle        SubscriptionStatus = iota // Initial state.
	SubscriptionStatusPending                               // SUBSCRIBE/PUBLISH sent, waiting for OK/ERROR.
	SubscriptionStatusEstablished                           // Handshake successful, data can flow.
	SubscriptionStatusTerminated                            // Subscription ended or rejected.
)

// Subscription represents the state of a single subscription within the session.
type Subscription struct {
	ID uint64 // The Request ID used in the SUBSCRIBE/PUBLISH message.

	Alias uint64

	FullTrackName *model.MoqtFullTrackName

	Filter model.SubscriptionFilter

	Status SubscriptionStatus

	Parameters []model.MoqtKeyValuePair

	Publisher         *Publisher               // Publisher for this subscription
	Dispatcher        *Dispatcher              // the track dispatcher this subscription was issued for
	DispatcherChannel <-chan *model.MoqtObject // The channel publisher use to receive objects from subscribed track's object dispatcher It's important that this is a buffered channel
	// Note: closing this channel is the responsibility of the
}
