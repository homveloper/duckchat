package models

import (
	"regexp"
	"strings"
	"time"
)

// ResponsiveLayoutState tracks current responsive design state
type ResponsiveLayoutState struct {
	ViewportWidth   int         `json:"viewportWidth"`
	ViewportHeight  int         `json:"viewportHeight"`
	DeviceType      DeviceType  `json:"deviceType"`
	Orientation     Orientation `json:"orientation"`
	IsTouchDevice   bool        `json:"isTouchDevice"`
	UserAgent       string      `json:"userAgent,omitempty"`
	LastUpdated     time.Time   `json:"lastUpdated"`
	BreakpointName  string      `json:"breakpointName"`
	DensityRatio    float64     `json:"densityRatio"`
	IsHighDensity   bool        `json:"isHighDensity"`
	IsReducedMotion bool        `json:"isReducedMotion"`
	Theme           Theme       `json:"theme"`
	PreferredTheme  Theme       `json:"preferredTheme"`
}

// DeviceType represents the device category
type DeviceType string

const (
	DeviceTypeMobile      DeviceType = "mobile"
	DeviceTypeTablet      DeviceType = "tablet"
	DeviceTypeDesktop     DeviceType = "desktop"
	DeviceTypeLargeScreen DeviceType = "large"
)

// Orientation represents screen orientation
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// Theme represents UI theme preference
type Theme string

const (
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
	ThemeAuto  Theme = "auto"
)

// Viewport breakpoint constants (matching Tailwind CSS defaults)
const (
	BreakpointMobile  = 0
	BreakpointTablet  = 768
	BreakpointDesktop = 1024
	BreakpointLarge   = 1280
	BreakpointXLarge  = 1536

	// Touch-specific breakpoints for better mobile experience
	MobileMaxWidth  = 767
	TabletMinWidth  = 768
	TabletMaxWidth  = 1023
	DesktopMinWidth = 1024

	// High density display threshold
	HighDensityThreshold = 1.5
)

// Breakpoint names for template rendering
const (
	BreakpointNameMobile  = "mobile"
	BreakpointNameTablet  = "tablet"
	BreakpointNameDesktop = "desktop"
	BreakpointNameLarge   = "large"
	BreakpointNameXLarge  = "xl"
)

// NewResponsiveLayoutState creates a new responsive layout state with mobile-first defaults
func NewResponsiveLayoutState() *ResponsiveLayoutState {
	return &ResponsiveLayoutState{
		ViewportWidth:   320, // Default mobile width for SSR
		ViewportHeight:  568, // Default mobile height for SSR
		DeviceType:      DeviceTypeMobile,
		Orientation:     OrientationPortrait,
		IsTouchDevice:   true, // Default to touch for mobile-first
		UserAgent:       "",
		LastUpdated:     time.Now(),
		BreakpointName:  BreakpointNameMobile,
		DensityRatio:    1.0,
		IsHighDensity:   false,
		IsReducedMotion: false,
		Theme:           ThemeLight,
		PreferredTheme:  ThemeAuto,
	}
}

// NewResponsiveLayoutStateFromRequest creates layout state from HTTP request headers
func NewResponsiveLayoutStateFromRequest(userAgent string, clientHints map[string]string) *ResponsiveLayoutState {
	state := NewResponsiveLayoutState()

	// Set user agent
	state.UserAgent = userAgent

	// Parse user agent for device detection
	state.parseUserAgent()

	// Parse client hints if available
	state.parseClientHints(clientHints)

	// Determine initial viewport and device type
	state.determineDeviceType()
	state.updateBreakpointName()

	return state
}

// NewResponsiveLayoutStateFromUserAgent creates layout state from user agent string
func NewResponsiveLayoutStateFromUserAgent(userAgent string) *ResponsiveLayoutState {
	state := NewResponsiveLayoutState()
	state.UserAgent = userAgent
	state.detectFromUserAgent()
	return state
}

// UpdateViewport updates viewport dimensions and recalculates dependent properties
func (s *ResponsiveLayoutState) UpdateViewport(width, height int) {
	s.ViewportWidth = width
	s.ViewportHeight = height
	s.LastUpdated = time.Now()

	// Recalculate dependent properties
	s.determineDeviceType()
	s.determineOrientation()
	s.updateBreakpointName()
}

