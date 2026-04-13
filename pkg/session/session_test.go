package session

import (
	"go-moq/pkg/model"
	"go-moq/pkg/transport"

	"testing"
)

func TestTerminateIfTerminationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		terminated bool
	}{
		{
			name:       "No error",
			err:        nil,
			terminated: false,
		},
		{
			name:       "Protocol Violation",
			err:        &model.MOQT_SESSION_TERMINATION_ERROR{ErrorCode: model.MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION, ReasonPhrase: "Protocol Violation"},
			terminated: true,
		},
		{
			name:       "Invalid Authority",
			err:        &model.MOQT_SESSION_TERMINATION_ERROR{ErrorCode: model.MOQT_SESSION_TERMINATION_ERROR_CODE_INVALID_AUTHORITY, ReasonPhrase: "Invalid Authority"},
			terminated: true,
		},
		// TODO: when added new types of error (non-termination errors) test them here too!
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{
				Conn: &transport.MockMOQTConnection{CloseWithErrorReturn: nil},
			}
			didTerminate := s.TerminateIfTerminationError(tt.err)
			if didTerminate != tt.terminated {
				t.Errorf("terminateIfTerminationError() terminated = %v, want %v", didTerminate, tt.terminated)
			}
		})
	}

}
