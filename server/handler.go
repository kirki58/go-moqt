package server

import (
	"fmt"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
)

// func (s *Server) HandleSubscribe(sess *session.Session, msg control.ControlMessage) error {
// 	// Cast into concrete message type
// 	subMsg, ok := msg.(*control.SubscribeMessage)
// 	if !ok {
// 		return fmt.Errorf("expected SubscribeMessage, got %T", msg)
// 	}

// 	// Since this is a request initiating control message we MUST validate and increment the next expected requestid from the peer
// 	// This may result in a session termination
// 	sess.TerminateIfTerminationError(sess.ValidateAndIncrementIncomingRequestId(subMsg.RequestId))

// 	// Check if the TrackRegistry has the requested track
// 	if subMsg.FullTrackName == nil {
// 		fmt.Println("[DEBUG] Received subscribe message with nil FullTrackName")
// 	}
// 	fmt.Print(subMsg.FullTrackName.ToString())
	
// 	if !s.TrackRegistry.HasTrack(*subMsg.FullTrackName) {
// 		err := model.MOQT_REQUEST_ERROR{
// 			ErrorCode:    model.MOQT_REQUEST_ERROR_CODE_DOES_NOT_EXIST,
// 			ReasonPhrase: model.NewReasonPhrase("Track does not exist"),
// 		}
// 		reqErrSubMsg := &control.RequestErrorMessage{
// 			RequestId:   subMsg.RequestId,
// 			ErrorCode:   err.ErrorCode,
// 			ErrorReason: err.ReasonPhrase,
// 		}

// 		sess.Cmf.WriteControlMessage(reqErrSubMsg)
// 		return err
// 	}

// 	// Check if a duplicate subscription (to the same track) exists in this session
// 	ftnKey := subMsg.FullTrackName.ToString()
// 	if _, exists := sess.ActiveIncomingSubscriptionNames[ftnKey]; exists {
// 		err := &model.MOQT_SESSION_TERMINATION_ERROR{
// 			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
// 			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Duplicate subscription to tracks are not allowed triggered by '%s'", ftnKey)),
// 		}
// 		sess.TerminateIfTerminationError(err)
// 		return err
// 	}

// 	// Generate Track Alias
// 	sess.State.AliasMutex.Lock()
// 	trackAlias := sess.State.NextTrackAlias
// 	sess.State.NextTrackAlias++
// 	sess.State.AliasMutex.Unlock()

// 	// Create and establish the subscription
// 	// subscription := &session.Subscription{
// 	// 	ID:            subMsg.RequestId,
// 	// 	FullTrackName: subMsg.FullTrackName,
// 	// 	TrackAlias:    trackAlias,
// 	// 	Status:        session.SubscriptionStatusEstablished,
// 	// 	Parameters:    subMsg.Parameters,
// 	// }

// 	// // Store subscription state in the session
// 	// sess.SubInNamesMutex.Lock()
// 	// sess.ActiveIncomingSubscriptionNames[subscription.FullTrackName.ToString()] = subscription
// 	// sess.SubInNamesMutex.Unlock()

// 	// sess.SubInAliasesMutex.Lock()
// 	// sess.ActiveIncomingSubscriptionAliases[subscription.TrackAlias] = subscription
// 	// sess.SubInAliasesMutex.Unlock()

// 	// data.GetTrackLargestObject...

// 	// Send SUBSCRIBE_OK
// 	subOkMsg := &control.SubscribeOkMessage{
// 		RequestId:  subMsg.RequestId,
// 		TrackAlias: trackAlias,
// 		Parameters: []model.MoqtKeyValuePair{}, // TODO: Determine LARGEST_OBJECT for the track and put it as a parameter here
// 	}

// 	// data.StartStream(filters-bla-bla)

// 	// data should have track structs that keeps of it's subscribers, largest object
// 	// data should have streamtrack functions that constantly streams track to subscribers

// 	if err := sess.Cmf.WriteControlMessage(subOkMsg); err != nil {
// 		return fmt.Errorf("failed to send SUBSCRIBE_OK: %w", err)
// 	}

// 	return nil
// }

func (s *Server) HandleSubscribe(sess *session.Session, msg control.ControlMessage) error {
	fmt.Printf("Received control message of type %T\n", msg)

	sMsg := msg.(*control.SubscribeMessage)
	if sMsg.RequestId == 0{
		fmt.Printf("Received invalid request id 0, terminating session\n")
		// termination error test
		err := model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase("Request ID 0 is invalid for client-initiated requests"),
		}
		sess.TerminateIfTerminationError(err)
		return err
		
	}

	return nil
}