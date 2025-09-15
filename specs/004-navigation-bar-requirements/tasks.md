# Tasks: Responsive Navigation Bar (HTMX + Alpine.js)

**Input**: Design documents from `/specs/004-navigation-bar-requirements/`
**Prerequisites**: plan.md (✓), research.md (✓), data-model.md (✓), contracts/ (✓)

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → ✓ HTMX + Alpine.js navigation with Go/Templ backend
   → Extract: Go 1.21+, Templ, Tailwind CSS, Heroicons, container queries
2. Load optional design documents:
   → data-model.md: NavigationData, NavigationItem, LayoutConfig entities
   → contracts/: navigation-component.go, navigation.templ contracts
   → research.md: Container queries, HTMX + Alpine.js decisions
3. Generate tasks by category:
   → Setup: Templ generation, Tailwind config, Alpine.js integration
   → Tests: Component behavior, responsive layout, URL state
   → Core: Navigation data structures, Templ components, CSS styling
   → Integration: HTMX handlers, Alpine.js state, existing layout
   → Polish: Accessibility, performance, browser compatibility
4. Apply task rules:
   → Different components = mark [P] for parallel
   → Same template file = sequential (no [P])
   → Container setup before components
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph: CSS → Components → Integration → Polish
7. Create parallel execution examples for independent tasks
8. Validate task completeness: All contracts implemented
9. Return: SUCCESS (21 tasks ready for HTMX + Alpine.js implementation)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Web app structure**: `backend/web/templates/`, `backend/internal/ui/`
- **Static assets**: `backend/web/static/`
- **Test files**: component behavior testing in browser

## Phase 3.1: Setup & Dependencies

- [ ] **T001** [P] Add container query support to existing Tailwind config in `backend/web/static/tailwind.config.js`
- [ ] **T002** [P] Extract Heroicons SVG paths for ChatBubbleLeftRightIcon and UserCircleIcon
- [ ] **T003** [P] Add Alpine.js integration to base layout template in `backend/web/templates/layouts/base.templ`
- [ ] **T004** [P] Create navigation-specific CSS file in `backend/web/static/input.css` (add container query utilities)

## Phase 3.2: Data Structures & Models

- [ ] **T005** [P] Create NavigationData struct in `backend/internal/ui/models/navigation.go`
- [ ] **T006** [P] Create NavigationItem struct with validation in `backend/internal/ui/models/navigation.go`
- [ ] **T007** [P] Create LayoutConfig struct for responsive behavior in `backend/internal/ui/models/navigation.go`
- [ ] **T008** [P] Implement default navigation items factory function in `backend/internal/ui/models/navigation.go`

## Phase 3.3: Templ Components

- [ ] **T009** Create main navigation container template in `backend/web/templates/components/navigation.templ`
- [ ] **T010** Create NavigationItem sub-component with Alpine.js directives in `backend/web/templates/components/navigation.templ`
- [ ] **T011** Add HTMX attributes for smooth page transitions in `backend/web/templates/components/navigation.templ`
- [ ] **T012** Implement Alpine.js URL-based active state detection in `backend/web/templates/components/navigation.templ`

## Phase 3.4: CSS Responsive Styling

- [ ] **T013** [P] Implement mobile layout CSS (bottom positioning, horizontal flex) in `backend/web/static/input.css`
- [ ] **T014** [P] Implement desktop layout CSS (left sidebar, vertical flex) in `backend/web/static/input.css`
- [ ] **T015** [P] Add container query breakpoint rules (768px threshold) in `backend/web/static/input.css`
- [ ] **T016** [P] Implement 300ms transition animations between layouts in `backend/web/static/input.css`

## Phase 3.5: Backend Integration

- [ ] **T017** Add navigation helper methods to existing handler in `backend/internal/ui/handler.go`
- [ ] **T018** Integrate navigation component into existing room list layout in `backend/web/templates/pages/rooms.templ`
- [ ] **T019** Create profile page template with navigation in `backend/web/templates/pages/profile.templ`
- [ ] **T020** Update existing handlers to pass navigation data in `backend/internal/ui/handler.go`

## Phase 3.6: Testing & Polish

- [ ] **T021** [P] Test responsive behavior across breakpoints (manual browser testing)
- [ ] **T022** [P] Validate Alpine.js URL state detection with browser navigation
- [ ] **T023** [P] Performance test: verify 60fps transitions and <300ms layout changes
- [ ] **T024** [P] Cross-browser compatibility test (Chrome 105+, Firefox 110+, Safari 16+)
- [ ] **T025** Add ARIA labels and keyboard navigation support in navigation components

