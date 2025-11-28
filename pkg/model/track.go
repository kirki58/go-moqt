package model

import (
	"fmt"
	"strings"
)

type MoqtTrackNamespace [][]byte // --> Must contain between 1 and 32 fields.

func (ns MoqtTrackNamespace) IsValid() bool {
	// Check number of fields
	return (len(ns) >= minFieldsInNamespace && len(ns) <= maxFieldsInNamespace)
}

const minFieldsInNamespace = 1
const maxFieldsInNamespace = 32
const maxBytesInFullTrackName = 4096

type MoqtFullTrackName struct {
	Namespace MoqtTrackNamespace
	Name      []byte
}

func (ftn MoqtFullTrackName) GetLength() int {
	length := len(ftn.Name)
	for _, field := range ftn.Namespace {
		length += len(field)
	}
	return length
}

func (ftn MoqtFullTrackName) IsValid() bool {
	// Namespace is valid AND length of name > 0 AND total ftn length < maxBytesInFullTrackName
	return (ftn.Namespace.IsValid() && len(ftn.Name) > 0 && ftn.GetLength() <= maxBytesInFullTrackName)
}

func StringToMoqtFullTrackName(str string) (MoqtFullTrackName, error) {
	// Split by delimeters '/'
	parts := strings.Split(str, "/")

	// "+1"s account for the additional name to fields
	if len(parts) < minFieldsInNamespace+1 || len(parts) > maxFieldsInNamespace+1 {
		return MoqtFullTrackName{}, MOQT_SESSION_TERMINATION_ERROR{
			// If an endpoint receives a Track Namespace consisting of 0 or greater than 32 Track Namespace Fields, it MUST close the session with a PROTOCOL_VIOLATION.
			// Cite 2.4.1
			ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
			ReasonPhrase: NewReasonPhrase(fmt.Sprintf("Number of Fields in Namespace must be between %d and %d", minFieldsInNamespace, maxFieldsInNamespace)),
		}
	}

	// Last part is name, others are fields
	name := []byte(parts[len(parts)-1])
	fields := parts[:len(parts)-1]

	ftnLen := len(name)

	namespace := make(MoqtTrackNamespace, len(fields))
	for i, field := range fields {
		ftnLen += len(field)
		if ftnLen > maxBytesInFullTrackName {
			return MoqtFullTrackName{}, MOQT_SESSION_TERMINATION_ERROR{
				// The maximum total length of a Full Track Name is 4,096 bytes.
				// If an endpoint receives a Full Track Name exceeding this length, it MUST close the session with a PROTOCOL_VIOLATION.
				// Cite 2.4.1
				ErrorCode:    MOQT_SESSION_TERMINATION_ERROR_CODE_PROTOCOL_VIOLATION,
				ReasonPhrase: NewReasonPhrase(fmt.Sprintf("Combined length of all Fields.Value and Name.Value must not exceed %d bytes", maxBytesInFullTrackName)),
			}
		}
		namespace[i] = []byte(field)
	}

	return MoqtFullTrackName{
		Namespace: namespace,
		Name:      name,
	}, nil
}

func (ftn MoqtFullTrackName) ToString() string {
	res := ""
	for _, field := range ftn.Namespace {
		res += fmt.Sprintf("%s/", string(field))
	}

	res += string(ftn.Name)
	return res
}
