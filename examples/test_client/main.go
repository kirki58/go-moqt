package main

import (
	"go-moq"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
)
func main() {
    client := moqt.Client{}

    // Ensure the Connect function uses the updated quicConf with MaxIncomingStreams!
    conn, err := client.Connect("moqt://localhost:4443")
    if err != nil {
        panic(err)
    }

    sess, err := client.InitiateSession(conn, []model.MoqtKeyValuePair{
        internal.Must(model.NewMoqtKeyValuePair(control.SetupParamMaxRequestID, uint64(100))),
    })
    if err != nil {
        panic(err)
    }
    _ = sess
    // Keep the client alive long enough to receive a response or for the server to process it
    // In a real app, you'd be reading from the stream or using a context.
    select {} 
}