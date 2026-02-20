package loop

import (
	"fmt"
	moqt "go-moq"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/session/control/handler"
	"time"
)

const generalControlMessageTimeout time.Duration = 15 * time.Second

func RunControlLoop(sess *session.Session, trackRegistry moqt.TrackRegistry){
	// Constantly read on the control stream, call the appropriate handlers upon receiving messages
	// Act upon the control message type and contents
	
	for{
		cMsg, err := sess.Cmf.ReadControlMessage()
		if err != nil{
			if sess.TerminateIfTerminationError(err) {
				return
			}
			// Non-termination error, just log and continue
			fmt.Printf("Error reading control message: %v\n", err)
			continue
		}

		switch cMsg.Type(){
		case control.SUBSCRIBE:
			subMsg := cMsg.(*control.SubscribeMessage)
			subHandler := &handler.SubscribeHandler{}
			go subHandler.Handle(sess, subMsg, trackRegistry)
		}
	}
}