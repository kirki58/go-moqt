package integration_test

import (
	"context"
	"fmt"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/server"
	"testing"
	"time"

	"go.uber.org/goleak"
)

var maxIncomingReqIdClient uint64 = 100

func Test_Integration(t *testing.T) {
	defer goleak.VerifyNone(t)

	srv := server.NewServer(nil)
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
					Name: []byte("track"),
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
