# Implementation Plan: Chat System Core Features


**Branch**: `006-i-m-planning` | **Date**: 2025-09-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-i-m-planning/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
4. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
5. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, or `GEMINI.md` for Gemini CLI).
6. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
7. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
8. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
A real-time chat system enabling users to create chat rooms, join existing rooms, send/receive messages with real-time synchronization, and scroll through message history. Core architecture uses REST APIs for CRUD operations, Watermill CQRS for event-driven message distribution, and Server-Sent Events (SSE) for real-time delivery to connected clients.

## Technical Context
**Language/Version**: Go 1.21+ with Templ templating system
**Primary Dependencies**: Redis (with RedisJSON), Watermill CQRS (Redis Streams transport), Server-Sent Events (SSE), HTMX (dynamic interactions), Tailwind CSS
**Storage**: Redis with RedisJSON for documents, Redis Streams for messages and events
**Testing**: Go testing framework (`go test`)
**Target Platform**: Web application (server-side rendering)
**Project Type**: web (frontend + backend combined via Templ SSR)
**Performance Goals**: Real-time message delivery, infinite scroll pagination via Redis Streams
**Constraints**: Distributed server support via Redis Streams event sourcing, reliable message broadcasting
**Scale/Scope**: Multi-user real-time chat with room-based organization

**Architecture Details from User Input**:
- **API Protocol**: JSON-RPC 2.0 hybrid with all requests via POST, always returning HTTP 200 with JSON-RPC 2.0 response structure
- **Error Handling**: All errors returned as JSON-RPC 2.0 error responses with standard and custom error codes, including middleware layer errors
- **Storage**: Redis as persistent storage using RedisJSON for structured data and Redis Streams for message ordering
- **Message Queue**: Watermill CQRS with Redis Streams transport for event processing
- **API Methods**: rooms.list, rooms.create, rooms.join, rooms.leave, messages.send, messages.history, sse.connect
- **Message Flow**: Client → JSON-RPC 2.0 API → Redis persistence → Watermill Redis Stream event → SSE broadcast to room participants
- **Event Processing**: MessageSent events published to Redis Streams, consumed by event processors for SSE broadcasting
- **Real-time**: SSE connections managed via Redis Sets, ensuring all participants receive messages immediately across distributed servers

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: [#] (max 3 - e.g., api, cli, tests)
- Using framework directly? (no wrapper classes)
- Single data model? (no DTOs unless serialization differs)
- Avoiding patterns? (no Repository/UoW without proven need)

**Architecture**:
- EVERY feature as library? (no direct app code)
- Libraries listed: [name + purpose for each]
- CLI per library: [commands with --help/--version/--format]
- Library docs: llms.txt format planned?

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? (test MUST fail first)
- Git commits show tests before implementation?
- Order: Contract→Integration→E2E→Unit strictly followed?
- Real dependencies used? (actual DBs, not mocks)
- Integration tests for: new libraries, contract changes, shared schemas?
- FORBIDDEN: Implementation before test, skipping RED phase

**Observability**:
- Structured logging included?
- Frontend logs → backend? (unified stream)
- Error context sufficient?

**Versioning**:
- Version number assigned? (MAJOR.MINOR.BUILD)
- BUILD increments on every change?
- Breaking changes handled? (parallel tests, migration plan)

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
# Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure]
```

**Structure Decision**: Using existing DuckChat structure with backend/ containing Go + Templ templates, web/templates/ for UI components, since this is a web application using server-side rendering

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `/scripts/bash/update-agent-context.sh claude` for your AI assistant
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
Based on the contracts (chat-api.yaml), data model (3 entities + views), and quickstart scenarios, the /tasks command will:

1. **Contract Tests** (from chat-api.yaml):
   - POST /api/rooms.list contract test [P]
   - POST /api/rooms.create contract test [P]
   - POST /api/rooms.join contract test [P]
   - POST /api/rooms.leave contract test [P]
   - POST /api/messages.history contract test [P]
   - POST /api/messages.send contract test [P]
   - POST /api/sse.connect contract test [P]

2. **Data Model Tasks** (from data-model.md):
   - Create ChatRoom entity and validation [P]
   - Create Message entity and validation [P]
   - Create RoomParticipant entity and validation [P]
   - Create database migration scripts
   - Create view models (MessageWithUser, RoomWithMetadata)

3. **Integration Tests** (from quickstart scenarios):
   - Room creation and auto-join integration test
   - Join existing room integration test
   - Send/receive messages integration test
   - Real-time SSE delivery integration test
   - Message history pagination integration test
   - Error handling validation integration test

4. **Implementation Tasks** (to make tests pass):
   - Implement room creation handler
   - Implement room join handler
   - Implement message sending handler
   - Implement message history handler
   - Implement SSE connection management
   - Implement Watermill CQRS event processing
   - Create Templ templates for chat UI
   - Integrate HTMX for dynamic interactions

**Ordering Strategy**:
1. **Phase A - Foundation**: Database migration, entity models (TDD: tests → implementation)
2. **Phase B - Core APIs**: Contract tests → REST endpoint implementations
3. **Phase C - Real-time**: SSE + Watermill integration tests → implementations
4. **Phase D - Frontend**: UI template tests → Templ template implementations
5. **Phase E - Integration**: End-to-end scenario tests → full system validation

**Parallel Execution Markers**:
- [P] Contract tests (independent endpoints)
- [P] Entity model creation (separate structs)
- [P] UI template creation (separate components)

**Estimated Task Count**: 30-35 tasks
- 7 contract tests (JSON-RPC 2.0 hybrid endpoints)
- 5 data model tasks (Redis-based entities)
- 6 integration test scenarios
- 9 implementation tasks (including Redis operations)
- 3-8 UI template tasks (Templ components)

**Dependencies**:
- Database entities before service handlers
- Service handlers before SSE integration
- SSE integration before UI templates
- All components before end-to-end tests

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (using existing DuckChat architecture)
- [x] Post-Design Constitution Check: PASS (simple REST + SSE, minimal complexity)
- [x] All NEEDS CLARIFICATION resolved (via research.md)
- [x] Complexity deviations documented (none required - follows existing patterns)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*