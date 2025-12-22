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
func (c *Client) performHandshake(sess *session.Session, setupParams []model.MoqtKeyValuePair) error{ 
	// Validate the parameters
	// If the underlying transport is WebTransport, than Setup params PATH and AUTHORITY must be omitted!
	if sess.Conn.IsWebTransport(){
		for _, param := range setupParams{
			if param.Type == control.SetupParamPath || param.Type == control.SetupParamAuthority{
				return fmt.Errorf("Setup parameters are not valid for the handshake: When using WebTransport, Setup Parameters PATH and AUTHORITY must be omitted")
			}
		}
	}

	// TODO: Implement the rest of the handshake
}

// Initiate a MOQT session on top of the given transport.
// Opens a bidirectional control stream
// Creates session struct
// Performs handshake

func (c *Client) InitiateSession(conn transport.MOQTConnection, setupParams []model.MoqtKeyValuePair) (*session.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transportOpenStreamTimeout)*time.Second)
	defer cancel()

	s, err := conn.OpenStreamSync(ctx) // Open the bidirectional control stream.
	if err != nil{
		return nil, err // err, here might be a general error or a MOQT_SESSION_TERMINATION_ERROR if the given connection already has a control stream open. The caller of this function should handle that.
		// ex: if MOQT_SESSION_TERMINATION_ERROR_CODE == MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION // callar can distinguish between general error and a protocol violation error
	}

	sess := &session.Session{
		Conn: conn,
		ControlStream: s,
		Cmf: control.NewControlMessageFactory(s),
	}

	// TODO: Implement the rest of the session initiation
}