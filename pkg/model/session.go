package model

import (
	"context"
	"io"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
)

// MoqTransport defines the abstracted functions MOQT needs
// from its underlying transport (QUIC or WebTransport).
type Transport interface {
	// Functions wrapping underlying transport protocol

	// Accept a bidirectional stream (like the control stream)
	AcceptBidirectionalStream(ctx context.Context) (io.ReadWriteCloser, error)
}

type QuicTransport struct {
	conn *quic.Conn
}

// Implements Moqtransport methods

type WebTransport struct {
	sess *webtransport.Session
}

// Implements Moqtransport methods too

type MoqtSession struct {
	// Underlying transport protocol abstractions
	Transport Transport

	// The single bidirectional control stream for this session
	ControlStream io.ReadWriteCloser

	// Role (client or server) determines Request ID parity, etc.
	IsClient bool

	// Manages subscription states
	// Subscriptions map[uint64]*SubscriptionState

	// // Manages fetch states
	// Fetches map[uint64]*FetchState

	// Tracks aliases provided by the peer
	TrackAliases map[uint64]MoqtFullTrackName // Alias -> FullTrackName
}

func NewMoqtSession(tp Transport, isClient bool) (*MoqtSession, error) {
	controlStream, err := tp.AcceptBidirectionalStream(context.Background())
	if err != nil {
		return nil, err
	}

	s := &MoqtSession{
		Transport:     tp,
		ControlStream: controlStream,
		TrackAliases:  make(map[uint64]MoqtFullTrackName),
		// Subscriptions: make(map[uint64]*SubscriptionState),

		// Fetches:       make(map[uint64]*FetchState),
		IsClient: isClient,
	}

	// ... start control stream read loop, handle SETUP, etc.
	// go s.readControlMessages()
	return s, nil
}
