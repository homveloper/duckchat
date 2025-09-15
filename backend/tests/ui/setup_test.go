package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// TestMain sets up Playwright for UI testing
func TestMain(m *testing.M) {
	// Install Playwright browsers if needed
	err := playwright.Install()
	if err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// BrowserTestSuite provides common browser testing functionality
type BrowserTestSuite struct {
	server  *httptest.Server
	browser playwright.Browser
	context playwright.BrowserContext
	page    playwright.Page
	pw      *playwright.Playwright
}

// SetupBrowserTest initializes a browser test environment
func SetupBrowserTest(t *testing.T, handler http.Handler) *BrowserTestSuite {
	// Start test server
	server := httptest.NewServer(handler)

	// Launch Playwright
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("Failed to start Playwright: %v", err)
	}

	// Launch browser (headless for CI, can be configured)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Set to false for debugging
	})
	if err != nil {
		t.Fatalf("Failed to launch browser: %v", err)
	}

	// Create browser context with mobile viewport by default
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport:  &playwright.Size{Width: 375, Height: 812}, // iPhone X dimensions
		UserAgent: playwright.String("Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15"),
		IsMobile:  playwright.Bool(true),
		HasTouch:  playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("Failed to create browser context: %v", err)
	}

	// Create page
	page, err := context.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	return &BrowserTestSuite{
		server:  server,
		browser: browser,
		context: context,
		page:    page,
		pw:      pw,
	}
}

// TeardownBrowserTest cleans up browser test resources
func (suite *BrowserTestSuite) TeardownBrowserTest() {
	if suite.page != nil {
		suite.page.Close()
	}
	if suite.context != nil {
		suite.context.Close()
	}
	if suite.browser != nil {
		suite.browser.Close()
	}
	if suite.pw != nil {
		suite.pw.Stop()
	}
	if suite.server != nil {
		suite.server.Close()
	}
}

// NavigateToPath navigates to a path on the test server
func (suite *BrowserTestSuite) NavigateToPath(path string) error {
	url := suite.server.URL + path
	_, err := suite.page.Goto(url)
	return err
}

// WaitForSelector waits for an element to appear
func (suite *BrowserTestSuite) WaitForSelector(selector string, timeout ...time.Duration) playwright.Locator {
	defaultTimeout := 5 * time.Second
	if len(timeout) > 0 {
		defaultTimeout = timeout[0]
	}

	locator := suite.page.Locator(selector)
	locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	return locator
}

// FillForm fills a form with the given data
func (suite *BrowserTestSuite) FillForm(formData map[string]string) error {
	for selector, value := range formData {
		if err := suite.page.Fill(selector, value); err != nil {
			return err
		}
	}
	return nil
}

// ClickAndWait clicks an element and waits for navigation
func (suite *BrowserTestSuite) ClickAndWait(selector string) error {
	// Wait for navigation after click
	return suite.page.Click(selector)
}

// TakeScreenshot captures a screenshot for debugging
func (suite *BrowserTestSuite) TakeScreenshot(name string) error {
	_, err := suite.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String("screenshots/" + name + ".png"),
	})
	return err
}

// SetMobileViewport sets the viewport to mobile dimensions
func (suite *BrowserTestSuite) SetMobileViewport() error {
	return suite.page.SetViewportSize(375, 812) // iPhone X
}

// SetTabletViewport sets the viewport to tablet dimensions
func (suite *BrowserTestSuite) SetTabletViewport() error {
	return suite.page.SetViewportSize(768, 1024) // iPad
}

// SetDesktopViewport sets the viewport to desktop dimensions
func (suite *BrowserTestSuite) SetDesktopViewport() error {
	return suite.page.SetViewportSize(1280, 720) // Desktop
}

// CheckResponsiveDesign tests responsive breakpoints
func (suite *BrowserTestSuite) CheckResponsiveDesign(t *testing.T, path string) {
	// Test mobile viewport
	if err := suite.SetMobileViewport(); err != nil {
		t.Fatalf("Failed to set mobile viewport: %v", err)
	}
	if err := suite.NavigateToPath(path); err != nil {
		t.Fatalf("Failed to navigate to %s: %v", path, err)
	}
	// Add mobile-specific assertions here

	// Test tablet viewport
	if err := suite.SetTabletViewport(); err != nil {
		t.Fatalf("Failed to set tablet viewport: %v", err)
	}
	if err := suite.NavigateToPath(path); err != nil {
		t.Fatalf("Failed to navigate to %s: %v", path, err)
	}
	// Add tablet-specific assertions here

	// Test desktop viewport
	if err := suite.SetDesktopViewport(); err != nil {
		t.Fatalf("Failed to set desktop viewport: %v", err)
	}
	if err := suite.NavigateToPath(path); err != nil {
		t.Fatalf("Failed to navigate to %s: %v", path, err)
	}
	// Add desktop-specific assertions here
}

