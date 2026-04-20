package main

import (
	"context"
	"fmt"
	"go-moq/internal"
	"go-moq/peer"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"log"
	"os"
	"strconv"
	"time"
)

func setupSession() *session.Session{
	port, err := strconv.ParseUint(os.Args[1], 10, 16)
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}

	// Spin up a client
	client := peer.Client{}
	bkgCtx := context.Background()
	dialCtx, dialCancel := context.WithTimeout(bkgCtx, 5*time.Second)
	defer dialCancel()

	conn, err := client.Connect(dialCtx, fmt.Sprintf("moqt://localhost:%d", port))

	// ASSERTION 1: Should successfully connect in the provided happy path
	if err != nil {
		log.Fatalf("Client Failed to connect to MoQT server at %s: %v", fmt.Sprintf("moqt://localhost:%d", port), err)
	}

	// Client initiates an handshake
	sessInitCtx, sessInitCancel := context.WithTimeout(bkgCtx, 20*time.Second) // Ping frames are sent every 15 seconds, so 20 sec max for session init is ideal
	defer sessInitCancel()

	sess, err := client.InitiateSession(sessInitCtx, conn, []model.MoqtKeyValuePair{ // this is the client's view of the session
		internal.Must(model.NewMoqtKeyValuePair(control.SetupParamMaxRequestID, uint64(100))),
	})

	// Assertion 2: Should successfully complete the handshake
	if err != nil {
		log.Fatalf("Client failed to initiate session: %v", err)
	}

	// Assertion 3: Verify session state
	if sess.State.MaxOutgoingRequestID != 100 {
		log.Fatalf("Max Outgoing Request Id mismatch")
	}

	fmt.Printf("[CLIENT] Session sucessfully initiated with %s\n", sess.Conn.RemoteHost())

	// Register SUBSCRIBE_OK message handler
	sess.RegisterHandler(control.SUBSCRIBE_OK, session.MessageHandler(client.HandleSubscribeOk))
	sess.RegisterHandler(control.PUBLISH_DONE, session.MessageHandler(client.HandlePublishDone))
	go sess.RunControlLoop()
	go sess.Subscriber.ListenAndRedispatch()

	return sess
}

func main() {
	ftn := internal.Must(model.StringToMoqtFullTrackName("test/track"))
	sess := setupSession()

	// Send a SUBSCRIBE message with NexGroupStart Filter

	// Create and encode the filter
	filter := model.NewNextGroupStartFilter(model.MoqtLocation{GroupId: 0, ObjectId: 0})
	var filterBytesArr [8]byte
	filterBytes := filterBytesArr[:0]
	message.EncodeSubscriptionFilter(&filterBytes, filter)

	// Create and write the Subscribe message
	subMsg := control.SubscribeMessage{
		RequestId:     0, // First request-initiating message for the client-side starts at id 0
		FullTrackName: &ftn,
		Parameters: []model.MoqtKeyValuePair{
			internal.Must(model.NewMoqtKeyValuePair(control.ParamSubscriptionFilter, filterBytes)),
		},
	}
	err := sess.Cmf.WriteControlMessage(&subMsg) // Send the subscribe control message over the control stream
	if err != nil {
		log.Fatalf("SUBSCRIBE message couldn't be sent over the control stream: %v", err)
	}

	// Increment the next outgoing request id counter
	sess.State.RequestIDMutex.Lock()
	sess.State.NextOutgoingRequestID++
	sess.State.RequestIDMutex.Unlock()

	// Session's subscriber registers a pending subscription until it receives a SUBSCRIBE_OK message
	_ = sess.Subscriber.NewSubscription(subMsg.RequestId, &ftn, *filter, subMsg.Parameters)

	// After SUBSCRIBE_OK, subscription should be registered in subscriber's active subscriptions cache
	time.Sleep(50 * time.Millisecond)

	sub, ok := sess.Subscriber.ActiveOutgoingSubscriptionIDs[0]
	if !ok {
		log.Fatalf("Subscription with Request ID 0 is not found in subscriber's ActiveOutgoingSubscriptionIDs after waiting 50 ms to receive SUBSCRIBE_OK")
	}

	dataPipe, err := os.OpenFile("/tmp/h264_stream", os.O_WRONLY, 0600)

	if err != nil {
		log.Fatalf("Failed to open data pipe for writing: %v", err)
		defer dataPipe.Close()
	}
	
	Loop:
		for {
			select {
			case obj, ok := <-sub.ObjectReceiveChannel:
				if !ok { // EOT, subscription terminated
					fmt.Println("Subscription ended (EOT)")
					break Loop
				}
				
				dataPipe.Write(obj.Payload)
			case <-time.After(5 * time.Second):
				log.Fatal("Data plane timed out")
			case <-sess.Conn.Context().Done():
				log.Fatal("Session context done before receiving all objects")
			}
		}
	
	fmt.Println("Client finished receiving objects. Closing session.")
	sess.Conn.CloseWithError(0x0, "Client done")
}
