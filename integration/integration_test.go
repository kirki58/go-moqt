package integration_test

import (
	"context"
	"go-moq/h264"
	"go-moq/internal"
	"go-moq/peer"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"testing"
	"time"
)

var maxIncomingReqIdClient uint64 = 100
var seed1, seed2 uint64 = 2, 4
var trackSize, groupSize uint16 = 1800, 60

func Test_Integration(t *testing.T) {
	// defer goleak.VerifyNone(t)

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
			enc := h264.NewMockEncoder(disp, trackSize, groupSize, &ftn)
			go enc.Encode()

			// subscriber sends a SUBSCRIBE message, which shoudl trigger the server's registered message handler
			filter := model.NewNextGroupStartFilter(model.MoqtLocation{GroupId: 0, ObjectId: 0})
			var filterBytesArr [8]byte
			filterBytes := filterBytesArr[:0]
			message.EncodeSubscriptionFilter(&filterBytes, filter)
			// location is just a place holder, publisher.JoinEstablishedSubscription will determine the correct start location on publisher side
			// subscriber can determine the start location after receiving a SUBSCRIBE_OK message with LARGEST_OBJECT parameter attached.
			subMsg := control.SubscribeMessage{
				RequestId:     0, // First request-initiating message for the client-side starts at id 0
				FullTrackName: &ftn,
				Parameters: []model.MoqtKeyValuePair{
					internal.Must(model.NewMoqtKeyValuePair(control.ParamSubscriptionFilter, filterBytes)),
				},
			}
			err := sess.Cmf.WriteControlMessage(&subMsg) // Send the subscribe control message over the control stream
			if err != nil {
				t.Errorf("SUBSCRIBE message couldn't be sent over the control stream: %v", err)
			}

			sess.State.RequestIDMutex.Lock()
			sess.State.NextOutgoingRequestID++
			sess.State.RequestIDMutex.Unlock()

			_ = sess.Subscriber.NewSubscription(subMsg.RequestId, &ftn, *filter, subMsg.Parameters)

			// After SUBSCRIBE_OK, subscription should be registered in subscriber cache
			time.Sleep(50 * time.Millisecond)
			sub, ok := sess.Subscriber.ActiveOutgoingSubscriptionIDs[0]
			if !ok {
				t.Errorf("Subscription with Request ID 0 is not found in subscriber's ActiveOutgoingSubscriptionIDs after waiting 50 ms to receive SUBSCRIBE_OK")
			}

			t.Run("AfterSubscribeOk_StartReceivingObjects", func(t *testing.T) {
				t.Run("FirstObjectReceived_IsGroupStart", func(t *testing.T) {
					select {
					case obj := <-sub.ObjectReceiveChannel:
						if obj.Location.ObjectId != 0 {
							t.Errorf("Expected first object to have ObjectId 0 (NextGroupStart), got %d", obj.Location.ObjectId)
						}
					case <-time.After(5 * time.Second):
						t.Fatal("Timed out waiting for the first object from the subscriber channel")
					}
				})
			})
		})

		// // Test Case 2: TerminateIfTerminationError

		// t.Run("TerminateIfTerminationError_Wrapped_Error_Terminates", func(t *testing.T) {
		// 	terminationError := model.MOQT_SESSION_TERMINATION_ERROR{
		// 		ReasonPhrase: "Test termination",
		// 		ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
		// 	}
		// 	sess.TerminateIfTerminationError(fmt.Errorf("Wrapped error with internal session termination error: %w", &terminationError))

		// 	// Assertion 1
		// 	// sess.Conn.Context() should be closed
		// 	select {
		// 	case <-sess.Conn.Context().Done():
		// 		// Success: Connection context is closed
		// 	case <-time.After(2 * time.Second):
		// 		t.Error("Session connection context was not closed after termination error")
		// 	}
		// })

		// // Test Case 3: Client is unable to write on terminated connection's control stream
		// t.Run("Client_ReadControlMessage_Terminated_Session_Fails", func(t *testing.T) {
		// 	// Client tries to send control message
		// 	cMsg := control.SubscribeMessage{
		// 		RequestId: 1,
		// 		FullTrackName: &model.MoqtFullTrackName{
		// 			Namespace: [][]byte{[]byte("test")},
		// 			Name:      []byte("track"),
		// 		},
		// 		Parameters: []model.MoqtKeyValuePair{},
		// 	}
		// 	err := sess.Cmf.WriteControlMessage(&cMsg)
		// 	fmt.Printf("Sucessfully received error after trying to write to terminated control stream %v\n", err)
		// 	if err == nil {
		// 		t.Error("Expected error when writing to a terminated session's control stream, but got nil")
		// 	}
		// })
	})

	t.Cleanup(func() {
		cancel()
	})
}
