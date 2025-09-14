# Redis Stack Data Schema

## Overview
DuckChat leverages Redis Stack features for efficient data storage and querying:
- **RedisJSON**: Native JSON document storage
- **RediSearch**: Full-text search and indexing
- **Redis Streams**: Message ordering and persistence

## Data Models

### Users (RedisJSON)
**Key Pattern**: `user:{user_id}`
```json
{
  "id": "uuid",
  "username": "string",
  "type": "guest",
  "created_at": "2023-12-01T10:00:00Z",
  "last_active": "2023-12-01T10:00:00Z"
}
```

### Rooms (RedisJSON)
**Key Pattern**: `room:{room_id}`
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "created_by": "user_id",
  "created_at": "2023-12-01T10:00:00Z",
  "user_count": 0,
  "last_message_at": "2023-12-01T10:00:00Z"
}
```

### Messages (RedisJSON)
**Key Pattern**: `message:{message_id}`
```json
{
  "id": "uuid",
  "room_id": "uuid",
  "user_id": "uuid",
  "username": "string",
  "content": "string",
  "timestamp": "2023-12-01T10:00:00Z",
  "edited": false,
  "edited_at": null
}
```

## RediSearch Indexes

### Room Index
```redis
FT.CREATE idx:rooms 
ON JSON PREFIX 1 room: 
SCHEMA 
  $.name AS name TEXT SORTABLE
  $.description AS description TEXT
  $.created_at AS created_at NUMERIC SORTABLE
  $.user_count AS user_count NUMERIC SORTABLE
  $.last_message_at AS last_message_at NUMERIC SORTABLE
```

### Message Index
```redis
FT.CREATE idx:messages 
ON JSON PREFIX 1 message: 
SCHEMA 
  $.room_id AS room_id TAG
  $.user_id AS user_id TAG  
  $.username AS username TEXT
  $.content AS content TEXT
  $.timestamp AS timestamp NUMERIC SORTABLE
```

### User Index
```redis
FT.CREATE idx:users 
ON JSON PREFIX 1 user: 
SCHEMA 
  $.username AS username TEXT SORTABLE
  $.type AS type TAG
  $.created_at AS created_at NUMERIC SORTABLE
  $.last_active AS last_active NUMERIC SORTABLE
```

## Query Examples

### Search Messages by Content
```redis
FT.SEARCH idx:messages "@content:(hello world)" 
SORTBY timestamp DESC 
LIMIT 0 50
```

### Get Room Messages with Pagination
```redis
FT.SEARCH idx:messages "@room_id:{room_uuid}" 
SORTBY timestamp DESC 
LIMIT 0 50
```

### Search Rooms by Name
```redis
FT.SEARCH idx:rooms "@name:(general|chat)" 
SORTBY last_message_at DESC
```

### Get Active Users
```redis
FT.SEARCH idx:users "@type:{guest}" 
SORTBY last_active DESC 
LIMIT 0 20
```

## Redis Streams (Optional)

### Message Stream
**Key**: `stream:messages:{room_id}`
- Real-time message broadcasting
- Message ordering guarantee
- Consumer groups for SSE connections

```redis
XADD stream:messages:room_uuid * 
message_id uuid 
username "user" 
content "Hello world" 
timestamp 1701000000
```

## Performance Optimizations

### Indexing Strategy
- **Primary**: RedisJSON for document storage
- **Secondary**: RediSearch indexes for fast queries
- **Caching**: Frequently accessed data in Redis strings/hashes

### Query Patterns
- **Recent Messages**: Use timestamp sorting in RediSearch
- **Message Search**: Full-text search on content field
- **Room Discovery**: Search by name/description
- **User Activity**: Sort by last_active timestamp