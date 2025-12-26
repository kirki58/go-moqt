package transport

import (
	"context"
	"io"

	"github.com/quic-go/quic-go"
)

// Stream is a bidirectional stream interface that matches both quic.Stream and webtransport.Stream
type Stream interface {
	ReceiveStream
	SendStream
}

// ReceiveStream is a unidirectional stream for reading only
type ReceiveStream interface {
	io.Reader
	CancelRead(quic.StreamErrorCode) // Transport layer, tell the publisher to shut up! (Sends a STOP_SENDING frame)
}

// WriteStream is a unidirectional stream for writing only
type SendStream interface {
	io.Writer
	io.Closer // Gracefully closes the stream, I'm done sending. (Sends a STREAM frame with the FIN bit set)
	CancelWrite(quic.StreamErrorCode)
}

// MOQTConnection abstracts the underlying transport (QUIC or WebTransport)
type MOQTConnection interface {
	OpenStream() (Stream, error)
	OpenStreamSync(context.Context) (Stream, error)
	OpenUniStream() (SendStream, error)
	OpenUniStreamSync(context.Context) (SendStream, error) // The endpoint that opens a unidirectional stream is the one that writes to it. So this function must open a SendStream
	AcceptStream(context.Context) (Stream, error)
	AcceptUniStream(context.Context) (ReceiveStream, error) // Receiver accepts unistream (ReceiveStream)
	IsWebTransport() bool // Returns true if the underlying transport is WebTransport false if QUIC
	CloseWithError(uint64 , string) error // Terminates the session with the given error information
	Context() context.Context // Returns a context that lives throughout the connection (until it's closed)
	RemoteHost() string // Returns the remote host address
}