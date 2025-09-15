# Tasks: Create Templ UI Components for DuckChat

**Input**: Design documents from `/specs/003-create-templ-ui/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Tech stack: Go 1.21+, Templ, Tailwind CSS, HTMX
   → Project structure: Web app - extending existing backend
2. Load design documents:
   → data-model.md: 5 UI state entities → model tasks
   → contracts/: 2 contract files → test tasks
   → quickstart.md: 5 user stories → integration tests
3. Generate tasks by category:
   → Setup: Templ installation, Tailwind config, project structure
   → Tests: contract tests, component tests, integration tests
   → Core: template components, state models, HTMX handlers
   → Integration: responsive design, SSE integration, authentication
   → Polish: performance tests, accessibility, cross-browser validation
4. Apply task rules:
   → Different template files = mark [P] for parallel
   → Shared CSS/config files = sequential (no [P])
   → Tests before template implementation (TDD)
5. Total tasks: 28 tasks across 5 phases
6. Parallel execution: 15 tasks marked [P] for concurrent execution
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Web templates**: `web/templates/` for .templ files
- **Static assets**: `web/static/` for CSS, images
- **Tests**: `tests/ui/`, `tests/contract/`, `tests/integration/`
- **Backend handlers**: `internal/*/handler.go` (existing pattern)

## Phase 3.1: Setup
- [ ] T001 Install Templ CLI and verify Go-Templ toolchain setup
- [ ] T002 Configure Tailwind CSS with mobile-first breakpoints in `web/static/tailwind.config.js`
- [ ] T003 [P] Create web templates directory structure `web/templates/components/`, `web/templates/pages/`, `web/templates/layouts/`
- [ ] T004 [P] Set up browser automation testing with Go and Playwright in `tests/ui/setup_test.go`

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests
- [ ] T005 [P] Contract test GET /login endpoint in `tests/contract/test_login_get.go`
- [ ] T006 [P] Contract test POST /login endpoint in `tests/contract/test_login_post.go`
- [ ] T007 [P] Contract test GET /rooms endpoint in `tests/contract/test_rooms_get.go`
- [ ] T008 [P] Contract test POST /rooms endpoint in `tests/contract/test_rooms_post.go`
- [ ] T009 [P] Contract test GET /chat/{roomId} endpoint in `tests/contract/test_chat_get.go`
- [ ] T010 [P] Contract test POST /messages HTMX endpoint in `tests/contract/test_messages_post.go`
- [ ] T011 [P] Contract test GET /events/{roomId} SSE endpoint in `tests/contract/test_events_sse.go`

### Component Tests
- [ ] T012 [P] Template rendering test for BaseLayout component in `tests/ui/test_base_layout.go`
- [ ] T013 [P] Template rendering test for LoginForm component in `tests/ui/test_login_form.go`
- [ ] T014 [P] Template rendering test for RoomList component in `tests/ui/test_room_list.go`
- [ ] T015 [P] Template rendering test for ChatPage component in `tests/ui/test_chat_page.go`
- [ ] T016 [P] Template rendering test for MessageInput component in `tests/ui/test_message_input.go`

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### UI State Models
- [ ] T017 [P] LoginFormState struct and validation in `internal/ui/models/login_state.go`
- [ ] T018 [P] ChatPageState struct and methods in `internal/ui/models/chat_state.go`
- [ ] T019 [P] MessageInputState struct and validation in `internal/ui/models/message_state.go`
- [ ] T020 [P] RoomCreationState struct and validation in `internal/ui/models/room_state.go`
- [ ] T021 [P] ResponsiveLayoutState struct and detection in `internal/ui/models/layout_state.go`

### Templ Components
- [ ] T022 [P] BaseLayout template with responsive container in `web/templates/layouts/base.templ`
- [ ] T023 [P] LoginForm template with validation display in `web/templates/components/login_form.templ`
- [ ] T024 [P] RoomList template with creation interface in `web/templates/components/room_list.templ`
- [ ] T025 [P] ChatPage template with message display in `web/templates/pages/chat.templ`
- [ ] T026 [P] MessageInput template with HTMX integration in `web/templates/components/message_input.templ`

## Phase 3.4: Integration
- [ ] T027 Integrate templates with existing auth handlers in `internal/auth/handler.go`
- [ ] T028 Add HTMX response handling to message handlers in `internal/message/handler.go`
- [ ] T029 Implement SSE event streaming for real-time updates in `internal/sse/handler.go`
- [ ] T030 Configure responsive breakpoints and CSS optimization in Tailwind build process

## Phase 3.5: Polish
- [ ] T031 [P] Cross-browser compatibility tests in `tests/ui/test_browser_compatibility.go`
- [ ] T032 [P] Mobile viewport and touch interaction tests in `tests/ui/test_mobile_responsive.go`
- [ ] T033 [P] Accessibility testing (keyboard nav, screen readers) in `tests/ui/test_accessibility.go`
- [ ] T034 [P] Performance testing (template render times) in `tests/performance/test_template_performance.go`
- [ ] T035 Optimize CSS bundle size and remove unused Tailwind classes

## Dependencies

### Setup Dependencies
- T001 blocks all template and CSS tasks
- T002 blocks all styling tasks (T022-T026, T030)
- T003 blocks all template creation tasks (T022-T026)
- T004 blocks all UI testing tasks (T012-T016, T031-T033)

### TDD Dependencies
- Tests (T005-T016) MUST complete and FAIL before implementation (T017-T026)
- Contract tests (T005-T011) before handler integration (T027-T029)
- Component tests (T012-T016) before template implementation (T022-T026)

### Implementation Dependencies
- State models (T017-T021) before template implementation (T022-T026)
- Templates (T022-T026) before integration (T027-T029)
- Integration (T027-T029) before optimization (T030, T035)
- Core implementation (T017-T029) before polish (T031-T034)

## Parallel Execution Examples

### Phase 3.1 Setup (can run T003-T004 together)
```bash
# Launch T003-T004 together:
Task: "Create web templates directory structure"
Task: "Set up browser automation testing with Go and Playwright"
```

### Phase 3.2 Contract Tests (can run T005-T011 together)
```bash
# Launch all contract tests in parallel:
Task: "Contract test GET /login endpoint in tests/contract/test_login_get.go"
Task: "Contract test POST /login endpoint in tests/contract/test_login_post.go"
Task: "Contract test GET /rooms endpoint in tests/contract/test_rooms_get.go"
Task: "Contract test POST /rooms endpoint in tests/contract/test_rooms_post.go"
Task: "Contract test GET /chat/{roomId} endpoint in tests/contract/test_chat_get.go"
Task: "Contract test POST /messages HTMX endpoint in tests/contract/test_messages_post.go"
Task: "Contract test GET /events/{roomId} SSE endpoint in tests/contract/test_events_sse.go"
```

### Phase 3.2 Component Tests (can run T012-T016 together)
```bash
# Launch all component tests in parallel:
Task: "Template rendering test for BaseLayout component in tests/ui/test_base_layout.go"
Task: "Template rendering test for LoginForm component in tests/ui/test_login_form.go"
Task: "Template rendering test for RoomList component in tests/ui/test_room_list.go"
Task: "Template rendering test for ChatPage component in tests/ui/test_chat_page.go"
Task: "Template rendering test for MessageInput component in tests/ui/test_message_input.go"
```

### Phase 3.3 State Models (can run T017-T021 together)
```bash
# Launch all state model implementations in parallel:
Task: "LoginFormState struct and validation in internal/ui/models/login_state.go"
Task: "ChatPageState struct and methods in internal/ui/models/chat_state.go"
Task: "MessageInputState struct and validation in internal/ui/models/message_state.go"
Task: "RoomCreationState struct and validation in internal/ui/models/room_state.go"
Task: "ResponsiveLayoutState struct and detection in internal/ui/models/layout_state.go"
```

### Phase 3.3 Template Components (can run T022-T026 together)
```bash
# Launch all template implementations in parallel:
Task: "BaseLayout template with responsive container in web/templates/layouts/base.templ"
Task: "LoginForm template with validation display in web/templates/components/login_form.templ"
Task: "RoomList template with creation interface in web/templates/components/room_list.templ"
Task: "ChatPage template with message display in web/templates/pages/chat.templ"
Task: "MessageInput template with HTMX integration in web/templates/components/message_input.templ"
```

### Phase 3.5 Polish Tests (can run T031-T034 together)
```bash
# Launch all polish tests in parallel:
Task: "Cross-browser compatibility tests in tests/ui/test_browser_compatibility.go"
Task: "Mobile viewport and touch interaction tests in tests/ui/test_mobile_responsive.go"
Task: "Accessibility testing in tests/ui/test_accessibility.go"
Task: "Performance testing in tests/performance/test_template_performance.go"
```

## User Story Mapping

### Story 1: Mobile Login Experience
- **Tests**: T005, T006, T013 (login endpoints and form component)
- **Implementation**: T017, T023, T027 (login state, template, integration)
- **Validation**: T031, T032 (browser compatibility, mobile responsive)

### Story 2: Room Creation Interface
- **Tests**: T007, T008, T014 (rooms endpoints and list component)
- **Implementation**: T020, T024, T027 (room state, template, integration)
- **Validation**: T032, T033 (mobile responsive, accessibility)

### Story 3: Chat Interface Usability
- **Tests**: T009, T010, T015, T016 (chat endpoint, message handling, components)
- **Implementation**: T018, T019, T025, T026, T028 (chat/message state, templates, HTMX)
- **Validation**: T032, T034 (mobile responsive, performance)

### Story 4: Responsive Layout Transition
- **Tests**: T012, T032 (base layout component, mobile responsive)
- **Implementation**: T021, T022, T030 (layout state, base template, CSS optimization)
- **Validation**: T031, T032 (cross-browser, responsive design)

### Story 5: Real-time Message Updates
- **Tests**: T011, T015, T016 (SSE endpoint, chat page, message input)
- **Implementation**: T018, T026, T029 (chat state, message input template, SSE integration)
- **Validation**: T034 (performance testing for real-time updates)

## File Structure Created
```
web/
├── templates/
│   ├── layouts/
│   │   └── base.templ                    # T022
│   ├── pages/
│   │   └── chat.templ                    # T025
│   └── components/
│       ├── login_form.templ              # T023
│       ├── room_list.templ               # T024
│       └── message_input.templ           # T026
├── static/
│   └── tailwind.config.js                # T002

internal/
└── ui/
    └── models/
        ├── login_state.go                # T017
        ├── chat_state.go                 # T018
        ├── message_state.go              # T019
        ├── room_state.go                 # T020
        └── layout_state.go               # T021

tests/
├── contract/
│   ├── test_login_get.go                 # T005
│   ├── test_login_post.go                # T006
│   ├── test_rooms_get.go                 # T007
│   ├── test_rooms_post.go                # T008
│   ├── test_chat_get.go                  # T009
│   ├── test_messages_post.go             # T010
│   └── test_events_sse.go                # T011
├── ui/
│   ├── setup_test.go                     # T004
│   ├── test_base_layout.go               # T012
│   ├── test_login_form.go                # T013
│   ├── test_room_list.go                 # T014
│   ├── test_chat_page.go                 # T015
│   ├── test_message_input.go             # T016
│   ├── test_browser_compatibility.go     # T031
│   ├── test_mobile_responsive.go         # T032
│   └── test_accessibility.go             # T033
└── performance/
    └── test_template_performance.go      # T034
```

## Validation Checklist
*GATE: Checked before task execution*

- [x] All contracts have corresponding tests (T005-T011)
- [x] All UI state entities have model tasks (T017-T021)
- [x] All components have template tests (T012-T016)
- [x] All tests come before implementation (Phase 3.2 before 3.3)
- [x] Parallel tasks truly independent (different files/packages)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] User stories mapped to specific tasks
- [x] Dependencies clearly documented
- [x] Total: 35 tasks, 20 marked [P] for parallel execution