# Tasks: Chat System Core Features

**Input**: Design documents from `/specs/006-i-m-planning/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Load plan.md from feature directory
   ✅ Found: Go 1.21+ with Redis, Watermill CQRS, SSE, Templ, HTMX
   ✅ Structure: DuckChat existing backend/ structure
2. Load optional design documents:
   ✅ data-model.md: 4 entities (ChatRoom, Message, RoomParticipant, UserSession)
   ✅ contracts/: chat-api.yaml with 7 JSON-RPC 2.0 methods
   ✅ quickstart.md: 6 integration test scenarios
3. Generate tasks by category: Setup, Tests, Core, Integration, Polish
4. Apply task rules: Different files = [P], Same file = sequential, Tests before implementation
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph and parallel execution examples
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Exact file paths included in descriptions

## Phase 3.1: Setup

- [x] T001 Install Redis with RedisJSON module and configure for development
- [x] T002 Add Go dependencies: Redis client, Watermill CQRS, UUID generation to backend/go.mod
- [x] T003 [P] Configure Redis connection and health check in backend/internal/config/redis.go
- [x] T004 [P] Set up JSON-RPC 2.0 middleware in backend/internal/jsonrpc/middleware.go

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests (JSON-RPC 2.0 Methods)
- [x] T005 [P] Contract test POST /api/rooms.list in backend/tests/contract/rooms_list_test.go
- [x] T006 [P] Contract test POST /api/rooms.create in backend/tests/contract/rooms_create_test.go
- [x] T007 [P] Contract test POST /api/rooms.join in backend/tests/contract/rooms_join_test.go
- [x] T008 [P] Contract test POST /api/rooms.leave in backend/tests/contract/rooms_leave_test.go
- [x] T009 [P] Contract test POST /api/messages.send in backend/tests/contract/messages_send_test.go
- [x] T010 [P] Contract test POST /api/messages.history in backend/tests/contract/messages_history_test.go
- [x] ~~T011 [P] Contract test POST /api/sse.connect in backend/tests/contract/sse_connect_test.go~~ (CANCELLED - SSE not JSON-RPC)

### Integration Tests (Quickstart Scenarios)
- [x] T012 [P] Integration test room creation and auto-join in backend/tests/integration/room_creation_test.go
- [x] T013 [P] Integration test join existing room in backend/tests/integration/room_join_test.go
- [x] T014 [P] Integration test send/receive messages in backend/tests/integration/messaging_test.go
- [x] T015 [P] Integration test real-time SSE delivery in backend/tests/integration/sse_realtime_test.go
- [x] T016 [P] Integration test message history pagination in backend/tests/integration/message_pagination_test.go
- [x] T017 [P] Integration test error handling validation in backend/tests/integration/error_handling_test.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Data Models & Redis Operations
- [x] T018 [P] ChatRoom entity and Redis operations in backend/internal/models/chatroom.go
- [x] T019 [P] Message entity and Redis Streams operations in backend/internal/models/message.go
- [x] T020 [P] RoomParticipant entity and Redis operations in backend/internal/models/participant.go
- [x] T021 [P] UserSession entity and Redis operations in backend/internal/models/session.go

### Service Layer
- [x] T022 [P] RoomService with Redis operations in backend/internal/services/room_service.go
- [x] T023 [P] MessageService with Redis Streams in backend/internal/services/message_service.go
- [x] T024 [P] ParticipantService with Redis operations in backend/internal/services/participant_service.go
- [x] T025 [P] SessionService with Redis operations in backend/internal/services/session_service.go

### JSON-RPC 2.0 API Handlers
- [x] T026 rooms.list and rooms.create handlers in backend/internal/handlers/room_handlers.go
- [x] T027 rooms.join and rooms.leave handlers in backend/internal/handlers/room_handlers.go
- [x] T028 messages.send and messages.history handlers in backend/internal/handlers/message_handlers.go
- [x] ~~T029 sse.connect handler in backend/internal/handlers/sse_handlers.go~~ (CANCELLED - SSE already exists)

### Watermill CQRS Event Processing
- [ ] T030 [P] Event publisher for Redis Streams in backend/internal/events/publisher.go
- [ ] T031 [P] Event consumer and SSE broadcaster in backend/internal/events/consumer.go
- [ ] T032 [P] Event definitions (MessageSent, UserJoined, etc.) in backend/internal/events/types.go

## Phase 3.4: Integration

### Real-time & SSE Management
- [ ] T033 SSE connection manager with Redis Sets in backend/internal/sse/manager.go
- [ ] T034 Watermill CQRS integration and event routing in backend/internal/events/router.go
- [ ] T035 Redis connection pooling and error handling in backend/internal/redis/client.go

### Frontend Templates
- [ ] T036 [P] Chat room list page with HTMX in backend/web/templates/rooms/list.templ
- [ ] T037 [P] Room creation dialog component in backend/web/templates/rooms/create.templ
- [ ] T038 [P] Chat room interface with message display in backend/web/templates/rooms/chat.templ
- [ ] T039 [P] Message input component with HTMX in backend/web/templates/components/message_input.templ
- [ ] T040 [P] Real-time message component in backend/web/templates/components/message.templ

### Authentication & Middleware
- [ ] T041 Session validation middleware for JSON-RPC in backend/internal/middleware/auth.go
- [ ] T042 Request logging and error handling middleware in backend/internal/middleware/logging.go

## Phase 3.5: Polish

### Unit Tests
- [ ] T043 [P] Unit tests for Redis operations in backend/tests/unit/redis_ops_test.go
- [ ] T044 [P] Unit tests for JSON-RPC validation in backend/tests/unit/jsonrpc_validation_test.go
- [ ] T045 [P] Unit tests for event processing in backend/tests/unit/event_processing_test.go

### Performance & Validation
- [ ] T046 Performance test Redis Streams pagination in backend/tests/performance/pagination_test.go
- [ ] T047 Load test concurrent SSE connections in backend/tests/performance/sse_load_test.go
- [ ] T048 Execute complete quickstart validation scenarios
- [ ] T049 [P] Update CLAUDE.md with chat system context
- [ ] T050 Clean up code duplication and optimize Redis queries

## Dependencies

### Critical Path Dependencies
- Setup (T001-T004) → All other tasks
- Contract tests (T005-T011) → Implementation tasks (T018+)
- Integration tests (T012-T017) → Implementation tasks (T018+)
- Models (T018-T021) → Services (T022-T025) → Handlers (T026-T029)
- Event types (T032) → Publisher/Consumer (T030-T031)
- Services (T022-T025) → Integration (T033-T042)

### Parallel Groups
**Phase 3.2 Tests**: T005-T017 can all run in parallel (different test files)
**Phase 3.3 Models**: T018-T021 can run in parallel (separate Go files)
**Phase 3.3 Services**: T022-T025 can run in parallel (separate Go files)
**Phase 3.3 Events**: T030-T032 can run in parallel (separate Go files)
**Phase 3.4 Templates**: T036-T040 can run in parallel (separate .templ files)
**Phase 3.5 Unit Tests**: T043-T045 can run in parallel (separate test files)

## Parallel Execution Examples

### Phase 3.2: Launch all contract tests together
```bash
# Contract Tests (7 parallel tasks)
Task: "Contract test POST /api/rooms.list in backend/tests/contract/rooms_list_test.go"
Task: "Contract test POST /api/rooms.create in backend/tests/contract/rooms_create_test.go"
Task: "Contract test POST /api/rooms.join in backend/tests/contract/rooms_join_test.go"
Task: "Contract test POST /api/rooms.leave in backend/tests/contract/rooms_leave_test.go"
Task: "Contract test POST /api/messages.send in backend/tests/contract/messages_send_test.go"
Task: "Contract test POST /api/messages.history in backend/tests/contract/messages_history_test.go"
Task: "Contract test POST /api/sse.connect in backend/tests/contract/sse_connect_test.go"

