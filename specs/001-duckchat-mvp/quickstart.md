# DuckChat Quickstart

## Prerequisites
- Go 1.21+
- Redis Stack 7.0+
- Modern web browser with SSE support

## Quick Start

### 1. Start Redis
```bash
docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack:latest
```

### 2. Run DuckChat Server
```bash
go run main.go
```

### 3. Open DuckChat
Navigate to `http://localhost:8080` in your web browser

### 4. Basic User Flow Test

#### User Authentication
1. Enter a username (1-50 characters)
2. Click "Login" - should receive JWT token
3. Should be redirected to main chat interface

#### Create Chat Room
1. Click "Create Room"
2. Enter room name (e.g., "General Chat")
3. Should be automatically joined to the new room
4. Note the room ID for sharing

#### Join Existing Room
1. Enter room ID in "Join Room" field
2. Click "Join" - should see existing message history
3. Should see user count update
4. Other users in room should see join notification

#### Send Messages
1. Type message in text input (1-1000 characters)
2. Press Enter or click Send
3. Message should appear at bottom of chat
4. Other users should see message in real-time
5. Timestamp should be displayed

#### View Message History
1. Scroll up in chat area
2. Should see previous messages
3. Should load more history when scrolling to top
4. Messages should remain in chronological order

### 5. Multi-User Testing

#### Open Multiple Browser Tabs/Windows
1. Tab 1: Create room as "Alice"
2. Tab 2: Join same room as "Bob"
3. Send messages from both tabs
4. Verify real-time updates in both

#### Test Connection Handling
1. Disconnect network briefly
2. Reconnect - should auto-reconnect SSE
3. Send message - should work after reconnection

### 6. Validation Checklist

**Authentication Flow**:
- [ ] Username validation (length, characters)
- [ ] JWT token generation and storage
- [ ] Protected routes require authentication
- [ ] Invalid tokens rejected

**Room Management**:
- [ ] Room creation with unique ID
- [ ] Room joining with valid ID
- [ ] Non-existent room joining fails gracefully
- [ ] Participant count updates correctly

**Real-time Messaging**:
- [ ] Messages appear instantly for all users
- [ ] Message ordering preserved
- [ ] SSE connection resilient to network issues
- [ ] User join/leave notifications

**Message History**:
- [ ] Scrolling loads previous messages
- [ ] Pagination works correctly
- [ ] Message persistence across sessions
- [ ] Chronological order maintained

**UI/UX**:
- [ ] Messages auto-scroll to bottom
- [ ] Loading states during operations
- [ ] Error messages for failures
- [ ] Responsive design basics

### 7. Performance Testing

#### Load Testing (Optional)
```bash
# Test concurrent users (requires testing tool)
# Simulate 10 users joining same room
# Send 100 messages rapidly
# Verify all messages received by all users
```

**Expected Performance**:
- Message latency: <100ms
- Room join time: <500ms
- History load time: <200ms
- SSE reconnect time: <1s

### 8. Expected Error Scenarios

**Should Handle Gracefully**:
- Invalid room IDs
- Network disconnections
- Duplicate usernames (should work fine)
- Very long messages (should be rejected)
- Rapid message sending (should not crash)
- Redis connection loss (should show error)

### 9. Development Quick Commands

```bash
# Start development server with auto-reload
go run main.go -dev

# Run all tests
go test ./...

# Check Redis data
redis-cli
> KEYS *
> XRANGE messages:room-123 - +
> SMEMBERS room_participants:room-123

# Clear Redis data for fresh start
redis-cli FLUSHALL
```

## Architecture Quick Reference

**Frontend**: Server-side rendered templates (Templ) + HTMX for interactions
**Backend**: Go with REST API endpoints + SSE for real-time updates
**Storage**: Redis Stack (Hashes for rooms/sessions, Streams for messages, Sets for participants)
**Auth**: JWT tokens with claims for user identification
**Communication**: JSON-RPC 2.0 for all API requests/responses and SSE notifications

## Troubleshooting

**Can't connect to Redis**: Check Docker container is running on port 6379
**SSE not working**: Verify browser supports Server-Sent Events, check network/proxy settings
**Messages not appearing**: Check JWT token validity, verify user is joined to room
**Performance issues**: Monitor Redis memory usage, check network latency