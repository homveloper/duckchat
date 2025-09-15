# Quickstart Guide: Chat System Core Features

## Overview
This quickstart validates the complete chat system implementation through realistic user scenarios. All tests should pass to ensure the feature is working correctly.

## Prerequisites
- DuckChat server running on localhost:8080
- Redis server running with RedisJSON module enabled
- At least 2 test user accounts available
- Watermill CQRS configured with Redis Streams transport
- SSE endpoints available

## Test Scenario 1: Room Creation and Auto-Join

### Setup
- User: `alice@example.com` (authenticated)

### Steps
1. **Navigate to main chat page**
   ```bash
   curl -c cookies.txt -X GET http://localhost:8080/chat
   ```
   ✅ Expect: Chat interface with '+' button visible

2. **Click create room button**
   ```bash
   # Simulated button click - opens room creation dialog
   curl -b cookies.txt -X GET http://localhost:8080/chat/create
   ```
   ✅ Expect: Room creation form displayed

3. **Submit room creation form**
   ```bash
   curl -b cookies.txt -X POST http://localhost:8080/api/rooms.create \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "rooms.create",
       "params": {
         "title": "Project Alpha Discussion"
       },
       "id": 1
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 success response with result.room and result.redirect_url
   ✅ Expect: Alice automatically added as participant

4. **Verify room creation in Redis**
   ```bash
   # Check room document
   redis-cli JSON.GET room:<room_id>

   # Check room participants
   redis-cli JSON.GET room:<room_id>:participants

   # Check room index
   redis-cli JSON.GET rooms:index .<room_id>
   ```
   ✅ Expect: Room exists with Alice as creator
   ✅ Expect: Alice is active participant
   ✅ Expect: Room appears in global index

5. **Verify redirect to room**
   ```bash
   curl -b cookies.txt -X GET http://localhost:8080/rooms/<room_id>
   ```
   ✅ Expect: Room interface displayed
   ✅ Expect: Message input and send button present
   ✅ Expect: Empty message area (new room)

## Test Scenario 2: Join Existing Room

### Setup
- User: `bob@example.com` (authenticated)
- Room: "Project Alpha Discussion" exists (created by Alice)

### Steps
1. **View available rooms list**
   ```bash
   curl -b cookies.txt -X POST http://localhost:8080/api/rooms.list \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "rooms.list",
       "params": {},
       "id": 2
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 success response with result.rooms array
   ✅ Expect: "Project Alpha Discussion" in list
   ✅ Expect: participant_count = 1

