package integration_test

import (
	"context"
	"fmt"
	"go-moq/client"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"go-moq/server"
	"testing"
	"time"
)

var maxIncomingReqIdClient uint64 = 100

func Test_Integration(t *testing.T) {
	srv := server.NewServer(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// It shouldn't take more than 5 seconds for the server to establish a Listener and bind to a port
	
	go RunServerOverQUIC(ctx, srv, "moqt://localhost") // Port will be assigned by OS if we don't specify it, we can be aware of the value in srv.Addr or srv.Port

	select {
	case <-srv.Ready:
		// The server is up.
	case <-time.After(5 * time.Second):
		t.Fatal("Server took too long to start")
	}

	// Test Case 1: Handshake
	t.Run("Client/Initiate_Success", func(t *testing.T) {
		// Spin up a client
		client := client.Client{}
		bkgCtx := context.Background()
		dialCtx, dialCancel := context.WithTimeout(bkgCtx, 5*time.Second)
		defer dialCancel()

		conn, err := client.Connect(dialCtx, fmt.Sprintf("moqt://localhost:%d", srv.Port))

		// ASSERTION 1: Should successfully connect in the provided happy path
		if err != nil {
			t.Fatalf("Client Failed to connect to MoQT server at %s: %v", srv.Addr, err)
		}

		// Client initiates an handshake
		sessInitCtx, sessInitCancel := context.WithTimeout(bkgCtx, 20 *time.Second) // Ping frames are sent every 15 seconds, so 20 sec max for session init is ideal
		defer sessInitCancel()

		sess, err := client.InitiateSession(sessInitCtx, conn, []model.MoqtKeyValuePair{ // this is the client's view of the session
			internal.Must(model.NewMoqtKeyValuePair(control.SetupParamMaxRequestID, maxIncomingReqIdClient)),
		})

		// Assertion 2: Should successfully complete the handshake
		if err != nil {
			t.Fatalf("Client failed to initiate session: %v", err)
		}

		if sess == nil {
			t.Fatal("Session is nil after successful initiation")
		}

		// Assertion 3: Verify session state
		if sess.State.MaxOutgoingRequestID != 100 {
			t.Errorf("Expected MaxOutgoingRequestId %d, got %d", maxIncomingReqIdClient, sess.State.MaxOutgoingRequestID)
		}

		// Test Case 2

		t.Run("TerminateIfTerminationError_Sucess", func(t *testing.T) {
			terminationError := model.MOQT_SESSION_TERMINATION_ERROR{
				ReasonPhrase: "Test termination",
				ErrorCode: model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
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
	})

	t.Cleanup(func() {
		cancel()
	})
}
