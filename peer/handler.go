package peer

import (
	"fmt"
	"go-moq/internal"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
)

const OBJ_BUF_SIZE uint = 1024 

func (s *Server) HandleSubscribe(sess *session.Session, msg control.ControlMessage) error {
	// Cast into concrete message type
	subMsg, ok := msg.(*control.SubscribeMessage)
	if !ok {
		return fmt.Errorf("expected SubscribeMessage, got %T", msg)
	}

	// Check if the TrackRegistry has the requested track
	if subMsg.FullTrackName == nil {
		return fmt.Errorf("[DEBUG] Received subscribe message with nil FullTrackName for message with Request ID: %d\n", subMsg.RequestId)
	}

	disp, ok := s.TrackRegistry.GetTrack(subMsg.FullTrackName);
	if !ok {
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

	if _, exists := sess.Publisher.GetSubscriptionByName(subMsg.FullTrackName); exists {
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Duplicate subscription to tracks are not allowed triggered by '%s'", ftnKey)),
		}
		return err
	}

	// Search for subscription filters in parameters
	var didFindOne bool
	var filter model.SubscriptionFilter
	for i := 0; i < len(subMsg.Parameters); i++ {
		if subMsg.Parameters[i].Type == control.ParamSubscriptionFilter {
			// A duplicate provided, it's a protocol violation
			if didFindOne {
				err := &model.MOQT_SESSION_TERMINATION_ERROR{
					ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
					ReasonPhrase: model.MoqtReasonPhrase(fmt.Sprintf("Multiple subscription filters are provided in SUBSCRIBE message with RequestId: %d, Protocol Violation!\n", subMsg.RequestId)),
				}
				return err
			}
			didFindOne = true
			subFilter, n, err := message.DecodeSubscriptionFilter(subMsg.Parameters[i].ValueBytes)
			filter = *subFilter
			if err != nil{
				return fmt.Errorf("Failed to parse subscription filter after parsing %d bytes, error: %w", n, err) // might or might not be a termination error, RunControlLoop will act accordingly.
			}
		}
	}

	// No subsciription filters are provided in the message
	if !didFindOne{
		err := model.MOQT_REQUEST_ERROR{
			ErrorCode:    model.MOQT_REQUEST_ERROR_CODE_INTERNAL_ERROR,
			ReasonPhrase: model.NewReasonPhrase("Unfiltered subscriptions are not supported, pass in a subscription filter and try again"),
		}
		reqErrSubMsg := &control.RequestErrorMessage{
			RequestId:   subMsg.RequestId,
			ErrorCode:   err.ErrorCode,
			ErrorReason: err.ReasonPhrase,
		}

		sess.Cmf.WriteControlMessage(reqErrSubMsg)
		return err
	}

	// Create and establish the subscription
	// Register subscription
	sub := sess.Publisher.NewSubscription(subMsg.RequestId, subMsg.FullTrackName, filter, subMsg.Parameters, disp)

	objBufCh, dropNotCh := disp.RegisterNewSubChannel(sub, OBJ_BUF_SIZE)
	sub.DispatcherChannel = objBufCh
	sub.DropChannel = dropNotCh

	largestObjLoc, ok := sess.Publisher.LargestObjectLocation(sub)
	if !ok{
		return fmt.Errorf("Track was over before joining")
	}

	var locBytesArr [128]byte
	locBytes := locBytesArr[:0]
	message.EncodeMoqtLocation(&locBytes, *largestObjLoc)

	subOkMsg := &control.SubscribeOkMessage{
		RequestId:  subMsg.RequestId,
		TrackAlias: sub.Alias,
		Parameters: []model.MoqtKeyValuePair{internal.Must(model.NewMoqtKeyValuePair(control.ParamLargestObject, locBytes))},
	}

	if err := sess.Cmf.WriteControlMessage(subOkMsg); err != nil {
		return fmt.Errorf("failed to send SUBSCRIBE_OK: %w", err)
	}

	if err := sess.Publisher.JoinEstablishedSubscription(sub); err != nil{
		return fmt.Errorf("Failed to establish subscription after sending SUBSCRIBE_OK message, error: %w", err)
	}

	return nil
}

