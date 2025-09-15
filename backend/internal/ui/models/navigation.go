package models

import (
	"strings"
	"time"
)

// NavigationSection represents a navigation section identifier
type NavigationSection string

const (
	NavigationSectionChat    NavigationSection = "chat"
	NavigationSectionProfile NavigationSection = "profile"
)

// NavigationData represents the complete navigation state and configuration
type NavigationData struct {
	Items         []NavigationItem  `json:"items"`
	ActiveSection NavigationSection `json:"activeSection"`
	LayoutConfig  LayoutConfig      `json:"layoutConfig"`
	LastUpdated   time.Time         `json:"lastUpdated"`
}

// NavigationItem represents a single navigation item
type NavigationItem struct {
	ID          string            `json:"id"`
	Section     NavigationSection `json:"section"`
	Label       string            `json:"label"`
	Icon        NavigationIcon    `json:"icon"`
	URL         string            `json:"url"`
	IsActive    bool              `json:"isActive"`
	IsEnabled   bool              `json:"isEnabled"`
	BadgeCount  int               `json:"badgeCount,omitempty"`
	Tooltip     string            `json:"tooltip,omitempty"`
	OrderIndex  int               `json:"orderIndex"`
	LastUpdated time.Time         `json:"lastUpdated"`
}

// NavigationIcon represents an icon with SVG path data
type NavigationIcon struct {
	Name     string `json:"name"`
	SVGPath  string `json:"svgPath"`
	ViewBox  string `json:"viewBox"`
	Stroke   string `json:"stroke,omitempty"`
	Fill     string `json:"fill,omitempty"`
	Outlined bool   `json:"outlined"`
}

// LayoutConfig controls responsive navigation behavior
type LayoutConfig struct {
	ContainerBreakpoint int             `json:"containerBreakpoint"` // 768px for mobile/desktop switch
	MobilePosition      MobilePosition  `json:"mobilePosition"`      // bottom
	DesktopPosition     DesktopPosition `json:"desktopPosition"`     // left
	TransitionDuration  int             `json:"transitionDuration"`  // 300ms
	ShowLabelsOnMobile  bool            `json:"showLabelsOnMobile"`  // true for small labels
	ShowLabelsOnDesktop bool            `json:"showLabelsOnDesktop"` // true for full labels
	CompactMode         bool            `json:"compactMode"`         // reduces sizes
	UseContainerQueries bool            `json:"useContainerQueries"` // true
	PreferReducedMotion bool            `json:"preferReducedMotion"` // accessibility setting
	Theme               NavigationTheme `json:"theme"`
}

// MobilePosition defines where navigation appears on mobile
type MobilePosition string

const (
	MobilePositionBottom MobilePosition = "bottom"
	MobilePositionTop    MobilePosition = "top"
)

// DesktopPosition defines where navigation appears on desktop
type DesktopPosition string

const (
	DesktopPositionLeft  DesktopPosition = "left"
	DesktopPositionRight DesktopPosition = "right"
)

// NavigationTheme defines visual theme for navigation
type NavigationTheme string

const (
	NavigationThemeLight NavigationTheme = "light"
	NavigationThemeDark  NavigationTheme = "dark"
	NavigationThemeAuto  NavigationTheme = "auto"
)

// NewNavigationData creates a new navigation data instance with defaults
func NewNavigationData() *NavigationData {
	return &NavigationData{
		Items:         GetDefaultNavigationItems(),
		ActiveSection: NavigationSectionChat, // Default to chat
		LayoutConfig:  GetDefaultLayoutConfig(),
		LastUpdated:   time.Now(),
	}
}

// GetDefaultNavigationItems returns the standard navigation items for DuckChat
func GetDefaultNavigationItems() []NavigationItem {
	return []NavigationItem{
		{
			ID:      "chat",
			Section: NavigationSectionChat,
			Label:   "Chat",
			Icon: NavigationIcon{
				Name:     "ChatBubbleLeftRightIcon",
				SVGPath:  "M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 0 1-.825-.242m9.345-8.334a2.126 2.126 0 0 0-.476-.095 48.64 48.64 0 0 0-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0 0 11.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155",
				ViewBox:  "0 0 24 24",
				Stroke:   "currentColor",
				Fill:     "none",
				Outlined: true,
			},
			URL:         "/rooms",
			IsActive:    true,
			IsEnabled:   true,
			BadgeCount:  0,
			Tooltip:     "Navigate to chat rooms",
			OrderIndex:  1,
			LastUpdated: time.Now(),
		},
		{
			ID:      "profile",
			Section: NavigationSectionProfile,
			Label:   "Profile",
			Icon: NavigationIcon{
				Name:     "UserCircleIcon",
				SVGPath:  "M17.982 18.725A7.488 7.488 0 0 0 12 15.75a7.488 7.488 0 0 0-5.982 2.975m11.963 0a9 9 0 1 0-11.963 0m11.963 0A8.966 8.966 0 0 1 12 21a8.966 8.966 0 0 1-5.982-2.275M15 9.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z",
				ViewBox:  "0 0 24 24",
				Stroke:   "currentColor",
				Fill:     "none",
				Outlined: true,
			},
			URL:         "/profile",
			IsActive:    false,
			IsEnabled:   true,
			BadgeCount:  0,
			Tooltip:     "View and edit your profile",
			OrderIndex:  2,
			LastUpdated: time.Now(),
		},
	}
}