// AssertElementVisible checks if an element is visible
func (suite *BrowserTestSuite) AssertElementVisible(t *testing.T, selector string) {
	locator := suite.page.Locator(selector)
	visible, err := locator.IsVisible()
	if err != nil {
		t.Fatalf("Failed to check visibility of %s: %v", selector, err)
	}
	if !visible {
		t.Errorf("Element %s is not visible", selector)
	}
}

// AssertElementNotVisible checks if an element is not visible
func (suite *BrowserTestSuite) AssertElementNotVisible(t *testing.T, selector string) {
	locator := suite.page.Locator(selector)
	visible, err := locator.IsVisible()
	if err != nil {
		t.Fatalf("Failed to check visibility of %s: %v", selector, err)
	}
	if visible {
		t.Errorf("Element %s should not be visible", selector)
	}
}

// AssertText checks if an element contains expected text
func (suite *BrowserTestSuite) AssertText(t *testing.T, selector, expectedText string) {
	locator := suite.page.Locator(selector)
	actualText, err := locator.TextContent()
	if err != nil {
		t.Fatalf("Failed to get text content of %s: %v", selector, err)
	}
	if actualText != expectedText {
		t.Errorf("Element %s has text '%s', expected '%s'", selector, actualText, expectedText)
	}
}

// SimulateSlowNetwork simulates slow network conditions
func (suite *BrowserTestSuite) SimulateSlowNetwork() error {
	// Simulate slow 3G network
	cdpSession, err := suite.context.NewCDPSession(suite.page)
	if err != nil {
		return err
	}

	_, err = cdpSession.Send("Network.emulateNetworkConditions", map[string]interface{}{
		"offline":            false,
		"latency":            100,                   // 100ms latency
		"downloadThroughput": 1.6 * 1024 * 1024 / 8, // 1.6 Mbps down
		"uploadThroughput":   750 * 1024 / 8,        // 750 Kbps up
	})
	return err
}

// TestAccessibility performs basic accessibility checks
func (suite *BrowserTestSuite) TestAccessibility(t *testing.T) {
	// Check for basic accessibility requirements

	// 1. Check if page has a title
	title, err := suite.page.Title()
	if err != nil {
		t.Errorf("Failed to get page title: %v", err)
	}
	if title == "" {
		t.Error("Page should have a title for accessibility")
	}

	// 2. Check for alt text on images
	images, _ := suite.page.QuerySelectorAll("img")
	for i, img := range images {
		alt, _ := img.GetAttribute("alt")
		if alt == "" {
			t.Errorf("Image %d is missing alt text", i)
		}
	}

	// 3. Check for form labels
	inputs, _ := suite.page.QuerySelectorAll("input[type='text'], input[type='password'], input[type='email'], textarea")
	for i, input := range inputs {
		// Check if input has associated label
		id, _ := input.GetAttribute("id")
		ariaLabel, _ := input.GetAttribute("aria-label")

		if id != "" {
			// Check for label with matching 'for' attribute
			label := suite.page.Locator(fmt.Sprintf("label[for='%s']", id))
			if count, _ := label.Count(); count == 0 && ariaLabel == "" {
				t.Errorf("Input %d lacks proper labeling", i)
			}
		} else if ariaLabel == "" {
			t.Errorf("Input %d lacks proper labeling", i)
		}
	}

	// 4. Check color contrast (basic check for text visibility)
	// This would require more sophisticated color analysis in a real implementation
}

// MockHTTTP creates a mock HTTP handler for testing
func MockHTTP() http.Handler {
	mux := http.NewServeMux()

	// Mock login page
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>DuckChat - Login</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="/static/output.css" rel="stylesheet">
</head>
<body>
    <div class="login-form">
        <h1>Login</h1>
        <form method="POST">
            <input type="text" name="username" placeholder="Username" class="login-input" required>
            <input type="password" name="password" placeholder="Password" class="login-input" required>
            <button type="submit" class="login-button">Sign In</button>
        </form>
    </div>
</body>
</html>`))
		}
	})

	// Mock rooms page
	mux.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>DuckChat - Rooms</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="/static/output.css" rel="stylesheet">
</head>
<body>
    <div class="room-list">
        <div class="room-item">
            <span>General</span>
            <span class="text-gray-500">5 online</span>
        </div>
    </div>
    <button class="create-room-button">+</button>
</body>
</html>`))
	})

	// Mock static CSS
	mux.HandleFunc("/static/output.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte("/* Mock CSS */"))
	})

	return mux
}
