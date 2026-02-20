package session

import "go-moq/pkg/model"

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

	FullTrackName *model.MoqtFullTrackName

	// TrackAlias is the session-specific identifier for the track (negotiated in SUBSCRIBE_OK or PUBLISH).
	TrackAlias uint64

	Status SubscriptionStatus

	Parameters []model.MoqtKeyValuePair
}
