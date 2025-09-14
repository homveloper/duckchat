# DuckChat API Specification

## Base URL
- Development: `http://localhost:8080`
- Production: TBD

## Endpoints

### Chat Messages

#### GET /api/messages
Get recent chat messages
```json
Response:
{
  "messages": [
    {
      "id": "uuid",
      "username": "string",
      "content": "string", 
      "timestamp": "2023-12-01T10:00:00Z"
    }
  ]
}
```

#### POST /api/messages
Send a new message
```json
Request:
{
  "username": "string",
  "content": "string"
}

Response:
{
  "id": "uuid",
  "username": "string", 
  "content": "string",
  "timestamp": "2023-12-01T10:00:00Z"
}
```

### Web Pages (Server-Side Rendered)

#### GET /
Main chat interface (HTML)

#### GET /health
Health check endpoint