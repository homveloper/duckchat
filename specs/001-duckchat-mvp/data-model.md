# Data Model: DuckChat

## Core Entities

### ChatRoom
**Purpose**: Represents a conversation space where multiple users can exchange messages

**Fields**:
- `id`: string (UUID) - Unique room identifier
- `name`: string - Human-readable room name
- `created_at`: timestamp - Room creation time
- `created_by`: string - User ID who created the room
- `participant_count`: integer - Current active participants
- `last_activity`: timestamp - Last message timestamp

**Validation Rules**:
- `id` must be valid UUID format
- `name` must be 1-100 characters, non-empty
- `participant_count` must be >= 0
- `created_by` must be valid user ID

**State Transitions**:
- Created → Active (when first user joins)
- Active → Active (users join/leave, messages sent)
- Active → Inactive (when last user leaves, optional cleanup)

**Redis Storage**: JSON document `room:{room_id}` using RedisJSON module

### Message
**Purpose**: Individual chat message within a room

**Fields**:
- `id`: string (UUID) - Unique message identifier
- `room_id`: string - Reference to chat room
- `user_id`: string - Message sender ID (username NOT stored here)
- `content`: string - Message text content
- `timestamp`: timestamp - Message creation time
- `message_type`: enum - "text", "system" (join/leave notifications)

**Validation Rules**:
- `content` must be 1-1000 characters for text messages
- `room_id` must reference existing room
- `user_id` must be authenticated user
- `timestamp` must not be future date

**State Transitions**:
- Draft → Sent (immutable after sending)
- No updates or deletes in MVP

**Redis Storage**: JSON document `message:{message_id}` using RedisJSON module, plus Stream `messages:{room_id}` with message IDs for ordering

**Note**: Username is resolved at display time by looking up `user:{user_id}` to get current nickname

### User
**Purpose**: User profile information, separate from session data

**Fields**:
- `user_id`: string (UUID) - Unique user identifier
- `username`: string - Current display name/nickname
- `created_at`: timestamp - Account creation time
- `updated_at`: timestamp - Last profile update time

**Validation Rules**:
- `username` must be 1-50 characters, alphanumeric + spaces
- `user_id` must be valid UUID format
- `username` can be changed anytime (affects all past messages)

**State Transitions**:
- Created → Active (when user first logs in)
- Active → Active (nickname changes, activity)
- No deletion in MVP

**Redis Storage**: JSON document `user:{user_id}` using RedisJSON module

### UserSession
**Purpose**: Tracks authenticated user's connection and room participation

**Fields**:
- `user_id`: string (UUID) - Unique user identifier
- `active_rooms`: set of strings - Room IDs user has joined
- `last_seen`: timestamp - Last activity time
- `jwt_claims`: object - Decoded JWT data
- `connection_id`: string - SSE connection identifier

**Validation Rules**:
- `active_rooms` maximum 10 concurrent rooms
- `jwt_claims` must contain valid user data
- `user_id` must match JWT subject

**State Transitions**:
- Authenticated → Active (JWT validated)
- Active → Active (joins/leaves rooms, sends messages)
- Active → Disconnected (SSE connection closed)
- Disconnected → Active (reconnection)

**Redis Storage**: JSON document `session:{user_id}` using RedisJSON module, plus Set `room_participants:{room_id}`

## Relationships

### User ↔ UserSession (1:1)
- Each user has one active session
- Session references user for profile data
- Username changes in User entity affect all message displays

### ChatRoom ↔ Message (1:N)
- One room contains many messages
- Messages belong to exactly one room
- Ordered by timestamp within room

### ChatRoom ↔ User (M:N)
- Users can join multiple rooms
- Rooms can have multiple participants
- Tracked via `room_participants:{room_id}` sets containing user_ids

### User → Message (1:N)
- User can send many messages
- Each message references user_id only
- Username resolved at display time from User entity

## Data Access Patterns

### Read Operations
1. **Room Message History**: `XRANGE messages:{room_id} - +` to get message IDs, then `JSON.GET message:{message_id}` for each message, then `JSON.GET user:{user_id} $.username` to resolve usernames
2. **Room Participants**: `SMEMBERS room_participants:{room_id}` to get user_ids, then `JSON.GET user:{user_id} $.username` for each to get current nicknames
3. **User Active Rooms**: `JSON.GET session:{user_id} $.active_rooms` (RedisJSON)
4. **Room Metadata**: `JSON.GET room:{room_id}` (RedisJSON)
5. **Message by ID**: `JSON.GET message:{message_id}` (RedisJSON)
6. **User Profile**: `JSON.GET user:{user_id}` (RedisJSON)
7. **Current Username**: `JSON.GET user:{user_id} $.username` (RedisJSON)

