package moqt

import (
	"context"
	"crypto/tls"
	"fmt"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"go-moq/pkg/session/control"
	"go-moq/pkg/transport"
	moqtquic "go-moq/pkg/transport/quic" // this alias is important to prevent confusion with "quic-go"
	"net/url"
	"time"

	"github.com/quic-go/quic-go"
)

const transportDialTimeout = 30       // time (seconds) limit to get a response to quic or WT dial.
const transportOpenStreamTimeout = 10 // time limit to a request of opening a stream being accpeted.
const maxUniStreams = 100             // Maximum number of concurrent unidirectional streams (incoming, because it's usually the server opening uni streams)
const defaultMaxIncomingRequestId = 1000
const defaultMaxLocalTokenCacheSize = 0

// MOQT Client functionality

type Client struct {
	// Right now this is empty, but it might be needed in the future to hold things like auth tokens etc.
}

// Establish a transport with the fiven URI
func (c *Client) Connect(uri string) (transport.MOQTConnection, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Client.Connect(): Failed to parse URI: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transportDialTimeout)*time.Second)
	defer cancel()

	switch u.Scheme {
	case "moqt": // QUIC Connection
		// Configure TLS with the correct ALPN (moqt-15) [Cite: Section 3.1]
		tlsConf := &tls.Config{
			NextProtos: []string{"moqt-15"},
		}

		quicConf := &quic.Config{
			EnableDatagrams:       true,          // The QUIC Datagram extension MUST be supported. [Cite: Section 3.1]
			MaxIncomingStreams:    1,             // Only 1 bidirectional stream allowed that is the control stream.
			MaxIncomingUniStreams: maxUniStreams, // Temporary hard limit.
		}

		// 2. Dial the QUIC Connection
		// If port is missing, default to 443 [cite: Section 3.1.2]
		addr := u.Host
		if u.Port() == "" {
			addr = u.Host + ":443"
		}

		qConn, err := quic.DialAddr(ctx, addr, tlsConf, quicConf)
		if err != nil {
			return nil, fmt.Errorf("Client.Connect(): Failed to dial QUIC connection: %w", err)
		}
		return &moqtquic.Connection{Conn: qConn}, nil

	case "https": // WebTransport Connection
		// TODO: Implement webtransport connection
		// The client performs an HTTP/3 CONNECT request. It MUST include the header WT-Available-Protocols: moqt-15
		return nil, nil

	default:
		return nil, fmt.Errorf("Client.Connect(): Unsupported URI scheme: %s", u.Scheme)
	}
}

// Performs a handshake in the given session
// 1. Sends CLIENT_SETUP message on the control stream
// 2. Waits for SERVER_SETUP message
func (c *Client) performHandshake(sess *session.Session, setupParams []model.MoqtKeyValuePair) error {
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

	// 1. Send CLIENT_SETUP message
	csMsg := control.ClientSetupMessage{
		Parameters: setupParams,
	}

	err := sess.Cmf.WriteControlMessage(&csMsg)

	if err != nil {
		return fmt.Errorf("Client.performHandshake(): Failed to send CLIENT_SETUP message: %w", err)
	}

	// 2. Wait for SERVER_SETUP message
	msg, err := sess.Cmf.ReadControlMessage()
	if err != nil {
		return fmt.Errorf("Client.performHandshake(): Failed to read SERVER_SETUP message: %w", err)
	}
	fmt.Printf("[DEBUG] Received message from the control stream: %#v\n", msg)

	serverSetupMsg, ok := msg.(*control.ServerSetupMessage)
	if !ok {
		return model.MOQT_SESSION_TERMINATION_ERROR{
			ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: model.NewReasonPhrase("Expected SERVER_SETUP message after sending CLIENT_SETUP, got something else."),
		}
	}

	// The server should not include "Authority" or "Param" in no way possible.
	for _, param := range serverSetupMsg.Parameters {
		// TODO: Implement MALFORMED_PATH and MALFORMED_AUTHORITY errors later.
		if param.Type == control.SetupParamPath {
			return model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_INVALID_PATH,
				ReasonPhrase: model.NewReasonPhrase("SERVER_SETUP message MUST NOT include Path parameter."),
			}

		} else if param.Type == control.SetupParamAuthority {
			return model.MOQT_SESSION_TERMINATION_ERROR{
				ErrorCode:    model.MOQT_SESSION_TERMINATION_ERROR_CODE_INVALID_AUTHORITY,
				ReasonPhrase: model.NewReasonPhrase("SERVER_SETUP message MUST NOT include Authority parameter."),
			}
		}
	}

	// Populate session state's peer values from obtained parameters in SERVER_SETUP
	sess.State.FromParams(serverSetupMsg.Parameters)
	return nil
}

// Initiate a MOQT session on top of the given transport.
// Opens a bidirectional control stream
// Creates session struct
// Performs handshake

func (c *Client) InitiateSession(conn transport.MOQTConnection, setupParams []model.MoqtKeyValuePair) (*session.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transportOpenStreamTimeout)*time.Second)
	defer cancel()

	// The first stream opened is a client-initiated bidirectional control stream where the endpoints exchange Setup messages (Section 9.3), followed by other messages defined in Section 9.
	s, err := conn.OpenStreamSync(ctx) // Open the bidirectional control stream.
	if err != nil {
		return nil, err // err, here might be a general error or a MOQT_SESSION_TERMINATION_ERROR if the given connection already has a control stream open. The caller of this function should handle that.
		// ex: if MOQT_SESSION_TERMINATION_ERROR_CODE == MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION // caller can distinguish between general error and a protocol violation error
	}

	sess := &session.Session{
		State:         session.NewSessionState(session.RoleClient, defaultMaxIncomingRequestId, defaultMaxLocalTokenCacheSize),
		Conn:          conn,
		ControlStream: s,
		Cmf:           control.NewControlMessageFactory(s),
	}

	err = c.performHandshake(sess, setupParams)
	if err != nil {
		return nil, err // err might be a general error or a MOQT_SESSION_TERMINATION_ERROR. The caller of this function should handle that.
	}
	return sess, nil
}
