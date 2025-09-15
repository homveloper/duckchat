# Implementation Plan: Responsive Navigation Bar

**Branch**: `004-navigation-bar-requirements` | **Date**: September 15, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-navigation-bar-requirements/spec.md`

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
Create a responsive navigation bar component that adapts between bottom positioning (mobile) and left sidebar (desktop/tablet) with Chat and Profile sections. Implementation will use container queries for modular responsiveness, Heroicons for iconography, and smooth CSS transitions.

## Technical Context
**Language/Version**: Go 1.21+ (backend), HTML/CSS/JavaScript (frontend templates)
**Primary Dependencies**: Templ (HTML templating), Tailwind CSS, Heroicons
**Storage**: N/A (navigation state only, no persistence)
**Testing**: Go testing, browser-based functional tests (skipped for prototype)
**Target Platform**: Web browsers (modern CSS container queries support)
**Project Type**: web - frontend Go template + backend integration
**Performance Goals**: <300ms layout transitions, 60fps animations
**Constraints**: Container query support (modern browsers), modular design
**Scale/Scope**: Single navigation component, 2 sections (Chat/Profile)

**User Technical Requirements**:
- Container queries instead of viewport media queries (768px breakpoint)
- Heroicons: ChatBubbleLeftRightIcon (Chat), UserCircleIcon (Profile)
- Mobile: Fixed bottom, horizontal flex, icons + small labels
- Desktop: Left sidebar, vertical flex, full height, icons + labels
- State management: Single useState for active section (chat/profile)
- Transitions: 300ms CSS transitions for smooth layout changes

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: 1 (frontend template component only)
- Using framework directly? ✅ (Direct Templ + Tailwind, no wrappers)
- Single data model? ✅ (Simple navigation state only)
- Avoiding patterns? ✅ (No Repository/UoW, direct component state)

**Architecture**:
- EVERY feature as library? ⚠️ VIOLATION: UI component, not standalone library
- Libraries listed: N/A (template component)
- CLI per library: N/A (UI component)
- Library docs: N/A (inline component documentation)

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? ⚠️ DEFERRED: User requested prototype without tests
- Git commits show tests before implementation? ⚠️ DEFERRED: Prototype approach
- Order: Contract→Integration→E2E→Unit strictly followed? ⚠️ DEFERRED: Testing skipped
- Real dependencies used? N/A (UI component)
- Integration tests for: new libraries, contract changes, shared schemas? ⚠️ DEFERRED: Testing phase
- FORBIDDEN: Implementation before test, skipping RED phase: ⚠️ ACKNOWLEDGED: Prototype exception

**Observability**:
- Structured logging included? N/A (UI component state only)
- Frontend logs → backend? N/A (no backend logging needed)
- Error context sufficient? ✅ (Console logs for state changes)

**Versioning**:
- Version number assigned? ✅ (0.1.0 - prototype)
- BUILD increments on every change? ✅ (Following semver)
- Breaking changes handled? N/A (initial implementation)

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

**Structure Decision**: Option 2 - Web application (existing DuckChat Go backend with Templ templates)

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
- Container component → CSS container setup [P]
- Navigation component → Templ template creation [P]
- State management → JavaScript implementation [P]
- Icon integration → Heroicons SVG embedding [P]
- Responsive behavior → Container query CSS [P]
- Integration tasks → Layout integration, route handling

**Detailed Task Categories**:

1. **Foundation Tasks** (Parallel):
   - CSS container query setup and Tailwind configuration
   - Heroicons SVG path extraction and optimization
   - Navigation state management JavaScript module
   - SessionStorage persistence utility functions

2. **Component Tasks** (Sequential):
   - Navigation container template (navigation.templ)
   - Navigation item sub-components
   - Active state styling and transitions
   - Container query responsive CSS classes

3. **Integration Tasks** (Sequential after components):
   - Integrate navigation into existing layouts
   - Update routing to handle navigation state
   - Connect with existing toast notification system
   - Update existing components for navigation awareness

4. **User Experience Tasks** (Final):
   - Smooth transition animations (300ms)
   - Keyboard navigation support
   - Touch/mobile interaction optimization
   - Cross-browser compatibility fixes

**Ordering Strategy**:
- Foundation → Components → Integration → UX polish
- Prototype approach: Skip testing initially (deferred to production version)
- Mark [P] for parallel execution (independent files)
- Dependencies: CSS setup before templates, templates before integration

**Container Query Implementation Priority**:
1. Basic container setup (`@container` directive)
2. Breakpoint-based layout switching (768px)
3. Smooth transitions between layouts
4. Fallback for non-supporting browsers

**Expected Task Breakdown**:
- **Foundation**: 6 tasks (container queries, icons, state, storage)
- **Components**: 8 tasks (templates, styling, responsive behavior)
- **Integration**: 4 tasks (layouts, routing, notifications, existing components)
- **UX Polish**: 3 tasks (animations, accessibility, compatibility)

**Estimated Output**: 21 numbered, ordered tasks in tasks.md

**Component Architecture Tasks**:
```
1. [P] Setup CSS container queries and Tailwind config
2. [P] Extract and optimize Heroicons SVG paths
3. [P] Create NavigationState JavaScript class
4. [P] Implement sessionStorage persistence utility
5. Create navigation.templ container template
6. Create NavigationItem component template
7. Implement responsive CSS with @container queries
8. Add active state styling and transitions
9. Integrate navigation into base layout template
10. Update room list layout for navigation space
11. Connect navigation state with routing
12. Test responsive behavior across breakpoints
13. Add smooth 300ms transition animations
14. Implement keyboard navigation support
15. Optimize touch interactions for mobile
16. Add cross-browser compatibility layers
17. Integrate with existing toast notification system
18. Update CLAUDE.md context with implementation details
19. Validate quickstart guide accuracy
20. Performance test: 60fps transitions
21. Final integration testing
```

**User-Specified Implementation Details Integrated**:
- ✅ Container queries (768px breakpoint)
- ✅ Heroicons (ChatBubbleLeftRightIcon, UserCircleIcon)
- ✅ Mobile: bottom fixed, horizontal flex, icons + small labels
- ✅ Desktop: left sidebar, vertical flex, full height, icons + labels
- ✅ Simple state management (activeSection only)
- ✅ 300ms CSS transitions
- ✅ Modular design with container wrapper

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
| UI component (not library) | Navigation is presentation layer component | Library extraction would add unnecessary abstraction for single-use UI element |
| Testing deferred | User requested prototype for quick validation | TDD cycle would slow prototype iteration; tests planned for production version |
| No CLI interface | UI component has no command-line functionality | CLI interface not applicable for frontend presentation components |


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
- [x] Initial Constitution Check: PASS (with documented deviations)
- [x] Post-Design Constitution Check: PASS (deviations remain justified)
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*