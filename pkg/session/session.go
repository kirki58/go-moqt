package session

import (
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

	// MaxOutgoingRequestID is the limit the PEER has imposed on US.
	// We cannot send a request if NextOutgoingRequestID >= MaxOutgoingRequestID.
	// Initialized from the `MAX_REQUEST_ID` param in the received SETUP message.
	// Updated via received MAX_REQUEST_ID control messages.
	MaxOutgoingRequestID uint64

	// MaxIncomingRequestID is the limit WE have imposed on the PEER.
	// If the peer sends a request with ID >= this value, we close the session with TOO_MANY_REQUESTS.
	// We send updates to this value via MAX_REQUEST_ID control messages.
	MaxIncomingRequestID uint64

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
func NewSessionState(localRole Role, maxIncomingRequestId uint64, localTokenCacheSize uint64) *SessionState{
	state := &SessionState{
		LocalRole:             localRole,
		NextOutgoingRequestID: uint64(localRole), // Client starts at 0, Server at 1
		MaxIncomingRequestID:  maxIncomingRequestId,
		LocalTokenCacheSize:   localTokenCacheSize,
	}
	return state
}

// Populates session state's peer values (not local) with given setup parameters from the peer
func (state *SessionState) FromParams(params []model.MoqtKeyValuePair){
	for _, param := range params{
		switch param.Type{
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

	trackAliases map[uint64]model.MoqtFullTrackName
}

