package integration_test

import (
	"context"
	"fmt"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"go-moq/server"
	"log"
	"time"
)

func RunServerOverQUIC(ctx context.Context, srv *server.Server, addr string){
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

			// Run control loop
			go sess.RunControlLoop()
			// Run data loop
			//...
		}(conn)
	}
}