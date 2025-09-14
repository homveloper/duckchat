# DuckChat JSON-RPC 2.0 API Specification

## Protocol Overview

- **Transport**: HTTP POST requests to `/api/v1/{method}`
- **Content-Type**: `application/json`
- **Body Format**: JSON-RPC 2.0
- **Authentication**: JWT Bearer token in Authorization header
- **SSE Notifications**: JSON-RPC 2.0 Notifications via `/api/v1/events`

## JSON-RPC 2.0 Methods

### Authentication

#### `auth.CreateGuestSession`
**Endpoint**: `POST /api/v1/auth.CreateGuestSession`

Create a guest JWT session
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "auth.CreateGuestSession",
  "params": {
    "username": "string"
  },
  "id": 1
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "token": "jwt_token_string",
    "user": {
      "id": "uuid",
      "username": "string",
      "type": "guest"
    },
    "expires_at": "2023-12-01T10:00:00Z"
  },
  "id": 1
}
```

#### `auth.ValidateToken`
**Endpoint**: `POST /api/v1/auth.ValidateToken`

Validate JWT token
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "auth.ValidateToken",
  "params": {
    "token": "jwt_token_string"
  },
  "id": 2
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "valid": true,
    "user": {
      "id": "uuid",
      "username": "string",
      "type": "guest"
    }
  },
  "id": 2
}
```

### Room Management

#### `room.CreateRoom`
**Endpoint**: `POST /api/v1/room.CreateRoom`

Create a new chat room
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "room.CreateRoom",
  "params": {
    "name": "string",
    "description": "string"
  },
  "id": 1
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "id": "uuid",
    "name": "string",
    "description": "string",
    "created_at": "2023-12-01T10:00:00Z"
  },
  "id": 1
}
```

#### `room.JoinRoom`
**Endpoint**: `POST /api/v1/room.JoinRoom`

Join a chat room
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "room.JoinRoom",
  "params": {
    "room_id": "uuid",
    "username": "string"
  },
  "id": 2
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "room": {
      "id": "uuid",
      "name": "string",
      "description": "string"
    }
  },
  "id": 2
}
```

#### `room.GetRooms`
**Endpoint**: `POST /api/v1/room.GetRooms`

Get list of available rooms
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "room.GetRooms",
  "params": {},
  "id": 3
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "rooms": [
      {
        "id": "uuid",
        "name": "string",
        "description": "string",
        "user_count": 0,
        "created_at": "2023-12-01T10:00:00Z"
      }
    ]
  },
  "id": 3
}
```

### Chat Methods

#### `chat.SendMessage`
**Endpoint**: `POST /api/v1/chat.SendMessage`

Send a new chat message to a room
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "chat.SendMessage",
  "params": {
    "room_id": "uuid",
    "username": "string",
    "content": "string"
  },
  "id": 4
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "id": "uuid",
    "room_id": "uuid",
    "username": "string",
    "content": "string", 
    "timestamp": "2023-12-01T10:00:00Z"
  },
  "id": 4
}
```

#### `chat.GetMessages`
**Endpoint**: `POST /api/v1/chat.GetMessages`

Get chat history for a room (with pagination)
```json
Request:
{
  "jsonrpc": "2.0",
  "method": "chat.GetMessages",
  "params": {
    "room_id": "uuid",
    "limit": 50,
    "before": "2023-12-01T10:00:00Z"
  },
  "id": 5
}

Response:
{
  "jsonrpc": "2.0",
  "result": {
    "messages": [
      {
        "id": "uuid",
        "room_id": "uuid",
        "username": "string",
        "content": "string",
        "timestamp": "2023-12-01T10:00:00Z"
      }
    ],
    "has_more": true
  },
  "id": 5
}
```

## Server-Sent Events (SSE)

### Endpoint: `GET /api/v1/events`

#### `chat.messageReceived` Notification
Real-time message notification
```json
{
  "jsonrpc": "2.0",
  "method": "chat.messageReceived",
  "params": {
    "id": "uuid",
    "username": "string", 
    "content": "string",
    "timestamp": "2023-12-01T10:00:00Z"
  }
}
```

#### `chat.userJoined` Notification
User joined notification
```json
{
  "jsonrpc": "2.0",
  "method": "chat.userJoined",
  "params": {
    "username": "string",
    "timestamp": "2023-12-01T10:00:00Z"
  }
}
```

#### `chat.userLeft` Notification
User left notification
```json
{
  "jsonrpc": "2.0", 
  "method": "chat.userLeft",
  "params": {
    "username": "string",
    "timestamp": "2023-12-01T10:00:00Z"
  }
}
```

## Error Handling

### JSON-RPC 2.0 Error Response
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": "Additional error information"
  },
  "id": null
}
```

### Error Codes
- `-32700`: Parse error
- `-32600`: Invalid Request  
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `-32000 to -32099`: Server error (custom)