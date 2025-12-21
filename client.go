package moqt

import (
	"context"
	"crypto/tls"
	"fmt"
	"go-moq/pkg/session"
	"go-moq/pkg/transport"
	moqtquic "go-moq/pkg/transport/quic" // this alias is important to prevent confusion with "quic-go"
	"net/url"
	"time"

	"github.com/quic-go/quic-go"
)

var transportDialTimeout int = 30 // time (seconds) limit to get a response to quic or WT dial.

// MOQT Client functionality

type Client struct{
	// Active sessions
	sessions map[string]*session.Session // string represents the connection URI
}

func (c *Client) Connect(uri string) (transport.MOQTConnection, error){
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Client.Connect(): Failed to parse URI: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transportDialTimeout)*time.Second)
	defer cancel()

	switch u.Scheme{
	case "moqt": // QUIC Connection
		// Configure TLS with the correct ALPN (moqt-15) [Cite: Section 3.1]
		tlsConf := &tls.Config{
			NextProtos: []string{"moqt-15"},
		}

		quicConf := &quic.Config{
			EnableDatagrams: true, // The QUIC Datagram extension MUST be supported. [Cite: Section 3.1]
		}

		// 2. Dial the QUIC Connection
        // If port is missing, default to 443 [cite: Section 3.1.2]
        addr := u.Host
        if u.Port() == "" {
            addr = u.Host + ":443"
        }

		qConn, err := quic.DialAddr(ctx, addr, tlsConf, quicConf)
		if err != nil{
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