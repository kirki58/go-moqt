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
	client := client.Client{}

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

	// Send subscribe message

	subMsg := control.SubscribeMessage{
		RequestId: 1,
		FullTrackName: &model.MoqtFullTrackName{
			Namespace: model.MoqtTrackNamespace{[]byte("field1"), []byte("field2")}, // existing track
			Name:      []byte("track"),
		},
		Parameters: []model.MoqtKeyValuePair{},
	}
	sess.Cmf.WriteControlMessage(&subMsg)

	fmt.Print("Written subscribe message (reqId = 1) to the control stream\n")

	subMsg2 := control.SubscribeMessage{
		RequestId: 0,
		FullTrackName: &model.MoqtFullTrackName{
			Namespace: model.MoqtTrackNamespace{[]byte("non"), []byte("existing")}, // non-existent track
			Name:      []byte("track"),
		},
		Parameters: []model.MoqtKeyValuePair{},
	}

	// wait for key signal
	var input string
	fmt.Print("Hit enter to send a subscribe message with invalid request id (reqId = 0) and non-existent track\n")
	fmt.Scanln(&input)

	sess.Cmf.WriteControlMessage(&subMsg2)

	fmt.Print("Written subscribe message (reqId = 1) to the control stream\n")

	// wait for key signal
	fmt.Print("Hit enter to send a subscribe message with invalid request id (reqId = 0) and non-existent track\n")
	fmt.Scanln(&input)

	sess.Cmf.WriteControlMessage(&subMsg2)

	fmt.Print("Written subscribe message (reqId = 1) to the control stream\n")

	// Keep the client alive
	select {}
}
