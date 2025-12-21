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
	NegotiatedExtensions map[uint64]bool
}

type Session struct {
	Conn          *transport.MOQTConnection
	controlStream *transport.Stream

	cmf *control.ControlMessageFactory // This is so that the session is able to read and write control messages

	State *SessionState

	trackAliases map[uint64]model.MoqtFullTrackName
}

func NewSession(conn *transport.MOQTConnection, state *SessionState) *Session {
	return &Session{
		Conn:  conn,
		State: state,
	}
}

// // Handshake performs the MOQT handshake.
// func (s *Session) Handshake(ctx context.Context) error {
// 	if s.isServer {
// 		return s.serverHandshake(ctx)
// 	}
// 	return s.clientHandshake(ctx)
// }

// func (s *Session) clientHandshake(ctx context.Context) error {
// 	// The client opens a bidirectional stream, which is called the control stream.
// 	stream, err := s.conn.OpenStreamSync(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to open control stream: %w", err)
// 	}
// 	s.controlStream = stream

// 	csMsg := &control.ClientSetupMessage{
// 		Parameters: []model.MoqtKeyValuePair{},
// 	}

// 	payload, err := csMsg.Encode()
// 	if err != nil {
// 		return fmt.Errorf("failed to encode CLIENT_SETUP: %w", err)
// 	}

// 	if err := s.writeControlMessage(stream, csMsg.Type(), payload); err != nil {
// 		return fmt.Errorf("failed to write CLIENT_SETUP: %w", err)
// 	}

// 	// Wait for SERVER_SETUP
// 	return s.waitForServerSetup(stream)
// }

// func (s *Session) serverHandshake(ctx context.Context) error {
// 	// The server accepts a bidirectional stream.
// 	stream, err := s.conn.AcceptStream(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to accept control stream: %w", err)
// 	}
// 	s.controlStream = stream

// 	// Wait for CLIENT_SETUP
// 	if err := s.waitForClientSetup(stream); err != nil {
// 		return err
// 	}

// 	// Send SERVER_SETUP
// 	ssMsg := &control.ServerSetupMessage{
// 		Parameters: []model.MoqtKeyValuePair{},
// 	}
// 	payload, err := ssMsg.Encode()
// 	if err != nil {
// 		return fmt.Errorf("failed to encode SERVER_SETUP: %w", err)
// 	}

// 	if err := s.writeControlMessage(stream, ssMsg.Type(), payload); err != nil {
// 		return fmt.Errorf("failed to write SERVER_SETUP: %w", err)
// 	}

// 	return nil
// }

// func (s *Session) writeControlMessage(w io.Writer, msgType control.ControlMessageType, payload []byte) error {
// 	// Format: Type(i) Length(i) Payload(...)
// 	buf := make([]byte, 0, 16+len(payload))
// 	buf = quicvarint.Append(buf, uint64(msgType))
// 	buf = quicvarint.Append(buf, uint64(len(payload)))
// 	buf = append(buf, payload...)

// 	_, err := w.Write(buf)
// 	return err
// }

// func (s *Session) waitForServerSetup(r io.Reader) error {
// 	// We need to use valid bufio.Reader.
// 	// The factory creates its own bufio.Reader from the io.Reader we pass.
// 	// NOTE: If we reuse the factory for subsequent messages, we must reuse the same factory instance/bufio.Reader
// 	// because bufio.Reader buffers data. Creating a new factory for every message is dangerous if they share the same underlying stream!
// 	// For this task "Only go as far as exchanging control setup messages", it's fine to create one factory here.
// 	// But in a real session we should store the factory or reader in the session struct.

// 	factory := control.NewControlMessageFactory(r)
// 	msg, err := factory.ReadControlMessage()
// 	if err != nil {
// 		return fmt.Errorf("failed to read control message: %w", err)
// 	}

// 	switch m := msg.(type) {
// 	case *control.ServerSetupMessage:
// 		// TODO: Validate parameters
// 		_ = m
// 		return nil
// 	default:
// 		return fmt.Errorf("expected SERVER_SETUP, got %d", msg.Type())
// 	}
// }

// func (s *Session) waitForClientSetup(r io.Reader) error {
// 	factory := control.NewControlMessageFactory(r)
// 	msg, err := factory.ReadControlMessage()
// 	if err != nil {
// 		return fmt.Errorf("failed to read control message: %w", err)
// 	}

// 	switch m := msg.(type) {
// 	case *control.ClientSetupMessage:
// 		// TODO: Validate parameters
// 		_ = m
// 		return nil
// 	default:
// 		return fmt.Errorf("expected CLIENT_SETUP, got %d", msg.Type())
// 	}
// }
