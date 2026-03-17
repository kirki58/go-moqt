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
		RequestId: 0,
		FullTrackName: &model.MoqtFullTrackName{
			Namespace: model.MoqtTrackNamespace{[]byte("field1"), []byte("field2")}, // existing track
			Name:      []byte("track"),
		},
		Parameters: []model.MoqtKeyValuePair{},
	}
	sess.Cmf.WriteControlMessage(&subMsg)

	fmt.Print("Written subscribe message to the control stream")

	// Wait for a response from the control stream:
	msg, err := sess.Cmf.ReadControlMessage()
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
	} else {
		fmt.Printf("Received response from server: %#v\n", msg)
	}

	subMsg2 := control.SubscribeMessage{
		RequestId: 0,
		FullTrackName: &model.MoqtFullTrackName{
			Namespace: model.MoqtTrackNamespace{[]byte("non"), []byte("existing")}, // non-existent track
			Name:      []byte("track"),
		},
		Parameters: []model.MoqtKeyValuePair{},
	}
	sess.Cmf.WriteControlMessage(&subMsg2)

	// Wait for a response from the control stream:
	msg, err = sess.Cmf.ReadControlMessage()
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
	} else {
		fmt.Printf("Received response from server: %#v\n", msg)
	}

	// Keep the client alive
	select {}
}