# Integration Tests (6 parallel tasks)
Task: "Integration test room creation and auto-join in backend/tests/integration/room_creation_test.go"
Task: "Integration test join existing room in backend/tests/integration/room_join_test.go"
Task: "Integration test send/receive messages in backend/tests/integration/messaging_test.go"
Task: "Integration test real-time SSE delivery in backend/tests/integration/sse_realtime_test.go"
Task: "Integration test message history pagination in backend/tests/integration/message_pagination_test.go"
Task: "Integration test error handling validation in backend/tests/integration/error_handling_test.go"
```

### Phase 3.3: Launch model creation tasks together
```bash
# Data Models (4 parallel tasks)
Task: "ChatRoom entity and Redis operations in backend/internal/models/chatroom.go"
Task: "Message entity and Redis Streams operations in backend/internal/models/message.go"
Task: "RoomParticipant entity and Redis operations in backend/internal/models/participant.go"
Task: "UserSession entity and Redis operations in backend/internal/models/session.go"
```

## Notes
- **Redis Setup**: Ensure RedisJSON module is installed and enabled
- **JSON-RPC 2.0**: All responses return HTTP 200 with JSON-RPC error/success format
- **TDD Critical**: Tests must fail before writing any implementation code
- **Redis Keys**: Follow patterns from data-model.md (room:{id}, room:{id}:participants, etc.)
- **Event Sourcing**: Messages stored in Redis Streams, events published to chat:events stream
- **SSE Management**: Connection tracking via Redis Sets for distributed server support
- **Templ Integration**: UI components use existing DuckChat Templ + HTMX + Tailwind patterns

## Validation Checklist
*GATE: Verified before task execution*

- [x] All 7 JSON-RPC methods have contract tests (T005-T011)
- [x] All 4 entities have model tasks (T018-T021)
- [x] All 6 quickstart scenarios have integration tests (T012-T017)
- [x] All tests come before implementation (Phase 3.2 → 3.3)
- [x] Parallel tasks target different files ([P] markers verified)
- [x] Each task specifies exact file path and is independently executable
- [x] Dependencies properly sequenced (models → services → handlers → integration)

**Total Tasks**: 50 tasks across 5 phases
**Parallel Opportunities**: 19 tasks can run in parallel across 5 groups
**Estimated Completion**: 35-45 development hours following TDD principles