# Research: Responsive Navigation Bar

## Container Queries Implementation

**Decision**: Use CSS `@container` queries with Tailwind CSS container query utilities
**Rationale**:
- More modular than viewport-based media queries
- Component-responsive rather than viewport-responsive
- Better encapsulation and reusability
- Supported in modern browsers (Chrome 105+, Firefox 110+, Safari 16+)

**Alternatives considered**:
- Viewport media queries: Less flexible, couples to viewport size
- JavaScript-based responsive: More complex, performance overhead
- CSS-in-JS solutions: Adds complexity for a template-based system

**Implementation approach**:
```css
/* Container setup */
.nav-container {
  container-type: inline-size;
  container-name: navigation;
}

/* Container queries */
@container navigation (max-width: 767px) {
  /* Mobile: bottom bar */
}

@container navigation (min-width: 768px) {
  /* Desktop: left sidebar */
}
```

## Heroicons Integration

**Decision**: Use Heroicons SVG directly in Templ templates
**Rationale**:
- Made by Tailwind team, perfect integration
- No additional dependencies
- Inline SVG for performance
- Easy customization with Tailwind classes

**Icons chosen**:
- Chat: `ChatBubbleLeftRightIcon` (outline version)
- Profile: `UserCircleIcon` (outline version)

**Implementation**:
```go
// Inline SVG in templ templates
<svg class="w-6 h-6" fill="none" stroke="currentColor">
  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="..." />
</svg>
```

## State Management Approach

**Decision**: Client-side JavaScript state management with sessionStorage persistence
**Rationale**:
- Simple useState pattern familiar to developers
- Persist across page reloads with sessionStorage
- No complex state library needed for 2-option navigation
- Integrates with existing tab-specific authentication

**State structure**:
```javascript
const navigationState = {
  activeSection: 'chat', // 'chat' | 'profile'
  layout: 'mobile' // 'mobile' | 'desktop' - derived from container size
}
```

## Transition Animation Strategy

**Decision**: CSS transitions with 300ms duration and easing
**Rationale**:
- Smooth user experience without being slow
- CSS transitions are performant (GPU accelerated)
- 300ms is industry standard for UI transitions
- `ease-in-out` provides natural feel

**Animation properties**:
```css
.navigation {
  transition: all 300ms ease-in-out;
  transition-property: transform, opacity, width, height;
}
```

## Tailwind CSS Container Queries Setup

**Decision**: Use Tailwind's built-in container query utilities (v3.2+)
**Rationale**:
- Native Tailwind support eliminates custom CSS
- Consistent with existing project approach
- Better developer experience with utility classes

**Container query classes**:
```html
<div class="@container">
  <nav class="@[768px]:flex-col @[768px]:h-screen flex-row fixed bottom-0 @[768px]:left-0 @[768px]:bottom-auto">
    <!-- Navigation content -->
  </nav>
</div>
```

## Browser Support Considerations

**Decision**: Target modern browsers with container query support
**Rationale**:
- DuckChat is a modern web app
- Container queries supported in Chrome 105+, Firefox 110+, Safari 16+
- Graceful degradation: fallback to mobile layout if not supported

**Fallback strategy**:
```css
/* Fallback for browsers without container query support */
@supports not (container-type: inline-size) {
  .navigation {
    /* Default to mobile layout */
    position: fixed;
    bottom: 0;
    flex-direction: row;
  }
}
```

## Component Architecture

**Decision**: Single responsive navigation component with container wrapper
**Rationale**:
- Keep it simple - single component handles all responsive behavior
- Container wrapper provides query context
- Easy integration into existing layout system

**Component structure**:
```
NavigationContainer (provides container context)
├── Navigation (responsive layout logic)
    ├── NavigationItem (Chat)
    └── NavigationItem (Profile)
```

## Integration with Existing DuckChat Architecture

**Decision**: Integrate as templ component in existing template system
**Rationale**:
- Consistent with existing toast and room list components
- Leverages existing Tailwind CSS setup
- Fits into current Go backend + templ frontend architecture

**Files to create**:
- `backend/web/templates/components/navigation.templ`
- Update existing layout templates to include navigation
- Add JavaScript for state management

## Container Query Gotchas and Solutions

**Key Gotchas Identified**:

1. **Container context required**: Parent element must have `container-type` set
   - Solution: Explicit container wrapper component

2. **Size containment affects layout**: Container queries can affect parent sizing
   - Solution: Use `inline-size` containment only (not `size`)

3. **Nested containers**: Inner containers inherit outer container context
   - Solution: Named containers with `container-name` property

4. **JavaScript size detection**: Container queries in CSS don't expose size to JS
   - Solution: Use ResizeObserver if JS needs to know container size

5. **Transition timing**: Layout shifts during container size changes
   - Solution: Transition timing carefully coordinated with layout changes

**Implementation safeguards**:
```css
.nav-container {
  container-type: inline-size; /* Only width, not height */
  container-name: navigation; /* Named container */
}

/* Specific container targeting */
@container navigation (max-width: 767px) {
  /* Mobile styles */
}
```

## Performance Considerations

**Decision**: Optimize for 60fps transitions and minimal layout thrashing
**Rationale**:
- Navigation is core UX - must be performant
- Frequent layout changes require optimization

**Optimizations planned**:
- Transform-based positioning (not layout properties)
- Will-change hints for GPU acceleration
- Minimal DOM manipulations during transitions
- Debounced resize handling

```css
.navigation {
  will-change: transform, opacity;
  transform: translateX(0); /* Use transform for positioning */
}
```