// GetDefaultLayoutConfig returns the default layout configuration
func GetDefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		ContainerBreakpoint: 768,
		MobilePosition:      MobilePositionBottom,
		DesktopPosition:     DesktopPositionLeft,
		TransitionDuration:  300,
		ShowLabelsOnMobile:  true,
		ShowLabelsOnDesktop: true,
		CompactMode:         false,
		UseContainerQueries: true,
		PreferReducedMotion: false,
		Theme:               NavigationThemeAuto,
	}
}

// SetActiveSection updates the active navigation section
func (n *NavigationData) SetActiveSection(section NavigationSection) {
	n.ActiveSection = section
	n.LastUpdated = time.Now()

	// Update active state for all items
	for i := range n.Items {
		n.Items[i].IsActive = n.Items[i].Section == section
		n.Items[i].LastUpdated = time.Now()
	}
}

// GetActiveItem returns the currently active navigation item
func (n *NavigationData) GetActiveItem() *NavigationItem {
	for i := range n.Items {
		if n.Items[i].IsActive {
			return &n.Items[i]
		}
	}
	return nil
}

// GetItemBySection returns the navigation item for a specific section
func (n *NavigationData) GetItemBySection(section NavigationSection) *NavigationItem {
	for i := range n.Items {
		if n.Items[i].Section == section {
			return &n.Items[i]
		}
	}
	return nil
}

// UpdateBadgeCount updates the badge count for a specific section
func (n *NavigationData) UpdateBadgeCount(section NavigationSection, count int) {
	if item := n.GetItemBySection(section); item != nil {
		item.BadgeCount = count
		item.LastUpdated = time.Now()
		n.LastUpdated = time.Now()
	}
}

// GetMobileCSS returns CSS classes for mobile layout
func (n *NavigationData) GetMobileCSS() string {
	classes := []string{
		"@container",
		"fixed",
		"bottom-0",
		"left-0",
		"right-0",
		"flex",
		"flex-row",
		"justify-center",
		"items-center",
		"bg-white",
		"border-t",
		"border-gray-200",
		"safe-area-padding",
		"z-50",
	}

	if n.LayoutConfig.PreferReducedMotion {
		return joinCSS(classes)
	}

	classes = append(classes, "transition-all", "duration-300", "ease-in-out")
	return joinCSS(classes)
}

// GetDesktopCSS returns CSS classes for desktop layout
func (n *NavigationData) GetDesktopCSS() string {
	classes := []string{
		"@container",
		"fixed",
		"top-0",
		"left-0",
		"bottom-0",
		"flex",
		"flex-col",
		"justify-start",
		"items-stretch",
		"bg-white",
		"border-r",
		"border-gray-200",
		"w-64",
		"z-50",
	}

	if n.LayoutConfig.PreferReducedMotion {
		return joinCSS(classes)
	}

	classes = append(classes, "transition-all", "duration-300", "ease-in-out")
	return joinCSS(classes)
}

// GetItemCSS returns CSS classes for navigation items
func (n *NavigationData) GetItemCSS(item NavigationItem, isMobile bool) string {
	baseClasses := []string{
		"flex",
		"items-center",
		"justify-center",
		"cursor-pointer",
		"text-gray-600",
		"hover:text-duck-blue-600",
		"hover:bg-gray-50",
		"focus:outline-none",
		"focus:ring-2",
		"focus:ring-duck-blue-500",
		"focus:ring-offset-2",
	}

	if item.IsActive {
		baseClasses = append(baseClasses, "text-duck-blue-600", "bg-duck-blue-50")
	}

	if isMobile {
		baseClasses = append(baseClasses,
			"flex-1",
			"flex-col",
			"py-2",
			"px-1",
			"min-h-touch",
		)
	} else {
		baseClasses = append(baseClasses,
			"flex-row",
			"w-full",
			"px-4",
			"py-3",
			"space-x-3",
		)
	}

	if !n.LayoutConfig.PreferReducedMotion {
		baseClasses = append(baseClasses, "transition-colors", "duration-200")
	}

	return joinCSS(baseClasses)
}

