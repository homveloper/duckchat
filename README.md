# DuckChat

A minimal chat service with unified REST API, supporting multiple server implementations.

## Project Structure

```
duckchat/
├── servers/
│   ├── cpp/           # C++ server implementation
│   └── golang/        # Go server implementation (with templ-like HTML components)
├── client/
│   └── web/           # Web client (server-side rendered)
├── docs/              # API documentation
└── shared/            # Shared resources (API specs, types)
```

## Technology Stack

### Backend
- **Protocol**: REST API with JSON-RPC 2.0 body
- **Persistent Storage**: Redis Stack (RedisJSON + RediSearch)
- **Real-time Communication**: Server-Sent Events (SSE) with JSON-RPC 2.0 Notifications

### C++ Server
- **Language**: C++23 (modern style, auto-oriented)
- **HTTP Library**: httplib
- **Architecture**: 4-tier (Presentation → Application → Domain → Infrastructure)

### Go Server  
- **Language**: Go
- **HTML Rendering**: templ-like components
- **Architecture**: Clean architecture

## Architecture (C++ Implementation)

```
┌─────────────────┐
│ Presentation    │ ← HTTP handlers, SSE endpoints
├─────────────────┤
│ Application     │ ← Use cases, orchestration
├─────────────────┤  
│ Domain          │ ← Business logic, entities
├─────────────────┤
│ Infrastructure  │ ← Redis, external services
└─────────────────┘
```

## MVP Features

### Authentication
1. **게스트 JWT 인증**
   - 게스트 JWT 토큰 생성 (임시 사용자)
   - 닉네임 기반 사용자 식별
   - JWT 기반 상태 관리 (stateless)

### Core Chat Functionality
2. **채팅방 생성/입장**
   - 채팅방 생성 및 참여
   - 채팅방별 독립적인 메시지 스트림

3. **실시간 채팅**
   - 메시지 전송 및 수신
   - Server-Sent Events를 통한 실시간 업데이트

4. **채팅 내역 보존**
   - 이전 채팅 내역 스크롤링 (RediSearch 페이지네이션)
   - RedisJSON 기반 메시지 영구 저장
   - 메시지 전문 검색 (RediSearch)

5. **채팅방 관리**
   - 채팅방 리스트 표시 (RediSearch 정렬)
   - 채팅방 검색 (이름/설명 기반)
   - 채팅방 정보 조회 (RedisJSON)