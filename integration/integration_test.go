package integration_test

import (
	"context"
	"fmt"
	"go-moq/h264"
	"go-moq/internal"
	"go-moq/peer"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"testing"
	"time"

	"go.uber.org/goleak"
)

var maxIncomingReqIdClient uint64 = 100
var seed1, seed2 uint64 = 2, 4
var trackSize, groupSize uint16= 1800, 60

func Test_Integration(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Create server and manage it's tracks
	srv := peer.NewServer(peer.NewTrackRegistry())
	ftn := internal.Must(model.StringToMoqtFullTrackName("test/track"))
	disp := session.NewDispatcher(&ftn)
	srv.TrackRegistry.AddTrack(&ftn, disp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// It shouldn't take more than 5 seconds for the server to establish a Listener and bind to a port

	go runServerOverQUIC(ctx, srv, "moqt://localhost") // Port will be assigned by OS if we don't specify it, we can be aware of the value in srv.Addr or srv.Port

	select {
	case <-srv.Ready:
		// The server is up.
	case <-time.After(5 * time.Second):
		t.Fatal("Server took too long to start")
	}

	// Test Case 1: Handshake
	t.Run("Client/Connection_Establishment_And_Handshake_Success", func(t *testing.T) {

		var sess *session.Session = setupSession(t, srv)

		t.Run("ValidSubscribe_ReceivesSubscribeOk", func(t *testing.T) {
			// Encoder starts to produce and dispatch (via the given dispatcher) objects
			enc := h264.NewMockEncoder(disp)
			go enc.Encode(seed1, seed2, trackSize, groupSize, &ftn)

			// subscriber sends a SUBSCRIBE message, which shoudl trigger the server's registered message handler
			filter := model.NewNextGroupStartFilter(model.MoqtLocation{GroupId: 0, ObjectId: 0})
			var filterBytesArr [8]byte
			filterBytes := filterBytesArr[:0] 
			message.EncodeSubscriptionFilter(&filterBytes, filter)
			// location is just a place holder, publisher.JoinEstablishedSubscription will determine the correct start location on publisher side
			// subscriber can determine the start location after receiving a SUBSCRIBE_OK message with LARGEST_OBJECT parameter attached.
			subMsg := control.SubscribeMessage{
				RequestId: 0, // First request-initiating message for the client-side starts at id 0
				FullTrackName: &ftn,
				Parameters: []model.MoqtKeyValuePair{
					internal.Must(model.NewMoqtKeyValuePair(control.ParamSubscriptionFilter, filterBytes)),
				},
			}
			err := sess.Cmf.WriteControlMessage(&subMsg) // Send the subscribe control message over the control stream
			if err != nil{
				t.Errorf("SUBSCRIBE message couldn't be sent over the control stream: %v", err)
			}

			subOk, err := sess.Cmf.ReadControlMessage()
			if err != nil{
				t.Errorf("Valid SUBSCRIBE message couldn't receive back a control message: %v", err)
			}
			subOkMsg, ok := subOk.(*control.SubscribeOkMessage)
			if !ok{
				t.Errorf("Expected SUBSCRIBE_OK message but got: %v", subOk.Type())
			}

			// Search for largest object parameter only
			didFindOne := false
			// var largestObjLoc model.MoqtLocation
			for i := 0; i < len(subOkMsg.Parameters); i++ {
				if subOkMsg.Parameters[i].Type == control.ParamLargestObject{
					if didFindOne{
						t.Error("SUBSCRIBE_OK message got duplicate LARGEST_OBJECT parameters")
					}
					didFindOne = true
					_ , n, err := message.DecodeMoqtLocation(subOkMsg.Parameters[i].ValueBytes)
					if err != nil{
						t.Errorf("Failed to parse LARGEST_OBJECT parameter after reading %d bytes, error: %v", n, err)
					}
					// largestObjLoc = largestObj
				}
			}

			if !didFindOne{
				t.Error("SUBSCRIBE_OK received had no LARGEST_OBJECT parameter attached")
			}

			// sub := session.Subscription{
			// 	ID: subOkMsg.RequestId,
			// 	Alias: subOkMsg.TrackAlias,
			// 	FullTrackName: &ftn,
			// 	Filter: *model.NewNextGroupStartFilter(model.MoqtLocation{GroupId: largestObjLoc.GroupId + 1, ObjectId: 0}),
			// 	Status: session.SubscriptionStatusEstablished,
			// 	Parameters: subMsg.Parameters,
			// }
		})

		// Test Case 2: TerminateIfTerminationError

		t.Run("TerminateIfTerminationError_Wrapped_Error_Terminates", func(t *testing.T) {
			terminationError := model.MOQT_SESSION_TERMINATION_ERROR{
				ReasonPhrase: "Test termination",
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			}
			sess.TerminateIfTerminationError(fmt.Errorf("Wrapped error with internal session termination error: %w", &terminationError))

			// Assertion 1
			// sess.Conn.Context() should be closed
			select {
			case <-sess.Conn.Context().Done():
				// Success: Connection context is closed
			case <-time.After(2 * time.Second):
				t.Error("Session connection context was not closed after termination error")
			}
		})

		// Test Case 3: Client is unable to write on terminated connection's control stream
		t.Run("Client_ReadControlMessage_Terminated_Session_Fails", func(t *testing.T) {
			// Client tries to send control message
			cMsg := control.SubscribeMessage{
				RequestId: 1,
				FullTrackName: &model.MoqtFullTrackName{
					Namespace: [][]byte{[]byte("test")},
					Name:      []byte("track"),
				},
				Parameters: []model.MoqtKeyValuePair{},
			}
			err := sess.Cmf.WriteControlMessage(&cMsg)
			fmt.Printf("Sucessfully received error after trying to write to terminated control stream %v\n", err)
			if err == nil {
				t.Error("Expected error when writing to a terminated session's control stream, but got nil")
			}
		})
	})

	t.Cleanup(func() {
		cancel()
	})
}
