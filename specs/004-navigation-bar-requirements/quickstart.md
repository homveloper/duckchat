# Quickstart: Responsive Navigation Bar

## Overview

This quickstart guide walks through implementing and testing the responsive navigation bar component for DuckChat. The navigation provides seamless switching between Chat and Profile sections with responsive behavior using container queries.

## Prerequisites

- Go 1.21+
- Modern browser with container query support (Chrome 105+, Firefox 110+, Safari 16+)
- DuckChat project with Templ and Tailwind CSS configured
- Basic familiarity with container queries

## Quick Start (5 minutes)

### 1. Create the Navigation Component

```bash
# Create the component file
touch backend/web/templates/components/navigation.templ
```

Add the basic component structure:

```go
// backend/web/templates/components/navigation.templ
package components

import "duckchat/internal/ui/models"

templ Navigation(currentUser *models.UserContext) {
    <div class="@container navigation-container">
        <nav class="navigation">
            <button class="nav-item active" data-section="chat">
                <!-- Chat icon -->
                <span class="nav-label">Chat</span>
            </button>
            <button class="nav-item" data-section="profile">
                <!-- Profile icon -->
                <span class="nav-label">Profile</span>
            </button>
        </nav>
    </div>
}
```

### 2. Add Container Query CSS

Add to your Tailwind CSS configuration or custom styles:

```css
/* Ensure Tailwind container queries are enabled */
@tailwind base;
@tailwind components;
@tailwind utilities;

.navigation-container {
  container-type: inline-size;
  container-name: navigation;
}

.navigation {
  /* Mobile: bottom bar */
  @apply fixed bottom-0 left-0 right-0 flex flex-row;
  @apply bg-white border-t border-gray-200 px-4 py-3 gap-4;
  @apply transition-all duration-300 ease-in-out;
}

@container navigation (min-width: 768px) {
  .navigation {
    /* Desktop: left sidebar */
    @apply left-0 bottom-auto top-0 h-screen w-60 flex-col;
    @apply border-r border-t-0 px-6 py-6 gap-6;
  }
}
```

### 3. Test Basic Layout

1. **Start the server**: `go run ./cmd/server/. -static ./web/static`
2. **Open browser**: Navigate to `http://localhost:8080`
3. **Test responsive behavior**:
   - **Mobile view**: Resize to < 768px → navigation at bottom
   - **Desktop view**: Resize to ≥ 768px → navigation on left
   - **Verify**: Layout transitions smoothly

### 4. Add JavaScript State Management

```javascript
// Add to navigation.templ or separate JS file
<script>
class NavigationManager {
  constructor() {
    this.activeSection = 'chat';
    this.init();
  }

  init() {
    document.querySelectorAll('.nav-item').forEach(item => {
      item.addEventListener('click', (e) => {
        const section = e.currentTarget.dataset.section;
        this.setActiveSection(section);
      });
    });

    // Restore from sessionStorage
    const stored = sessionStorage.getItem('duckchat_navigation_state');
    if (stored) {
      const state = JSON.parse(stored);
      this.setActiveSection(state.activeSection);
    }
  }

  setActiveSection(section) {
    // Update UI
    document.querySelectorAll('.nav-item').forEach(item => {
      item.classList.toggle('active', item.dataset.section === section);
    });

    // Update state
    this.activeSection = section;

    // Persist
    sessionStorage.setItem('duckchat_navigation_state', JSON.stringify({
      activeSection: section,
      timestamp: Date.now()
    }));

    // Navigate (if needed)
    if (section === 'chat') window.location.href = '/rooms';
    if (section === 'profile') window.location.href = '/profile';
  }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  new NavigationManager();
});
</script>
```

## Manual Testing Checklist (10 minutes)

### Responsive Behavior Tests

1. **Container Query Test**:
   ```bash
   # Open browser dev tools
   # Resize container element to < 768px
   ✓ Navigation appears at bottom
   ✓ Horizontal layout (flex-row)
   ✓ Full width

   # Resize container element to ≥ 768px
   ✓ Navigation appears on left
   ✓ Vertical layout (flex-column)
   ✓ Fixed width sidebar
   ```

2. **Transition Test**:
   ```bash
   # Slowly resize across 768px breakpoint
   ✓ Smooth 300ms transition
   ✓ No layout flash or jump
   ✓ Elements maintain proportions
   ```

