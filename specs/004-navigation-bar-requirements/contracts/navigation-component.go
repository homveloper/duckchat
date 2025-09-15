/*
Package contracts defines the Go contracts for the responsive navigation bar component.
This serves as documentation and interface definition for implementation.

Note: This is a documentation file. Actual implementation will be in:
- backend/web/templates/components/navigation.templ
- backend/internal/ui/models/ (for data structures)
- backend/internal/ui/handlers/ (for HTTP handlers)

Import paths shown are for reference and will be resolved during implementation.
*/

// package contracts

// import (
// 	"duckchat/internal/ui/models" // Will be available during implementation
// 	"time"
// )

// ============================================================================
// Navigation Data Structures (Go Structs)
// ============================================================================

// NavigationData contains all data needed for rendering the navigation component
type NavigationData struct {
	CurrentUser    *models.UserContext `json:"currentUser,omitempty"`
	ActiveSection  NavigationSection   `json:"activeSection"`
	Items          []NavigationItem    `json:"items"`
	LayoutConfig   LayoutConfig        `json:"layoutConfig"`
	ContainerClass string              `json:"containerClass"`
}

// NavigationSection represents the available navigation sections
type NavigationSection string

const (
	SectionChat    NavigationSection = "chat"
	SectionProfile NavigationSection = "profile"
)

// IsValid checks if the navigation section is valid
func (s NavigationSection) IsValid() bool {
	return s == SectionChat || s == SectionProfile
}

// NavigationItem represents a single navigation item
type NavigationItem struct {
	ID          NavigationSection `json:"id"`
	Label       string            `json:"label"`
	Icon        HeroIcon          `json:"icon"`
	IconPath    string            `json:"iconPath"`    // SVG path data
	Href        string            `json:"href"`        // Target route
	IsActive    bool              `json:"isActive"`    // Computed from ActiveSection
	AccessLevel string            `json:"accessLevel"` // "public", "authenticated"
}

// HeroIcon represents Heroicons used in navigation
type HeroIcon string

const (
	IconChat    HeroIcon = "ChatBubbleLeftRightIcon"
	IconProfile HeroIcon = "UserCircleIcon"
)

// ============================================================================
// Layout Configuration (Go Structs)
// ============================================================================

// LayoutConfig defines responsive layout configurations
type LayoutConfig struct {
	Breakpoint    int                    `json:"breakpoint"`    // Container width threshold (768px)
	MobileLayout  LayoutConfiguration    `json:"mobileLayout"`  // Mobile-specific config
	DesktopLayout LayoutConfiguration    `json:"desktopLayout"` // Desktop-specific config
	Transitions   TransitionConfig       `json:"transitions"`   // Animation config
}

// LayoutConfiguration defines layout-specific settings
type LayoutConfiguration struct {
	Position      string            `json:"position"`      // "fixed"
	Placement     string            `json:"placement"`     // "bottom" | "left"
	FlexDirection string            `json:"flexDirection"` // "row" | "column"
	Dimensions    DimensionConfig   `json:"dimensions"`    // Width/height
	Spacing       SpacingConfig     `json:"spacing"`       // Padding/gap
	ItemDisplay   ItemDisplayConfig `json:"itemDisplay"`   // Icon/label settings
	CSSClasses    []string          `json:"cssClasses"`    // Tailwind classes
}

// DimensionConfig defines layout dimensions
type DimensionConfig struct {
	Width  string `json:"width,omitempty"`  // CSS width value
	Height string `json:"height,omitempty"` // CSS height value
}

// SpacingConfig defines layout spacing
type SpacingConfig struct {
	Padding string `json:"padding"` // CSS padding value
	Gap     string `json:"gap"`     // CSS gap value
}

// ItemDisplayConfig defines how navigation items are displayed
type ItemDisplayConfig struct {
	ShowLabels bool   `json:"showLabels"` // Whether to show text labels
	IconSize   string `json:"iconSize"`   // "w-5 h-5" | "w-6 h-6"
	LabelSize  string `json:"labelSize"`  // "text-xs" | "text-sm"
}

// TransitionConfig defines animation settings
type TransitionConfig struct {
	Duration string `json:"duration"` // "300ms"
	Easing   string `json:"easing"`   // "ease-in-out"
	Property string `json:"property"` // "all"
}

// ============================================================================
// Templ Component Contracts
// ============================================================================

// NavigationTemplData represents data passed to navigation templates
type NavigationTemplData struct {
	Data       *NavigationData       `json:"data"`
	User       *models.UserContext   `json:"user"`
	LayoutData *models.LayoutState   `json:"layoutData,omitempty"`
	CSRFToken  string                `json:"csrfToken,omitempty"`
}

// NavigationItemTemplData represents data for individual navigation items
type NavigationItemTemplData struct {
	Item         *NavigationItem     `json:"item"`
	IsActive     bool                `json:"isActive"`
	LayoutMode   string              `json:"layoutMode"`   // "mobile" | "desktop"
	CSSClasses   []string            `json:"cssClasses"`
}

