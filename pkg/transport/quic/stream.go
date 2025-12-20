package moqtquic // Package name is not made "quic" in order to prevent confusion with "quic-go" "quic" package.

import "github.com/quic-go/quic-go" // Concrete stream implementations for

// "transport.Stream", "transport.SendStream", "transport.ReceiveStream" are provided with quic here.

type SendStream struct {
	stream *quic.SendStream
}

// moqtquic.SendStream implements transport.SendStream

// io.Writer implementation
func (s *SendStream) Write(p []byte) (n int, err error) {
	return s.stream.Write(p)
}

// io.Closer implementation
func (s *SendStream) Close() error {
	return s.stream.Close()
}

// CancelWrite implementation
func (s *SendStream) CancelWrite(code quic.StreamErrorCode) {
	s.stream.CancelWrite(code)
}

type ReceiveStream struct {
	stream *quic.ReceiveStream
}

// moqtquic.ReceiveStream implements transport.ReceiveStream

// io.Reader implementation
func (s *ReceiveStream) Read(p []byte) (n int, err error) {
	return s.stream.Read(p)
}

// CancelRead implementation
func (s *ReceiveStream) CancelRead(code quic.StreamErrorCode) {
	s.stream.CancelRead(code)
}

type Stream struct {
	stream *quic.Stream
}

// moqtquic.Stream implements transport.Stream

// io.Reader implementation
func (s *Stream) Read(p []byte) (n int, err error) {
	return s.stream.Read(p)
}

// CancelRead implementation
func (s *Stream) CancelRead(code quic.StreamErrorCode) {
	s.stream.CancelRead(code)
}

// io.Writer implementation
func (s *Stream) Write(p []byte) (n int, err error) {
	return s.stream.Write(p)
}

// io.Closer implementation
func (s *Stream) Close() error {
	return s.stream.Close()
}

// CancelWrite implementation
func (s *Stream) CancelWrite(code quic.StreamErrorCode) {
	s.stream.CancelWrite(code)
}