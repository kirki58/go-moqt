package integration_test

import (
	"context"
	"fmt"
	"go-moq/client"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"go-moq/server"
	"log"
	"testing"
	"time"
)

func runServerOverQUIC(ctx context.Context, srv *server.Server, addr string){
	connsCh := make(chan transport.MOQTConnection, 100) // 100 connections can handshake at the same time without blocking the accepting of the new connections.

	// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // real-world implementations would use this context
	// defer stop()

	// 1. The server "accept connections loop" in it's dedicated goroutine
	go func() {
		defer close(connsCh)
		// accepted connections are pushed on connsCh to be consumed
		err := srv.Run(ctx, addr, "../local_certs/localhost.pem", "../local_certs/localhost-key.pem", connsCh)
		if err != nil {
			if err == context.Canceled {
				return
			}
			panic(err)
		}
	}()

	// 2. The consumer loop should spawn dedicated handlers goroutines for each connection established and handshake them
	for conn := range connsCh {
		// Spawn a handler for this specific connection
		go func(c transport.MOQTConnection) {
			timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // give 5 seconds max. for session initializiton, this can be made dynamic later
			defer cancel()
			sess, err := srv.InitateSession(timeCtx, c, []model.MoqtKeyValuePair{
				internal.Must(model.NewMoqtKeyValuePair(control.SetupParamMaxRequestID, uint64(100))),
			})
			if err != nil {
				log.Printf("Failed to initiate session with %s, Closing the connection: %v", c.RemoteHost(), err)
				c.CloseWithError(0x0, "Handshake failed")
				return
			}
			fmt.Printf("Session initiated with %s\n", sess.Conn.RemoteHost())

			// Control loop and data loop needs to inherently respect sess.Conn.Context
			// Run control loop
			go sess.RunControlLoop()
			// Run data loop
			//...
		}(conn)
	}
}

func setupSession(t *testing.T, srv *server.Server) (*session.Session){
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

		return sess
}