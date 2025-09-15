# Data Model: Chat System Core Features

## Entity Definitions

### ChatRoom
**Purpose**: Container for group conversations with metadata and participant tracking

**Fields**:
- `id` (string): Unique identifier (UUID)
- `title` (string): Human-readable room name (3-100 chars)
- `created_at` (timestamp): Room creation time
- `created_by` (string): User ID of room creator
- `is_active` (bool): Room status (for future soft deletion)

**Validation Rules**:
- Title: 3-100 characters, non-empty after trimming
- Created_by: Must reference valid user ID
- ID: UUID v4 format

**Relationships**:
- One-to-many with Message
- Many-to-many with User (via RoomParticipant)

### Message
**Purpose**: Individual chat messages with content and metadata

**Fields**:
- `id` (string): Unique identifier (UUID)
- `room_id` (string): Reference to ChatRoom
- `user_id` (string): Message sender ID
- `content` (string): Message text content (1-1000 chars)
- `created_at` (timestamp): Message timestamp
- `edited_at` (timestamp, nullable): Last edit time (future feature)

**Validation Rules**:
- Content: 1-1000 characters, non-empty after trimming
- Room_id: Must reference existing ChatRoom
- User_id: Must reference valid user
- Created_at: Cannot be future timestamp

**Relationships**:
- Many-to-one with ChatRoom
- Many-to-one with User

### RoomParticipant
**Purpose**: Junction table tracking user membership in chat rooms

**Fields**:
- `id` (string): Unique identifier (UUID)
- `room_id` (string): Reference to ChatRoom
- `user_id` (string): Reference to User
- `joined_at` (timestamp): When user joined room
- `last_seen_at` (timestamp): Last activity time (for future features)
- `is_active` (bool): Participation status

**Validation Rules**:
- Room_id: Must reference existing ChatRoom
- User_id: Must reference valid user
- Unique constraint on (room_id, user_id) pairs

**Relationships**:
- Many-to-one with ChatRoom
- Many-to-one with User

### User
**Purpose**: References existing user system, no new fields required

**Assumed Fields** (from existing authentication system):
- `id` (string): Unique user identifier
- `username` (string): Display name
- `email` (string): User email
- `created_at` (timestamp): Account creation time

## State Transitions

### ChatRoom Lifecycle
```
[Created] → [Active] → [Archived] (future)
    ↓
[Participants Join/Leave]
    ↓
[Messages Posted]
```

### Message Lifecycle
```
[Composed] → [Sent] → [Delivered] → [Read] (future)
                ↓
           [Edited] (future)
```

### Participant Lifecycle
```
[Invited] (future) → [Joined] → [Active] → [Left]
                                   ↑
                              [Last Seen Updated]
```

## Redis Data Structure

### RedisJSON Documents

**Chat Rooms** - Key pattern: `room:{room_id}`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Project Discussion",
  "created_at": "2025-09-15T09:30:00Z",
  "created_by": "123e4567-e89b-12d3-a456-426614174000",
  "is_active": true,
  "participant_count": 5,
  "last_message_at": "2025-09-15T10:25:00Z"
}
```

**Room Participants** - Key pattern: `room:{room_id}:participants`
```json
{
  "123e4567-e89b-12d3-a456-426614174000": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "joined_at": "2025-09-15T09:30:00Z",
    "last_seen_at": "2025-09-15T10:30:00Z",
    "is_active": true,
    "username": "alice"
  },
  "456e7890-e12b-34d5-b678-901234567890": {
    "user_id": "456e7890-e12b-34d5-b678-901234567890",
    "joined_at": "2025-09-15T09:35:00Z",
    "last_seen_at": "2025-09-15T10:28:00Z",
    "is_active": true,
    "username": "bob"
  }
}
```

**Room Index** - Key: `rooms:index`
```json
{
  "550e8400-e29b-41d4-a716-446655440000": {
    "title": "Project Discussion",
    "created_at": "2025-09-15T09:30:00Z",
    "participant_count": 5,
    "last_message_at": "2025-09-15T10:25:00Z",
    "is_active": true
  },
  "660f9500-f30c-52e5-b827-557766551111": {
    "title": "General Chat",
    "created_at": "2025-09-15T08:00:00Z",
    "participant_count": 12,
    "last_message_at": "2025-09-15T10:30:00Z",
    "is_active": true
  }
}
```

### Redis Streams

**Messages** - Stream key pattern: `room:{room_id}:messages`
```
Stream: room:550e8400-e29b-41d4-a716-446655440000:messages
1694772600000-0 user_id="123e4567-e89b-12d3-a456-426614174000" content="Hello everyone!" username="alice"
1694772605000-0 user_id="456e7890-e12b-34d5-b678-901234567890" content="Hi Alice! How are you?" username="bob"
1694772610000-0 user_id="123e4567-e89b-12d3-a456-426614174000" content="Great! Ready for the meeting?" username="alice"
```

**Events** - Stream key: `chat:events`
```
Stream: chat:events
1694772600000-0 event="MessageSent" room_id="550e8400-e29b-41d4-a716-446655440000" user_id="123e4567..." message_id="1694772600000-0" content="Hello everyone!" username="alice"
1694772605000-0 event="UserJoinedRoom" room_id="550e8400-e29b-41d4-a716-446655440000" user_id="789e0123..." username="charlie"
1694772610000-0 event="MessageSent" room_id="550e8400-e29b-41d4-a716-446655440000" user_id="456e7890..." message_id="1694772610000-0" content="Hi Alice!" username="bob"
```

### Additional Redis Keys

**User Sessions** - Key pattern: `session:{session_id}`
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "alice",
  "active_rooms": ["550e8400-e29b-41d4-a716-446655440000", "660f9500-f30c-52e5-b827-557766551111"],
  "connected_at": "2025-09-15T09:30:00Z"
}
```