// UpdateDeviceCapabilities updates device-specific capabilities
func (s *ResponsiveLayoutState) UpdateDeviceCapabilities(isTouchDevice bool, densityRatio float64, isReducedMotion bool) {
	s.IsTouchDevice = isTouchDevice
	s.DensityRatio = densityRatio
	s.IsHighDensity = densityRatio >= HighDensityThreshold
	s.IsReducedMotion = isReducedMotion
	s.LastUpdated = time.Now()
}

// SetTheme sets the current theme
func (s *ResponsiveLayoutState) SetTheme(theme Theme) {
	s.Theme = theme
	s.LastUpdated = time.Now()
}

// SetPreferredTheme sets the preferred theme (user preference)
func (s *ResponsiveLayoutState) SetPreferredTheme(preferredTheme Theme) {
	s.PreferredTheme = preferredTheme

	// If preferred theme is not auto, apply it
	if preferredTheme != ThemeAuto {
		s.Theme = preferredTheme
	}

	s.LastUpdated = time.Now()
}

// determineDeviceType determines device type based on viewport width
func (s *ResponsiveLayoutState) determineDeviceType() {
	switch {
	case s.ViewportWidth <= MobileMaxWidth:
		s.DeviceType = DeviceTypeMobile
	case s.ViewportWidth >= TabletMinWidth && s.ViewportWidth <= TabletMaxWidth:
		s.DeviceType = DeviceTypeTablet
	case s.ViewportWidth >= DesktopMinWidth && s.ViewportWidth < BreakpointLarge:
		s.DeviceType = DeviceTypeDesktop
	default:
		s.DeviceType = DeviceTypeLargeScreen
	}
}

// determineOrientation determines orientation based on viewport dimensions
func (s *ResponsiveLayoutState) determineOrientation() {
	if s.ViewportWidth > s.ViewportHeight {
		s.Orientation = OrientationLandscape
	} else {
		s.Orientation = OrientationPortrait
	}
}

// updateBreakpointName updates the breakpoint name based on viewport width
func (s *ResponsiveLayoutState) updateBreakpointName() {
	switch {
	case s.ViewportWidth < BreakpointTablet:
		s.BreakpointName = BreakpointNameMobile
	case s.ViewportWidth < BreakpointDesktop:
		s.BreakpointName = BreakpointNameTablet
	case s.ViewportWidth < BreakpointLarge:
		s.BreakpointName = BreakpointNameDesktop
	case s.ViewportWidth < BreakpointXLarge:
		s.BreakpointName = BreakpointNameLarge
	default:
		s.BreakpointName = BreakpointNameXLarge
	}
}

// parseUserAgent parses user agent string for device information
func (s *ResponsiveLayoutState) parseUserAgent() {
	if s.UserAgent == "" {
		return
	}

	userAgent := strings.ToLower(s.UserAgent)

	// Mobile device detection patterns
	mobilePatterns := []string{
		`mobile`, `android`, `iphone`, `ipod`, `blackberry`, `windows phone`,
		`opera mini`, `opera mobi`, `firefox mobile`, `chrome mobile`,
	}

	// Tablet device detection patterns
	tabletPatterns := []string{
		`tablet`, `ipad`, `android.*mobile`, `kindle`, `silk`, `playbook`,
		`gt-p\d{4}`, `sm-t\d{3}`, `nexus \d{1,2}`,
	}

	// Touch device detection patterns
	touchPatterns := []string{
		`touch`, `mobile`, `tablet`, `android`, `ios`, `iphone`, `ipad`,
		`windows phone`, `blackberry`,
	}

	// Check for mobile devices
	for _, pattern := range mobilePatterns {
		if matched, _ := regexp.MatchString(pattern, userAgent); matched {
			s.DeviceType = DeviceTypeMobile
			s.IsTouchDevice = true
			return
		}
	}

	// Check for tablet devices
	for _, pattern := range tabletPatterns {
		if matched, _ := regexp.MatchString(pattern, userAgent); matched {
			s.DeviceType = DeviceTypeTablet
			s.IsTouchDevice = true
			return
		}
	}

	// Check for touch capability
	for _, pattern := range touchPatterns {
		if matched, _ := regexp.MatchString(pattern, userAgent); matched {
			s.IsTouchDevice = true
			break
		}
	}

	// Default to desktop if no mobile/tablet patterns matched
	if s.DeviceType == DeviceTypeMobile && !s.IsTouchDevice {
		s.DeviceType = DeviceTypeDesktop
		s.ViewportWidth = DesktopMinWidth
		s.ViewportHeight = 768
	}
}