2. **Join existing room**
   ```bash
   curl -b cookies.txt -X POST http://localhost:8080/api/rooms.join \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "rooms.join",
       "params": {
         "room_id": "<room_id>"
       },
       "id": 3
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 success response with result.redirect_url
   ✅ Expect: Bob added as participant

3. **Verify room participant count updated**
   ```bash
   curl -b cookies.txt -X POST http://localhost:8080/api/rooms.list \
     -H "Content-Type: application/json" \
     -d '{}'
   ```
   ✅ Expect: "Project Alpha Discussion" shows participant_count = 2

4. **Access joined room**
   ```bash
   curl -b cookies.txt -X GET http://localhost:8080/rooms/<room_id>
   ```
   ✅ Expect: Room interface displayed for Bob
   ✅ Expect: Same message history as Alice sees

## Test Scenario 3: Send and Receive Messages

### Setup
- Users: Alice and Bob both in "Project Alpha Discussion"
- Both users have room interface open

### Steps
1. **Alice sends first message**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.send \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "messages.send",
       "params": {
         "room_id": "<room_id>",
         "content": "Hello everyone! Ready to start the project discussion?"
       },
       "id": 4
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 success response with result.message containing Redis Stream ID
   ✅ Expect: result.message.created_at timestamp present

2. **Verify message appears in room history**
   ```bash
   curl -b bob_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>"}'
   ```
   ✅ Expect: HTTP 200 with result.messages array
   ✅ Expect: Alice's message present with username
   ✅ Expect: Message content and timestamp correct

3. **Bob responds with message**
   ```bash
   curl -b bob_cookies.txt -X POST http://localhost:8080/api/messages.send \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "content": "Hi Alice! Yes, let'\''s discuss the project roadmap."}'
   ```
   ✅ Expect: HTTP 200, message created successfully

4. **Verify conversation thread**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>"}'

   # Also verify in Redis Streams directly
   redis-cli XREVRANGE room:<room_id>:messages + - COUNT 10
   ```
   ✅ Expect: 2 messages in chronological order
   ✅ Expect: Oldest message (Alice's) first by Stream ID
   ✅ Expect: Newest message (Bob's) last by Stream ID
   ✅ Expect: Both messages include username
   ✅ Expect: Redis Stream contains same messages with proper ordering

## Test Scenario 4: Real-time Message Delivery (SSE)

### Setup
- Alice and Bob in same room
- SSE connections established for both users

### Steps
1. **Alice establishes SSE connection**
   ```bash
   curl -b alice_cookies.txt -N -H "Accept: text/event-stream" \
     -X POST http://localhost:8080/api/sse.connect \
     -H "Content-Type: application/json" \
     -d '{"room_ids": ["<room_id>"]}'
   ```
   ✅ Expect: SSE stream established (HTTP 200)
   ✅ Expect: Connection remains open

2. **Bob establishes SSE connection**
   ```bash
   curl -b bob_cookies.txt -N -H "Accept: text/event-stream" \
     -X POST http://localhost:8080/api/sse.connect \
     -H "Content-Type: application/json" \
     -d '{"room_ids": ["<room_id>"]}'
   ```
   ✅ Expect: SSE stream established independently

3. **Alice sends message via REST API**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.send \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "content": "This should appear in real-time!"}'
   ```
   ✅ Expect: Message posted successfully (HTTP 200)

4. **Verify Bob receives message via SSE**
   ```
   event: message
   data: {"id":"1694772600000-0","room_id":"<room_id>","user_id":"alice_id","content":"This should appear in real-time!","created_at":"2025-09-15T10:30:00Z","username":"alice"}
   ```
   ✅ Expect: Bob's SSE stream receives message event
   ✅ Expect: Message data includes all required fields
   ✅ Expect: Delivery within 100ms of sending

5. **Bob responds and verify Alice receives**
   ```bash
   curl -b bob_cookies.txt -X POST http://localhost:8080/api/messages.send \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "content": "Got it! Real-time is working perfectly."}'
   ```
   ✅ Expect: Alice's SSE stream receives Bob's message
   ✅ Expect: Bidirectional real-time communication confirmed

## Test Scenario 5: Message History Pagination

### Setup
- Room with 75 messages (exceeds default 50-message limit)
- Alice accessing room

### Steps
1. **Load initial messages (first page)**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "limit": 50}'

   # Verify Redis Stream directly
   redis-cli XREVRANGE room:<room_id>:messages + - COUNT 50
   ```
   ✅ Expect: 50 most recent messages returned
   ✅ Expect: result.has_more = true
   ✅ Expect: result.next_cursor with Redis Stream ID provided
   ✅ Expect: Messages in reverse chronological order (newest first)

2. **Load older messages (scroll up)**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "before": "<next_cursor>", "limit": 20}'

   # Verify with Redis XREVRANGE using cursor
   redis-cli XREVRANGE room:<room_id>:messages "(<next_cursor>" - COUNT 20
   ```
   ✅ Expect: 20 older messages returned
   ✅ Expect: Messages have earlier Stream IDs than first batch
   ✅ Expect: No overlap with previous batch
   ✅ Expect: Correct chronological ordering maintained

3. **Load final batch of messages**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>", "before": "<cursor>", "limit": 20}'
   ```
   ✅ Expect: Remaining 5 messages returned
   ✅ Expect: result.has_more = false
   ✅ Expect: result.next_cursor not provided or null

