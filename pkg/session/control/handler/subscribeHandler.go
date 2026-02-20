package handler

import (
	"fmt"
	moqt "go-moq"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
)

// SubscribeHandler handles incoming SUBSCRIBE messages.

type SubscribeHandler struct{}

func (h *SubscribeHandler) Handle(sess *session.Session, subMsg *control.SubscribeMessage, trackRegistry moqt.TrackRegistry) error {
	// Check if the TrackRegistry has the requested track
	if !trackRegistry.HasTrack(*subMsg.FullTrackName) {
		err := model.MOQT_REQUEST_ERROR{
			ErrorCode:    model.MOQT_REQUEST_ERROR_CODE_DOES_NOT_EXIST,
			ReasonPhrase: model.NewReasonPhrase("Track does not exist"),
		}
		reqErrMsg := &control.RequestErrorMessage{
			RequestId:   subMsg.RequestId,
			ErrorCode:   err.ErrorCode,
			ErrorReason: err.ReasonPhrase,
		}
		
		sess.Cmf.WriteControlMessage(reqErrMsg)
		return err
	}

	// Check if a duplicate subscription (to the same track) exists in this session
	ftnKey := subMsg.FullTrackName.ToString()
	if _, exists := sess.ActiveSubscriptionNames[ftnKey]; exists {
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Duplicate subscription to tracks are not allowed triggered by '%s'", ftnKey)),
		}
		sess.TerminateIfTerminationError(err)
		return err
	}

	// Validate RequestId against session's MAX_REQUEST_ID
	sess.State.RequestIDMutex.Lock()
	if subMsg.RequestId >= sess.State.MaxIncomingRequestID {
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_TOO_MANY_REQUESTS,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Request ID %d exceeds MaxIncomingRequestID %d", subMsg.RequestId, sess.State.MaxIncomingRequestID)),
		}
		sess.State.RequestIDMutex.Unlock()
		sess.TerminateIfTerminationError(err)
		return err
	}
	sess.State.RequestIDMutex.Unlock()

	// Generate Track Alias
	sess.State.AliasMutex.Lock()
	trackAlias := sess.State.NextTrackAlias
	sess.State.NextTrackAlias++
	sess.State.AliasMutex.Unlock()

	// Create and establish the subscription
	subscription := &session.Subscription{
		ID:            subMsg.RequestId,
		FullTrackName: subMsg.FullTrackName,
		TrackAlias:    trackAlias,
		Status:        session.SubscriptionStatusEstablished,
		Parameters:    subMsg.Parameters,
	}

	// Store subscription state in the session
	sess.ReceivingSubscriptions[subscription.ID] = subscription
	sess.ActiveSubscriptionNames[ftnKey] = subscription.ID

	// Send SUBSCRIBE_OK
	subOkMsg := &control.SubscribeOkMessage{
		RequestId:  subMsg.RequestId,
		TrackAlias: trackAlias,
		Parameters: []model.MoqtKeyValuePair{}, // Could include LARGEST_OBJECT here if available
	}

	if err := sess.Cmf.WriteControlMessage(subOkMsg); err != nil {
		return fmt.Errorf("failed to send SUBSCRIBE_OK: %w", err)
	}

	return nil
}