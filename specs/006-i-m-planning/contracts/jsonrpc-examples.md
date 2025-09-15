# JSON-RPC 2.0 Request/Response Examples

## Standard Request Format
```json
{
  "jsonrpc": "2.0",
  "method": "method.name",
  "params": {
    "param1": "value1",
    "param2": "value2"
  },
  "id": 1
}
```

## Standard Success Response Format
```json
{
  "jsonrpc": "2.0",
  "result": {
    "data": "response_data"
  },
  "id": 1
}
```

## Standard Error Response Format
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32002,
    "message": "Validation failed",
    "data": {
      "error_type": "VALIDATION_ERROR",
      "field": "title",
      "details": "Title must be between 3 and 100 characters"
    }
  },
  "id": 1
}
```

## Error Codes

### JSON-RPC 2.0 Standard Error Codes
- `-32700`: Parse error - Invalid JSON was received by the server
- `-32600`: Invalid Request - The JSON sent is not a valid Request object
- `-32601`: Method not found - The method does not exist / is not available
- `-32602`: Invalid params - Invalid method parameter(s)
- `-32603`: Internal error - Internal JSON-RPC error

### Custom Application Error Codes
- `-32001`: Authentication required - User not authenticated
- `-32002`: Validation failed - Request validation failed
- `-32003`: Resource not found - Requested resource does not exist
- `-32004`: Resource conflict - Resource already exists or conflict state
- `-32005`: Permission denied - User lacks required permissions
- `-32006`: Rate limit exceeded - Too many requests

## Method Examples

### rooms.list
```bash
curl -X POST http://localhost:8080/api/rooms.list \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "rooms.list",
    "params": {
      "limit": 50
    },
    "id": 1
  }'
```

### rooms.create
```bash
curl -X POST http://localhost:8080/api/rooms.create \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "rooms.create",
    "params": {
      "title": "Project Discussion"
    },
    "id": 2
  }'
```

### messages.send
```bash
curl -X POST http://localhost:8080/api/messages.send \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "messages.send",
    "params": {
      "room_id": "550e8400-e29b-41d4-a716-446655440000",
      "content": "Hello everyone!"
    },
    "id": 3
  }'
```

## Response Examples

### Success Response
```json
{
  "jsonrpc": "2.0",
  "result": {
    "room": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Project Discussion",
      "created_at": "2025-09-15T09:30:00Z",
      "created_by": "123e4567-e89b-12d3-a456-426614174000",
      "is_active": true
    },
    "redirect_url": "/rooms/550e8400-e29b-41d4-a716-446655440000"
  },
  "id": 2
}
```

### Error Response
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32002,
    "message": "Validation failed",
    "data": {
      "error_type": "VALIDATION_ERROR",
      "field": "title",
      "details": "Title must be between 3 and 100 characters"
    }
  },
  "id": 2
}
```