### Write Operations
1. **Create User**: `JSON.SET user:{user_id} $ '{user_json_data}'` (RedisJSON)
2. **Update Username**: `JSON.SET user:{user_id} $.username '{new_username}'` and `JSON.SET user:{user_id} $.updated_at '{timestamp}'` (RedisJSON) - affects all past messages
3. **Create Room**: `JSON.SET room:{room_id} $ '{room_json_data}'` (RedisJSON)
4. **Send Message**: `JSON.SET message:{message_id} $ '{message_json_data}'` then `XADD messages:{room_id} * message_id {message_id}` (RedisJSON + Streams)
5. **Join Room**: `SADD room_participants:{room_id} {user_id}` and `JSON.SET session:{user_id} $.last_seen '{timestamp}'` (Sets + RedisJSON)
6. **Leave Room**: `SREM room_participants:{room_id} {user_id}` (Redis Sets)
7. **Update Session**: `JSON.SET session:{user_id} $.{field} '{value}'` (RedisJSON)
8. **Update Room Activity**: `JSON.SET room:{room_id} $.last_activity '{timestamp}'` (RedisJSON)

### Event Patterns
1. **Message Sent**: Command → Event → Stream write → SSE broadcast (username resolved at broadcast time)
2. **User Joined**: Command → Event → Set update → SSE notification
3. **User Left**: Command → Event → Set update → SSE notification
4. **Username Changed**: Command → Event → SSE broadcast to all active rooms → Frontend re-renders all messages from that user

## Performance Considerations
- Redis Streams provide O(log N) message insertion for ordering
- RedisJSON operations are O(1) for direct path access, O(N) for complex queries
- Set operations for participants are O(1) average case
- Message history pagination via XRANGE with COUNT limit for IDs, then batch JSON.GET
- JSON documents allow complex nested queries and partial updates
- JSONPath expressions enable efficient field access without full document retrieval

## Redis JSON Data Structure Examples

### ChatRoom JSON Document
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "General Chat",
  "created_at": "2025-09-14T18:00:00Z",
  "created_by": "user123",
  "participant_count": 5,
  "last_activity": "2025-09-14T18:05:30Z",
  "metadata": {
    "max_participants": 50,
    "created_via": "web"
  }
}
```
**Redis Key**: `room:550e8400-e29b-41d4-a716-446655440000`

### Message JSON Document
```json
{
  "id": "msg-001",
  "room_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user123",
  "content": "Hello everyone!",
  "timestamp": "2025-09-14T18:05:30Z",
  "message_type": "text",
  "metadata": {
    "client": "web",
    "edited": false
  }
}
```
**Redis Key**: `message:msg-001`
**Note**: Username is resolved by looking up `user:user123` at display time

### User JSON Document
```json
{
  "user_id": "user123",
  "username": "Alice",
  "created_at": "2025-09-14T17:00:00Z",
  "updated_at": "2025-09-14T18:00:00Z"
}
```
**Redis Key**: `user:user123`

### UserSession JSON Document
```json
{
  "user_id": "user123",
  "active_rooms": [
    "550e8400-e29b-41d4-a716-446655440000",
    "other-room-id"
  ],
  "last_seen": "2025-09-14T18:05:30Z",
  "jwt_claims": {
    "sub": "user123",
    "exp": 1694745330
  },
  "connection_id": "conn-abc123",
  "metadata": {
    "user_agent": "Mozilla/5.0...",
    "ip_address": "192.168.1.1"
  }
}
```
**Redis Key**: `session:user123`
**Note**: Username is resolved by looking up `user:user123` when needed

## JSONPath Query Examples

### Common Queries
```bash
# Get current username (affects all past messages)
JSON.GET user:user123 $.username

# Update username (affects all past message displays)
JSON.SET user:user123 $.username '"NewNickname"'
JSON.SET user:user123 $.updated_at '"2025-09-14T18:10:00Z"'

# Get room name
JSON.GET room:550e8400-e29b-41d4-a716-446655440000 $.name

# Get user's active rooms
JSON.GET session:user123 $.active_rooms

# Get message content and user_id (username resolved separately)
JSON.GET message:msg-001 $.content $.user_id

# Resolve username for message display
JSON.GET user:user123 $.username

# Update room participant count
JSON.SET room:550e8400-e29b-41d4-a716-446655440000 $.participant_count 6

# Add room to user session
JSON.ARRAPPEND session:user123 $.active_rooms '"new-room-id"'

# Update last activity
JSON.SET room:550e8400-e29b-41d4-a716-446655440000 $.last_activity '"2025-09-14T18:10:00Z"'
```

### Username Change Impact
```bash
# When user changes nickname from "Alice" to "SuperAlice":
JSON.SET user:user123 $.username '"SuperAlice"'
JSON.SET user:user123 $.updated_at '"2025-09-14T18:10:00Z"'

# All past messages from user123 will now show "SuperAlice" when displayed
# No message documents need to be updated - only the User document changes
# Frontend re-renders affected messages via SSE notification
```

## Consistency Model
- Eventually consistent across room participants
- Strong consistency within single Redis instance
- Event ordering preserved per room via Streams
- User session state may lag briefly during reconnection
- JSON document updates are atomic per key
- Complex transactions possible with Redis transactions + JSON operations