**SSE Connections** - Key pattern: `sse:room:{room_id}`
```
SET members: session_id_1, session_id_2, session_id_3
```

## Go Struct Definitions

```go
// RedisJSON document structures
type ChatRoom struct {
    ID               string    `json:"id" redis:"id"`
    Title            string    `json:"title" redis:"title"`
    CreatedAt        time.Time `json:"created_at" redis:"created_at"`
    CreatedBy        string    `json:"created_by" redis:"created_by"`
    IsActive         bool      `json:"is_active" redis:"is_active"`
    ParticipantCount int       `json:"participant_count" redis:"participant_count"`
    LastMessageAt    *time.Time `json:"last_message_at,omitempty" redis:"last_message_at"`
}

type RoomParticipant struct {
    UserID     string    `json:"user_id" redis:"user_id"`
    JoinedAt   time.Time `json:"joined_at" redis:"joined_at"`
    LastSeenAt time.Time `json:"last_seen_at" redis:"last_seen_at"`
    IsActive   bool      `json:"is_active" redis:"is_active"`
    Username   string    `json:"username" redis:"username"`
}

type RoomParticipants map[string]RoomParticipant

// Redis Stream message structures
type StreamMessage struct {
    ID       string            `json:"id"`
    Fields   map[string]string `json:"fields"`
    RoomID   string            `json:"room_id"`
}

type Message struct {
    ID        string    `json:"id"`        // Redis Stream ID
    RoomID    string    `json:"room_id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"created_at"` // Derived from Stream ID
}

// Session management
type UserSession struct {
    UserID      string    `json:"user_id" redis:"user_id"`
    Username    string    `json:"username" redis:"username"`
    ActiveRooms []string  `json:"active_rooms" redis:"active_rooms"`
    ConnectedAt time.Time `json:"connected_at" redis:"connected_at"`
}

// View models for API responses
type RoomWithMetadata struct {
    ChatRoom
    LastMessage string `json:"last_message,omitempty"`
}

type MessagePage struct {
    Messages   []Message `json:"messages"`
    HasMore    bool      `json:"has_more"`
    NextCursor string    `json:"next_cursor,omitempty"`
}

// Event structures for Watermill
type MessageSentEvent struct {
    Event     string    `json:"event"`
    MessageID string    `json:"message_id"`
    RoomID    string    `json:"room_id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    Username  string    `json:"username"`
    Timestamp time.Time `json:"timestamp"`
}

type UserJoinedRoomEvent struct {
    Event    string    `json:"event"`
    RoomID   string    `json:"room_id"`
    UserID   string    `json:"user_id"`
    Username string    `json:"username"`
    JoinedAt time.Time `json:"joined_at"`
}
```

## Business Rules

### Room Creation
1. User must be authenticated to create room
2. Room title must be unique within the system
3. Creator automatically becomes first participant
4. Room is immediately active upon creation

### Message Sending
1. User must be authenticated and active participant in room
2. Message content cannot be empty after trimming
3. Messages are immutable once sent (editing is future feature)
4. Timestamps are server-generated, not client-provided

### Room Participation
1. Any authenticated user can join public rooms
2. Users cannot join the same room twice (unique constraint)
3. Room creators cannot leave their own rooms (business rule)
4. Last seen timestamp updates on any room activity

### Message History
1. Messages are ordered chronologically by Redis Stream ID
2. Pagination uses XRANGE with Stream ID cursors
3. Message history is persistent in Redis Streams (configurable retention)
4. Users can only see messages from after they joined (future feature)

## Integration Points

### Watermill Events
**Events to Publish**:
- `MessageSent`: When message is persisted to database
- `UserJoinedRoom`: When user becomes room participant
- `UserLeftRoom`: When user leaves room (future)
- `RoomCreated`: When new room is created

**Event Payload Example**:
```go
type MessageSentEvent struct {
    MessageID string    `json:"message_id"`
    RoomID    string    `json:"room_id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

### SSE Integration
- SSE connections stored in Redis Sets (`sse:room:{room_id}`)
- Events from Redis Streams trigger SSE broadcasts
- Connection management uses Redis for distributed session tracking
- Disconnected users automatically removed from Redis Sets

### Redis Operations Summary

**Room Operations**:
- Create: `JSON.SET room:{id}` + `JSON.SET room:{id}:participants` + `JSON.SET rooms:index .{id}`
- Join: `JSON.SET room:{id}:participants .{user_id}` + `SADD sse:room:{id} {session_id}`
- List: `JSON.GET rooms:index`

**Message Operations**:
- Send: `XADD room:{id}:messages * user_id {id} content {text} username {name}` + `XADD chat:events *`
- History: `XREVRANGE room:{id}:messages {end} {start} COUNT {limit}`
- Real-time: Watermill consumes `chat:events` stream → SSE broadcast

**Session Operations**:
- Connect: `JSON.SET session:{id}` + `SADD sse:room:{room_id} {session_id}`
- Disconnect: `DEL session:{id}` + `SREM sse:room:{room_id} {session_id}`

**Data Model Complete**: Redis-based architecture with RedisJSON documents, Redis Streams for messages, and native pub/sub capabilities.