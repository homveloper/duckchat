package ui

import (
	"net/http"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// LoginFormTestProps represents mock data for LoginForm component
type LoginFormTestProps struct {
	CSRFToken       string
	FormState       LoginFormStateTest
	RedirectURL     string
	ErrorMessage    string
}

type LoginFormStateTest struct {
	Username          string
	IsSubmitting      bool
	ValidationErrors  map[string]string
}

// MockLoginFormHandler creates a failing mock handler for LoginForm component
// This implements TDD RED phase - tests will fail initially
func MockLoginFormHandler(props LoginFormTestProps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Intentionally broken/minimal HTML that will cause tests to fail (TDD RED phase)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Login</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
    <div>
        <h1>Login</h1>
        <form>
            <input type="text" name="username" placeholder="Username">
            <input type="password" name="password" placeholder="Password">
            <button type="submit">Sign In</button>
        </form>
    </div>
</body>
</html>`))
	})
}

func TestLoginForm_Rendering(t *testing.T) {
	testCases := []struct {
		name  string
		props LoginFormTestProps
	}{
		{
			name: "empty form",
			props: LoginFormTestProps{
				CSRFToken:   "csrf_empty_123",
				RedirectURL: "/rooms",
				FormState: LoginFormStateTest{
					Username:     "",
					IsSubmitting: false,
				},
			},
		},
		{
			name: "form with validation errors",
			props: LoginFormTestProps{
				CSRFToken:    "csrf_errors_456",
				ErrorMessage: "Invalid credentials",
				FormState: LoginFormStateTest{
					Username:     "invalid_user",
					IsSubmitting: false,
					ValidationErrors: map[string]string{
						"username": "Username must be at least 3 characters",
						"password": "Password is required",
					},
				},
			},
		},
		{
			name: "submitting form",
			props: LoginFormTestProps{
				CSRFToken: "csrf_submit_789",
				FormState: LoginFormStateTest{
					Username:     "validuser",
					IsSubmitting: true,
				},
			},
		},
		{
			name: "form with prefilled data",
			props: LoginFormTestProps{
				CSRFToken:   "csrf_prefill_abc",
				RedirectURL: "/dashboard",
				FormState: LoginFormStateTest{
					Username:     "johndoe",
					IsSubmitting: false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := MockLoginFormHandler(tc.props)
			suite := SetupBrowserTest(t, handler)
			defer suite.TeardownBrowserTest()

			if err := suite.NavigateToPath("/"); err != nil {
				t.Fatalf("Failed to navigate: %v", err)
			}

			t.Run("form_structure", func(t *testing.T) {
				// These will fail as mock doesn't have proper form structure
				form := suite.page.Locator("[data-testid='login-form']")
				suite.AssertElementVisible(t, "[data-testid='login-form']")

				// Check form method and action
				method, _ := form.GetAttribute("method")
				if method != "POST" {
					t.Errorf("Expected form method 'POST', got '%s'", method)
				}

				action, _ := form.GetAttribute("action")
				expectedAction := "/api/auth/login"
				if action != expectedAction {
					t.Errorf("Expected form action '%s', got '%s'", expectedAction, action)
				}
			})

			t.Run("csrf_token", func(t *testing.T) {
				// This will fail as mock doesn't include proper CSRF token
				csrfInput := suite.page.Locator("input[name='csrf_token']")
				suite.AssertElementVisible(t, "input[name='csrf_token']")

				value, _ := csrfInput.GetAttribute("value")
				if value != tc.props.CSRFToken {
					t.Errorf("Expected CSRF token '%s', got '%s'", tc.props.CSRFToken, value)
				}

				// CSRF input should be hidden
				inputType, _ := csrfInput.GetAttribute("type")
				if inputType != "hidden" {
					t.Errorf("CSRF input should be hidden, got type '%s'", inputType)
				}
			})

			t.Run("username_field", func(t *testing.T) {
				// This will fail as mock doesn't have proper username field
				usernameInput := suite.page.Locator("[data-testid='username-input']")
				suite.AssertElementVisible(t, "[data-testid='username-input']")

				// Check input attributes
				name, _ := usernameInput.GetAttribute("name")
				if name != "username" {
					t.Errorf("Expected username input name 'username', got '%s'", name)
				}

				inputType, _ := usernameInput.GetAttribute("type")
				if inputType != "text" {
					t.Errorf("Expected username input type 'text', got '%s'", inputType)
				}

				required, _ := usernameInput.GetAttribute("required")
				if required == "" {
					t.Error("Username input should be required")
				}

				// Check prefilled value
				value, _ := usernameInput.InputValue()
				if value != tc.props.FormState.Username {
					t.Errorf("Expected username value '%s', got '%s'", tc.props.FormState.Username, value)
				}
			})

			t.Run("password_field", func(t *testing.T) {
				// This will fail as mock doesn't have proper password field
				passwordInput := suite.page.Locator("[data-testid='password-input']")
				suite.AssertElementVisible(t, "[data-testid='password-input']")

				// Check input attributes
				name, _ := passwordInput.GetAttribute("name")
				if name != "password" {
					t.Errorf("Expected password input name 'password', got '%s'", name)
				}

				inputType, _ := passwordInput.GetAttribute("type")
				if inputType != "password" {
					t.Errorf("Expected password input type 'password', got '%s'", inputType)
				}

				required, _ := passwordInput.GetAttribute("required")
				if required == "" {
					t.Error("Password input should be required")
				}
			})

			t.Run("submit_button", func(t *testing.T) {
				// This will fail as mock doesn't have proper submit button
				submitButton := suite.page.Locator("[data-testid='submit-button']")
				suite.AssertElementVisible(t, "[data-testid='submit-button']")

				buttonType, _ := submitButton.GetAttribute("type")
				if buttonType != "submit" {
					t.Errorf("Expected submit button type 'submit', got '%s'", buttonType)
				}

				// Check button state based on form state
				if tc.props.FormState.IsSubmitting {
					disabled, _ := submitButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Submit button should be disabled when form is submitting")
					}

					// Check loading text/indicator
					buttonText, _ := submitButton.TextContent()
					if buttonText != "Signing In..." {
						t.Errorf("Expected loading text 'Signing In...', got '%s'", buttonText)
					}
				} else {
					disabled, _ := submitButton.GetAttribute("disabled")
					if disabled != "" {
						t.Error("Submit button should not be disabled when form is not submitting")
					}

					buttonText, _ := submitButton.TextContent()
					if buttonText != "Sign In" {
						t.Errorf("Expected button text 'Sign In', got '%s'", buttonText)
					}
				}
			})

			t.Run("redirect_url", func(t *testing.T) {
				if tc.props.RedirectURL != "" {
					// This will fail as mock doesn't include redirect URL
					redirectInput := suite.page.Locator("input[name='redirect_url']")
					suite.AssertElementVisible(t, "input[name='redirect_url']")

					value, _ := redirectInput.GetAttribute("value")
					if value != tc.props.RedirectURL {
						t.Errorf("Expected redirect URL '%s', got '%s'", tc.props.RedirectURL, value)
					}
				}
			})

			t.Run("error_message", func(t *testing.T) {
				if tc.props.ErrorMessage != "" {
					// This will fail as mock doesn't display error messages
					errorDiv := suite.page.Locator("[data-testid='login-error']")
					suite.AssertElementVisible(t, "[data-testid='login-error']")

					errorText, _ := errorDiv.TextContent()
					if errorText != tc.props.ErrorMessage {
						t.Errorf("Expected error message '%s', got '%s'", tc.props.ErrorMessage, errorText)
					}

					// Check error styling
					errorClass, _ := errorDiv.GetAttribute("class")
					if !contains(errorClass, "text-red-500") {
						t.Error("Error message should have red text styling")
					}
				} else {
					// Should not show error div when no error
					suite.AssertElementNotVisible(t, "[data-testid='login-error']")
				}
			})

			t.Run("validation_errors", func(t *testing.T) {
				for field, message := range tc.props.FormState.ValidationErrors {
					// This will fail as mock doesn't show validation errors
					errorSelector := "[data-testid='" + field + "-error']"
					errorElement := suite.page.Locator(errorSelector)
					suite.AssertElementVisible(t, errorSelector)

					errorText, _ := errorElement.TextContent()
					if errorText != message {
						t.Errorf("Expected %s validation error '%s', got '%s'", field, message, errorText)
					}

					// Check that corresponding input has error styling
					inputSelector := "[data-testid='" + field + "-input']"
					inputElement := suite.page.Locator(inputSelector)
					inputClass, _ := inputElement.GetAttribute("class")
					if !contains(inputClass, "border-red-500") {
						t.Errorf("Input %s should have error border styling", field)
					}
				}
			})
		})
	}
}

func TestLoginForm_ResponsiveDesign(t *testing.T) {
	props := LoginFormTestProps{
		CSRFToken:   "csrf_responsive",
		RedirectURL: "/rooms",
		FormState: LoginFormStateTest{
			Username:     "testuser",
			IsSubmitting: false,
		},
	}

	handler := MockLoginFormHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	t.Run("mobile_layout", func(t *testing.T) {
		if err := suite.SetMobileViewport(); err != nil {
			t.Fatalf("Failed to set mobile viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// These will fail as mock doesn't implement responsive design
		formContainer := suite.page.Locator("[data-testid='login-container']")
		suite.AssertElementVisible(t, "[data-testid='login-container']")

		// Check mobile-specific styling
		containerClass, _ := formContainer.GetAttribute("class")
		expectedClasses := []string{"w-full", "px-4", "py-8"}
		for _, class := range expectedClasses {
			if !contains(containerClass, class) {
				t.Errorf("Missing mobile class '%s' in container", class)
			}
		}

		// Check input sizing for mobile
		inputs := suite.page.Locator("input[type='text'], input[type='password']")
		inputCount, _ := inputs.Count()
		for i := 0; i < inputCount; i++ {
			input := inputs.Nth(i)
			inputClass, _ := input.GetAttribute("class")
			if !contains(inputClass, "w-full") {
				t.Errorf("Input %d should be full width on mobile", i)
			}
		}
	})

	t.Run("tablet_layout", func(t *testing.T) {
		if err := suite.SetTabletViewport(); err != nil {
			t.Fatalf("Failed to set tablet viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Check tablet-specific layout
		formContainer := suite.page.Locator("[data-testid='login-container']")
		suite.AssertElementVisible(t, "[data-testid='login-container']")

		containerClass, _ := formContainer.GetAttribute("class")
		if !contains(containerClass, "max-w-md") {
			t.Error("Login container should have max width on tablet")
		}
	})

	t.Run("touch_friendly", func(t *testing.T) {
		// Test touch-friendly design on mobile
		if err := suite.SetMobileViewport(); err != nil {
			t.Fatalf("Failed to set mobile viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Check minimum touch target sizes (44px)
		submitButton := suite.page.Locator("[data-testid='submit-button']")
		buttonBox, _ := submitButton.BoundingBox()
		if buttonBox.Height < 44 {
			t.Errorf("Submit button height %f is less than 44px minimum for touch", buttonBox.Height)
		}

		// Check input field heights
		inputs := suite.page.Locator("input[type='text'], input[type='password']")
		inputCount, _ := inputs.Count()
		for i := 0; i < inputCount; i++ {
			input := inputs.Nth(i)
			inputBox, _ := input.BoundingBox()
			if inputBox.Height < 44 {
				t.Errorf("Input %d height %f is less than 44px minimum for touch", i, inputBox.Height)
			}
		}
	})
}

func TestLoginForm_Accessibility(t *testing.T) {
	props := LoginFormTestProps{
		CSRFToken:   "csrf_a11y",
		RedirectURL: "/rooms",
		FormState: LoginFormStateTest{
			ValidationErrors: map[string]string{
				"username": "Username is required",
			},
		},
	}

	handler := MockLoginFormHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("form_labels", func(t *testing.T) {
		// These will fail as mock doesn't have proper labels
		usernameLabel := suite.page.Locator("label[for='username']")
		suite.AssertElementVisible(t, "label[for='username']")

		passwordLabel := suite.page.Locator("label[for='password']")
		suite.AssertElementVisible(t, "label[for='password']")

		// Check label text
		usernameLabelText, _ := usernameLabel.TextContent()
		if usernameLabelText != "Username" {
			t.Errorf("Expected username label 'Username', got '%s'", usernameLabelText)
		}

		passwordLabelText, _ := passwordLabel.TextContent()
		if passwordLabelText != "Password" {
			t.Errorf("Expected password label 'Password', got '%s'", passwordLabelText)
		}
	})

	t.Run("input_associations", func(t *testing.T) {
		// Check input-label associations
		usernameInput := suite.page.Locator("[data-testid='username-input']")
		usernameId, _ := usernameInput.GetAttribute("id")
		if usernameId != "username" {
			t.Errorf("Expected username input id 'username', got '%s'", usernameId)
		}

		passwordInput := suite.page.Locator("[data-testid='password-input']")
		passwordId, _ := passwordInput.GetAttribute("id")
		if passwordId != "password" {
			t.Errorf("Expected password input id 'password', got '%s'", passwordId)
		}
	})

	t.Run("aria_labels", func(t *testing.T) {
		// Check ARIA labels for better screen reader support
		form := suite.page.Locator("[data-testid='login-form']")
		ariaLabel, _ := form.GetAttribute("aria-label")
		if ariaLabel != "Login form" {
			t.Errorf("Expected form aria-label 'Login form', got '%s'", ariaLabel)
		}

		submitButton := suite.page.Locator("[data-testid='submit-button']")
		buttonAriaLabel, _ := submitButton.GetAttribute("aria-label")
		if buttonAriaLabel != "Sign in to your account" {
			t.Errorf("Expected button aria-label 'Sign in to your account', got '%s'", buttonAriaLabel)
		}
	})

	t.Run("error_announcements", func(t *testing.T) {
		// Check that errors are properly announced to screen readers
		for field := range props.FormState.ValidationErrors {
			errorElement := suite.page.Locator("[data-testid='" + field + "-error']")
			ariaLive, _ := errorElement.GetAttribute("aria-live")
			if ariaLive != "polite" {
				t.Errorf("Error element for %s should have aria-live='polite'", field)
			}

			role, _ := errorElement.GetAttribute("role")
			if role != "alert" {
				t.Errorf("Error element for %s should have role='alert'", field)
			}
		}
	})

	t.Run("keyboard_navigation", func(t *testing.T) {
		// Test tab order
		usernameInput := suite.page.Locator("[data-testid='username-input']")
		tabIndex, _ := usernameInput.GetAttribute("tabindex")
		if tabIndex != "1" && tabIndex == "" {
			// Either explicit tabindex=1 or natural tab order
		}

		passwordInput := suite.page.Locator("[data-testid='password-input']")
		submitButton := suite.page.Locator("[data-testid='submit-button']")

		// Test that Enter key submits form from password field
		if err := passwordInput.Focus(); err != nil {
			t.Fatalf("Failed to focus password input: %v", err)
		}

		// Simulate Enter keypress (this would trigger form submission in real implementation)
		if err := suite.page.Keyboard().Press("Enter"); err != nil {
			t.Fatalf("Failed to press Enter: %v", err)
		}
	})
}

func TestLoginForm_HTMXIntegration(t *testing.T) {
	props := LoginFormTestProps{
		CSRFToken:   "csrf_htmx",
		RedirectURL: "/rooms",
		FormState: LoginFormStateTest{
			Username:     "testuser",
			IsSubmitting: false,
		},
	}

	handler := MockLoginFormHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("htmx_attributes", func(t *testing.T) {
		// These will fail as mock doesn't include HTMX attributes
		form := suite.page.Locator("[data-testid='login-form']")

		// Check HTMX form submission
		hxPost, _ := form.GetAttribute("hx-post")
		if hxPost != "/api/auth/login" {
			t.Errorf("Expected hx-post '/api/auth/login', got '%s'", hxPost)
		}

		// Check HTMX target for response
		hxTarget, _ := form.GetAttribute("hx-target")
		if hxTarget != "#login-response" {
			t.Errorf("Expected hx-target '#login-response', got '%s'", hxTarget)
		}

		// Check HTMX swap strategy
		hxSwap, _ := form.GetAttribute("hx-swap")
		if hxSwap != "innerHTML" {
			t.Errorf("Expected hx-swap 'innerHTML', got '%s'", hxSwap)
		}
	})

	t.Run("htmx_indicators", func(t *testing.T) {
		// Check for HTMX loading indicators
		loadingIndicator := suite.page.Locator("[data-testid='login-loading']")
		suite.AssertElementVisible(t, "[data-testid='login-loading']")

		// Check htmx-indicator class
		indicatorClass, _ := loadingIndicator.GetAttribute("class")
		if !contains(indicatorClass, "htmx-indicator") {
			t.Error("Loading indicator should have htmx-indicator class")
		}

		// Should be initially hidden
		style, _ := loadingIndicator.GetAttribute("style")
		if !contains(style, "display: none") && !contains(indicatorClass, "hidden") {
			t.Error("Loading indicator should be initially hidden")
		}
	})

	t.Run("csrf_header", func(t *testing.T) {
		// Check HTMX CSRF header configuration
		form := suite.page.Locator("[data-testid='login-form']")
		hxHeaders, _ := form.GetAttribute("hx-headers")

		expectedHeaders := `{"X-CSRF-Token": "` + props.CSRFToken + `"}`
		if hxHeaders != expectedHeaders {
			t.Errorf("Expected hx-headers '%s', got '%s'", expectedHeaders, hxHeaders)
		}
	})

	t.Run("validation_triggers", func(t *testing.T) {
		// Check for client-side validation triggers
		usernameInput := suite.page.Locator("[data-testid='username-input']")
		hxTrigger, _ := usernameInput.GetAttribute("hx-trigger")
		if hxTrigger != "blur, input delay:500ms" {
			t.Errorf("Expected username hx-trigger 'blur, input delay:500ms', got '%s'", hxTrigger)
		}

		passwordInput := suite.page.Locator("[data-testid='password-input']")
		hxTrigger, _ = passwordInput.GetAttribute("hx-trigger")
		if hxTrigger != "blur" {
			t.Errorf("Expected password hx-trigger 'blur', got '%s'", hxTrigger)
		}
	})
}

func TestLoginForm_TailwindStyling(t *testing.T) {
	props := LoginFormTestProps{
		CSRFToken:   "csrf_tailwind",
		RedirectURL: "/rooms",
		FormState: LoginFormStateTest{
			Username:     "testuser",
			IsSubmitting: false,
			ValidationErrors: map[string]string{
				"username": "Username too short",
			},
		},
	}

	handler := MockLoginFormHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("container_styling", func(t *testing.T) {
		// These will fail as mock doesn't apply Tailwind classes
		container := suite.page.Locator("[data-testid='login-container']")
		containerClass, _ := container.GetAttribute("class")

		expectedClasses := []string{
			"min-h-screen",
			"flex",
			"items-center",
			"justify-center",
			"bg-gray-50",
			"py-12",
			"px-4",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(containerClass, expectedClass) {
				t.Errorf("Container missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("form_styling", func(t *testing.T) {
		form := suite.page.Locator("[data-testid='login-form']")
		formClass, _ := form.GetAttribute("class")

		expectedClasses := []string{
			"max-w-md",
			"w-full",
			"space-y-8",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(formClass, expectedClass) {
				t.Errorf("Form missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("input_styling", func(t *testing.T) {
		usernameInput := suite.page.Locator("[data-testid='username-input']")
		inputClass, _ := usernameInput.GetAttribute("class")

		expectedClasses := []string{
			"block",
			"w-full",
			"px-3",
			"py-2",
			"border",
			"border-gray-300",
			"placeholder-gray-500",
			"text-gray-900",
			"rounded-md",
			"focus:outline-none",
			"focus:ring-indigo-500",
			"focus:border-indigo-500",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(inputClass, expectedClass) {
				t.Errorf("Input missing Tailwind class: %s", expectedClass)
			}
		}

		// Check error state styling if validation errors exist
		if len(props.FormState.ValidationErrors) > 0 {
			if !contains(inputClass, "border-red-500") {
				t.Error("Input with error should have red border")
			}
		}
	})

	t.Run("button_styling", func(t *testing.T) {
		submitButton := suite.page.Locator("[data-testid='submit-button']")
		buttonClass, _ := submitButton.GetAttribute("class")

		expectedClasses := []string{
			"group",
			"relative",
			"w-full",
			"flex",
			"justify-center",
			"py-2",
			"px-4",
			"border",
			"border-transparent",
			"text-sm",
			"font-medium",
			"rounded-md",
			"text-white",
			"bg-indigo-600",
			"hover:bg-indigo-700",
			"focus:outline-none",
			"focus:ring-2",
			"focus:ring-offset-2",
			"focus:ring-indigo-500",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(buttonClass, expectedClass) {
				t.Errorf("Button missing Tailwind class: %s", expectedClass)
			}
		}

		// Check disabled state styling
		if props.FormState.IsSubmitting {
			if !contains(buttonClass, "opacity-50") {
				t.Error("Submitting button should have opacity-50 class")
			}
		}
	})

	t.Run("error_styling", func(t *testing.T) {
		for field := range props.FormState.ValidationErrors {
			errorElement := suite.page.Locator("[data-testid='" + field + "-error']")
			errorClass, _ := errorElement.GetAttribute("class")

			expectedClasses := []string{
				"mt-2",
				"text-sm",
				"text-red-600",
			}

			for _, expectedClass := range expectedClasses {
				if !contains(errorClass, expectedClass) {
					t.Errorf("Error element missing Tailwind class: %s", expectedClass)
				}
			}
		}
	})
}

func TestLoginForm_InteractionBehavior(t *testing.T) {
	props := LoginFormTestProps{
		CSRFToken:   "csrf_interaction",
		RedirectURL: "/rooms",
		FormState: LoginFormStateTest{
			IsSubmitting: false,
		},
	}

	handler := MockLoginFormHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("form_submission", func(t *testing.T) {
		// Test form submission behavior
		usernameInput := suite.page.Locator("[data-testid='username-input']")
		passwordInput := suite.page.Locator("[data-testid='password-input']")
		submitButton := suite.page.Locator("[data-testid='submit-button']")

		// Fill form
		if err := usernameInput.Fill("testuser"); err != nil {
			t.Fatalf("Failed to fill username: %v", err)
		}
		if err := passwordInput.Fill("testpass"); err != nil {
			t.Fatalf("Failed to fill password: %v", err)
		}

		// Submit form
		if err := submitButton.Click(); err != nil {
			t.Fatalf("Failed to click submit: %v", err)
		}

		// In a real implementation, we would:
		// 1. Check that loading state is shown
		// 2. Wait for response
		// 3. Verify redirect or error handling
		t.Log("Form submission behavior would be tested with actual implementation")
	})

	t.Run("input_validation", func(t *testing.T) {
		usernameInput := suite.page.Locator("[data-testid='username-input']")

		// Test minimum length validation
		if err := usernameInput.Fill("ab"); err != nil {
			t.Fatalf("Failed to fill short username: %v", err)
		}

		// Trigger validation (blur event)
		if err := usernameInput.Blur(); err != nil {
			t.Fatalf("Failed to blur username input: %v", err)
		}

		// In real implementation, validation error would appear
		time.Sleep(100 * time.Millisecond) // Allow time for validation
		t.Log("Input validation would be tested with actual HTMX validation endpoints")
	})

	t.Run("progressive_enhancement", func(t *testing.T) {
		// Test that form works without JavaScript (progressive enhancement)
		// This is important for accessibility and robustness

		form := suite.page.Locator("[data-testid='login-form']")
		action, _ := form.GetAttribute("action")
		method, _ := form.GetAttribute("method")

		// Form should have proper action and method for non-JS submission
		if action == "" || method != "POST" {
			t.Error("Form should work without JavaScript (progressive enhancement)")
		}
	})
}