// parseClientHints parses client hints for more accurate device detection
func (s *ResponsiveLayoutState) parseClientHints(clientHints map[string]string) {
	if clientHints == nil {
		return
	}

	// Parse viewport width hint
	if widthHint, exists := clientHints["Viewport-Width"]; exists {
		if width := parseIntSafe(widthHint); width > 0 {
			s.ViewportWidth = width
		}
	}

	// Parse device pixel ratio hint
	if dprHint, exists := clientHints["DPR"]; exists {
		if dpr := parseFloatSafe(dprHint); dpr > 0 {
			s.DensityRatio = dpr
			s.IsHighDensity = dpr >= HighDensityThreshold
		}
	}

	// Parse mobile hint
	if mobileHint, exists := clientHints["Mobile"]; exists {
		if mobileHint == "?1" {
			s.DeviceType = DeviceTypeMobile
			s.IsTouchDevice = true
		}
	}

	// Parse prefers-reduced-motion
	if motionHint, exists := clientHints["Prefers-Reduced-Motion"]; exists {
		s.IsReducedMotion = motionHint == "reduce"
	}

	// Parse color scheme preference
	if colorSchemeHint, exists := clientHints["Prefers-Color-Scheme"]; exists {
		switch colorSchemeHint {
		case "dark":
			s.PreferredTheme = ThemeDark
			s.Theme = ThemeDark
		case "light":
			s.PreferredTheme = ThemeLight
			s.Theme = ThemeLight
		}
	}
}

// IsMobile returns true if device is mobile
func (s *ResponsiveLayoutState) IsMobile() bool {
	return s.DeviceType == DeviceTypeMobile
}

// IsTablet returns true if device is tablet
func (s *ResponsiveLayoutState) IsTablet() bool {
	return s.DeviceType == DeviceTypeTablet
}

// IsDesktop returns true if device is desktop or larger
func (s *ResponsiveLayoutState) IsDesktop() bool {
	return s.DeviceType == DeviceTypeDesktop || s.DeviceType == DeviceTypeLargeScreen
}

// IsMobileOrTablet returns true if device is mobile or tablet
func (s *ResponsiveLayoutState) IsMobileOrTablet() bool {
	return s.IsMobile() || s.IsTablet()
}

// IsPortrait returns true if in portrait orientation
func (s *ResponsiveLayoutState) IsPortrait() bool {
	return s.Orientation == OrientationPortrait
}

// IsLandscape returns true if in landscape orientation
func (s *ResponsiveLayoutState) IsLandscape() bool {
	return s.Orientation == OrientationLandscape
}

// ShouldShowMobileLayout returns true if mobile layout should be used
func (s *ResponsiveLayoutState) ShouldShowMobileLayout() bool {
	return s.IsMobile() || (s.IsTablet() && s.IsPortrait())
}

// ShouldShowSidebar returns true if sidebar should be shown
func (s *ResponsiveLayoutState) ShouldShowSidebar() bool {
	return s.IsDesktop() || (s.IsTablet() && s.IsLandscape())
}

// ShouldUseCompactMode returns true if compact UI mode should be used
func (s *ResponsiveLayoutState) ShouldUseCompactMode() bool {
	return s.IsMobile() || s.ViewportWidth < 600 || s.ViewportHeight < 500
}

// GetTailwindClasses returns Tailwind CSS classes based on current state
func (s *ResponsiveLayoutState) GetTailwindClasses() []string {
	classes := make([]string, 0)

	// Device type classes
	switch s.DeviceType {
	case DeviceTypeMobile:
		classes = append(classes, "mobile-layout")
	case DeviceTypeTablet:
		classes = append(classes, "tablet-layout")
	case DeviceTypeDesktop:
		classes = append(classes, "desktop-layout")
	case DeviceTypeLargeScreen:
		classes = append(classes, "large-layout")
	}

	// Orientation classes
	switch s.Orientation {
	case OrientationPortrait:
		classes = append(classes, "portrait")
	case OrientationLandscape:
		classes = append(classes, "landscape")
	}

	// Touch classes
	if s.IsTouchDevice {
		classes = append(classes, "touch-device")
	} else {
		classes = append(classes, "no-touch")
	}

	// Density classes
	if s.IsHighDensity {
		classes = append(classes, "high-density", "retina")
	}

	// Motion classes
	if s.IsReducedMotion {
		classes = append(classes, "reduced-motion")
	}

	// Theme classes
	switch s.Theme {
	case ThemeDark:
		classes = append(classes, "dark")
	case ThemeLight:
		classes = append(classes, "light")
	}

	return classes
}

