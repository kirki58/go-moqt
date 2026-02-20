package handler

import (
	"go-moq/pkg/session/control"
	"go-moq/pkg/session"
)

// SessionHandler defines the behavior for processing a specific control message.
// Implementations should be stateless and safe for concurrent use if the session handles messages concurrently.
type SessionHandler interface {
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
	Handle(sess *session.Session, msg control.ControlMessage) error
}