// ============================================================================
// State Management Contracts (JavaScript Integration)
// ============================================================================

// NavigationStateJS defines the JavaScript state structure
// This will be serialized to JSON and used in client-side JavaScript
type NavigationStateJS struct {
	ActiveSection   NavigationSection `json:"activeSection"`
	IsTransitioning bool              `json:"isTransitioning"`
	LayoutMode      string            `json:"layoutMode"`      // "mobile" | "desktop"
	ContainerWidth  int               `json:"containerWidth"`  // Current container width
	Timestamp       int64             `json:"timestamp"`       // For sessionStorage
	Version         string            `json:"version"`         // For future compatibility
}

// ToJSON converts NavigationStateJS to JSON string for JavaScript consumption
func (n *NavigationStateJS) ToJSON() ([]byte, error) {
	// Implementation would use json.Marshal
	return nil, nil
}

// ============================================================================
// HTTP Handler Contracts
// ============================================================================

// NavigationHandler defines methods for handling navigation-related requests
type NavigationHandler interface {
	// RenderNavigation renders the navigation component with current state
	RenderNavigation(user *models.UserContext, activeSection NavigationSection) ([]byte, error)

	// UpdateNavigationState handles AJAX requests to update navigation state
	UpdateNavigationState(userID string, section NavigationSection) error

	// GetNavigationData returns navigation data for HTMX partial updates
	GetNavigationData(user *models.UserContext) (*NavigationData, error)
}

// ============================================================================
// Repository/Service Contracts (if needed for state persistence)
// ============================================================================

// NavigationPreferences represents user's navigation preferences
type NavigationPreferences struct {
	UserID          string            `json:"userId"`
	PreferredLayout string            `json:"preferredLayout"` // "auto" | "mobile" | "desktop"
	LastSection     NavigationSection `json:"lastSection"`     // Last visited section
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// NavigationService defines business logic operations
type NavigationService interface {
	// GetNavigationData builds navigation data for a user
	GetNavigationData(user *models.UserContext) (*NavigationData, error)

	// UpdateActiveSection updates the user's active navigation section
	UpdateActiveSection(userID string, section NavigationSection) error

	// GetUserPreferences retrieves user's navigation preferences
	GetUserPreferences(userID string) (*NavigationPreferences, error)

	// SaveUserPreferences saves user's navigation preferences
	SaveUserPreferences(prefs *NavigationPreferences) error
}

// ============================================================================
// Default Configurations
// ============================================================================

// DefaultNavigationItems returns the standard navigation items
func DefaultNavigationItems() []NavigationItem {
	return []NavigationItem{
		{
			ID:          SectionChat,
			Label:       "Chat",
			Icon:        IconChat,
			IconPath:    "M20 2H4a2 2 0 00-2 2v12a2 2 0 002 2h4l4 2 4-2h4a2 2 0 002-2V4a2 2 0 00-2-2z",
			Href:        "/rooms",
			AccessLevel: "authenticated",
		},
		{
			ID:          SectionProfile,
			Label:       "Profile",
			Icon:        IconProfile,
			IconPath:    "M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z",
			Href:        "/profile",
			AccessLevel: "authenticated",
		},
	}
}

// DefaultLayoutConfig returns the standard layout configuration
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		Breakpoint: 768, // Tailwind 'md' breakpoint
		MobileLayout: LayoutConfiguration{
			Position:      "fixed",
			Placement:     "bottom",
			FlexDirection: "row",
			Dimensions: DimensionConfig{
				Width:  "100%",
				Height: "auto",
			},
			Spacing: SpacingConfig{
				Padding: "12px 16px",
				Gap:     "8px",
			},
			ItemDisplay: ItemDisplayConfig{
				ShowLabels: true,
				IconSize:   "w-5 h-5",
				LabelSize:  "text-xs",
			},
			CSSClasses: []string{
				"fixed", "bottom-0", "left-0", "right-0",
				"flex", "flex-row", "justify-center", "items-center",
				"bg-white", "border-t", "border-gray-200",
				"px-4", "py-3", "gap-2",
				"transition-all", "duration-300", "ease-in-out",
			},
		},
		DesktopLayout: LayoutConfiguration{
			Position:      "fixed",
			Placement:     "left",
			FlexDirection: "column",
			Dimensions: DimensionConfig{
				Width:  "240px",
				Height: "100vh",
			},
			Spacing: SpacingConfig{
				Padding: "24px 16px",
				Gap:     "16px",
			},
			ItemDisplay: ItemDisplayConfig{
				ShowLabels: true,
				IconSize:   "w-6 h-6",
				LabelSize:  "text-sm",
			},
			CSSClasses: []string{
				"fixed", "left-0", "top-0", "h-screen", "w-60",
				"flex", "flex-col", "justify-start", "items-start",
				"bg-white", "border-r", "border-gray-200",
				"px-6", "py-6", "gap-4",
				"transition-all", "duration-300", "ease-in-out",
			},
		},
		Transitions: TransitionConfig{
			Duration: "300ms",
			Easing:   "ease-in-out",
			Property: "all",
		},
	}
}