func (c *Client) HandleSubscribeOk(sess *session.Session, msg control.ControlMessage) error{
	subOkMsg, ok := msg.(*control.SubscribeOkMessage)
	if !ok{
		return fmt.Errorf("expected SubscribeOkMessage, got %T", msg)
	}

	// The identifer used for this track in Subgroups or Datagrams (see Section 10.1). The same Track Alias MUST NOT be used to refer to two different Tracks simultaneously.
	// If a subscriber receives a SUBSCRIBE_OK that uses the same Track Alias as a different track with an Established subscription, it MUST close the session with error DUPLICATE_TRACK_ALIAS.

	_ , ok = sess.Subscriber.GetSubscriptionByAlias(subOkMsg.TrackAlias)
	if ok{ // subscription already exists
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode: model.MOQT_SESSION_TERMINATION_ERROR_CODE_DUPLICATE_TRACK_ALIAS,
			ReasonPhrase: model.MoqtReasonPhrase(fmt.Sprintf("Subscriber received a SUBSCRIBE_OK for existing subscription to track alias %d", subOkMsg.TrackAlias)),
		}
		return err
	}

	// Look for largest object parameter
	var didFindOne bool
	var largestObj model.MoqtLocation
	for i := 0; i < len(subOkMsg.Parameters); i++ {
		if subOkMsg.Parameters[i].Type == control.ParamLargestObject {
			if didFindOne { // A duplicate provided, it's a protocol violation
				err := &model.MOQT_SESSION_TERMINATION_ERROR{
					ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
					ReasonPhrase: model.MoqtReasonPhrase(fmt.Sprintf("Multiple Largest Object parameters found in SUBSCRIBE_OK message for subscription with request id %d, Protocol violation!", subOkMsg.RequestId)),
				}
				return err
			}
			didFindOne = true
			largest, n, err := message.DecodeMoqtLocation(subOkMsg.Parameters[i].ValueBytes)
			largestObj = largest
			if err != nil{
				return fmt.Errorf("Failed to parse largest object parameter after parsing %d bytes, error: %w", n, err) // might or might not be a termination error, RunControlLoop will act accordingly.
			}
		}
	}

	if !didFindOne{
		err := &model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode: model.MOQT_SESSION_TERMINATION_ERROR_CODE_INTERNAL_ERROR,
			ReasonPhrase: model.MoqtReasonPhrase("This implementation expects LARGEST_OBJECT parameter in SUBSCRIBE_OK messages"),
		}
		return err
	}

	err := sess.Subscriber.ActivateSubscription(subOkMsg.RequestId, subOkMsg.TrackAlias, largestObj)
	if err != nil{
		return fmt.Errorf("SUBSCRIBE_OK handler could not activate pending subscription created by SUBSCRIBE message with request id: %d, error: %w", subOkMsg.RequestId, err)
	}

	return nil
}

func (c *Client) HandlePublishDone(sess *session.Session, msg control.ControlMessage) error{
	pubDoneMsg, ok := msg.(*control.PublishDoneMessage)
	if !ok{
		return fmt.Errorf("Expected PublishDoneMessage, got %T\n", msg)
	}

	sess.Subscriber.TerminateSubscription(pubDoneMsg.RequestId)

	fmt.Printf("Subscription ended with stream count %d, Reason: [%d], %v", pubDoneMsg.StreamCount, pubDoneMsg.StatusCode, pubDoneMsg.ErrorReason)
	return nil
}

func (c *Client) HandleRequestError(sess *session.Session, msg control.ControlMessage) error{
	reqErrMsg, ok := msg.(*control.RequestErrorMessage)
	if !ok {
		return fmt.Errorf("Expected RequestErrorMessage, got %T\n", msg)
	}

	fmt.Printf("Received Request Error for Request ID: %d, Error Code: %d, Reason: %s\n", reqErrMsg.RequestId, reqErrMsg.ErrorCode, reqErrMsg.ErrorReason)
	sess.Subscriber.CancelPendingSubscription(reqErrMsg.RequestId)
	return nil
}
