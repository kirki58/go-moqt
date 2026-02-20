# Project Context

This is a project that implements the IETF's new "Media Over QUIC Transport" (MOQT) streaming protocol with full and strict compliance to the draft-ietf-moq-transport-15 document in go.
We aim to provide a functional and compliant codebase for the use of MOQT-draft-15 efficiently in a go environment.

# Coding Style

- Use interfaces and structs when necessary to abstract and reuse the concepts mentioned in the document (draft-ietf-moq-transport-15)
- Always scan and respect with the current structure of the codebase follow it's trail
- Most of reusable code should be under `pkg/`

# Dependancies

- All dependancies of this codebase can be found in `go.mod`
- `quic-go` for QUIC abstractions
- `webtransport-go` for WebTransport abstractions
- `gonullv2` for nullable types

# Architecture

- `pkg/model/` directory contains the high-level abstractions of data modelling mentioned in the document like objects, error definitions, tracks etc.
- `pkg/message/` directory contains the wire encoding and decoding logic of the protocol's objects `object_datagram.go` and `subgroup_header.go` are low-level object definitions and are exceptions
- `pkg/session/` contains code that are related to the management of a MOQT session between 2 pairs
- `pkg/session/control/` contains the definition, wire encoding and decoding, stream-handling of "control messages" sent by peers to each other, used to manage sessions between 2 peers
- `pkg/session/handler.go` contains control message handlers that define the behaviour when a control message is received from the control stream of a session
- `pkg/transport/` contains QUIC and WebTransport transport-level logic necessary for session establishment, datagrams, actual transport stream handling etc. the `connection.go` file provides an interface abstraction for both QUIC and WebTransport to implement.
- `server.go` and `client.go` provides logic for both types of peers to perform MOQT communication at the very high-level, These are the "true" exports of the library we are implementing. They both use abstractions, functions, structs etc. defined under `pkg/`, some or most of their common session-handling logic can be placed in `pkg/session/session.go`

# Protocol Specification
Always use the following file as the primary source of truth for protocol implementation:
@./draft-ietf-moq-transport-15.txt