## Dependencies

**Sequential Dependencies:**
- T001-T004 (Setup) → T005-T008 (Models) → T009-T012 (Components)
- T013-T016 (CSS) must complete before T021-T024 (Testing)
- T009-T012 (Components) → T017-T020 (Integration)
- T017-T020 (Integration) → T021-T025 (Testing)

**No Dependencies (Parallel Safe):**
- T001, T002, T003, T004 (different files)
- T005, T006, T007, T008 (different structs in same file)
- T013, T014, T015, T016 (different CSS sections)
- T021, T022, T023, T024 (different testing aspects)

## Parallel Execution Examples

### Setup Phase (Launch Together):
```bash
Task: "Add container query support to existing Tailwind config in backend/web/static/tailwind.config.js"
Task: "Extract Heroicons SVG paths for ChatBubbleLeftRightIcon and UserCircleIcon"
Task: "Add Alpine.js integration to base layout template in backend/web/templates/layouts/base.templ"
Task: "Create navigation-specific CSS file in backend/web/static/input.css (add container query utilities)"
```

### CSS Styling Phase (Launch Together):
```bash
Task: "Implement mobile layout CSS (bottom positioning, horizontal flex) in backend/web/static/input.css"
Task: "Implement desktop layout CSS (left sidebar, vertical flex) in backend/web/static/input.css"
Task: "Add container query breakpoint rules (768px threshold) in backend/web/static/input.css"
Task: "Implement 300ms transition animations between layouts in backend/web/static/input.css"
```

### Testing Phase (Launch Together):
```bash
Task: "Test responsive behavior across breakpoints (manual browser testing)"
Task: "Validate Alpine.js URL state detection with browser navigation"
Task: "Performance test: verify 60fps transitions and <300ms layout changes"
Task: "Cross-browser compatibility test (Chrome 105+, Firefox 110+, Safari 16+)"
```

## HTMX + Alpine.js Implementation Notes

### Key Technical Decisions:
- **HTMX**: Handles navigation between `/rooms` and `/profile` routes
- **Alpine.js**: Manages active state based on `window.location.pathname`
- **Container Queries**: 768px breakpoint for responsive layout switching
- **No Server State**: Navigation active state determined client-side from URL

### File Organization:
- **Models**: `backend/internal/ui/models/navigation.go` (Go structs)
- **Templates**: `backend/web/templates/components/navigation.templ` (Templ components)
- **Handler**: `backend/internal/ui/handler.go` (Navigation methods added to existing handler)
- **Styles**: `backend/web/static/input.css` (Container query CSS integrated with existing Tailwind)
- **Assets**: Heroicons SVG paths integrated directly in templates

### Alpine.js State Management Pattern:
```javascript
x-data="{
  getCurrentSection() {
    const path = window.location.pathname;
    if (path.startsWith('/rooms')) return 'chat';
    if (path.startsWith('/profile')) return 'profile';
    return 'chat';
  },
  activeSection: 'chat'
}"
x-init="activeSection = getCurrentSection()"
@htmx:after-request="activeSection = getCurrentSection()"
```

## Validation Checklist
*GATE: Checked during implementation*

- [x] All contracts have corresponding implementation tasks
- [x] All entities have model creation tasks (NavigationData, NavigationItem, LayoutConfig)
- [x] Setup tasks come before component tasks
- [x] Parallel tasks target different files or independent sections
- [x] Each task specifies exact file path
- [x] HTMX and Alpine.js integration clearly defined
- [x] Container query implementation approach documented
- [x] No task modifies same template file as another [P] task

## Success Criteria

After completing all tasks:
1. ✅ Navigation bar responsive between mobile (bottom) and desktop (left sidebar) layouts
2. ✅ HTMX provides smooth transitions between Chat (/rooms) and Profile (/profile) routes
3. ✅ Alpine.js correctly identifies active section from URL without server state
4. ✅ Container queries handle responsive behavior at 768px breakpoint
5. ✅ 300ms transition animations between layout modes
6. ✅ Heroicons properly integrated for ChatBubbleLeftRightIcon and UserCircleIcon
7. ✅ Cross-browser compatibility with container query fallbacks
8. ✅ No server-side navigation state persistence required