package main

import (
	"go-moq"
	"go-moq/internal"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"time"

	"golang.org/x/net/context"
)
func main() {
    client := moqt.Client{MaxIncomingUniStreamsPerConn: 100}

    bkgContext := context.Background()

    dialUpCtx, cancel := context.WithTimeout(bkgContext, 5*time.Second)
    defer cancel()

    conn, err := client.Connect(dialUpCtx ,"moqt://localhost:4443")
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
    _ = sess
    // Keep the client alive long enough to receive a response or for the server to process it
    // In a real app, you'd be reading from the stream or using a context.
    select {} 
}