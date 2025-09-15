package ui

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// MessageInputTestProps represents mock data for MessageInput component
type MessageInputTestProps struct {
	RoomID        string
	CSRFToken     string
	InputState    MessageInputStateTest
	MaxLength     int
	IsConnected   bool
	CurrentUser   UserContextTest
}

// MockMessageInputHandler creates a failing mock handler for MessageInput component
// This implements TDD RED phase - tests will fail initially
func MockMessageInputHandler(props MessageInputTestProps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Intentionally broken/minimal HTML that will cause tests to fail (TDD RED phase)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Message Input Test</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
    <div>
        <form>
            <input type="text" placeholder="Type a message...">
            <button>Send</button>
        </form>
    </div>
</body>
</html>`))
	})
}

func TestMessageInput_Rendering(t *testing.T) {
	testCases := []struct {
		name  string
		props MessageInputTestProps
	}{
		{
			name: "empty message input",
			props: MessageInputTestProps{
				RoomID:    "room_123",
				CSRFToken: "csrf_empty_token",
				InputState: MessageInputStateTest{
					Content:        "",
					CharacterCount: 0,
					IsSending:      false,
				},
				MaxLength:   1000,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_123",
					Username:        "testuser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "message input with content",
			props: MessageInputTestProps{
				RoomID:    "room_456",
				CSRFToken: "csrf_content_token",
				InputState: MessageInputStateTest{
					Content:        "Hello everyone! How is everyone doing today?",
					CharacterCount: 44,
					IsSending:      false,
				},
				MaxLength:   1000,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_456",
					Username:        "contentuser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "sending message state",
			props: MessageInputTestProps{
				RoomID:    "room_789",
				CSRFToken: "csrf_sending_token",
				InputState: MessageInputStateTest{
					Content:        "This message is being sent...",
					CharacterCount: 30,
					IsSending:      true,
				},
				MaxLength:   1000,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_789",
					Username:        "sendinguser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "disconnected state",
			props: MessageInputTestProps{
				RoomID:    "room_disconnected",
				CSRFToken: "csrf_disconnected_token",
				InputState: MessageInputStateTest{
					Content:        "Can't send while disconnected",
					CharacterCount: 29,
					IsSending:      false,
				},
				MaxLength:   1000,
				IsConnected: false,
				CurrentUser: UserContextTest{
					ID:              "user_disconnected",
					Username:        "disconnecteduser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "validation error state",
			props: MessageInputTestProps{
				RoomID:    "room_validation",
				CSRFToken: "csrf_validation_token",
				InputState: MessageInputStateTest{
					Content:         "",
					CharacterCount:  0,
					IsSending:       false,
					ValidationError: "Message cannot be empty",
				},
				MaxLength:   1000,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_validation",
					Username:        "validationuser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "near character limit",
			props: MessageInputTestProps{
				RoomID:    "room_limit",
				CSRFToken: "csrf_limit_token",
				InputState: MessageInputStateTest{
					Content:        "This is a very long message that approaches the character limit for testing purposes and should show a warning when near the maximum allowed length",
					CharacterCount: 152,
					IsSending:      false,
				},
				MaxLength:   200,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_limit",
					Username:        "limituser",
					IsAuthenticated: true,
				},
			},
		},
		{
			name: "over character limit",
			props: MessageInputTestProps{
				RoomID:    "room_over_limit",
				CSRFToken: "csrf_over_limit_token",
				InputState: MessageInputStateTest{
					Content:         "This message exceeds the maximum character limit and should show an error",
					CharacterCount:  73,
					IsSending:       false,
					ValidationError: "Message exceeds maximum length of 50 characters",
				},
				MaxLength:   50,
				IsConnected: true,
				CurrentUser: UserContextTest{
					ID:              "user_over_limit",
					Username:        "overlimituser",
					IsAuthenticated: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := MockMessageInputHandler(tc.props)
			suite := SetupBrowserTest(t, handler)
			defer suite.TeardownBrowserTest()

			if err := suite.NavigateToPath("/"); err != nil {
				t.Fatalf("Failed to navigate: %v", err)
			}

			t.Run("form_structure", func(t *testing.T) {
				// These will fail as mock doesn't have proper form structure
				messageForm := suite.page.Locator("[data-testid='message-input-form']")
				suite.AssertElementVisible(t, "[data-testid='message-input-form']")

				// Check form attributes
				method, _ := messageForm.GetAttribute("method")
				if method != "POST" {
					t.Errorf("Expected form method 'POST', got '%s'", method)
				}

				action, _ := messageForm.GetAttribute("action")
				expectedAction := fmt.Sprintf("/api/rooms/%s/messages", tc.props.RoomID)
				if action != expectedAction {
					t.Errorf("Expected form action '%s', got '%s'", expectedAction, action)
				}

				// Check form container
				formContainer := suite.page.Locator("[data-testid='message-input-container']")
				suite.AssertElementVisible(t, "[data-testid='message-input-container']")
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

			t.Run("room_id_field", func(t *testing.T) {
				// Room ID should be included as hidden field
				roomIdInput := suite.page.Locator("input[name='room_id']")
				suite.AssertElementVisible(t, "input[name='room_id']")

				value, _ := roomIdInput.GetAttribute("value")
				if value != tc.props.RoomID {
					t.Errorf("Expected room ID '%s', got '%s'", tc.props.RoomID, value)
				}

				inputType, _ := roomIdInput.GetAttribute("type")
				if inputType != "hidden" {
					t.Errorf("Room ID input should be hidden, got type '%s'", inputType)
				}
			})

			t.Run("message_textarea", func(t *testing.T) {
				// This will fail as mock doesn't have proper textarea
				messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
				suite.AssertElementVisible(t, "[data-testid='message-textarea']")

				// Check textarea attributes
				name, _ := messageTextarea.GetAttribute("name")
				if name != "content" {
					t.Errorf("Expected textarea name 'content', got '%s'", name)
				}

				placeholder, _ := messageTextarea.GetAttribute("placeholder")
				expectedPlaceholder := "Type a message..."
				if placeholder != expectedPlaceholder {
					t.Errorf("Expected textarea placeholder '%s', got '%s'", expectedPlaceholder, placeholder)
				}

				maxLength, _ := messageTextarea.GetAttribute("maxlength")
				expectedMaxLength := fmt.Sprintf("%d", tc.props.MaxLength)
				if maxLength != expectedMaxLength {
					t.Errorf("Expected textarea maxlength '%s', got '%s'", expectedMaxLength, maxLength)
				}

				// Check current value
				value, _ := messageTextarea.InputValue()
				if value != tc.props.InputState.Content {
					t.Errorf("Expected textarea value '%s', got '%s'", tc.props.InputState.Content, value)
				}

				// Check required attribute
				required, _ := messageTextarea.GetAttribute("required")
				if required == "" {
					t.Error("Message textarea should be required")
				}

				// Check disabled state based on connection
				if !tc.props.IsConnected {
					disabled, _ := messageTextarea.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Textarea should be disabled when not connected")
					}
				} else {
					disabled, _ := messageTextarea.GetAttribute("disabled")
					if disabled != "" && !tc.props.InputState.IsSending {
						t.Error("Textarea should not be disabled when connected and not sending")
					}
				}

				// Check disabled state when sending
				if tc.props.InputState.IsSending {
					disabled, _ := messageTextarea.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Textarea should be disabled when sending message")
					}
				}
			})

			t.Run("send_button", func(t *testing.T) {
				// This will fail as mock doesn't have proper send button
				sendButton := suite.page.Locator("[data-testid='send-button']")
				suite.AssertElementVisible(t, "[data-testid='send-button']")

				// Check button attributes
				buttonType, _ := sendButton.GetAttribute("type")
				if buttonType != "submit" {
					t.Errorf("Expected send button type 'submit', got '%s'", buttonType)
				}

				// Check button state based on conditions
				if tc.props.InputState.IsSending {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Send button should be disabled when sending message")
					}

					buttonText, _ := sendButton.TextContent()
					if buttonText != "Sending..." {
						t.Errorf("Expected sending button text 'Sending...', got '%s'", buttonText)
					}
				} else if !tc.props.IsConnected {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Send button should be disabled when not connected")
					}

					buttonText, _ := sendButton.TextContent()
					if buttonText != "Disconnected" {
						t.Errorf("Expected disconnected button text 'Disconnected', got '%s'", buttonText)
					}
				} else if tc.props.InputState.Content == "" {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Send button should be disabled when message is empty")
					}

					buttonText, _ := sendButton.TextContent()
					if buttonText != "Send" {
						t.Errorf("Expected button text 'Send', got '%s'", buttonText)
					}
				} else if tc.props.InputState.CharacterCount > tc.props.MaxLength {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Send button should be disabled when message exceeds max length")
					}
				} else {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled != "" {
						t.Error("Send button should be enabled when conditions are met")
					}

					buttonText, _ := sendButton.TextContent()
					if buttonText != "Send" {
						t.Errorf("Expected button text 'Send', got '%s'", buttonText)
					}
				}
			})

			t.Run("character_counter", func(t *testing.T) {
				// This will fail as mock doesn't include character counter
				charCounter := suite.page.Locator("[data-testid='character-counter']")
				suite.AssertElementVisible(t, "[data-testid='character-counter']")

				// Check counter text
				counterText, _ := charCounter.TextContent()
				expectedCounterText := fmt.Sprintf("%d/%d", tc.props.InputState.CharacterCount, tc.props.MaxLength)
				if counterText != expectedCounterText {
					t.Errorf("Expected character counter '%s', got '%s'", expectedCounterText, counterText)
				}

				// Check counter color based on character count
				charCounterClass, _ := charCounter.GetAttribute("class")
				percentage := float64(tc.props.InputState.CharacterCount) / float64(tc.props.MaxLength)

				if percentage > 1.0 {
					// Over limit - red
					if !contains(charCounterClass, "text-red-500") && !contains(charCounterClass, "text-red-600") {
						t.Error("Character counter should be red when over limit")
					}
				} else if percentage >= 0.9 {
					// Near limit - amber/yellow
					if !contains(charCounterClass, "text-amber-500") && !contains(charCounterClass, "text-yellow-500") {
						t.Error("Character counter should be amber/yellow when near limit")
					}
				} else {
					// Normal - gray
					if !contains(charCounterClass, "text-gray-500") && !contains(charCounterClass, "text-gray-400") {
						t.Error("Character counter should be gray when within normal range")
					}
				}
			})

			t.Run("validation_error", func(t *testing.T) {
				if tc.props.InputState.ValidationError != "" {
					// Should show validation error
					errorElement := suite.page.Locator("[data-testid='message-input-error']")
					suite.AssertElementVisible(t, "[data-testid='message-input-error']")

					errorText, _ := errorElement.TextContent()
					if errorText != tc.props.InputState.ValidationError {
						t.Errorf("Expected validation error '%s', got '%s'", tc.props.InputState.ValidationError, errorText)
					}

					// Check error styling
					errorClass, _ := errorElement.GetAttribute("class")
					if !contains(errorClass, "text-red-500") && !contains(errorClass, "text-red-600") {
						t.Error("Validation error should have red text styling")
					}

					// Textarea should have error border
					messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
					textareaClass, _ := messageTextarea.GetAttribute("class")
					if !contains(textareaClass, "border-red-500") && !contains(textareaClass, "border-red-300") {
						t.Error("Textarea should have red border when validation error exists")
					}
				} else {
					// Should not show validation error when no error
					suite.AssertElementNotVisible(t, "[data-testid='message-input-error']")
				}
			})

			t.Run("connection_indicator", func(t *testing.T) {
				connectionIndicator := suite.page.Locator("[data-testid='connection-indicator']")
				suite.AssertElementVisible(t, "[data-testid='connection-indicator']")

				if tc.props.IsConnected {
					indicatorClass, _ := connectionIndicator.GetAttribute("class")
					if !contains(indicatorClass, "text-green-500") && !contains(indicatorClass, "bg-green-500") {
						t.Error("Connection indicator should show green when connected")
					}
				} else {
					indicatorClass, _ := connectionIndicator.GetAttribute("class")
					if !contains(indicatorClass, "text-red-500") && !contains(indicatorClass, "bg-red-500") {
						t.Error("Connection indicator should show red when disconnected")
					}

					// Should show disconnection message
					disconnectMessage := suite.page.Locator("[data-testid='disconnect-message']")
					suite.AssertElementVisible(t, "[data-testid='disconnect-message']")

					messageText, _ := disconnectMessage.TextContent()
					expectedMessage := "Connection lost. Messages cannot be sent."
					if messageText != expectedMessage {
						t.Errorf("Expected disconnect message '%s', got '%s'", expectedMessage, messageText)
					}
				}
			})

			t.Run("keyboard_shortcuts", func(t *testing.T) {
				// Check for keyboard shortcut hints
				shortcutHint := suite.page.Locator("[data-testid='keyboard-shortcuts']")
				suite.AssertElementVisible(t, "[data-testid='keyboard-shortcuts']")

				hintText, _ := shortcutHint.TextContent()
				expectedHint := "Press Enter to send, Shift+Enter for new line"
				if hintText != expectedHint {
					t.Errorf("Expected keyboard shortcut hint '%s', got '%s'", expectedHint, hintText)
				}
			})

			t.Run("emoji_picker_trigger", func(t *testing.T) {
				// Check for emoji picker button
				emojiButton := suite.page.Locator("[data-testid='emoji-picker-button']")
				suite.AssertElementVisible(t, "[data-testid='emoji-picker-button']")

				buttonType, _ := emojiButton.GetAttribute("type")
				if buttonType != "button" {
					t.Errorf("Expected emoji button type 'button', got '%s'", buttonType)
				}

				ariaLabel, _ := emojiButton.GetAttribute("aria-label")
				if ariaLabel != "Open emoji picker" {
					t.Errorf("Expected emoji button aria-label 'Open emoji picker', got '%s'", ariaLabel)
				}
			})

			t.Run("file_upload_trigger", func(t *testing.T) {
				// Check for file upload button
				fileButton := suite.page.Locator("[data-testid='file-upload-button']")
				suite.AssertElementVisible(t, "[data-testid='file-upload-button']")

				buttonType, _ := fileButton.GetAttribute("type")
				if buttonType != "button" {
					t.Errorf("Expected file button type 'button', got '%s'", buttonType)
				}

				ariaLabel, _ := fileButton.GetAttribute("aria-label")
				if ariaLabel != "Upload file" {
					t.Errorf("Expected file button aria-label 'Upload file', got '%s'", ariaLabel)
				}

				// Hidden file input
				fileInput := suite.page.Locator("input[type='file'][data-testid='file-input']")
				suite.AssertElementVisible(t, "input[type='file'][data-testid='file-input']")

				accept, _ := fileInput.GetAttribute("accept")
				expectedAccept := "image/*,video/*,.pdf,.doc,.docx,.txt"
				if accept != expectedAccept {
					t.Errorf("Expected file input accept '%s', got '%s'", expectedAccept, accept)
				}
			})
		})
	}
}

func TestMessageInput_ResponsiveDesign(t *testing.T) {
	props := MessageInputTestProps{
		RoomID:    "room_responsive",
		CSRFToken: "csrf_responsive_token",
		InputState: MessageInputStateTest{
			Content:        "Testing responsive design across different screen sizes",
			CharacterCount: 57,
			IsSending:      false,
		},
		MaxLength:   1000,
		IsConnected: true,
		CurrentUser: UserContextTest{
			ID:              "user_responsive",
			Username:        "responsiveuser",
			IsAuthenticated: true,
		},
	}

	handler := MockMessageInputHandler(props)
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
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		// Check mobile-specific classes
		expectedMobileClasses := []string{
			"fixed",
			"bottom-0",
			"left-0",
			"right-0",
			"bg-white",
			"border-t",
			"border-gray-200",
			"p-4",
			"safe-area-pb", // For devices with home indicator
		}

		for _, class := range expectedMobileClasses {
			if !contains(containerClass, class) {
				t.Errorf("Input container missing mobile class: %s", class)
			}
		}

		// Check textarea sizing on mobile
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaClass, _ := messageTextarea.GetAttribute("class")

		if !contains(textareaClass, "w-full") {
			t.Error("Message textarea should be full width on mobile")
		}

		// Check button layout - should stack or be appropriately sized
		sendButton := suite.page.Locator("[data-testid='send-button']")
		buttonBox, _ := sendButton.BoundingBox()
		if buttonBox.Height < 44 {
			t.Errorf("Send button height %f is less than 44px minimum for touch", buttonBox.Height)
		}

		// Additional action buttons should be appropriately sized
		emojiButton := suite.page.Locator("[data-testid='emoji-picker-button']")
		emojiBox, _ := emojiButton.BoundingBox()
		if emojiBox.Height < 44 {
			t.Errorf("Emoji button height %f is less than 44px minimum for touch", emojiBox.Height)
		}

		fileButton := suite.page.Locator("[data-testid='file-upload-button']")
		fileBox, _ := fileButton.BoundingBox()
		if fileBox.Height < 44 {
			t.Errorf("File button height %f is less than 44px minimum for touch", fileBox.Height)
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
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		// Should not be fixed positioned on tablet
		if contains(containerClass, "fixed") && !contains(containerClass, "md:relative") {
			t.Error("Input container should not be fixed positioned on tablet")
		}

		// Should have appropriate max-width
		if !contains(containerClass, "md:max-w-none") {
			t.Error("Input container should use full width on tablet")
		}

		// Textarea should have comfortable sizing
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaClass, _ := messageTextarea.GetAttribute("class")

		if !contains(textareaClass, "md:min-h-12") {
			t.Error("Textarea should have minimum height on tablet")
		}
	})

	t.Run("desktop_layout", func(t *testing.T) {
		if err := suite.SetDesktopViewport(); err != nil {
			t.Fatalf("Failed to set desktop viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Desktop layout should not be fixed
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		if contains(containerClass, "fixed") && !contains(containerClass, "lg:relative") {
			t.Error("Input container should not be fixed positioned on desktop")
		}

		// Should show all features
		suite.AssertElementVisible(t, "[data-testid='emoji-picker-button']")
		suite.AssertElementVisible(t, "[data-testid='file-upload-button']")
		suite.AssertElementVisible(t, "[data-testid='keyboard-shortcuts']")

		// Buttons should be inline on desktop
		buttonGroup := suite.page.Locator("[data-testid='input-button-group']")
		buttonGroupClass, _ := buttonGroup.GetAttribute("class")

		if !contains(buttonGroupClass, "lg:flex") || !contains(buttonGroupClass, "lg:items-center") {
			t.Error("Button group should use flex layout on desktop")
		}
	})

	t.Run("landscape_mobile", func(t *testing.T) {
		// Test landscape orientation on mobile
		if err := suite.page.SetViewportSize(812, 375); err != nil { // iPhone X landscape
			t.Fatalf("Failed to set landscape mobile viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Input should adapt to landscape orientation
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		// Might have reduced padding or different layout in landscape
		if !contains(containerClass, "landscape:py-2") {
			t.Error("Input container should have reduced padding in landscape mobile")
		}

		// Textarea height might be smaller in landscape
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaClass, _ := messageTextarea.GetAttribute("class")

		if !contains(textareaClass, "landscape:max-h-20") {
			t.Error("Textarea should have max height limit in landscape mobile")
		}
	})

	t.Run("keyboard_visible_mobile", func(t *testing.T) {
		// Test layout when virtual keyboard is visible
		if err := suite.SetMobileViewport(); err != nil {
			t.Fatalf("Failed to set mobile viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Focus the textarea to simulate keyboard appearance
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		if err := messageTextarea.Focus(); err != nil {
			t.Fatalf("Failed to focus textarea: %v", err)
		}

		// Input container should adjust for keyboard
		// Note: This is challenging to test automatically as it requires
		// actual virtual keyboard interaction, but we can check for classes
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		// Should have keyboard-aware positioning
		if !contains(containerClass, "keyboard-aware") && !contains(containerClass, "env-keyboard") {
			t.Log("Keyboard-aware positioning would be tested with actual virtual keyboard")
		}
	})
}

func TestMessageInput_Accessibility(t *testing.T) {
	props := MessageInputTestProps{
		RoomID:    "room_a11y",
		CSRFToken: "csrf_a11y_token",
		InputState: MessageInputStateTest{
			Content:         "Testing accessibility",
			CharacterCount:  21,
			IsSending:       false,
			ValidationError: "Message contains invalid characters",
		},
		MaxLength:   1000,
		IsConnected: true,
		CurrentUser: UserContextTest{
			ID:              "user_a11y",
			Username:        "a11yuser",
			IsAuthenticated: true,
		},
	}

	handler := MockMessageInputHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("form_labeling", func(t *testing.T) {
		// These will fail as mock doesn't include proper labels
		messageForm := suite.page.Locator("[data-testid='message-input-form']")
		ariaLabel, _ := messageForm.GetAttribute("aria-label")
		if ariaLabel != "Send message form" {
			t.Errorf("Expected form aria-label 'Send message form', got '%s'", ariaLabel)
		}

		// Textarea should have proper labeling
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaId, _ := messageTextarea.GetAttribute("id")
		if textareaId != "message-content" {
			t.Errorf("Expected textarea id 'message-content', got '%s'", textareaId)
		}

		// Check for associated label
		label := suite.page.Locator("label[for='message-content']")
		suite.AssertElementVisible(t, "label[for='message-content']")

		labelText, _ := label.TextContent()
		if labelText != "Message" {
			t.Errorf("Expected label text 'Message', got '%s'", labelText)
		}

		// Check aria-label as alternative
		ariaLabel, _ = messageTextarea.GetAttribute("aria-label")
		if textareaId == "" && ariaLabel != "Type your message" {
			t.Error("Textarea should have either proper id/label or aria-label")
		}
	})

	t.Run("button_accessibility", func(t *testing.T) {
		// Send button accessibility
		sendButton := suite.page.Locator("[data-testid='send-button']")
		ariaLabel, _ := sendButton.GetAttribute("aria-label")
		if ariaLabel != "Send message" {
			t.Errorf("Expected send button aria-label 'Send message', got '%s'", ariaLabel)
		}

		// Check aria-describedby for additional context
		ariaDescribedBy, _ := sendButton.GetAttribute("aria-describedby")
		if ariaDescribedBy == "" {
			t.Error("Send button should have aria-describedby for status information")
		}

		// Emoji picker button
		emojiButton := suite.page.Locator("[data-testid='emoji-picker-button']")
		emojiAriaLabel, _ := emojiButton.GetAttribute("aria-label")
		if emojiAriaLabel != "Open emoji picker" {
			t.Errorf("Expected emoji button aria-label 'Open emoji picker', got '%s'", emojiAriaLabel)
		}

		// File upload button
		fileButton := suite.page.Locator("[data-testid='file-upload-button']")
		fileAriaLabel, _ := fileButton.GetAttribute("aria-label")
		if fileAriaLabel != "Upload file" {
			t.Errorf("Expected file button aria-label 'Upload file', got '%s'", fileAriaLabel)
		}
	})

	t.Run("status_announcements", func(t *testing.T) {
		// Character counter should be accessible
		charCounter := suite.page.Locator("[data-testid='character-counter']")
		ariaLive, _ := charCounter.GetAttribute("aria-live")
		if ariaLive != "polite" {
			t.Error("Character counter should have aria-live='polite' for screen reader updates")
		}

		ariaLabel, _ := charCounter.GetAttribute("aria-label")
		expectedLabel := fmt.Sprintf("Character count: %d of %d", props.InputState.CharacterCount, props.MaxLength)
		if ariaLabel != expectedLabel {
			t.Errorf("Expected character counter aria-label '%s', got '%s'", expectedLabel, ariaLabel)
		}

		// Connection indicator accessibility
		connectionIndicator := suite.page.Locator("[data-testid='connection-indicator']")
		connAriaLive, _ := connectionIndicator.GetAttribute("aria-live")
		if connAriaLive != "assertive" {
			t.Error("Connection indicator should have aria-live='assertive' for immediate announcements")
		}

		connAriaLabel, _ := connectionIndicator.GetAttribute("aria-label")
		if props.IsConnected {
			if connAriaLabel != "Connected to chat" {
				t.Errorf("Expected connected aria-label 'Connected to chat', got '%s'", connAriaLabel)
			}
		} else {
			if connAriaLabel != "Disconnected from chat" {
				t.Errorf("Expected disconnected aria-label 'Disconnected from chat', got '%s'", connAriaLabel)
			}
		}
	})

	t.Run("error_accessibility", func(t *testing.T) {
		// Validation error should be accessible
		if props.InputState.ValidationError != "" {
			errorElement := suite.page.Locator("[data-testid='message-input-error']")

			// Error should have proper role
			errorRole, _ := errorElement.GetAttribute("role")
			if errorRole != "alert" {
				t.Error("Validation error should have role='alert'")
			}

			// Error should be live region
			ariaLive, _ := errorElement.GetAttribute("aria-live")
			if ariaLive != "assertive" {
				t.Error("Validation error should have aria-live='assertive'")
			}

			// Textarea should reference error
			messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
			ariaDescribedBy, _ := messageTextarea.GetAttribute("aria-describedby")
			if !contains(ariaDescribedBy, "message-input-error") {
				t.Error("Textarea should reference validation error via aria-describedby")
			}

			// Error should be marked as invalid
			ariaInvalid, _ := messageTextarea.GetAttribute("aria-invalid")
			if ariaInvalid != "true" {
				t.Error("Textarea should have aria-invalid='true' when validation error exists")
			}
		}
	})

	t.Run("keyboard_navigation", func(t *testing.T) {
		// Test tab order
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		if err := messageTextarea.Focus(); err != nil {
			t.Fatalf("Failed to focus textarea: %v", err)
		}

		// Tab should move to emoji button
		if err := suite.page.Keyboard().Press("Tab"); err != nil {
			t.Fatalf("Failed to press Tab: %v", err)
		}

		// Tab again should move to file button
		if err := suite.page.Keyboard().Press("Tab"); err != nil {
			t.Fatalf("Failed to press Tab: %v", err)
		}

		// Tab again should move to send button
		if err := suite.page.Keyboard().Press("Tab"); err != nil {
			t.Fatalf("Failed to press Tab: %v", err)
		}

		// In real implementation, we would check focus is on send button
		t.Log("Tab order would be verified with actual focus tracking")

		// Test keyboard shortcuts
		if err := messageTextarea.Focus(); err != nil {
			t.Fatalf("Failed to focus textarea: %v", err)
		}

		// Enter should submit (unless Shift+Enter)
		if err := suite.page.Keyboard().Press("Enter"); err != nil {
			t.Fatalf("Failed to press Enter: %v", err)
		}

		// Shift+Enter should create new line
		if err := messageTextarea.Focus(); err != nil {
			t.Fatalf("Failed to focus textarea: %v", err)
		}
		if err := suite.page.Keyboard().Press("Shift+Enter"); err != nil {
			t.Fatalf("Failed to press Shift+Enter: %v", err)
		}

		// In real implementation, these would trigger appropriate actions
		t.Log("Keyboard shortcuts would be tested with actual event handling")
	})

	t.Run("screen_reader_support", func(t *testing.T) {
		// Form should have descriptive fieldset/legend
		fieldset := suite.page.Locator("fieldset")
		if fieldset != nil {
			legend := fieldset.Locator("legend")
			legendText, _ := legend.TextContent()
			if legendText != "Compose Message" {
				t.Errorf("Expected fieldset legend 'Compose Message', got '%s'", legendText)
			}
		}

		// Instructions should be available
		instructions := suite.page.Locator("[data-testid='input-instructions']")
		suite.AssertElementVisible(t, "[data-testid='input-instructions']")

		instructionText, _ := instructions.TextContent()
		expectedInstructions := "Type your message and press Enter to send, or Shift+Enter for a new line"
		if instructionText != expectedInstructions {
			t.Errorf("Expected instructions '%s', got '%s'", expectedInstructions, instructionText)
		}

		// Instructions should be referenced by textarea
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		ariaDescribedBy, _ := messageTextarea.GetAttribute("aria-describedby")
		if !contains(ariaDescribedBy, "input-instructions") {
			t.Error("Textarea should reference instructions via aria-describedby")
		}
	})

	t.Run("high_contrast_support", func(t *testing.T) {
		// Check that elements have sufficient visual contrast
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaClass, _ := messageTextarea.GetAttribute("class")

		// Should have proper border and background for contrast
		if !contains(textareaClass, "border-") || !contains(textareaClass, "bg-") {
			t.Error("Textarea should have explicit border and background colors")
		}

		// Error state should be visually distinct
		if props.InputState.ValidationError != "" {
			if !contains(textareaClass, "border-red") {
				t.Error("Textarea should have red border in error state for high contrast")
			}
		}

		// Buttons should have sufficient contrast
		sendButton := suite.page.Locator("[data-testid='send-button']")
		buttonClass, _ := sendButton.GetAttribute("class")

		if !contains(buttonClass, "bg-") || !contains(buttonClass, "text-") {
			t.Error("Send button should have explicit background and text colors")
		}

		// Disabled state should be visually distinct
		if props.InputState.IsSending || !props.IsConnected {
			if !contains(buttonClass, "opacity-") && !contains(buttonClass, "bg-gray") {
				t.Error("Disabled button should have distinct visual styling")
			}
		}
	})
}

func TestMessageInput_HTMXIntegration(t *testing.T) {
	props := MessageInputTestProps{
		RoomID:    "room_htmx",
		CSRFToken: "csrf_htmx_token",
		InputState: MessageInputStateTest{
			Content:        "Testing HTMX integration",
			CharacterCount: 25,
			IsSending:      false,
		},
		MaxLength:   1000,
		IsConnected: true,
		CurrentUser: UserContextTest{
			ID:              "user_htmx",
			Username:        "htmxuser",
			IsAuthenticated: true,
		},
	}

	handler := MockMessageInputHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("form_submission_htmx", func(t *testing.T) {
		// These will fail as mock doesn't include HTMX attributes
		messageForm := suite.page.Locator("[data-testid='message-input-form']")

		// Check HTMX form submission
		hxPost, _ := messageForm.GetAttribute("hx-post")
		expectedURL := fmt.Sprintf("/api/rooms/%s/messages", props.RoomID)
		if hxPost != expectedURL {
			t.Errorf("Expected form hx-post '%s', got '%s'", expectedURL, hxPost)
		}

		// Check HTMX target - messages should be appended to message list
		hxTarget, _ := messageForm.GetAttribute("hx-target")
		if hxTarget != "#message-list" {
			t.Errorf("Expected form hx-target '#message-list', got '%s'", hxTarget)
		}

		// Check HTMX swap strategy
		hxSwap, _ := messageForm.GetAttribute("hx-swap")
		if hxSwap != "beforeend" {
			t.Errorf("Expected form hx-swap 'beforeend', got '%s'", hxSwap)
		}

		// Check form reset after successful submission
		hxOnSuccess, _ := messageForm.GetAttribute("hx-on::after-request")
		if !contains(hxOnSuccess, "this.reset()") {
			t.Error("Form should reset after successful message submission")
		}

		// Check HTMX headers for CSRF
		hxHeaders, _ := messageForm.GetAttribute("hx-headers")
		expectedHeaders := fmt.Sprintf(`{"X-CSRF-Token": "%s"}`, props.CSRFToken)
		if hxHeaders != expectedHeaders {
			t.Errorf("Expected hx-headers '%s', got '%s'", expectedHeaders, hxHeaders)
		}
	})

	t.Run("character_count_updates", func(t *testing.T) {
		// Character counter should update via HTMX as user types
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")

		// Check HTMX trigger for character count updates
		hxTrigger, _ := messageTextarea.GetAttribute("hx-trigger")
		if hxTrigger != "input delay:100ms" {
			t.Errorf("Expected textarea hx-trigger 'input delay:100ms', got '%s'", hxTrigger)
		}

		// Should update character counter
		hxPost, _ := messageTextarea.GetAttribute("hx-post")
		expectedCounterURL := "/components/character-counter"
		if hxPost != expectedCounterURL {
			t.Errorf("Expected textarea hx-post '%s', got '%s'", expectedCounterURL, hxPost)
		}

		hxTarget, _ := messageTextarea.GetAttribute("hx-target")
		if hxTarget != "#character-counter" {
			t.Errorf("Expected textarea hx-target '#character-counter', got '%s'", hxTarget)
		}

		// Include current content in request
		hxInclude, _ := messageTextarea.GetAttribute("hx-include")
		if hxInclude != "this" {
			t.Error("Character count update should include textarea content")
		}
	})

	t.Run("typing_indicators", func(t *testing.T) {
		// Typing indicators via HTMX
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")

		// Should send typing notifications
		hxTriggerTyping, _ := messageTextarea.GetAttribute("hx-trigger")
		expectedTriggers := "input delay:100ms, keydown delay:1s"
		if hxTriggerTyping != expectedTriggers && !contains(hxTriggerTyping, "keydown") {
			t.Error("Textarea should trigger typing notifications")
		}

		// Separate endpoint for typing notifications
		hxPost, _ := messageTextarea.GetAttribute("hx-post")
		typingURL := fmt.Sprintf("/api/rooms/%s/typing", props.RoomID)
		if hxPost != typingURL && !contains(hxPost, "/typing") {
			t.Error("Should have separate endpoint for typing notifications")
		}
	})

	t.Run("validation_htmx", func(t *testing.T) {
		// Real-time validation via HTMX
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")

		// Validation on blur or after delay
		hxTrigger, _ := messageTextarea.GetAttribute("hx-trigger")
		if !contains(hxTrigger, "blur") {
			t.Error("Textarea should validate on blur")
		}

		// Validation endpoint
		hxPost, _ := messageTextarea.GetAttribute("hx-post")
		validationURL := "/api/messages/validate"
		if hxPost != validationURL && !contains(hxPost, "/validate") {
			t.Error("Should have validation endpoint")
		}

		// Target error display
		hxTarget, _ := messageTextarea.GetAttribute("hx-target")
		if hxTarget != "#message-input-error" && !contains(hxTarget, "error") {
			t.Error("Validation should target error display element")
		}
	})

	t.Run("file_upload_htmx", func(t *testing.T) {
		// File upload via HTMX
		fileInput := suite.page.Locator("input[type='file'][data-testid='file-input']")

		// Should trigger on file selection
		hxTrigger, _ := fileInput.GetAttribute("hx-trigger")
		if hxTrigger != "change" {
			t.Error("File input should trigger HTMX on change")
		}

		// File upload endpoint
		hxPost, _ := fileInput.GetAttribute("hx-post")
		uploadURL := fmt.Sprintf("/api/rooms/%s/files", props.RoomID)
		if hxPost != uploadURL {
			t.Errorf("Expected file upload hx-post '%s', got '%s'", uploadURL, hxPost)
		}

		// Should show upload progress
		hxTarget, _ := fileInput.GetAttribute("hx-target")
		if hxTarget != "#file-upload-status" {
			t.Error("File upload should target status display")
		}

		// File upload should include form data
		hxEncoding, _ := fileInput.GetAttribute("hx-encoding")
		if hxEncoding != "multipart/form-data" {
			t.Error("File upload should use multipart/form-data encoding")
		}
	})

	t.Run("emoji_picker_htmx", func(t *testing.T) {
		// Emoji picker via HTMX
		emojiButton := suite.page.Locator("[data-testid='emoji-picker-button']")

		// Should load emoji picker component
		hxGet, _ := emojiButton.GetAttribute("hx-get")
		if hxGet != "/components/emoji-picker" {
			t.Errorf("Expected emoji button hx-get '/components/emoji-picker', got '%s'", hxGet)
		}

		// Target emoji picker container
		hxTarget, _ := emojiButton.GetAttribute("hx-target")
		if hxTarget != "#emoji-picker-container" {
			t.Error("Emoji picker should target picker container")
		}

		// Swap strategy for modal-like behavior
		hxSwap, _ := emojiButton.GetAttribute("hx-swap")
		if hxSwap != "innerHTML" {
			t.Error("Emoji picker should use innerHTML swap")
		}
	})

	t.Run("loading_states", func(t *testing.T) {
		// Loading indicators during HTMX requests
		loadingIndicator := suite.page.Locator("[data-testid='message-sending-indicator']")
		suite.AssertElementVisible(t, "[data-testid='message-sending-indicator']")

		indicatorClass, _ := loadingIndicator.GetAttribute("class")
		if !contains(indicatorClass, "htmx-indicator") {
			t.Error("Loading indicator should have htmx-indicator class")
		}

		// Send button should show loading state
		sendButton := suite.page.Locator("[data-testid='send-button']")
		hxIndicator, _ := sendButton.GetAttribute("hx-indicator")
		if hxIndicator != "#message-sending-indicator" {
			t.Error("Send button should reference loading indicator")
		}

		// Form should disable during submission
		messageForm := suite.page.Locator("[data-testid='message-input-form']")
		hxDisable, _ := messageForm.GetAttribute("hx-disable")
		if hxDisable == "" {
			t.Error("Form should disable inputs during submission")
		}
	})

	t.Run("error_handling", function(t *testing.T) {
		// HTMX error handling for form submission
		messageForm := suite.page.Locator("[data-testid='message-input-form']")

		// Error handling attributes
		hxOnError, _ := messageForm.GetAttribute("hx-on::response-error")
		if hxOnError == "" {
			t.Error("Form should have HTMX error handling")
		}

		// Error display target
		errorDisplay := suite.page.Locator("[data-testid='message-error-display']")
		suite.AssertElementVisible(t, "[data-testid='message-error-display']")

		// Should be initially hidden
		errorClass, _ := errorDisplay.GetAttribute("class")
		if !contains(errorClass, "hidden") {
			t.Error("Error display should be initially hidden")
		}

		// Connection error handling
		hxOnConnectionError, _ := messageForm.GetAttribute("hx-on::connection-error")
		if hxOnConnectionError == "" {
			t.Error("Form should handle connection errors")
		}

		// Timeout handling
		hxTimeout, _ := messageForm.GetAttribute("hx-timeout")
		if hxTimeout == "" {
			t.Error("Form should have request timeout")
		}
	})

	t.Run("progressive_enhancement", func(t *testing.T) {
		// Form should work without JavaScript
		messageForm := suite.page.Locator("[data-testid='message-input-form']")

		action, _ := messageForm.GetAttribute("action")
		method, _ := messageForm.GetAttribute("method")

		// Should have proper fallback action and method
		expectedAction := fmt.Sprintf("/rooms/%s/messages", props.RoomID)
		if action != expectedAction {
			t.Errorf("Expected fallback action '%s', got '%s'", expectedAction, action)
		}

		if method != "POST" {
			t.Errorf("Expected fallback method 'POST', got '%s'", method)
		}

		// Form should include all necessary fields for fallback
		roomIdInput := suite.page.Locator("input[name='room_id']")
		suite.AssertElementVisible(t, "input[name='room_id']")

		csrfInput := suite.page.Locator("input[name='csrf_token']")
		suite.AssertElementVisible(t, "input[name='csrf_token']")

		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		name, _ := messageTextarea.GetAttribute("name")
		if name != "content" {
			t.Error("Textarea should have name attribute for fallback submission")
		}
	})
}

func TestMessageInput_TailwindStyling(t *testing.T) {
	props := MessageInputTestProps{
		RoomID:    "room_styling",
		CSRFToken: "csrf_styling_token",
		InputState: MessageInputStateTest{
			Content:         "Testing Tailwind CSS styling",
			CharacterCount:  30,
			IsSending:       false,
			ValidationError: "Test validation error",
		},
		MaxLength:   1000,
		IsConnected: true,
		CurrentUser: UserContextTest{
			ID:              "user_styling",
			Username:        "stylinguser",
			IsAuthenticated: true,
		},
	}

	handler := MockMessageInputHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("container_styling", func(t *testing.T) {
		// These will fail as mock doesn't apply Tailwind classes
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		expectedContainerClasses := []string{
			"bg-white",
			"border-t",
			"border-gray-200",
			"p-4",
			"shadow-lg",
			"sm:p-6",
			"md:relative",
			"fixed",
			"bottom-0",
			"left-0",
			"right-0",
			"z-10",
		}

		for _, expectedClass := range expectedContainerClasses {
			if !contains(containerClass, expectedClass) {
				t.Errorf("Input container missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("form_styling", func(t *testing.T) {
		messageForm := suite.page.Locator("[data-testid='message-input-form']")
		formClass, _ := messageForm.GetAttribute("class")

		expectedFormClasses := []string{
			"flex",
			"flex-col",
			"space-y-3",
			"max-w-4xl",
			"mx-auto",
		}

		for _, expectedClass := range expectedFormClasses {
			if !contains(formClass, expectedClass) {
				t.Errorf("Form missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("textarea_styling", func(t *testing.T) {
		messageTextarea := suite.page.Locator("[data-testid='message-textarea']")
		textareaClass, _ := messageTextarea.GetAttribute("class")

		expectedTextareaClasses := []string{
			"flex-1",
			"border",
			"border-gray-300",
			"rounded-lg",
			"px-4",
			"py-3",
			"text-sm",
			"placeholder-gray-500",
			"focus:outline-none",
			"focus:ring-2",
			"focus:ring-blue-500",
			"focus:border-blue-500",
			"resize-none",
			"transition-colors",
			"duration-200",
		}

		for _, expectedClass := range expectedTextareaClasses {
			if !contains(textareaClass, expectedClass) {
				t.Errorf("Textarea missing Tailwind class: %s", expectedClass)
			}
		}

		// Error state styling
		if props.InputState.ValidationError != "" {
			if !contains(textareaClass, "border-red-500") || !contains(textareaClass, "focus:ring-red-500") {
				t.Error("Textarea should have red styling when validation error exists")
			}
		}

		// Disabled state styling
		if !props.IsConnected || props.InputState.IsSending {
			if !contains(textareaClass, "bg-gray-50") || !contains(textareaClass, "cursor-not-allowed") {
				t.Error("Textarea should have disabled styling when not available")
			}
		}
	})

	t.Run("button_group_styling", func(t *testing.T) {
		buttonGroup := suite.page.Locator("[data-testid='input-button-group']")
		groupClass, _ := buttonGroup.GetAttribute("class")

		expectedGroupClasses := []string{
			"flex",
			"items-center",
			"justify-between",
			"mt-3",
		}

		for _, expectedClass := range expectedGroupClasses {
			if !contains(groupClass, expectedClass) {
				t.Errorf("Button group missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("action_buttons_styling", func(t *testing.T) {
		// Emoji picker button
		emojiButton := suite.page.Locator("[data-testid='emoji-picker-button']")
		emojiClass, _ := emojiButton.GetAttribute("class")

		expectedActionButtonClasses := []string{
			"p-2",
			"text-gray-400",
			"hover:text-gray-600",
			"hover:bg-gray-100",
			"rounded-md",
			"transition-colors",
			"duration-200",
			"focus:outline-none",
			"focus:ring-2",
			"focus:ring-blue-500",
		}

		for _, expectedClass := range expectedActionButtonClasses {
			if !contains(emojiClass, expectedClass) {
				t.Errorf("Emoji button missing Tailwind class: %s", expectedClass)
			}
		}

		// File upload button
		fileButton := suite.page.Locator("[data-testid='file-upload-button']")
		fileClass, _ := fileButton.GetAttribute("class")

		for _, expectedClass := range expectedActionButtonClasses {
			if !contains(fileClass, expectedClass) {
				t.Errorf("File button missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("send_button_styling", func(t *testing.T) {
		sendButton := suite.page.Locator("[data-testid='send-button']")
		buttonClass, _ := sendButton.GetAttribute("class")

		expectedSendButtonClasses := []string{
			"px-4",
			"py-2",
			"ml-3",
			"bg-blue-600",
			"hover:bg-blue-700",
			"disabled:bg-blue-300",
			"text-white",
			"text-sm",
			"font-medium",
			"rounded-md",
			"transition-colors",
			"duration-200",
			"focus:outline-none",
			"focus:ring-2",
			"focus:ring-blue-500",
			"focus:ring-offset-2",
			"disabled:cursor-not-allowed",
		}

		for _, expectedClass := range expectedSendButtonClasses {
			if !contains(buttonClass, expectedClass) {
				t.Errorf("Send button missing Tailwind class: %s", expectedClass)
			}
		}

		// Loading state styling
		if props.InputState.IsSending {
			if !contains(buttonClass, "opacity-75") {
				t.Error("Send button should have loading opacity when sending")
			}
		}
	})

	t.Run("character_counter_styling", function(t *testing.T) {
		charCounter := suite.page.Locator("[data-testid='character-counter']")
		counterClass, _ := charCounter.GetAttribute("class")

		expectedCounterClasses := []string{
			"text-xs",
			"font-medium",
			"select-none",
		}

		for _, expectedClass := range expectedCounterClasses {
			if !contains(counterClass, expectedClass) {
				t.Errorf("Character counter missing Tailwind class: %s", expectedClass)
			}
		}

		// Color based on character count
		percentage := float64(props.InputState.CharacterCount) / float64(props.MaxLength)
		if percentage > 1.0 {
			if !contains(counterClass, "text-red-600") {
				t.Error("Character counter should be red when over limit")
			}
		} else if percentage >= 0.9 {
			if !contains(counterClass, "text-amber-600") {
				t.Error("Character counter should be amber when near limit")
			}
		} else {
			if !contains(counterClass, "text-gray-500") {
				t.Error("Character counter should be gray in normal range")
			}
		}
	})

	t.Run("validation_error_styling", function(t *testing.T) {
		if props.InputState.ValidationError != "" {
			errorElement := suite.page.Locator("[data-testid='message-input-error']")
			errorClass, _ := errorElement.GetAttribute("class")

			expectedErrorClasses := []string{
				"mt-2",
				"text-sm",
				"text-red-600",
				"flex",
				"items-center",
			}

			for _, expectedClass := range expectedErrorClasses {
				if !contains(errorClass, expectedClass) {
					t.Errorf("Validation error missing Tailwind class: %s", expectedClass)
				}
			}
		}
	})

	t.Run("connection_indicator_styling", func(t *testing.T) {
		connectionIndicator := suite.page.Locator("[data-testid='connection-indicator']")
		indicatorClass, _ := connectionIndicator.GetAttribute("class")

		expectedIndicatorClasses := []string{
			"flex",
			"items-center",
			"text-xs",
			"font-medium",
		}

		for _, expectedClass := range expectedIndicatorClasses {
			if !contains(indicatorClass, expectedClass) {
				t.Errorf("Connection indicator missing Tailwind class: %s", expectedClass)
			}
		}

		// Color based on connection status
		if props.IsConnected {
			if !contains(indicatorClass, "text-green-600") {
				t.Error("Connection indicator should be green when connected")
			}
		} else {
			if !contains(indicatorClass, "text-red-600") {
				t.Error("Connection indicator should be red when disconnected")
			}
		}
	})

	t.Run("keyboard_shortcuts_styling", func(t *testing.T) {
		shortcutHint := suite.page.Locator("[data-testid='keyboard-shortcuts']")
		hintClass, _ := shortcutHint.GetAttribute("class")

		expectedHintClasses := []string{
			"text-xs",
			"text-gray-400",
			"hidden",
			"sm:block",
		}

		for _, expectedClass := range expectedHintClasses {
			if !contains(hintClass, expectedClass) {
				t.Errorf("Keyboard shortcuts hint missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("loading_indicator_styling", func(t *testing.T) {
		loadingIndicator := suite.page.Locator("[data-testid='message-sending-indicator']")
		loadingClass, _ := loadingIndicator.GetAttribute("class")

		expectedLoadingClasses := []string{
			"htmx-indicator",
			"flex",
			"items-center",
			"text-sm",
			"text-gray-600",
			"opacity-0",
			"transition-opacity",
			"duration-200",
		}

		for _, expectedClass := range expectedLoadingClasses {
			if !contains(loadingClass, expectedClass) {
				t.Errorf("Loading indicator missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("responsive_adjustments", function(t *testing.T) {
		// Check responsive classes on different elements
		inputContainer := suite.page.Locator("[data-testid='message-input-container']")
		containerClass, _ := inputContainer.GetAttribute("class")

		expectedResponsiveClasses := []string{
			"sm:p-6",          // Larger padding on small screens
			"md:relative",     // Relative positioning on medium+ screens
			"lg:rounded-t-lg", // Rounded corners on large screens
		}

		for _, expectedClass := range expectedResponsiveClasses {
			if !contains(containerClass, expectedClass) {
				t.Errorf("Input container missing responsive class: %s", expectedClass)
			}
		}

		// Button group responsive layout
		buttonGroup := suite.page.Locator("[data-testid='input-button-group']")
		groupClass, _ := buttonGroup.GetAttribute("class")

		if !contains(groupClass, "sm:flex-row") || !contains(groupClass, "sm:items-center") {
			t.Error("Button group should have responsive flex layout classes")
		}
	})
}