// ============================================================================
// CSS Class Contracts
// ============================================================================

// CSSClasses defines all CSS classes used in the navigation component
type CSSClasses struct {
	// Container classes
	Container   []string `json:"container"`
	Navigation  []string `json:"navigation"`

	// Item classes
	Item        []string `json:"item"`
	ItemActive  []string `json:"itemActive"`
	ItemIcon    []string `json:"itemIcon"`
	ItemLabel   []string `json:"itemLabel"`

	// Responsive classes
	Mobile      []string `json:"mobile"`
	Desktop     []string `json:"desktop"`
	Transition  []string `json:"transition"`
}

// DefaultCSSClasses returns the standard CSS class configuration
func DefaultCSSClasses() CSSClasses {
	return CSSClasses{
		Container: []string{
			"@container", "navigation-container",
		},
		Navigation: []string{
			"navigation", "fixed", "flex", "bg-white", "border-gray-200",
			"transition-all", "duration-300", "ease-in-out",
			// Mobile-first (default)
			"bottom-0", "left-0", "right-0", "flex-row", "border-t",
			"px-4", "py-3", "justify-center",
			// Container queries for desktop
			"@[768px]:left-0", "@[768px]:bottom-auto", "@[768px]:top-0",
			"@[768px]:h-screen", "@[768px]:w-60", "@[768px]:flex-col",
			"@[768px]:border-r", "@[768px]:border-t-0", "@[768px]:justify-start",
			"@[768px]:px-6", "@[768px]:py-6",
		},
		Item: []string{
			"nav-item", "flex", "items-center", "px-4", "py-3",
			"text-gray-600", "hover:text-duck-blue-600", "hover:bg-gray-50",
			"transition-colors", "duration-200", "cursor-pointer",
			"rounded-lg", "min-w-0",
			// Mobile: center content
			"justify-center", "flex-col", "gap-1",
			// Desktop: left align with gap
			"@[768px]:justify-start", "@[768px]:flex-row", "@[768px]:gap-3",
			"@[768px]:px-6", "@[768px]:py-4", "@[768px]:w-full",
		},
		ItemActive: []string{
			"text-duck-blue-600", "bg-duck-blue-50", "font-medium",
		},
		ItemIcon: []string{
			"nav-icon", "w-5", "h-5", "stroke-current", "flex-shrink-0",
			"@[768px]:w-6", "@[768px]:h-6",
		},
		ItemLabel: []string{
			"nav-label", "text-xs", "font-medium", "truncate",
			"@[768px]:text-sm",
		},
		Mobile: []string{
			"@container", "(max-width: 767px)",
		},
		Desktop: []string{
			"@container", "(min-width: 768px)",
		},
		Transition: []string{
			"will-change-transform", "transform-gpu",
		},
	}
}

// ============================================================================
// Validation Contracts
// ============================================================================

// ValidationError represents validation errors in navigation data
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// NavigationValidator defines validation methods
type NavigationValidator interface {
	ValidateNavigationData(data *NavigationData) []ValidationError
	ValidateNavigationSection(section NavigationSection) error
	ValidateUserAccess(user *models.UserContext, section NavigationSection) error
}

// ============================================================================
// Integration Contracts with Existing DuckChat Components
// ============================================================================

// DuckChatIntegration defines how navigation integrates with existing components
type DuckChatIntegration struct {
	// Toast system integration
	ShowToast func(toastType, title, message string) error

	// User context integration
	GetCurrentUser func() *models.UserContext

	// Route handling integration
	HandleNavigation func(section NavigationSection, user *models.UserContext) error

	// Layout integration
	UpdateLayout func(hasNavigation bool) error
}

// ============================================================================
// Error Handling Contracts
// ============================================================================

// NavigationError represents navigation-specific errors
type NavigationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Section string `json:"section,omitempty"`
	UserID  string `json:"userId,omitempty"`
}

// Error codes
const (
	ErrInvalidSection     = "INVALID_SECTION"
	ErrUnauthorizedAccess = "UNAUTHORIZED_ACCESS"
	ErrStateCorrupted     = "STATE_CORRUPTED"
	ErrLayoutError        = "LAYOUT_ERROR"
	ErrPersistenceError   = "PERSISTENCE_ERROR"
)

// Error returns the error message
func (e *NavigationError) Error() string {
	return e.Message
}

// ============================================================================
// Testing Contracts (for future implementation)
// ============================================================================

// NavigationTestData provides test data structures
type NavigationTestData struct {
	ValidUser        *models.UserContext
	InvalidUser      *models.UserContext
	ValidSection     NavigationSection
	InvalidSection   NavigationSection
	ValidLayoutData  *NavigationData
	InvalidLayoutData *NavigationData
}

// NavigationTestInterface defines testing methods (for future TDD implementation)
type NavigationTestInterface interface {
	TestNavigationRendering(data *NavigationData) error
	TestResponsiveLayout(containerWidth int) error
	TestStateTransitions(fromSection, toSection NavigationSection) error
	TestUserPermissions(user *models.UserContext, section NavigationSection) error
}