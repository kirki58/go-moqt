package moqtquic

import (
	"context"
	"fmt"
	"go-moq/pkg/transport"

	"github.com/quic-go/quic-go"
)

type Connection struct{
	conn *quic.Conn
}

// moqtquic.Connection implements transport.Connection

func (c *Connection) OpenStream() (transport.Stream, error){
	s, err := c.conn.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("moqtquic.OpenStream():\n\t Failed to open stream:\n\t: %w", err)
	}

	return &Stream{ // keep in mind that this is of type moqtquic.Stream
		stream: s,
	}, nil
}

func (c *Connection) OpenStreamSync(ctx context.Context) (transport.Stream, error){
	s, err := c.conn.OpenStreamSync(ctx) // Internally handles mutex lock!
	if err != nil {
		return nil, fmt.Errorf("moqtquic.OpenStreamSync():\n\t Failed to open stream:\n\t: %w", err)
	}

	return &Stream{ // keep in mind that this is of type moqtquic.Stream
		stream: s,
	}, nil
}

func (c *Connection) OpenUniStream() (transport.SendStream, error){
	s, err := c.conn.OpenUniStream()
	if err != nil {
		return nil, fmt.Errorf("moqtquic.OpenUniStream():\n\t Failed to open unistream:\n\t: %w", err)
	}

	return &SendStream{ // keep in mind that this is of type moqtquic.SendStream
		stream: s,
	}, nil
}

func (c *Connection) OpenUniStreamSync(ctx context.Context) (transport.SendStream, error){
	s, err := c.conn.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("moqtquic.OpenUniStreamSync():\n\t Failed to open unistream:\n\t: %w", err)
	}

	return &SendStream{ // keep in mind that this is of type moqtquic.SendStream
		stream: s,
	}, nil
}

func (c *Connection) AcceptStream(ctx context.Context) (transport.Stream, error){
	s, err := c.conn.AcceptStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("moqtquic.AcceptStream():\n\t Failed to accept stream:\n\t: %w", err)
	}

	return &Stream{ // keep in mind that this is of type moqtquic.Stream
		stream: s,
	}, nil
}

func (c *Connection) AcceptUniStream(ctx context.Context) (transport.ReceiveStream, error){
	s, err := c.conn.AcceptUniStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("moqtquic.AcceptUniStream():\n\t Failed to accept unistream:\n\t: %w", err)
	}

	return &ReceiveStream{ // keep in mind that this is of type moqtquic.ReceiveStream
		stream: s,
	}, nil
}