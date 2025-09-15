# Implementation Plan: DuckChat Real-time Chat Application


**Branch**: `001-duckchat-mvp` | **Date**: 2025-09-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-duckchat-mvp/spec.md`

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
Real-time chat application allowing multiple users to create and join chat rooms with instant messaging capabilities. Users can view chat history through scrolling and see messages appear in real-time. Technical approach uses Go with REST API + SSE for communication, Redis for storage, HTMX + Templ for frontend, and JWT authentication in a CQRS event-driven architecture.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**: Go Watermill (CQRS), Templ (templates), HTMX (frontend), Redis Stack (storage)
**Storage**: Redis Stack (primary data store, chat history, sessions)
**Testing**: Go testing package with integration tests
**Target Platform**: Linux server, Web browsers
**Project Type**: web (SSR frontend + backend API)
**Performance Goals**: Real-time messaging (<100ms latency), concurrent multi-user support
**Constraints**: Minimal complexity, MVP-focused, distributed environment ready
**Scale/Scope**: Multiple concurrent users per room, persistent chat history

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: 1 (backend web app with embedded frontend)
- Using framework directly? (yes - standard Go HTTP, Redis native commands in repository layer)
- Single data model? (yes - domain entities map directly to Redis JSON structures)
- Architecture patterns? (DDD + 3-Tier for clear boundaries, Repository pattern for data access)

**Architecture**:
- EVERY feature as library? (yes - each domain as self-contained package)
- Libraries listed:
  - auth (authentication handlers, service, repository, entities)
  - user (user profile handlers, service, repository, entities)
  - room (chat room handlers, service, repository, entities)
  - message (messaging handlers, service, repository, entities)
  - sse (shared event broadcasting service)
- CLI per library: (N/A - web application, not CLI-focused)
- Library docs: (documentation in CLAUDE.md format)
- Domain-centric compliance: Each domain contains all related layers

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? (yes - contract tests written first)
- Git commits show tests before implementation? (yes - tests created in Phase 1)
- Order: Contract→Integration→E2E→Unit strictly followed? (yes - API contracts → room integration → user flows → units)
- Real dependencies used? (yes - actual Redis instance for integration tests)
- Integration tests for: new libraries, contract changes, shared schemas? (yes - all JSON-RPC endpoints)
- FORBIDDEN: Implementation before test, skipping RED phase

**Observability**:
- Structured logging included? (yes - Go standard log with structured fields)
- Frontend logs → backend? (yes - errors sent via API)
- Error context sufficient? (yes - JSON-RPC error objects with context)

**Versioning**:
- Version number assigned? (1.0.0 - MVP initial version)
- BUILD increments on every change? (yes - semantic versioning planned)
- Breaking changes handled? (yes - API versioning strategy documented)

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

# Option 2: Web application with Domain-Centric Architecture
backend/
├── cmd/
│   └── server/         # Main application entry point
├── internal/
│   ├── auth/           # Authentication domain
│   │   ├── handler.go  # JSON-RPC handlers
│   │   ├── service.go  # Business logic
│   │   ├── repository.go # Redis operations
│   │   └── entity.go   # Domain entities & value objects
│   ├── user/           # User management domain
│   │   ├── handler.go  # User profile handlers
│   │   ├── service.go  # User business logic
│   │   ├── repository.go # User Redis operations
│   │   └── entity.go   # User domain model
│   ├── room/           # Chat room domain
│   │   ├── handler.go  # Room management handlers
│   │   ├── service.go  # Room business logic
│   │   ├── repository.go # Room Redis operations
│   │   └── entity.go   # ChatRoom domain model
│   ├── message/        # Messaging domain
│   │   ├── handler.go  # Message handlers
│   │   ├── service.go  # Message business logic
│   │   ├── repository.go # Message Redis operations
│   │   └── entity.go   # Message domain model
│   └── sse/            # Server-Sent Events (shared)
│       ├── handler.go  # SSE stream management
│       └── service.go  # Event broadcasting
├── web/
│   ├── templates/      # Templ template files
│   └── static/         # CSS, JS, assets
└── tests/
    ├── contract/       # API contract tests
    ├── integration/    # Full system tests
    └── unit/           # Domain unit tests

# Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure]
```

**Structure Decision**: Option 2: Web application with Domain-Centric Architecture
- **Domain-Centric**: Each domain (auth, user, room, message) contains all related code
- **Vertical Slices**: handler.go, service.go, repository.go, entity.go per domain
- **3-Tier per Domain**: Each domain maintains presentation → application → data layers
- **No Infrastructure Abstraction**: Repository directly handles Redis operations

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
- Load `/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P] 
- Each user story → integration test task
- Implementation tasks to make tests pass

**Ordering Strategy**:
- TDD order: Tests before implementation 
- Dependency order: Models before services before UI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: 25-30 numbered, ordered tasks in tasks.md

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
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*