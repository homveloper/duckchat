# Research: DuckChat Technical Decisions

## Authentication Method Resolution
**Decision**: JWT-based authentication
**Rationale**:
- User specified JWT over session-based auth
- Supports distributed/stateless architecture
- Works well with SSE connections (bearer token in headers)
- Simpler than session management in distributed environment

**Alternatives considered**:
- Session-based: Rejected due to distributed environment complexity
- Anonymous users: Rejected as user identification needed for messaging

## Message Retention Policy Resolution
**Decision**: Persistent storage in Redis with no TTL (permanent retention)
**Rationale**:
- Chat history scrolling requirement from spec
- Redis Stack provides persistent storage
- MVP scope suggests simple permanent retention
- Future TTL policies can be added later

**Alternatives considered**:
- Session-only storage: Rejected due to chat history requirement
- Time-limited TTL: Deferred to post-MVP for simplicity

## User Identification Method Resolution
**Decision**: JWT claims with user display names
**Rationale**:
- Users need identification for message attribution
- JWT can carry user name/ID claims
- Display names sufficient for MVP
- No full user profile system needed

**Alternatives considered**:
- Anonymous with generated IDs: Insufficient for user recognition
- Full user accounts: Too complex for MVP

## Maximum Concurrent Users Resolution
**Decision**: Start with 50 users per room, 1000 concurrent users system-wide
**Rationale**:
- Reasonable limits for MVP testing
- Redis can handle this scale easily
- SSE connections manageable at this scale
- Limits prevent resource exhaustion

**Alternatives considered**:
- Unlimited: Risk of resource exhaustion
- Very low limits (10): Too restrictive for testing

## Real-time Communication Pattern
**Decision**: Server-Sent Events (SSE) for message delivery
**Rationale**:
- User specified SSE over WebSockets
- Simpler than WebSockets for one-way messaging
- Works with HTMX patterns
- HTTP-based, easier debugging

**Alternatives considered**:
- WebSockets: More complex bidirectional protocol
- Polling: Inefficient for real-time updates

## JSON-RPC 2.0 Implementation
**Decision**: JSON-RPC 2.0 for all API requests/responses and SSE notifications
**Rationale**:
- User specified this protocol
- Standardized request/response format
- Good error handling patterns
- Works well with HTMX

**Alternatives considered**:
- Plain REST JSON: Rejected per user requirements
- GraphQL: Too complex for MVP

## Go Watermill CQRS Architecture
**Decision**: Event-driven architecture with command/query separation
**Rationale**:
- User specified Watermill + CQRS
- Good for distributed systems
- Separates read/write operations
- Event sourcing for chat messages

**Alternatives considered**:
- Simple CRUD: Rejected for distributed requirements
- Other Go CQRS libraries: User specified Watermill

## Redis Data Model Strategy
**Decision**:
- Chat rooms: Redis Hash per room with metadata
- Messages: Redis Streams per room for ordered message history
- User sessions: Redis Hash with JWT claims and active rooms
- Room participants: Redis Sets for active user tracking

**Rationale**:
- Redis Streams perfect for chat message ordering
- Hashes provide efficient key-value storage
- Sets enable efficient membership tracking
- All structures support real-time operations

**Alternatives considered**:
- Single Redis structure: Insufficient for complex queries
- External database: Against user requirements