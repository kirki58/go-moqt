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

	// Since this is a request initiating control message we MUST validate and increment the next expected requestid from the peer
	// This may result in a session termination
	if err := sess.ValidateAndIncrementIncomingRequestId(subMsg.RequestId); err != nil {
		return err
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
				return fmt.Errorf("Failed to parse subscription filter after parsing %d bytes, error: %v", n, err) // might or might not be a termination error, RunControlLoop will act accordingly.
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
	sub, err := sess.Publisher.NewSubscription(subMsg.RequestId, subMsg.FullTrackName, filter, subMsg.Parameters, disp)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("Failed to establish subscription after sending SUBSCRIBE_OK message, error: %v", err)
	}

	return nil
}
