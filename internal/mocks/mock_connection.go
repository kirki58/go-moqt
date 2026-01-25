package mocks

import (
	"context"
	"go-moq/pkg/transport"
)

type MockMOQTConnection struct {
	// Right now, we only need to mock the CloseWithError() function to use in tests, rest can be mocked properly when needed
    // These fields allow you to inspect what was passed to the function
    CloseWithErrorCalled bool
    LastCloseCode        uint64
    LastCloseMsg         string
    // Allow the test to dictate what error this returns
    CloseWithErrorReturn error
}

func (m *MockMOQTConnection) CloseWithError(code uint64, msg string) error {
    m.CloseWithErrorCalled = true
    m.LastCloseCode = code
    m.LastCloseMsg = msg
    return m.CloseWithErrorReturn
}

func (m *MockMOQTConnection) OpenStream() (transport.Stream, error) { return nil, nil }
func (m *MockMOQTConnection) OpenStreamSync(ctx context.Context) (transport.Stream, error) { return nil, nil }
func (m *MockMOQTConnection) OpenUniStream() (transport.SendStream, error) { return nil, nil }
func (m *MockMOQTConnection) OpenUniStreamSync(ctx context.Context) (transport.SendStream, error) { return nil, nil }
func (m *MockMOQTConnection) AcceptStream(ctx context.Context) (transport.Stream, error) { return nil, nil }
func (m *MockMOQTConnection) AcceptUniStream(ctx context.Context) (transport.ReceiveStream, error) { return nil, nil }
func (m *MockMOQTConnection) IsWebTransport() bool { return false }
func (m *MockMOQTConnection) Context() context.Context { return context.Background() }
func (m *MockMOQTConnection) RemoteHost() string { return "" }
func (m *MockMOQTConnection) SendDatagram(data []byte) error { return nil }
func (m *MockMOQTConnection) ReceiveDatagram(ctx context.Context) ([]byte, error) { return nil, nil }