// GetBreakpointClasses returns responsive breakpoint classes
func (s *ResponsiveLayoutState) GetBreakpointClasses() map[string]bool {
	return map[string]bool{
		"sm":        s.ViewportWidth >= BreakpointTablet,
		"md":        s.ViewportWidth >= BreakpointDesktop,
		"lg":        s.ViewportWidth >= BreakpointLarge,
		"xl":        s.ViewportWidth >= BreakpointXLarge,
		"mobile":    s.IsMobile(),
		"tablet":    s.IsTablet(),
		"desktop":   s.IsDesktop(),
		"touch":     s.IsTouchDevice,
		"portrait":  s.IsPortrait(),
		"landscape": s.IsLandscape(),
	}
}

// GetViewportInfo returns viewport information for client-side JS
func (s *ResponsiveLayoutState) GetViewportInfo() map[string]interface{} {
	return map[string]interface{}{
		"width":       s.ViewportWidth,
		"height":      s.ViewportHeight,
		"deviceType":  string(s.DeviceType),
		"orientation": string(s.Orientation),
		"breakpoint":  s.BreakpointName,
		"isMobile":    s.IsMobile(),
		"isTablet":    s.IsTablet(),
		"isDesktop":   s.IsDesktop(),
		"isTouch":     s.IsTouchDevice,
		"isPortrait":  s.IsPortrait(),
		"dpr":         s.DensityRatio,
		"theme":       string(s.Theme),
	}
}

// ToTemplateData converts the state to data suitable for template rendering
func (s *ResponsiveLayoutState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"ViewportWidth":     s.ViewportWidth,
		"ViewportHeight":    s.ViewportHeight,
		"DeviceType":        string(s.DeviceType),
		"Orientation":       string(s.Orientation),
		"IsTouchDevice":     s.IsTouchDevice,
		"IsMobile":          s.IsMobile(),
		"IsTablet":          s.IsTablet(),
		"IsDesktop":         s.IsDesktop(),
		"IsMobileOrTablet":  s.IsMobileOrTablet(),
		"IsPortrait":        s.IsPortrait(),
		"IsLandscape":       s.IsLandscape(),
		"ShouldShowMobile":  s.ShouldShowMobileLayout(),
		"ShouldShowSidebar": s.ShouldShowSidebar(),
		"ShouldUseCompact":  s.ShouldUseCompactMode(),
		"BreakpointName":    s.BreakpointName,
		"BreakpointClasses": s.GetBreakpointClasses(),
		"TailwindClasses":   strings.Join(s.GetTailwindClasses(), " "),
		"DensityRatio":      s.DensityRatio,
		"IsHighDensity":     s.IsHighDensity,
		"IsReducedMotion":   s.IsReducedMotion,
		"Theme":             string(s.Theme),
		"PreferredTheme":    string(s.PreferredTheme),
		"IsDarkTheme":       s.Theme == ThemeDark,
		"IsLightTheme":      s.Theme == ThemeLight,
		"ViewportInfo":      s.GetViewportInfo(),
		"LastUpdated":       s.LastUpdated.Format(time.RFC3339),
	}
}

// ViewportChangeEvent represents a viewport change event
type ViewportChangeEvent struct {
	OldWidth      int        `json:"oldWidth"`
	OldHeight     int        `json:"oldHeight"`
	NewWidth      int        `json:"newWidth"`
	NewHeight     int        `json:"newHeight"`
	OldDeviceType DeviceType `json:"oldDeviceType"`
	NewDeviceType DeviceType `json:"newDeviceType"`
	Timestamp     time.Time  `json:"timestamp"`
	Significant   bool       `json:"significant"` // True if crosses major breakpoint
}

