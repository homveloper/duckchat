# Data Model: Responsive Navigation Bar

## Navigation State Entity

**Purpose**: Manages the current navigation state and responsive behavior

### Core Properties

| Field | Type | Description | Validation Rules |
|-------|------|-------------|------------------|
| `activeSection` | `string` | Currently selected navigation section | Must be one of: `"chat"`, `"profile"` |
| `isTransitioning` | `boolean` | Whether navigation is currently animating | Default: `false` |
| `layoutMode` | `string` | Current responsive layout mode | Computed: `"mobile"` or `"desktop"` |
| `containerWidth` | `number` | Current container width in pixels | Read-only, updated by ResizeObserver |

### State Transitions

```
[Initial Load]
    ↓
[Detect Container Size] → layoutMode = containerWidth < 768 ? "mobile" : "desktop"
    ↓
[Set Default Active] → activeSection = "chat"
    ↓
[Ready State]

[User Clicks Section]
    ↓
[Start Transition] → isTransitioning = true
    ↓
[Update Active Section] → activeSection = clickedSection
    ↓
[Complete Transition] → isTransitioning = false

[Container Resize]
    ↓
[Start Layout Transition] → isTransitioning = true
    ↓
[Update Layout Mode] → layoutMode = newWidth < 768 ? "mobile" : "desktop"
    ↓
[Complete Layout Transition] → isTransitioning = false
```

### Navigation Item Entity

**Purpose**: Represents individual navigation sections (Chat, Profile)

| Field | Type | Description | Validation Rules |
|-------|------|-------------|------------------|
| `id` | `string` | Unique identifier for the section | Must be one of: `"chat"`, `"profile"` |
| `label` | `string` | Display text for the navigation item | Required, non-empty string |
| `icon` | `SVGComponent` | Heroicon component reference | Must be valid Heroicons component |
| `isActive` | `boolean` | Whether this section is currently active | Computed from navigationState.activeSection |
| `href` | `string` | Navigation target URL/route | Required, valid URL/route |

### Navigation Items Configuration

```javascript
const navigationItems = [
  {
    id: 'chat',
    label: 'Chat',
    icon: 'ChatBubbleLeftRightIcon',
    href: '/rooms',
    iconPath: 'M20 2H4a2 2 0 00-2 2v12a2 2 0 002 2h4l4 2 4-2h4a2 2 0 002-2V4a2 2 0 00-2-2z'
  },
  {
    id: 'profile',
    label: 'Profile',
    icon: 'UserCircleIcon',
    href: '/profile',
    iconPath: 'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z'
  }
];
```

## Responsive Layout Entity

**Purpose**: Defines layout configurations for different screen sizes

| Field | Type | Description | Validation Rules |
|-------|------|-------------|------------------|
| `breakpoint` | `number` | Container width threshold in pixels | Must be 768 (md breakpoint) |
| `mobileLayout` | `LayoutConfig` | Configuration for mobile layout | Required |
| `desktopLayout` | `LayoutConfig` | Configuration for desktop layout | Required |

### Layout Configuration Schema

```typescript
interface LayoutConfig {
  position: 'fixed';
  placement: 'bottom' | 'left';
  flexDirection: 'row' | 'column';
  dimensions: {
    width?: string;   // CSS width value
    height?: string;  // CSS height value
  };
  spacing: {
    padding: string;  // CSS padding value
    gap: string;      // CSS gap value
  };
  itemDisplay: {
    showLabels: boolean;
    iconSize: 'sm' | 'md' | 'lg';
    labelSize: 'xs' | 'sm' | 'md';
  };
}

const layoutConfigs = {
  mobile: {
    position: 'fixed',
    placement: 'bottom',
    flexDirection: 'row',
    dimensions: {
      width: '100%',
      height: 'auto'
    },
    spacing: {
      padding: '12px 16px',
      gap: '8px'
    },
    itemDisplay: {
      showLabels: true,
      iconSize: 'sm', // w-5 h-5
      labelSize: 'xs' // text-xs
    }
  },
  desktop: {
    position: 'fixed',
    placement: 'left',
    flexDirection: 'column',
    dimensions: {
      width: '240px',
      height: '100vh'
    },
    spacing: {
      padding: '24px 16px',
      gap: '16px'
    },
    itemDisplay: {
      showLabels: true,
      iconSize: 'md', // w-6 h-6
      labelSize: 'sm' // text-sm
    }
  }
};
```

