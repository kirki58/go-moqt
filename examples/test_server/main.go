package main

import (
	"context"
	"fmt"
	moqt "go-moq"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"go-moq/server"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	srv := server.Server{
		TrackRegistry: moqt.NewSimpleTrackRegistry(),
	}
	srv.TrackRegistry.AddTrack(model.MoqtFullTrackName{
		Namespace: model.MoqtTrackNamespace{[]byte("field1"), []byte("field2")},
		Name:      []byte("track"),
	})

	// 1. Run the server in a separate goroutine
	connsCh := make(chan transport.MOQTConnection, 100) // 100 connections can handshake at the same time without blocking the accepting of the new connections.

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		defer close(connsCh)
		err := srv.Run(ctx, "moqt://localhost:4443", "../../local_certs/localhost.pem", "../../local_certs/localhost-key.pem", connsCh)
		if err != nil {
			if err == context.Canceled {
				return
			}
			panic(err)
		}
	}()
	fmt.Println("Server is running on moqt://localhost:4443")

	// 3. Consume connections and decide how to handle them
	for conn := range connsCh {
		// fmt.Println("Accepted new MOQT connection:", conn)

		// Note: can decide what to do with connection based on different features of the connection such as remote host address, webtransport or quic etc.

		// Spawn a handler for this specific connection
		go func(c transport.MOQTConnection) {
			timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // give 5 seconds max. for session initializiton
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

			// Register subscribe message handler
			subHandler := session.SubscribeHandler(srv.HandleSubscribe)
			sess.RegisterHandler(control.SUBSCRIBE, subHandler)
			fmt.Print("Subscribe handler registered for the session\n")

			// Run control loop
			sess.RunControlLoop()
		}(conn)
	}
}
