package session

import (
	"context"
	"errors"
	"fmt"
	"go-moq/pkg/data"
	"go-moq/pkg/message"
	"go-moq/pkg/model"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	"sync"
)

// Role defines whether we are the Client or Server.
type Role int

const (
	RoleClient Role = iota
	RoleServer
)

type SessionState struct {
	// --- Static Configuration (Set at initialization) ---

	// Role determines Request ID numbering (Client=Even, Server=Odd).
	LocalRole Role

	// --- Negotiated Setup Parameters (From Setup Handshake) ---

	// PeerImplementation stores the "MOQT_IMPLEMENTATION" string sent by the peer.
	// Useful for logging and debugging interoperability.
	PeerImplementation string

	// Path received in CLIENT_SETUP (Server-side only).
	// Used for routing or validation. MUST be ignored if transport is WebTransport.
	Path string

	// Authority received in CLIENT_SETUP (Server-side only).
	// Used for virtual hosting. MUST be ignored if transport is WebTransport.
	Authority string

	// --- Flow Control & Limits (Dynamic State) ---

	// RequestIDMutex protects the request ID counters below.
	RequestIDMutex sync.Mutex

	// NextOutgoingRequestID is the next ID we will assign to a new request we send.
	// Client starts at 0 (increments by 2), Server starts at 1 (increments by 2).
	NextOutgoingRequestID uint64

	// NextIncomingRequestID is the next ID we **EXPECT** to receive from the other peer
	// If the remote peer is client NextIncomingRequestID should always be even and it should be odd otherwise
	NextIncomingRequestID uint64

	// MaxOutgoingRequestID is the limit the PEER has imposed on US.
	// We cannot send a request if NextOutgoingRequestID >= MaxOutgoingRequestID.
	// Initialized from the `MAX_REQUEST_ID` param in the received SETUP message.
	// Updated via received MAX_REQUEST_ID control messages.
	MaxOutgoingRequestID uint64

	// MaxIncomingRequestID is the limit WE have imposed on the PEER.
	// If the peer sends a request with ID >= this value, we close the session with TOO_MANY_REQUESTS.
	// We send updates to this value via MAX_REQUEST_ID control messages.
	MaxIncomingRequestID uint64

	// Track Alias Generation
	NextTrackAlias uint64
	AliasMutex     sync.Mutex

	// --- Authorization State ---

	// PeerMaxTokenCacheSize is the limit of token data the PEER is willing to store.
	// We must track the size of active tokens we've registered to avoid AUTH_TOKEN_CACHE_OVERFLOW errors.
	PeerMaxTokenCacheSize uint64

	// LocalTokenCacheSize is the limit of token data WE are willing to store.
	// We use this to validate incoming REGISTER token requests from the peer.
	LocalTokenCacheSize uint64

	// --- Extension State ---

	// Extensions stores which optional features were successfully negotiated.
	// MOQT Draft-15 currently defines no specific extensions, but the handshake supports them.
	// NegotiatedExtensions map[uint64]bool
}

// Create a session state with local parameters
func NewSessionState(localRole Role, maxIncomingRequestId uint64, localTokenCacheSize uint64) *SessionState {
	state := &SessionState{
		LocalRole:             localRole,
		NextOutgoingRequestID: uint64(localRole), // Client starts at 0, Server at 1
		MaxIncomingRequestID:  maxIncomingRequestId,
		LocalTokenCacheSize:   localTokenCacheSize,
	}
	return state
}

// Populates session state's peer values (not local) with given setup parameters from the peer
func (state *SessionState) FromParams(params []model.MoqtKeyValuePair) {
	for _, param := range params {
		switch param.Type {
		case control.SetupParamMoqtImplementation:
			state.PeerImplementation = string(param.ValueBytes)
		case control.SetupParamPath:
			state.Path = string(param.ValueBytes)
		case control.SetupParamAuthority:
			state.Authority = string(param.ValueBytes)
		case control.SetupParamMaxRequestID:
			state.MaxOutgoingRequestID = param.ValueUInt64
		case control.SetupParamMaxAuthTokenCacheSize:
			state.PeerMaxTokenCacheSize = param.ValueUInt64
		default:
			continue // Unknown parameter type, just ignore

			// TODO: Implement the handling AuthToken setup parameter later on.
		}
	}
}

type Session struct {
	Conn          transport.MOQTConnection
	ControlStream transport.Stream

	Cmf *control.ControlMessageFactory // This is so that the session is able to read and write control messages

	State *SessionState
	
	Handlers map[control.ControlMessageType]Handler
}

