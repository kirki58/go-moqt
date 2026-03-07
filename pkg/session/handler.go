package session

import (
	"go-moq/pkg/session/control"
)

// SessionHandler defines the behavior for processing a specific control message.
// Implementations should be stateless and safe for concurrent use if the session handles messages concurrently.
type Handler interface {
	// Handle processes the incoming control message.
	//
	// Arguments:
	//   sess: The session receiving the message.
	//   msg:  The decoded control message.
	//
	// Returns:
	//   error: If an error is returned, the caller (RunControlLoop) should decide whether
	//          to log it or terminate the session. Protocol violations should generally
	//          result in sess.CloseWithError() being called internally before returning.
	Handle(sess *Session, msg control.ControlMessage) error
}

type SubscribeHandler func(sess *Session, msg control.ControlMessage) error

func (f SubscribeHandler) Handle(sess *Session, msg control.ControlMessage) error {
	return f(sess, msg)
}