## Session Persistence Entity

**Purpose**: Manages navigation state persistence across page reloads

| Field | Type | Description | Validation Rules |
|-------|------|-------------|------------------|
| `storageKey` | `string` | sessionStorage key for navigation state | `"duckchat_navigation_state"` |
| `persistedState` | `NavigationState` | Serialized navigation state | JSON-serializable |
| `expirationTime` | `number` | Timestamp when state expires | 24 hours from creation |

### Persistence Schema

```javascript
// Stored in sessionStorage
const persistedNavigationState = {
  activeSection: 'chat',
  timestamp: 1694789123456, // Date.now()
  version: '1.0' // For future migration compatibility
};

// Restore logic
function restoreNavigationState() {
  const stored = sessionStorage.getItem('duckchat_navigation_state');
  if (!stored) return getDefaultState();

  const parsed = JSON.parse(stored);
  const isExpired = Date.now() - parsed.timestamp > 24 * 60 * 60 * 1000;

  return isExpired ? getDefaultState() : {
    activeSection: parsed.activeSection,
    isTransitioning: false, // Never persist transition state
    layoutMode: 'mobile', // Always recompute on load
    containerWidth: 0 // Always recompute on load
  };
}
```

## Integration with Existing DuckChat Models

### UserContext Integration

The navigation component integrates with existing `UserContext` from the authentication system:

```go
// Existing UserContext from internal/ui/models/chat_state.go
type UserContext struct {
    ID           string     `json:"id"`
    Username     string     `json:"username"`
    AvatarURL    *string    `json:"avatarUrl,omitempty"`
    IsOnline     bool       `json:"isOnline"`
    LastActivity *time.Time `json:"lastActivity,omitempty"`
    IsTyping     bool       `json:"isTyping"`
}

// Navigation will access user context for:
// - Profile navigation (when user clicks Profile tab)
// - User status display (if needed in future)
// - Authentication state (ensure user is logged in)
```

### Route Integration

Navigation items map to existing application routes:

```go
// Chat section → Existing rooms interface
href: "/rooms" → UIHandler.RoomsGetHandler

// Profile section → Future profile interface
href: "/profile" → UIHandler.ProfileGetHandler (to be implemented)
```

## Validation Rules Summary

### Client-Side Validation

1. **Active Section**: Must be one of the configured navigation items
2. **Container Width**: Must be positive number
3. **Layout Mode**: Must be derived from container width and breakpoint
4. **Persistence**: State must be valid JSON and not expired

### Template Validation (Go/Templ)

1. **Navigation Items**: Must have valid icon components
2. **URLs**: Must be valid application routes
3. **CSS Classes**: Must be valid Tailwind CSS classes
4. **Container Queries**: Must have proper `@container` support

### Error Handling

```javascript
// Invalid active section
if (!['chat', 'profile'].includes(newActiveSection)) {
  console.warn('Invalid navigation section:', newActiveSection);
  return; // Ignore invalid state change
}

// Container query not supported
if (!CSS.supports('container-type', 'inline-size')) {
  console.warn('Container queries not supported, using mobile layout');
  layoutMode = 'mobile'; // Fallback to mobile
}

// SessionStorage errors
try {
  const state = JSON.parse(sessionStorage.getItem(storageKey));
} catch (error) {
  console.warn('Failed to parse navigation state:', error);
  return getDefaultState(); // Use defaults
}
```