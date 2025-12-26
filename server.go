package moqt

import (
	"context"
	"crypto/tls"
	"fmt"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	moqtquic "go-moq/pkg/transport/quic"
	"net/url"
	"time"

	"github.com/quic-go/quic-go"
)

const serverDefaultMaxIncomingRequestId = 1000
const serverDefaultMaxLocalTokenCacheSize = 0

type Server struct {
	MaxUniStreamsPerConn        int
	WaitForControlStreamTimeout time.Duration
}

// Starts a while-true loop that accepts connections, sends accepted connection over the channel to get handled by the caller
// Run starts the listener and pushes accepted connections to the connCh.
// It blocks until the listener closes or a fatal error occurs.
func (s *Server) Run(ctx context.Context, uri string, certFile string, keyFile string, connCh chan<- transport.MOQTConnection) error { // ctx is the parent context, likely would be a context.Background()
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("failed to parse URI: %w", err)
	}

	switch u.Scheme {
	case "moqt": // QUIC Connection
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		tlsConf := &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"moqt-15"},
		}
		quicConf := &quic.Config{
			EnableDatagrams:       true,
			MaxIncomingStreams:    1,
			MaxIncomingUniStreams: int64(s.MaxUniStreamsPerConn),
		}

		addr := u.Host
		if u.Port() == "" {
			addr = u.Host + ":443"
		}

		listener, err := quic.ListenAddr(addr, tlsConf, quicConf)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}

		fmt.Printf("[INFO] Listening for QUIC on %s\n", addr)

		for {
			qConn, err := listener.Accept(ctx)
			if err != nil {
				// Log and continue, or return if it's a permanent error
				fmt.Printf("[WARN] Accepting connection failed: %v\n", err)
				continue
			}

			// Wrap the raw QUIC connection in the MOQT adapter
			moqtConn := &moqtquic.Connection{Conn: qConn}

			// Send to the caller.
			// This will BLOCK if the caller is too slow and the channel is full.
			select {
			case connCh <- moqtConn:
				// Successfully handed off
			case <-ctx.Done():
				return ctx.Err()
			}
		}

	case "https":
		// WebTransport implementation would go here
		// It would need an http.Handler that pushes to connCh
		return nil

	default:
		return nil
	}
}

func (s *Server) InitateSession(parentCtx context.Context, conn transport.MOQTConnection, setupParams []model.MoqtKeyValuePair) (*session.Session, error) {
	// Accept the Control Stream
	// The Draft-15 spec requires the Client to open this stream immediately after the connection is established
	ctx, cancel := context.WithTimeout(parentCtx, s.WaitForControlStreamTimeout)
	defer cancel()

	stream, err := conn.AcceptStream(ctx) // Accept client-initiated control stream.
	if err != nil {
		fmt.Printf("[WARN]: Server.InitiateSession(): Failed to accept control stream from %s: %v\n", conn.RemoteHost(), err)
		return nil, fmt.Errorf("Server.InitiateSession(): Failed to accept control stream from %s: %w", conn.RemoteHost(), err)
	}
	fmt.Printf("[INFO]: Server.InitiateSession(): Accepted control stream from client: %s\n", conn.RemoteHost())

	sess := &session.Session{
		State:         session.NewSessionState(session.RoleServer, serverDefaultMaxIncomingRequestId, serverDefaultMaxLocalTokenCacheSize),
		Conn:          conn,
		ControlStream: stream,
		Cmf:           control.NewControlMessageFactory(stream),
	}

	err = s.performHandshake(sess, setupParams)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Server) performHandshake(sess *session.Session, setupParams []model.MoqtKeyValuePair) error { // setup params we
	// This specification only specifies two uses of bidirectional streams, the control stream, which begins with CLIENT_SETUP, and SUBSCRIBE_NAMESPACE. Bidirectional streams
	// MUST NOT begin with any other message type unless negotiated. If they do, the peer MUST close the Session with a Protocol Violation.
	// Objects are sent on unidirectional streams. [Cite: Section 3.3]

	// Validate the parameters
	// If the underlying transport is WebTransport, than Setup params PATH and AUTHORITY must be omitted!
	if sess.Conn.IsWebTransport() {
		for _, param := range setupParams {
			if param.Type == control.SetupParamPath || param.Type == control.SetupParamAuthority {
				return fmt.Errorf("Setup parameters are not valid for the handshake: When using WebTransport, Setup Parameters PATH and AUTHORITY must be omitted")
			}
		}
	}

	// Populate the session state's local values with given parameters
	for _, param := range setupParams {
		switch param.Type {
		case control.SetupParamPath:
			sess.State.Path = string(param.ValueBytes)
		case control.SetupParamAuthority:
			sess.State.Authority = string(param.ValueBytes)
		case control.SetupParamMaxRequestID:
			sess.State.MaxIncomingRequestID = param.ValueUInt64
		case control.SetupParamMaxAuthTokenCacheSize:
			sess.State.LocalTokenCacheSize = param.ValueUInt64
		default:
			continue
		}
	}

	// Wait for CLIENT_SETUP message
	msg, err := sess.Cmf.ReadControlMessage()
	if err != nil {
		return fmt.Errorf("Server.performHandshake(): Failed to read CLIENT_SETUP message: %w", err)
	}
	fmt.Printf("[DEBUG] Receiver message from the control stream: %#v\n", msg)

	clientSetupMsg, ok := msg.(*control.ClientSetupMessage)
	if !ok {
		return model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase("Expected CLIENT_SETUP message, got something else."),
		}
	}

	// Populate session state's peer values from obtained parameters in CLIENT_SETUP
	sess.State.FromParams(clientSetupMsg.Parameters)

	// Send SERVER_SETUP message
	ssMsg := control.ServerSetupMessage{
		Parameters: setupParams,
	}

	err = sess.Cmf.WriteControlMessage(&ssMsg)
	if err != nil {
		return fmt.Errorf("Server.performHandshake(): Failed to send SERVER_SETUP message: %w", err)
	}
	return nil
}