4. **Verify message continuity**
   - Combine all 3 batches
   ```bash
   # Verify total message count in Redis Stream
   redis-cli XLEN room:<room_id>:messages
   ```
   ✅ Expect: Total 75 unique messages
   ✅ Expect: No gaps in Redis Stream ID sequence
   ✅ Expect: Oldest message has earliest Stream ID

## Test Scenario 6: Error Handling and Validation

### Setup
- Alice authenticated, Bob not authenticated

### Steps
1. **Test room creation with invalid title**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/rooms.create \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "rooms.create",
       "params": {
         "title": "AB"
       },
       "id": 10
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 error response with code -32002 (Validation failed)
   ✅ Expect: Error message about title length in error.data

2. **Test sending empty message**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.send \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "messages.send",
       "params": {
         "room_id": "<room_id>",
         "content": "   "
       },
       "id": 11
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 error response with code -32002 (Validation failed)
   ✅ Expect: Error about empty content after trimming in error.data

3. **Test unauthenticated access**
   ```bash
   curl -X POST http://localhost:8080/api/rooms.list \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "rooms.list",
       "params": {},
       "id": 12
     }'
   ```
   ✅ Expect: HTTP 200 (always)
   ✅ Expect: JSON-RPC 2.0 error response with code -32001 (Authentication required)
   ✅ Expect: Authentication required error in error.message

4. **Test accessing non-existent room**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/messages.history \
     -H "Content-Type: application/json" \
     -d '{"room_id": "00000000-0000-0000-0000-000000000000"}'
   ```
   ✅ Expect: HTTP 404 Not Found
   ✅ Expect: Room not found error

5. **Test joining room user is already in**
   ```bash
   curl -b alice_cookies.txt -X POST http://localhost:8080/api/rooms.join \
     -H "Content-Type: application/json" \
     -d '{"room_id": "<room_id>"}'
   ```
   ✅ Expect: HTTP 409 Conflict
   ✅ Expect: Already participant error

## Performance Validation

### Load Test: Concurrent Message Sending
1. **Setup**: 10 users in same room
2. **Test**: Each user sends 5 messages simultaneously
3. **Expected Results**:
   - All 50 messages successfully posted
   - All users receive all 50 messages via SSE
   - Message delivery within 200ms average
   - No duplicate or lost messages
   - Correct chronological ordering maintained

### Stress Test: Message History
1. **Setup**: Room with 10,000 messages
2. **Test**: Multiple pagination requests
3. **Expected Results**:
   - Response time < 100ms for 50-message batches
   - Consistent performance across all pages
   - Accurate cursor-based pagination

## Success Criteria

### Functional Requirements Validated
- ✅ FR-001: Room creation with '+' button
- ✅ FR-002: Auto-join creator to new room
- ✅ FR-003: Join existing rooms from list
- ✅ FR-004: Text input and send button present
- ✅ FR-005: Send messages via button/Enter key
- ✅ FR-006: Latest messages at bottom with auto-scroll
- ✅ FR-007: Scroll up loads older messages
- ✅ FR-008: Chronological message order
- ✅ FR-009: Real-time message delivery to participants
- ✅ FR-010: Cross-participant synchronization

### Non-Functional Requirements
- ✅ Real-time delivery < 200ms
- ✅ Pagination handles large message volumes
- ✅ Proper error handling and validation
- ✅ Database constraints enforced
- ✅ SSE connections stable under load

## Cleanup
After testing, clean up test data:
```bash
# Remove test rooms and their data
redis-cli --eval cleanup_test_rooms.lua 0 "Test" "Project Alpha"

# Or manually:
# Delete room messages (Redis Streams)
redis-cli DEL room:<test_room_id>:messages

# Delete room participants
redis-cli DEL room:<test_room_id>:participants

# Delete room document
redis-cli DEL room:<test_room_id>

# Remove from rooms index
redis-cli JSON.DEL rooms:index .<test_room_id>

# Clean up SSE connections
redis-cli DEL sse:room:<test_room_id>

# Clean up test sessions
redis-cli DEL session:<test_session_id>
```

**Feature validation complete when all test scenarios pass successfully.**