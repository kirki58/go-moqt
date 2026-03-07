package main

import (
	"fmt"
	"go-moq/client"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"time"

	"golang.org/x/net/context"
)

func main() {
	client := client.Client{MaxIncomingUniStreamsPerConn: 100}

	bkgContext := context.Background()

	dialUpCtx, cancel := context.WithTimeout(bkgContext, 5*time.Second)
	defer cancel()

	conn, err := client.Connect(dialUpCtx, "moqt://localhost:4443")
	if err != nil {
		panic(err)
	}

	sessInitCtx, cancel := context.WithTimeout(bkgContext, 5*time.Second)
	defer cancel()

	sess, err := client.InitiateSession(sessInitCtx, conn, []model.MoqtKeyValuePair{
		internal.Must(model.NewMoqtKeyValuePair(control.SetupParamMaxRequestID, uint64(100))),
	})
	if err != nil {
		panic(err)
	}

	// Receive the "Welcome" object datagram from the server
	receiveWelcomeCtx, cancel := context.WithTimeout(bkgContext, 5*time.Second)
	defer cancel()
	objDg, err := sess.ReceiveObjectDatagram(receiveWelcomeCtx)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Received Object Datagram from the server: %+v Payload to string is: %s\n", objDg ,string(objDg.Payload.Val))

	// Keep the client alive
	select {}
}