// GetContainerQueryCSS returns container query CSS rules
func (n *NavigationData) GetContainerQueryCSS() string {
	return `
		@container (max-width: 767px) {
			.nav-container {
				/* Mobile layout */
				position: fixed;
				bottom: 0;
				left: 0;
				right: 0;
				flex-direction: row;
				height: auto;
				width: 100%;
				border-top: 1px solid #e5e7eb;
				border-right: none;
			}

			.nav-item {
				flex: 1;
				flex-direction: column;
				padding: 0.5rem 0.25rem;
				font-size: 0.75rem;
			}

			.nav-icon {
				width: 1.5rem;
				height: 1.5rem;
				margin-bottom: 0.25rem;
			}
		}

		@container (min-width: 768px) {
			.nav-container {
				/* Desktop layout */
				position: fixed;
				top: 0;
				left: 0;
				bottom: 0;
				flex-direction: column;
				width: 16rem;
				height: 100vh;
				border-right: 1px solid #e5e7eb;
				border-top: none;
			}

			.nav-item {
				flex-direction: row;
				width: 100%;
				padding: 0.75rem 1rem;
				font-size: 1rem;
				justify-content: flex-start;
			}

			.nav-icon {
				width: 1.25rem;
				height: 1.25rem;
				margin-right: 0.75rem;
				margin-bottom: 0;
			}
		}
	`
}

// ToTemplateData converts navigation data to template-friendly format
func (n *NavigationData) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Items":               n.Items,
		"ActiveSection":       string(n.ActiveSection),
		"LayoutConfig":        n.LayoutConfig,
		"LastUpdated":         n.LastUpdated.Format(time.RFC3339),
		"MobileCSS":           n.GetMobileCSS(),
		"DesktopCSS":          n.GetDesktopCSS(),
		"ContainerQueryCSS":   n.GetContainerQueryCSS(),
		"HasBadges":           n.hasBadges(),
		"TotalBadgeCount":     n.getTotalBadgeCount(),
		"ActiveItem":          n.GetActiveItem(),
		"TransitionDuration":  n.LayoutConfig.TransitionDuration,
		"UseContainerQueries": n.LayoutConfig.UseContainerQueries,
		"ContainerBreakpoint": n.LayoutConfig.ContainerBreakpoint,
	}
}

// Validation methods

// Validate checks if navigation data is valid
func (n *NavigationData) Validate() error {
	if len(n.Items) == 0 {
		return newValidationError("navigation must have at least one item")
	}

	// Validate each item
	for i, item := range n.Items {
		if err := item.Validate(); err != nil {
			return newValidationError("item %d: %v", i, err)
		}
	}

	// Validate layout config
	if err := n.LayoutConfig.Validate(); err != nil {
		return newValidationError("layout config: %v", err)
	}

	return nil
}

// Validate checks if navigation item is valid
func (item *NavigationItem) Validate() error {
	if item.ID == "" {
		return newValidationError("navigation item ID cannot be empty")
	}

	if item.Label == "" {
		return newValidationError("navigation item label cannot be empty")
	}

	if item.URL == "" {
		return newValidationError("navigation item URL cannot be empty")
	}

	if item.Icon.SVGPath == "" {
		return newValidationError("navigation item icon must have SVG path")
	}

	if item.BadgeCount < 0 {
		return newValidationError("badge count cannot be negative")
	}

	return nil
}

// Validate checks if layout config is valid
func (config *LayoutConfig) Validate() error {
	if config.ContainerBreakpoint <= 0 {
		return newValidationError("container breakpoint must be positive")
	}

	if config.TransitionDuration < 0 {
		return newValidationError("transition duration cannot be negative")
	}

	return nil
}

// Helper functions

func (n *NavigationData) hasBadges() bool {
	for _, item := range n.Items {
		if item.BadgeCount > 0 {
			return true
		}
	}
	return false
}

func (n *NavigationData) getTotalBadgeCount() int {
	total := 0
	for _, item := range n.Items {
		total += item.BadgeCount
	}
	return total
}

func joinCSS(classes []string) string {
	result := ""
	for i, class := range classes {
		if i > 0 {
			result += " "
		}
		result += class
	}
	return result
}

// NavigationError represents navigation-specific errors
type NavigationError struct {
	Message string
}

func (e *NavigationError) Error() string {
	return e.Message
}

func newValidationError(format string, args ...interface{}) error {
	message := format
	if len(args) > 0 {
		// Simple string formatting without external dependencies
		for i, arg := range args {
			placeholder := "%v"
			if i == 0 {
				placeholder = "%d"
			}
			oldMessage := message
			message = strings.Replace(message, placeholder, toString(arg), 1)
			if message == oldMessage {
				break // No more placeholders
			}
		}
	}
	return &NavigationError{Message: message}
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return intToString(val)
	case error:
		return val.Error()
	default:
		return "unknown"
	}
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}

	negative := i < 0
	if negative {
		i = -i
	}

	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte(i%10) + '0'}, digits...)
		i /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// IntToString converts an integer to string for template use
func IntToString(i int) string {
	return intToString(i)
}
