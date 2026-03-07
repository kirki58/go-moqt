package server

import (
	"fmt"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
)

func (s *Server) handleSubscribe(sess *session.Session, msg control.ControlMessage) error{
	// Cast into concrete message type
	subMsg, ok := msg.(*control.SubscribeMessage)
	if !ok {
		return fmt.Errorf("expected SubscribeMessage, got %T", msg)
	}
	
	// Since this is a request initiating control message we MUST validate and increment the next expected requestid from the peer
	// This may result in a session termination
	sess.TerminateIfTerminationError(sess.ValidateAndIncrementIncomingRequestId(subMsg.RequestId))

	// Check if the TrackRegistry has the requested track
	if !s.trackRegistry.HasTrack(*subMsg.FullTrackName) {
		err := model.MOQT_REQUEST_ERROR{
			ErrorCode:    model.MOQT_REQUEST_ERROR_CODE_DOES_NOT_EXIST,
			ReasonPhrase: model.NewReasonPhrase("Track does not exist"),
		}
		reqErrSubMsg := &control.RequestErrorMessage{
			RequestId:   subMsg.RequestId,
			ErrorCode:   err.ErrorCode,
			ErrorReason: err.ReasonPhrase,
		}
		
		sess.Cmf.WriteControlMessage(reqErrSubMsg)
		return err
	}

	// Check if a duplicate subscription (to the same track) exists in this session
	ftnKey := subMsg.FullTrackName.ToString()
	if _, exists := sess.ActiveIncomingSubscriptionNames[ftnKey]; exists {
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Duplicate subscription to tracks are not allowed triggered by '%s'", ftnKey)),
		}
		sess.TerminateIfTerminationError(err)
		return err
	}

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
	sess.ActiveIncomingSubscriptionNames[subscription.FullTrackName.ToString()] = subscription
	sess.ActiveIncomingSubscriptionAliases[subscription.TrackAlias] = subscription

	// Send SUBSCRIBE_OK
	subOkMsg := &control.SubscribeOkMessage{
		RequestId:  subMsg.RequestId,
		TrackAlias: trackAlias,
		Parameters: []model.MoqtKeyValuePair{}, // TODO: Determine LARGEST_OBJECT for the track and put it as a parameter here
	}

	// data.StartStream(filters-bla-bla)

	if err := sess.Cmf.WriteControlMessage(subOkMsg); err != nil {
		return fmt.Errorf("failed to send SUBSCRIBE_OK: %w", err)
	}

	return nil
}