// CreateViewportChangeEvent creates a viewport change event
func (s *ResponsiveLayoutState) CreateViewportChangeEvent(newWidth, newHeight int) *ViewportChangeEvent {
	oldDeviceType := s.DeviceType

	// Temporarily update to determine new device type
	oldWidth, oldHeight := s.ViewportWidth, s.ViewportHeight
	s.ViewportWidth, s.ViewportHeight = newWidth, newHeight
	s.determineDeviceType()
	newDeviceType := s.DeviceType

	// Restore original values
	s.ViewportWidth, s.ViewportHeight = oldWidth, oldHeight
	s.DeviceType = oldDeviceType

	// Determine if change is significant (crosses major breakpoint)
	significant := oldDeviceType != newDeviceType ||
		(oldWidth < BreakpointTablet && newWidth >= BreakpointTablet) ||
		(oldWidth >= BreakpointTablet && newWidth < BreakpointTablet) ||
		(oldWidth < BreakpointDesktop && newWidth >= BreakpointDesktop) ||
		(oldWidth >= BreakpointDesktop && newWidth < BreakpointDesktop)

	return &ViewportChangeEvent{
		OldWidth:      oldWidth,
		OldHeight:     oldHeight,
		NewWidth:      newWidth,
		NewHeight:     newHeight,
		OldDeviceType: oldDeviceType,
		NewDeviceType: newDeviceType,
		Timestamp:     time.Now(),
		Significant:   significant,
	}
}

// Utility functions for parsing client hints safely

func parseIntSafe(s string) int {
	// Simple integer parsing without external dependencies
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0 // Invalid character
		}
	}
	return result
}

func parseFloatSafe(s string) float64 {
	// Simple float parsing for density ratio (e.g., "1.5", "2.0")
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0.0 // Invalid format
	}

	integer := parseIntSafe(parts[0])
	if len(parts) == 1 {
		return float64(integer)
	}

	decimal := parseIntSafe(parts[1])
	decimalPlaces := len(parts[1])

	result := float64(integer)
	if decimalPlaces > 0 {
		for i := 0; i < decimalPlaces; i++ {
			decimal /= 10
		}
		result += float64(decimal)
	}

	return result
}


// detectFromUserAgent analyzes user agent string to set device properties
func (s *ResponsiveLayoutState) detectFromUserAgent() {
	userAgent := strings.ToLower(s.UserAgent)

	// Check for mobile devices
	mobileKeywords := []string{"mobile", "android", "iphone", "ipad", "ipod", "blackberry", "windows phone"}
	for _, keyword := range mobileKeywords {
		if strings.Contains(userAgent, keyword) {
			s.IsTouchDevice = true
			if strings.Contains(userAgent, "ipad") || strings.Contains(userAgent, "tablet") {
				s.DeviceType = DeviceTypeTablet
				s.ViewportWidth = 768
				s.ViewportHeight = 1024
			} else {
				s.DeviceType = DeviceTypeMobile
				s.ViewportWidth = 375
				s.ViewportHeight = 812
			}
			break
		}
	}

	// Detect preferred theme (basic detection)
	if strings.Contains(userAgent, "dark") {
		s.PreferredTheme = ThemeDark
		s.Theme = ThemeDark
	} else {
		s.PreferredTheme = ThemeLight
		s.Theme = ThemeAuto
	}

	s.updateBreakpointName()
}

// GetThemeClass returns the CSS class for current theme
func (s *ResponsiveLayoutState) GetThemeClass() string {
	switch s.Theme {
	case ThemeDark:
		return "theme-dark"
	case ThemeLight:
		return "theme-light"
	case ThemeAuto:
		return "theme-auto"
	default:
		return "theme-light"
	}
}

// GetDeviceClasses returns CSS classes for current device characteristics
func (s *ResponsiveLayoutState) GetDeviceClasses() string {
	classes := []string{}

	// Device type classes
	switch s.DeviceType {
	case DeviceTypeMobile:
		classes = append(classes, "device-mobile")
	case DeviceTypeTablet:
		classes = append(classes, "device-tablet")
	case DeviceTypeDesktop:
		classes = append(classes, "device-desktop")
	case DeviceTypeLargeScreen:
		classes = append(classes, "device-large")
	}

	// Orientation classes
	switch s.Orientation {
	case OrientationPortrait:
		classes = append(classes, "orientation-portrait")
	case OrientationLandscape:
		classes = append(classes, "orientation-landscape")
	}

	// Touch classes
	if s.IsTouchDevice {
		classes = append(classes, "touch-device")
	} else {
		classes = append(classes, "no-touch")
	}

	// High density class
	if s.IsHighDensity {
		classes = append(classes, "high-density")
	}

	// Reduced motion class
	if s.IsReducedMotion {
		classes = append(classes, "reduced-motion")
	}

	return strings.Join(classes, " ")
}
