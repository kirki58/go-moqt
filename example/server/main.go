package main

import (
	"context"
	"fmt"
	"go-moq/h264"
	"go-moq/internal"
	"go-moq/peer"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"log"
	"time"
)

func runServerOverQUIC(ctx context.Context, srv *peer.Server, addr string) {
	connsCh := make(chan transport.MOQTConnection, 100) // 100 connections can handshake at the same time without blocking the accepting of the new connections.

	// 1. The server "accept connections loop" in it's dedicated goroutine
	go func() {
		defer close(connsCh)
		// accepted connections are pushed on connsCh to be consumed
		err := srv.Run(ctx, addr, "./local_certs/localhost.pem", "./local_certs/localhost-key.pem", connsCh)
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

			// Register SUBSCRIBE message handler for server's session
			sess.RegisterHandler(control.SUBSCRIBE, session.MessageHandler(srv.HandleSubscribe))

			// Control loop and data loop needs to inherently respect sess.Conn.Context
			// Run control loop
			go sess.RunControlLoop()
			// Run data loop
			//...
		}(conn)
	}
}
func main() {
	// Create a server and manage it's tracks
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
		log.Fatal("Server took too long to start")
	case <-ctx.Done():
		log.Fatal("Context cancelled before server was ready")
	}

	// Encoder starts to produce and dispatch (via the given dispatcher) objects
	enc := h264.NewH264Encoder(disp, &ftn)
	go enc.Encode()

	<-ctx.Done()
}