3. **State Persistence Test**:
   ```bash
   # Select "Profile" section
   # Refresh page
   ✓ Profile remains active
   # Select "Chat" section
   # Navigate away and back
   ✓ Chat remains active
   ```

### Cross-Browser Tests

4. **Browser Compatibility**:
   ```bash
   Chrome (105+): ✓ Container queries work
   Firefox (110+): ✓ Container queries work
   Safari (16+): ✓ Container queries work
   Older browsers: ✓ Graceful degradation to mobile layout
   ```

5. **Device Tests**:
   ```bash
   Mobile (portrait): ✓ Bottom navigation
   Mobile (landscape): ✓ Bottom navigation
   Tablet (portrait): ✓ Left sidebar
   Tablet (landscape): ✓ Left sidebar
   Desktop: ✓ Left sidebar
   ```

## Performance Validation (5 minutes)

### Animation Performance

1. **Test 60fps transitions**:
   ```bash
   # Open dev tools > Performance tab
   # Record while resizing across breakpoint
   ✓ No frame drops during transition
   ✓ GPU acceleration active (green layers)
   ✓ < 300ms total transition time
   ```

2. **Memory usage**:
   ```bash
   # Check memory tab while interacting
   ✓ No memory leaks from event listeners
   ✓ SessionStorage size < 1KB
   ✓ No excessive DOM manipulation
   ```

## Integration Testing (5 minutes)

### With Existing DuckChat Components

1. **Toast Integration**:
   ```bash
   # Navigate between sections while toasts are visible
   ✓ Toasts remain positioned correctly
   ✓ Navigation doesn't interfere with toast container
   ```

2. **Room List Integration**:
   ```bash
   # Ensure navigation works with room list
   ✓ Navigation visible alongside room list
   ✓ No z-index conflicts
   ✓ Proper spacing maintained
   ```

3. **Authentication Integration**:
   ```bash
   # Test with different user states
   ✓ Works with guest users
   ✓ Works with authenticated users
   ✓ Persists across login/logout
   ```

## Troubleshooting Common Issues

### Container Queries Not Working

```bash
# Check browser support
console.log(CSS.supports('container-type', 'inline-size'));

# Expected: true in Chrome 105+, Firefox 110+, Safari 16+
# If false: Browser doesn't support container queries

# Fallback solution: Use viewport media queries
@media (min-width: 768px) {
  .navigation { /* desktop styles */ }
}
```

### Layout Not Transitioning

```bash
# Check CSS transitions are applied
# Inspect element > Computed styles
# Look for: transition: all 300ms ease-in-out

# Common issues:
1. Missing container-type: inline-size
2. Incorrect @container syntax
3. Conflicting CSS specificity
```

### State Not Persisting

```javascript
// Debug sessionStorage
console.log(sessionStorage.getItem('duckchat_navigation_state'));

// Common issues:
1. JSON.parse() errors (check console)
2. Storage quota exceeded (clear storage)
3. Private/incognito mode limitations
```

## Next Steps

### Immediate (Ready for Production)

1. **Add Heroicons**: Replace placeholder icons with actual ChatBubbleLeftRightIcon and UserCircleIcon
2. **Style polish**: Add hover effects, focus states, active indicators
3. **Accessibility**: Add ARIA labels, keyboard navigation support

### Future Enhancements

1. **Animation improvements**: Custom easing functions, stagger animations
2. **Theme support**: Dark mode, custom color schemes
3. **Testing**: Automated browser tests, accessibility tests
4. **Performance**: Virtual scrolling for many nav items (if needed)

## Expected Results

After completing this quickstart:

- ✅ Navigation bar responsive behavior working
- ✅ Smooth transitions between layouts
- ✅ State persistence across page loads
- ✅ Integration with existing DuckChat components
- ✅ Cross-browser compatibility
- ✅ Performance targets met (60fps, <300ms transitions)

## Success Criteria Validation

1. **FR-001 & FR-002**: ✅ Chat and Profile sections accessible on all screen sizes
2. **FR-003 & FR-004**: ✅ Automatic positioning (bottom mobile, left desktop)
3. **FR-005**: ✅ Visual active state indication
4. **FR-006**: ✅ Seamless transitions between layouts
5. **FR-007**: ✅ Accessibility maintained during changes
6. **FR-008**: ✅ Navigation state persisted across transitions

Total quickstart time: ~25 minutes for a working prototype.