// Checks for given error (unwraps wrapped errors sequentially with errors.As()), if it's a type of termination error it will terminate the session and the underlying transport
// Returns true if the session and transport is terminated, false if they are still alive.
func (sess *Session) TerminateIfTerminationError(err error) bool {
	var terminationError *model.MOQT_SESSION_TERMINATION_ERROR
	if errors.As(err, &terminationError) {
		// Error is a session termination error
		// Terminate the session, kill the transport layer (if it exists)
		if sess.Conn != nil {
			sess.Conn.CloseWithError(uint64(terminationError.ErrorCode), string(terminationError.ReasonPhrase))
		}

		return true
	}
	return false
}

func (sess *Session) RegisterHandler(msgType control.ControlMessageType, handler Handler) {
	sess.Handlers[msgType] = handler
}

func (sess *Session) RunControlLoop() {
	// Constantly read on the control stream, call the appropriate handlers upon receiving messages
	// Act upon the control message type and contents

	for {
		cMsg, err := sess.Cmf.ReadControlMessage()
		if err != nil {
			if sess.TerminateIfTerminationError(err) {
				return
			}
			// Non-termination error, just log and continue
			fmt.Printf("Error reading control message: %v\n", err)
			continue
		}
		fmt.Printf("[DEBUG] Received message from the control stream: %#v\n", cMsg)

		msgType := cMsg.Type()

		// Below listed message types belong to request initiator messages which increment the next expected request id counter for the peer by 2 when received.
		// Also upon sending one of these messages a peer must increment his own next request id tracker by 2

		// Request ID validity checks for the "RequestID" field they have should be performed in their handler implementation!
		// A ValidateRequestId function is provided below for this purpose
		// SUBSCRIBE
		// FETCH
		// PUBLISH
		// TRACK_STATUS
		// PUBLISH_NAMESPACE
		// SUBSCRIBE_NAMESPACE 
		// SUBSCRIBE_UPDATE

		handler, ok := sess.Handlers[msgType] // Check registered handlers for this message type
		if !ok {                                  // Message type is not supported by the peer
			// INTERNAL_ERROR termination error because the received control message is unsupported
			err := model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_INTERNAL_ERROR,
				ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("No handler for Control Message Type: %#X", cMsg.Type())),
			}
			sess.TerminateIfTerminationError(err)
			return
		}

		if err := handler.Handle(sess, cMsg); err != nil { // might run in a separate goroutine later
			if sess.TerminateIfTerminationError(err) {
				return
			}
		}
	}
}

// Validate the request id upon receiving request initiator control messages
func (sess *Session) ValidateAndIncrementIncomingRequestId(reqId uint64) error {
	sess.State.RequestIDMutex.Lock()
	defer sess.State.RequestIDMutex.Unlock()

	// Expected reqid check:
	// this also ensures the parity correctness
	// If we are the server, we expect even request ids from the client
	// if we are the client we expect odd request ids from the server

	if reqId != sess.State.NextIncomingRequestID{
		return model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_INVALID_REQUEST_ID,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Request ID %d does not match expected NextIncomingRequestID %d", reqId, sess.State.NextIncomingRequestID)),
		}
	}

	// Check if the received reqId exceeds the determined max request id for peers self.
	if reqId >= sess.State.MaxIncomingRequestID {
		return model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_TOO_MANY_REQUESTS,
			ReasonPhrase: model.NewReasonPhrase(fmt.Sprintf("Request ID %d exceeds MaxIncomingRequestID %d", reqId, sess.State.MaxIncomingRequestID)),
		}
	}

	// Increment to the next expected request id
	sess.State.NextIncomingRequestID += 2

	return nil
}

func (sess *Session) SendObjectDatagram(obj *data.ObjectDatagram) error {
	dgBuf := make([]byte, 0, 256) // Initial capacity of 256 bytes

	message.EncodeObjectDatagram(&dgBuf, obj)

	return sess.Conn.SendDatagram(dgBuf)
}

func (sess *Session) ReceiveObjectDatagram(ctx context.Context) (*data.ObjectDatagram, error) {
	msgBytes, err := sess.Conn.ReceiveDatagram(ctx)
	if err != nil {
		return nil, err
	}

	objDg, _, err := message.DecodeObjectDatagram(msgBytes)

	if err != nil {
		return nil, err
	}

	return objDg, nil
}
