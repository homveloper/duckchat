package ui

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// ChatPageTestProps represents mock data for ChatPage component
type ChatPageTestProps struct {
	CurrentRoom       ChatRoomContextTest
	CurrentUser       UserContextTest
	Messages          []MessageViewModelTest
	OnlineUsers       []UserContextTest
	MessageInputState MessageInputStateTest
	IsConnected       bool
	UnreadCount       int
}

type ChatRoomContextTest struct {
	ID               string
	Name             string
	Description      string
	IsPrivate        bool
	ParticipantCount int
}

type MessageViewModelTest struct {
	ID            string
	Content       string
	Username      string
	Timestamp     time.Time
	IsCurrentUser bool
	AvatarURL     string
	MessageType   string
}

type MessageInputStateTest struct {
	Content         string
	CharacterCount  int
	IsSending       bool
	ValidationError string
}

// MockChatPageHandler creates a failing mock handler for ChatPage component
// This implements TDD RED phase - tests will fail initially
func MockChatPageHandler(props ChatPageTestProps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Intentionally broken/minimal HTML that will cause tests to fail (TDD RED phase)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>DuckChat - Chat</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
    <div>
        <h1>Chat Room</h1>
        <div>
            <div>User: Hello there</div>
        </div>
        <div>
            <input type="text" placeholder="Type a message...">
            <button>Send</button>
        </div>
    </div>
</body>
</html>`))
	})
}

func TestChatPage_Rendering(t *testing.T) {
	testCases := []struct {
		name  string
		props ChatPageTestProps
	}{
		{
			name: "active chat with messages",
			props: ChatPageTestProps{
				CurrentRoom: ChatRoomContextTest{
					ID:               "room_general",
					Name:             "General Discussion",
					Description:      "Main chat room for general topics",
					IsPrivate:        false,
					ParticipantCount: 15,
				},
				CurrentUser: UserContextTest{
					ID:              "user_123",
					Username:        "johndoe",
					IsAuthenticated: true,
				},
				Messages: []MessageViewModelTest{
					{
						ID:            "msg_1",
						Content:       "Hello everyone! 👋",
						Username:      "alice",
						Timestamp:     time.Now().Add(-5 * time.Minute),
						IsCurrentUser: false,
						AvatarURL:     "/avatars/alice.jpg",
						MessageType:   "text",
					},
					{
						ID:            "msg_2",
						Content:       "Hey Alice! How's it going?",
						Username:      "johndoe",
						Timestamp:     time.Now().Add(-3 * time.Minute),
						IsCurrentUser: true,
						AvatarURL:     "/avatars/johndoe.jpg",
						MessageType:   "text",
					},
					{
						ID:            "msg_3",
						Content:       "alice joined the room",
						Username:      "system",
						Timestamp:     time.Now().Add(-10 * time.Minute),
						IsCurrentUser: false,
						MessageType:   "system",
					},
				},
				OnlineUsers: []UserContextTest{
					{ID: "user_alice", Username: "alice", IsAuthenticated: true},
					{ID: "user_123", Username: "johndoe", IsAuthenticated: true},
					{ID: "user_bob", Username: "bob", IsAuthenticated: true},
				},
				MessageInputState: MessageInputStateTest{
					Content:        "",
					CharacterCount: 0,
					IsSending:      false,
				},
				IsConnected: true,
				UnreadCount: 0,
			},
		},
		{
			name: "empty chat room",
			props: ChatPageTestProps{
				CurrentRoom: ChatRoomContextTest{
					ID:               "room_empty",
					Name:             "Empty Room",
					IsPrivate:        false,
					ParticipantCount: 1,
				},
				CurrentUser: UserContextTest{
					ID:              "user_456",
					Username:        "newuser",
					IsAuthenticated: true,
				},
				Messages:    []MessageViewModelTest{},
				OnlineUsers: []UserContextTest{{ID: "user_456", Username: "newuser", IsAuthenticated: true}},
				MessageInputState: MessageInputStateTest{
					Content:        "",
					CharacterCount: 0,
					IsSending:      false,
				},
				IsConnected: true,
				UnreadCount: 0,
			},
		},
		{
			name: "disconnected state",
			props: ChatPageTestProps{
				CurrentRoom: ChatRoomContextTest{
					ID:               "room_disconnected",
					Name:             "Test Room",
					IsPrivate:        false,
					ParticipantCount: 5,
				},
				CurrentUser: UserContextTest{
					ID:              "user_789",
					Username:        "disconnecteduser",
					IsAuthenticated: true,
				},
				Messages: []MessageViewModelTest{
					{
						ID:            "msg_old",
						Content:       "Last message before disconnect",
						Username:      "someuser",
						Timestamp:     time.Now().Add(-1 * time.Hour),
						IsCurrentUser: false,
						MessageType:   "text",
					},
				},
				OnlineUsers: []UserContextTest{},
				MessageInputState: MessageInputStateTest{
					Content:        "Trying to send...",
					CharacterCount: 17,
					IsSending:      false,
				},
				IsConnected: false,
				UnreadCount: 3,
			},
		},
		{
			name: "sending message state",
			props: ChatPageTestProps{
				CurrentRoom: ChatRoomContextTest{
					ID:               "room_sending",
					Name:             "Sending Test Room",
					IsPrivate:        false,
					ParticipantCount: 8,
				},
				CurrentUser: UserContextTest{
					ID:              "user_sending",
					Username:        "sendinguser",
					IsAuthenticated: true,
				},
				Messages: []MessageViewModelTest{
					{
						ID:            "msg_recent",
						Content:       "Previous message",
						Username:      "otheruser",
						Timestamp:     time.Now().Add(-2 * time.Minute),
						IsCurrentUser: false,
						MessageType:   "text",
					},
				},
				OnlineUsers: []UserContextTest{
					{ID: "user_sending", Username: "sendinguser", IsAuthenticated: true},
					{ID: "user_other", Username: "otheruser", IsAuthenticated: true},
				},
				MessageInputState: MessageInputStateTest{
					Content:        "This message is being sent right now",
					CharacterCount: 36,
					IsSending:      true,
				},
				IsConnected: true,
				UnreadCount: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := MockChatPageHandler(tc.props)
			suite := SetupBrowserTest(t, handler)
			defer suite.TeardownBrowserTest()

			if err := suite.NavigateToPath("/"); err != nil {
				t.Fatalf("Failed to navigate: %v", err)
			}

			t.Run("chat_header", func(t *testing.T) {
				// These will fail as mock doesn't have proper header structure
				chatHeader := suite.page.Locator("[data-testid='chat-header']")
				suite.AssertElementVisible(t, "[data-testid='chat-header']")

				// Check room name
				roomName := suite.page.Locator("[data-testid='room-name']")
				suite.AssertElementVisible(t, "[data-testid='room-name']")

				nameText, _ := roomName.TextContent()
				if nameText != tc.props.CurrentRoom.Name {
					t.Errorf("Expected room name '%s', got '%s'", tc.props.CurrentRoom.Name, nameText)
				}

				// Check participant count
				participantCount := suite.page.Locator("[data-testid='participant-count']")
				suite.AssertElementVisible(t, "[data-testid='participant-count']")

				countText, _ := participantCount.TextContent()
				expectedCount := fmt.Sprintf("%d participants", tc.props.CurrentRoom.ParticipantCount)
				if countText != expectedCount {
					t.Errorf("Expected participant count '%s', got '%s'", expectedCount, countText)
				}

				// Check room description if present
				if tc.props.CurrentRoom.Description != "" {
					roomDesc := suite.page.Locator("[data-testid='room-description']")
					suite.AssertElementVisible(t, "[data-testid='room-description']")

					descText, _ := roomDesc.TextContent()
					if descText != tc.props.CurrentRoom.Description {
						t.Errorf("Expected room description '%s', got '%s'", tc.props.CurrentRoom.Description, descText)
					}
				}

				// Check connection status
				connectionStatus := suite.page.Locator("[data-testid='connection-status']")
				if tc.props.IsConnected {
					statusClass, _ := connectionStatus.GetAttribute("class")
					if !contains(statusClass, "text-green-500") {
						t.Error("Connected status should show green indicator")
					}
				} else {
					suite.AssertElementVisible(t, "[data-testid='connection-status']")
					statusClass, _ := connectionStatus.GetAttribute("class")
					if !contains(statusClass, "text-red-500") {
						t.Error("Disconnected status should show red indicator")
					}
				}
			})

			t.Run("message_list", func(t *testing.T) {
				messageList := suite.page.Locator("[data-testid='message-list']")
				suite.AssertElementVisible(t, "[data-testid='message-list']")

				if len(tc.props.Messages) == 0 {
					// Should show empty state
					emptyState := suite.page.Locator("[data-testid='empty-message-list']")
					suite.AssertElementVisible(t, "[data-testid='empty-message-list']")

					emptyText, _ := emptyState.TextContent()
					expectedText := "No messages yet. Start the conversation!"
					if emptyText != expectedText {
						t.Errorf("Expected empty state text '%s', got '%s'", expectedText, emptyText)
					}
				} else {
					// Check individual messages
					for i, message := range tc.props.Messages {
						msgSelector := fmt.Sprintf("[data-testid='message-%s']", message.ID)
						msgElement := suite.page.Locator(msgSelector)
						suite.AssertElementVisible(t, msgSelector)

						// Check message content
						contentSelector := fmt.Sprintf("[data-testid='message-content-%s']", message.ID)
						contentElement := suite.page.Locator(contentSelector)
						suite.AssertElementVisible(t, contentSelector)

						contentText, _ := contentElement.TextContent()
						if contentText != message.Content {
							t.Errorf("Message %d content: expected '%s', got '%s'", i, message.Content, contentText)
						}

						// Check username (except for system messages)
						if message.MessageType != "system" {
							usernameSelector := fmt.Sprintf("[data-testid='message-username-%s']", message.ID)
							usernameElement := suite.page.Locator(usernameSelector)
							suite.AssertElementVisible(t, usernameSelector)

							usernameText, _ := usernameElement.TextContent()
							if usernameText != message.Username {
								t.Errorf("Message %d username: expected '%s', got '%s'", i, message.Username, usernameText)
							}
						}

						// Check timestamp
						timestampSelector := fmt.Sprintf("[data-testid='message-timestamp-%s']", message.ID)
						timestampElement := suite.page.Locator(timestampSelector)
						suite.AssertElementVisible(t, timestampSelector)

						// Check message alignment based on current user
						msgClass, _ := msgElement.GetAttribute("class")
						if message.IsCurrentUser {
							if !contains(msgClass, "justify-end") && !contains(msgClass, "ml-auto") {
								t.Errorf("Current user message %d should be right-aligned", i)
							}
						} else {
							if !contains(msgClass, "justify-start") && !contains(msgClass, "mr-auto") {
								t.Errorf("Other user message %d should be left-aligned", i)
							}
						}

						// Check message type styling
						if message.MessageType == "system" {
							if !contains(msgClass, "system-message") && !contains(msgClass, "text-gray-500") {
								t.Errorf("System message %d should have system styling", i)
							}
						}

						// Check avatar if present
						if message.AvatarURL != "" {
							avatarSelector := fmt.Sprintf("[data-testid='message-avatar-%s']", message.ID)
							avatarElement := suite.page.Locator(avatarSelector)
							suite.AssertElementVisible(t, avatarSelector)

							avatarSrc, _ := avatarElement.GetAttribute("src")
							if avatarSrc != message.AvatarURL {
								t.Errorf("Message %d avatar: expected '%s', got '%s'", i, message.AvatarURL, avatarSrc)
							}
						}
					}
				}
			})

			t.Run("message_input", func(t *testing.T) {
				// These will fail as mock doesn't have proper message input
				messageInput := suite.page.Locator("[data-testid='message-input']")
				suite.AssertElementVisible(t, "[data-testid='message-input']")

				// Check input field
				inputField := suite.page.Locator("[data-testid='message-text-input']")
				suite.AssertElementVisible(t, "[data-testid='message-text-input']")

				// Check current value
				inputValue, _ := inputField.InputValue()
				if inputValue != tc.props.MessageInputState.Content {
					t.Errorf("Expected input value '%s', got '%s'", tc.props.MessageInputState.Content, inputValue)
				}

				// Check input state
				if tc.props.IsConnected {
					disabled, _ := inputField.GetAttribute("disabled")
					if disabled != "" {
						t.Error("Message input should be enabled when connected")
					}
				} else {
					disabled, _ := inputField.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Message input should be disabled when disconnected")
					}
				}

				// Check send button
				sendButton := suite.page.Locator("[data-testid='send-button']")
				suite.AssertElementVisible(t, "[data-testid='send-button']")

				if tc.props.MessageInputState.IsSending {
					disabled, _ := sendButton.GetAttribute("disabled")
					if disabled == "" {
						t.Error("Send button should be disabled when sending")
					}

					buttonText, _ := sendButton.TextContent()
					if buttonText != "Sending..." {
						t.Errorf("Expected sending button text 'Sending...', got '%s'", buttonText)
					}
				} else {
					buttonText, _ := sendButton.TextContent()
					if buttonText != "Send" {
						t.Errorf("Expected button text 'Send', got '%s'", buttonText)
					}
				}

				// Check character count
				charCount := suite.page.Locator("[data-testid='character-count']")
				suite.AssertElementVisible(t, "[data-testid='character-count']")

				countText, _ := charCount.TextContent()
				expectedCountText := fmt.Sprintf("%d/1000", tc.props.MessageInputState.CharacterCount)
				if countText != expectedCountText {
					t.Errorf("Expected character count '%s', got '%s'", expectedCountText, countText)
				}

				// Check validation error
				if tc.props.MessageInputState.ValidationError != "" {
					errorElement := suite.page.Locator("[data-testid='message-input-error']")
					suite.AssertElementVisible(t, "[data-testid='message-input-error']")

					errorText, _ := errorElement.TextContent()
					if errorText != tc.props.MessageInputState.ValidationError {
						t.Errorf("Expected validation error '%s', got '%s'", tc.props.MessageInputState.ValidationError, errorText)
					}
				}
			})

			t.Run("online_users_sidebar", func(t *testing.T) {
				// Check online users list
				onlineUsersList := suite.page.Locator("[data-testid='online-users-list']")
				suite.AssertElementVisible(t, "[data-testid='online-users-list']")

				// Check online users count
				usersHeader := suite.page.Locator("[data-testid='online-users-header']")
				suite.AssertElementVisible(t, "[data-testid='online-users-header']")

				headerText, _ := usersHeader.TextContent()
				expectedHeaderText := fmt.Sprintf("Online (%d)", len(tc.props.OnlineUsers))
				if headerText != expectedHeaderText {
					t.Errorf("Expected online users header '%s', got '%s'", expectedHeaderText, headerText)
				}

				// Check individual online users
				for i, user := range tc.props.OnlineUsers {
					userSelector := fmt.Sprintf("[data-testid='online-user-%s']", user.ID)
					userElement := suite.page.Locator(userSelector)
					suite.AssertElementVisible(t, userSelector)

					usernameText, _ := userElement.TextContent()
					if usernameText != user.Username {
						t.Errorf("Online user %d: expected username '%s', got '%s'", i, user.Username, usernameText)
					}

					// Check if current user is highlighted
					if user.ID == tc.props.CurrentUser.ID {
						userClass, _ := userElement.GetAttribute("class")
						if !contains(userClass, "font-semibold") && !contains(userClass, "text-blue-600") {
							t.Errorf("Current user should be highlighted in online users list")
						}
					}
				}
			})

			t.Run("unread_messages_indicator", func(t *testing.T) {
				if tc.props.UnreadCount > 0 {
					// Should show unread count indicator
					unreadIndicator := suite.page.Locator("[data-testid='unread-count-indicator']")
					suite.AssertElementVisible(t, "[data-testid='unread-count-indicator']")

					countText, _ := unreadIndicator.TextContent()
					expectedCountText := fmt.Sprintf("%d unread", tc.props.UnreadCount)
					if countText != expectedCountText {
						t.Errorf("Expected unread count '%s', got '%s'", expectedCountText, countText)
					}
				} else {
					// Should not show unread indicator when no unread messages
					suite.AssertElementNotVisible(t, "[data-testid='unread-count-indicator']")
				}
			})

			t.Run("disconnection_banner", func(t *testing.T) {
				if !tc.props.IsConnected {
					// Should show disconnection banner
					disconnectBanner := suite.page.Locator("[data-testid='disconnect-banner']")
					suite.AssertElementVisible(t, "[data-testid='disconnect-banner']")

					bannerText, _ := disconnectBanner.TextContent()
					expectedBannerText := "Connection lost. Trying to reconnect..."
					if bannerText != expectedBannerText {
						t.Errorf("Expected disconnect banner text '%s', got '%s'", expectedBannerText, bannerText)
					}

					// Check banner styling
					bannerClass, _ := disconnectBanner.GetAttribute("class")
					if !contains(bannerClass, "bg-red-100") || !contains(bannerClass, "text-red-800") {
						t.Error("Disconnect banner should have red warning styling")
					}
				} else {
					// Should not show disconnect banner when connected
					suite.AssertElementNotVisible(t, "[data-testid='disconnect-banner']")
				}
			})
		})
	}
}

func TestChatPage_ResponsiveDesign(t *testing.T) {
	props := ChatPageTestProps{
		CurrentRoom: ChatRoomContextTest{
			ID:               "room_responsive",
			Name:             "Responsive Test Room",
			Description:      "Testing responsive design",
			IsPrivate:        false,
			ParticipantCount: 12,
		},
		CurrentUser: UserContextTest{
			ID:              "user_responsive",
			Username:        "responsiveuser",
			IsAuthenticated: true,
		},
		Messages: []MessageViewModelTest{
			{
				ID:            "msg_responsive1",
				Content:       "This is a test message for responsive design testing",
				Username:      "testuser1",
				Timestamp:     time.Now().Add(-5 * time.Minute),
				IsCurrentUser: false,
				MessageType:   "text",
			},
			{
				ID:            "msg_responsive2",
				Content:       "Another message from current user",
				Username:      "responsiveuser",
				Timestamp:     time.Now().Add(-2 * time.Minute),
				IsCurrentUser: true,
				MessageType:   "text",
			},
		},
		OnlineUsers: []UserContextTest{
			{ID: "user_1", Username: "testuser1", IsAuthenticated: true},
			{ID: "user_responsive", Username: "responsiveuser", IsAuthenticated: true},
			{ID: "user_2", Username: "testuser2", IsAuthenticated: true},
		},
		MessageInputState: MessageInputStateTest{
			Content:        "",
			CharacterCount: 0,
			IsSending:      false,
		},
		IsConnected: true,
		UnreadCount: 0,
	}

	handler := MockChatPageHandler(props)
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
		chatContainer := suite.page.Locator("[data-testid='chat-container']")
		suite.AssertElementVisible(t, "[data-testid='chat-container']")

		containerClass, _ := chatContainer.GetAttribute("class")
		expectedMobileClasses := []string{
			"flex",
			"flex-col",
			"h-screen",
			"w-full",
		}

		for _, class := range expectedMobileClasses {
			if !contains(containerClass, class) {
				t.Errorf("Chat container missing mobile class: %s", class)
			}
		}

		// Online users should be hidden or collapsed on mobile
		onlineUsersSidebar := suite.page.Locator("[data-testid='online-users-sidebar']")
		sidebarClass, _ := onlineUsersSidebar.GetAttribute("class")
		if !contains(sidebarClass, "hidden") && !contains(sidebarClass, "md:block") {
			t.Error("Online users sidebar should be hidden on mobile")
		}

		// Message input should be full width and fixed at bottom
		messageInput := suite.page.Locator("[data-testid='message-input']")
		inputClass, _ := messageInput.GetAttribute("class")
		if !contains(inputClass, "w-full") {
			t.Error("Message input should be full width on mobile")
		}

		// Check touch-friendly button sizes
		sendButton := suite.page.Locator("[data-testid='send-button']")
		buttonBox, _ := sendButton.BoundingBox()
		if buttonBox.Height < 44 {
			t.Errorf("Send button height %f is less than 44px minimum for touch", buttonBox.Height)
		}
	})

	t.Run("tablet_layout", func(t *testing.T) {
		if err := suite.SetTabletViewport(); err != nil {
			t.Fatalf("Failed to set tablet viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Online users sidebar should be visible on tablet
		onlineUsersSidebar := suite.page.Locator("[data-testid='online-users-sidebar']")
		suite.AssertElementVisible(t, "[data-testid='online-users-sidebar']")

		// Check tablet grid layout
		chatContainer := suite.page.Locator("[data-testid='chat-container']")
		containerClass, _ := chatContainer.GetAttribute("class")
		if !contains(containerClass, "md:grid") || !contains(containerClass, "md:grid-cols-4") {
			t.Error("Chat container should use grid layout on tablet")
		}

		// Message area should span appropriate columns
		messageArea := suite.page.Locator("[data-testid='message-area']")
		messageAreaClass, _ := messageArea.GetAttribute("class")
		if !contains(messageAreaClass, "md:col-span-3") {
			t.Error("Message area should span 3 columns on tablet")
		}
	})

	t.Run("desktop_layout", func(t *testing.T) {
		if err := suite.SetDesktopViewport(); err != nil {
			t.Fatalf("Failed to set desktop viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Should show full desktop layout with sidebar
		chatContainer := suite.page.Locator("[data-testid='chat-container']")
		containerClass, _ := chatContainer.GetAttribute("class")
		if !contains(containerClass, "lg:grid-cols-5") {
			t.Error("Chat container should use 5-column grid on desktop")
		}

		// Check that all components are visible
		suite.AssertElementVisible(t, "[data-testid='chat-header']")
		suite.AssertElementVisible(t, "[data-testid='message-list']")
		suite.AssertElementVisible(t, "[data-testid='message-input']")
		suite.AssertElementVisible(t, "[data-testid='online-users-sidebar']")

		// Message input should not be fixed on desktop
		messageInput := suite.page.Locator("[data-testid='message-input']")
		inputClass, _ := messageInput.GetAttribute("class")
		if contains(inputClass, "fixed") {
			t.Error("Message input should not be fixed positioned on desktop")
		}
	})

	t.Run("message_bubbles_responsive", func(t *testing.T) {
		// Test message bubble sizing across viewports
		viewports := []struct {
			name     string
			setFunc  func() error
			maxWidth string
		}{
			{"mobile", suite.SetMobileViewport, "max-w-xs"},
			{"tablet", suite.SetTabletViewport, "max-w-sm"},
			{"desktop", suite.SetDesktopViewport, "max-w-md"},
		}

		for _, viewport := range viewports {
			t.Run(viewport.name, func(t *testing.T) {
				if err := viewport.setFunc(); err != nil {
					t.Fatalf("Failed to set %s viewport: %v", viewport.name, err)
				}

				if err := suite.NavigateToPath("/"); err != nil {
					t.Fatalf("Failed to navigate: %v", err)
				}

				// Check message bubble max width
				for _, message := range props.Messages {
					msgSelector := fmt.Sprintf("[data-testid='message-%s']", message.ID)
					msgElement := suite.page.Locator(msgSelector)
					msgClass, _ := msgElement.GetAttribute("class")

					if !contains(msgClass, viewport.maxWidth) {
						t.Errorf("Message bubble should have %s class on %s", viewport.maxWidth, viewport.name)
					}
				}
			})
		}
	})
}

func TestChatPage_Accessibility(t *testing.T) {
	props := ChatPageTestProps{
		CurrentRoom: ChatRoomContextTest{
			ID:               "room_a11y",
			Name:             "Accessibility Test Room",
			Description:      "Testing accessibility features",
			IsPrivate:        false,
			ParticipantCount: 5,
		},
		CurrentUser: UserContextTest{
			ID:              "user_a11y",
			Username:        "a11yuser",
			IsAuthenticated: true,
		},
		Messages: []MessageViewModelTest{
			{
				ID:            "msg_a11y1",
				Content:       "Testing accessibility",
				Username:      "testuser",
				Timestamp:     time.Now().Add(-5 * time.Minute),
				IsCurrentUser: false,
				MessageType:   "text",
			},
			{
				ID:            "msg_a11y2",
				Content:       "My response message",
				Username:      "a11yuser",
				Timestamp:     time.Now().Add(-2 * time.Minute),
				IsCurrentUser: true,
				MessageType:   "text",
			},
		},
		OnlineUsers: []UserContextTest{
			{ID: "user_test", Username: "testuser", IsAuthenticated: true},
			{ID: "user_a11y", Username: "a11yuser", IsAuthenticated: true},
		},
		MessageInputState: MessageInputStateTest{
			Content:        "",
			CharacterCount: 0,
			IsSending:      false,
		},
		IsConnected: true,
		UnreadCount: 0,
	}

	handler := MockChatPageHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("semantic_html_structure", func(t *testing.T) {
		// These will fail as mock doesn't use semantic HTML
		main := suite.page.Locator("main")
		suite.AssertElementVisible(t, "main")

		header := suite.page.Locator("header")
		suite.AssertElementVisible(t, "header")

		// Message list should have proper list structure
		messageList := suite.page.Locator("[data-testid='message-list']")
		listRole, _ := messageList.GetAttribute("role")
		if listRole != "log" && listRole != "list" {
			t.Error("Message list should have role='log' or role='list' for accessibility")
		}

		// Message items should have listitem role
		messages := suite.page.Locator("[data-testid^='message-']")
		firstMessage := messages.First()
		itemRole, _ := firstMessage.GetAttribute("role")
		if itemRole != "listitem" {
			t.Error("Message items should have role='listitem'")
		}
	})

	t.Run("aria_labels_and_descriptions", func(t *testing.T) {
		// Chat container should have aria-label
		chatContainer := suite.page.Locator("[data-testid='chat-container']")
		ariaLabel, _ := chatContainer.GetAttribute("aria-label")
		expectedLabel := fmt.Sprintf("Chat room %s", props.CurrentRoom.Name)
		if ariaLabel != expectedLabel {
			t.Errorf("Expected chat container aria-label '%s', got '%s'", expectedLabel, ariaLabel)
		}

		// Message input should have proper labeling
		messageInput := suite.page.Locator("[data-testid='message-text-input']")
		ariaLabel, _ = messageInput.GetAttribute("aria-label")
		if ariaLabel != "Type a message" {
			t.Errorf("Expected message input aria-label 'Type a message', got '%s'", ariaLabel)
		}

		// Send button should have descriptive aria-label
		sendButton := suite.page.Locator("[data-testid='send-button']")
		ariaLabel, _ = sendButton.GetAttribute("aria-label")
		if ariaLabel != "Send message" {
			t.Errorf("Expected send button aria-label 'Send message', got '%s'", ariaLabel)
		}

		// Online users list should have proper aria-label
		onlineUsersList := suite.page.Locator("[data-testid='online-users-list']")
		ariaLabel, _ = onlineUsersList.GetAttribute("aria-label")
		if ariaLabel != "Online users" {
			t.Errorf("Expected online users list aria-label 'Online users', got '%s'", ariaLabel)
		}
	})

	t.Run("keyboard_navigation", func(t *testing.T) {
		// Message input should be focusable
		messageInput := suite.page.Locator("[data-testid='message-text-input']")
		if err := messageInput.Focus(); err != nil {
			t.Fatalf("Failed to focus message input: %v", err)
		}

		// Tab should move to send button
		if err := suite.page.Keyboard().Press("Tab"); err != nil {
			t.Fatalf("Failed to press Tab: %v", err)
		}

		focusedElement := suite.page.Locator(":focus")
		sendButton := suite.page.Locator("[data-testid='send-button']")

		// Check if send button is focused (this would work in real implementation)
		t.Log("Keyboard navigation would be tested with actual focus management")

		// Enter key should submit message from input
		if err := messageInput.Focus(); err != nil {
			t.Fatalf("Failed to focus message input: %v", err)
		}

		if err := suite.page.Keyboard().Press("Enter"); err != nil {
			t.Fatalf("Failed to press Enter: %v", err)
		}

		// In real implementation, this would trigger message send
		t.Log("Enter key submission would be tested with actual form submission")
	})

	t.Run("screen_reader_support", func(t *testing.T) {
		// Live region for new messages
		messageList := suite.page.Locator("[data-testid='message-list']")
		ariaLive, _ := messageList.GetAttribute("aria-live")
		if ariaLive != "polite" {
			t.Error("Message list should have aria-live='polite' for screen reader announcements")
		}

		// Connection status should be announced
		connectionStatus := suite.page.Locator("[data-testid='connection-status']")
		ariaLive, _ = connectionStatus.GetAttribute("aria-live")
		if ariaLive != "assertive" {
			t.Error("Connection status should have aria-live='assertive' for immediate announcements")
		}

		// Message timestamps should be accessible
		for _, message := range props.Messages {
			timestampSelector := fmt.Sprintf("[data-testid='message-timestamp-%s']", message.ID)
			timestampElement := suite.page.Locator(timestampSelector)

			ariaLabel, _ := timestampElement.GetAttribute("aria-label")
			if ariaLabel == "" {
				t.Errorf("Message %s timestamp should have descriptive aria-label", message.ID)
			}
		}
	})

	t.Run("message_accessibility", func(t *testing.T) {
		// Each message should have proper structure
		for i, message := range props.Messages {
			msgSelector := fmt.Sprintf("[data-testid='message-%s']", message.ID)
			msgElement := suite.page.Locator(msgSelector)

			// Message should have aria-labelledby pointing to username and content
			ariaLabelledBy, _ := msgElement.GetAttribute("aria-labelledby")
			expectedIds := fmt.Sprintf("message-username-%s message-content-%s", message.ID, message.ID)
			if ariaLabelledBy != expectedIds {
				t.Errorf("Message %d should have aria-labelledby='%s', got '%s'", i, expectedIds, ariaLabelledBy)
			}

			// Username should have proper id
			if message.MessageType != "system" {
				usernameSelector := fmt.Sprintf("[data-testid='message-username-%s']", message.ID)
				usernameElement := suite.page.Locator(usernameSelector)
				usernameId, _ := usernameElement.GetAttribute("id")
				expectedId := fmt.Sprintf("message-username-%s", message.ID)
				if usernameId != expectedId {
					t.Errorf("Message %d username should have id='%s', got '%s'", i, expectedId, usernameId)
				}
			}

			// Content should have proper id
			contentSelector := fmt.Sprintf("[data-testid='message-content-%s']", message.ID)
			contentElement := suite.page.Locator(contentSelector)
			contentId, _ := contentElement.GetAttribute("id")
			expectedContentId := fmt.Sprintf("message-content-%s", message.ID)
			if contentId != expectedContentId {
				t.Errorf("Message %d content should have id='%s', got '%s'", i, expectedContentId, contentId)
			}
		}
	})

	t.Run("high_contrast_support", func(t *testing.T) {
		// Check that messages have sufficient visual distinction
		for _, message := range props.Messages {
			msgSelector := fmt.Sprintf("[data-testid='message-%s']", message.ID)
			msgElement := suite.page.Locator(msgSelector)
			msgClass, _ := msgElement.GetAttribute("class")

			// Current user messages should have distinct styling
			if message.IsCurrentUser {
				if !contains(msgClass, "bg-blue") && !contains(msgClass, "bg-indigo") {
					t.Errorf("Current user message should have background color for distinction")
				}
			} else {
				if !contains(msgClass, "bg-white") && !contains(msgClass, "bg-gray") {
					t.Errorf("Other user message should have background color for distinction")
				}
			}

			// System messages should be visually distinct
			if message.MessageType == "system" {
				if !contains(msgClass, "italic") && !contains(msgClass, "text-gray") {
					t.Error("System messages should have distinct styling")
				}
			}
		}
	})
}

func TestChatPage_HTMXIntegration(t *testing.T) {
	props := ChatPageTestProps{
		CurrentRoom: ChatRoomContextTest{
			ID:               "room_htmx",
			Name:             "HTMX Test Room",
			IsPrivate:        false,
			ParticipantCount: 3,
		},
		CurrentUser: UserContextTest{
			ID:              "user_htmx",
			Username:        "htmxuser",
			IsAuthenticated: true,
		},
		Messages: []MessageViewModelTest{
			{
				ID:            "msg_htmx",
				Content:       "Testing HTMX functionality",
				Username:      "htmxuser",
				Timestamp:     time.Now(),
				IsCurrentUser: true,
				MessageType:   "text",
			},
		},
		OnlineUsers: []UserContextTest{
			{ID: "user_htmx", Username: "htmxuser", IsAuthenticated: true},
		},
		MessageInputState: MessageInputStateTest{
			Content:        "",
			CharacterCount: 0,
			IsSending:      false,
		},
		IsConnected: true,
		UnreadCount: 0,
	}

	handler := MockChatPageHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("message_submission_htmx", func(t *testing.T) {
		// These will fail as mock doesn't include HTMX attributes
		messageForm := suite.page.Locator("[data-testid='message-form']")
		suite.AssertElementVisible(t, "[data-testid='message-form']")

		// Check HTMX form submission
		hxPost, _ := messageForm.GetAttribute("hx-post")
		expectedURL := fmt.Sprintf("/api/rooms/%s/messages", props.CurrentRoom.ID)
		if hxPost != expectedURL {
			t.Errorf("Expected message form hx-post '%s', got '%s'", expectedURL, hxPost)
		}

		// Check HTMX target
		hxTarget, _ := messageForm.GetAttribute("hx-target")
		if hxTarget != "#message-list" {
			t.Errorf("Expected message form hx-target '#message-list', got '%s'", hxTarget)
		}

		// Check HTMX swap strategy
		hxSwap, _ := messageForm.GetAttribute("hx-swap")
		if hxSwap != "beforeend" {
			t.Errorf("Expected message form hx-swap 'beforeend', got '%s'", hxSwap)
		}

		// Check form reset after submission
		hxOnSuccess, _ := messageForm.GetAttribute("hx-on::after-request")
		if hxOnSuccess == "" {
			t.Error("Message form should have after-request handler to reset form")
		}
	})

	t.Run("real_time_updates_sse", func(t *testing.T) {
		// Check SSE connection for real-time messages
		messageList := suite.page.Locator("[data-testid='message-list']")

		hxSSE, _ := messageList.GetAttribute("hx-sse")
		expectedSSE := fmt.Sprintf("connect:/events/rooms/%s", props.CurrentRoom.ID)
		if hxSSE != expectedSSE {
			t.Errorf("Expected message list hx-sse '%s', got '%s'", expectedSSE, hxSSE)
		}

		// Check for message event listeners
		hxTrigger, _ := messageList.GetAttribute("hx-trigger")
		expectedTriggers := "sse:new-message,sse:message-updated,sse:message-deleted"
		if hxTrigger != expectedTriggers {
			t.Errorf("Expected message list hx-trigger '%s', got '%s'", expectedTriggers, hxTrigger)
		}

		// Check online users SSE updates
		onlineUsersList := suite.page.Locator("[data-testid='online-users-list']")
		hxSSE, _ = onlineUsersList.GetAttribute("hx-sse")
		if hxSSE != expectedSSE {
			t.Errorf("Expected online users list hx-sse '%s', got '%s'", expectedSSE, hxSSE)
		}

		usersTrigger, _ := onlineUsersList.GetAttribute("hx-trigger")
		expectedUsersTriggers := "sse:user-joined,sse:user-left,sse:user-updated"
		if usersTrigger != expectedUsersTriggers {
			t.Errorf("Expected users list hx-trigger '%s', got '%s'", expectedUsersTriggers, usersTrigger)
		}
	})

	t.Run("message_actions_htmx", func(t *testing.T) {
		// Check message action buttons (edit, delete, reply)
		for _, message := range props.Messages {
			if message.IsCurrentUser {
				// Edit button
				editSelector := fmt.Sprintf("[data-testid='edit-message-%s']", message.ID)
				editButton := suite.page.Locator(editSelector)
				suite.AssertElementVisible(t, editSelector)

				hxGet, _ := editButton.GetAttribute("hx-get")
				expectedEditURL := fmt.Sprintf("/components/edit-message-form?messageId=%s", message.ID)
				if hxGet != expectedEditURL {
					t.Errorf("Expected edit button hx-get '%s', got '%s'", expectedEditURL, hxGet)
				}

				// Delete button
				deleteSelector := fmt.Sprintf("[data-testid='delete-message-%s']", message.ID)
				deleteButton := suite.page.Locator(deleteSelector)
				suite.AssertElementVisible(t, deleteSelector)

				hxDelete, _ := deleteButton.GetAttribute("hx-delete")
				expectedDeleteURL := fmt.Sprintf("/api/messages/%s", message.ID)
				if hxDelete != expectedDeleteURL {
					t.Errorf("Expected delete button hx-delete '%s', got '%s'", expectedDeleteURL, hxDelete)
				}

				// Delete confirmation
				hxConfirm, _ := deleteButton.GetAttribute("hx-confirm")
				if hxConfirm != "Are you sure you want to delete this message?" {
					t.Error("Delete button should have confirmation dialog")
				}
			}
		}
	})

	t.Run("loading_indicators", func(t *testing.T) {
		// Message sending indicator
		sendingIndicator := suite.page.Locator("[data-testid='message-sending-indicator']")
		suite.AssertElementVisible(t, "[data-testid='message-sending-indicator']")

		indicatorClass, _ := sendingIndicator.GetAttribute("class")
		if !contains(indicatorClass, "htmx-indicator") {
			t.Error("Message sending indicator should have htmx-indicator class")
		}

		// Messages loading indicator
		messagesLoading := suite.page.Locator("[data-testid='messages-loading']")
		suite.AssertElementVisible(t, "[data-testid='messages-loading']")

		loadingClass, _ := messagesLoading.GetAttribute("class")
		if !contains(loadingClass, "htmx-indicator") {
			t.Error("Messages loading indicator should have htmx-indicator class")
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Check HTMX error handling for message form
		messageForm := suite.page.Locator("[data-testid='message-form']")

		hxOnError, _ := messageForm.GetAttribute("hx-on::response-error")
		if hxOnError == "" {
			t.Error("Message form should have HTMX error handling")
		}

		// Error display element
		errorDisplay := suite.page.Locator("[data-testid='message-error']")
		suite.AssertElementVisible(t, "[data-testid='message-error']")

		errorClass, _ := errorDisplay.GetAttribute("class")
		if !contains(errorClass, "hidden") {
			t.Error("Message error display should be initially hidden")
		}
	})

	t.Run("typing_indicators", func(t *testing.T) {
		// Typing indicator area
		typingIndicator := suite.page.Locator("[data-testid='typing-indicator']")
		suite.AssertElementVisible(t, "[data-testid='typing-indicator']")

		// Check SSE connection for typing events
		hxSSE, _ := typingIndicator.GetAttribute("hx-sse")
		expectedSSE := fmt.Sprintf("connect:/events/rooms/%s", props.CurrentRoom.ID)
		if hxSSE != expectedSSE {
			t.Errorf("Expected typing indicator hx-sse '%s', got '%s'", expectedSSE, hxSSE)
		}

		hxTrigger, _ := typingIndicator.GetAttribute("hx-trigger")
		if hxTrigger != "sse:user-typing,sse:user-stopped-typing" {
			t.Error("Typing indicator should listen for typing events")
		}

		// Message input should trigger typing notifications
		messageInput := suite.page.Locator("[data-testid='message-text-input']")
		inputTrigger, _ := messageInput.GetAttribute("hx-trigger")
		if inputTrigger != "input delay:1s" {
			t.Error("Message input should trigger typing notifications with delay")
		}

		hxPost, _ := messageInput.GetAttribute("hx-post")
		expectedTypingURL := fmt.Sprintf("/api/rooms/%s/typing", props.CurrentRoom.ID)
		if hxPost != expectedTypingURL {
			t.Errorf("Expected message input typing hx-post '%s', got '%s'", expectedTypingURL, hxPost)
		}
	})
}

func TestChatPage_TailwindStyling(t *testing.T) {
	props := ChatPageTestProps{
		CurrentRoom: ChatRoomContextTest{
			ID:               "room_styling",
			Name:             "Styling Test Room",
			IsPrivate:        false,
			ParticipantCount: 8,
		},
		CurrentUser: UserContextTest{
			ID:              "user_styling",
			Username:        "stylinguser",
			IsAuthenticated: true,
		},
		Messages: []MessageViewModelTest{
			{
				ID:            "msg_other",
				Content:       "Message from another user",
				Username:      "otheruser",
				Timestamp:     time.Now().Add(-5 * time.Minute),
				IsCurrentUser: false,
				MessageType:   "text",
			},
			{
				ID:            "msg_current",
				Content:       "My own message",
				Username:      "stylinguser",
				Timestamp:     time.Now().Add(-2 * time.Minute),
				IsCurrentUser: true,
				MessageType:   "text",
			},
			{
				ID:            "msg_system",
				Content:       "User joined the room",
				Username:      "system",
				Timestamp:     time.Now().Add(-10 * time.Minute),
				IsCurrentUser: false,
				MessageType:   "system",
			},
		},
		OnlineUsers: []UserContextTest{
			{ID: "user_other", Username: "otheruser", IsAuthenticated: true},
			{ID: "user_styling", Username: "stylinguser", IsAuthenticated: true},
		},
		MessageInputState: MessageInputStateTest{
			Content:        "Test message content",
			CharacterCount: 20,
			IsSending:      false,
		},
		IsConnected: true,
		UnreadCount: 0,
	}

	handler := MockChatPageHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("layout_styling", func(t *testing.T) {
		// These will fail as mock doesn't apply Tailwind classes
		chatContainer := suite.page.Locator("[data-testid='chat-container']")
		containerClass, _ := chatContainer.GetAttribute("class")

		expectedContainerClasses := []string{
			"flex",
			"h-screen",
			"bg-gray-50",
			"md:grid",
			"md:grid-cols-4",
			"lg:grid-cols-5",
		}

		for _, expectedClass := range expectedContainerClasses {
			if !contains(containerClass, expectedClass) {
				t.Errorf("Chat container missing Tailwind class: %s", expectedClass)
			}
		}

		// Message area styling
		messageArea := suite.page.Locator("[data-testid='message-area']")
		messageAreaClass, _ := messageArea.GetAttribute("class")

		expectedMessageAreaClasses := []string{
			"flex",
			"flex-col",
			"bg-white",
			"md:col-span-3",
			"lg:col-span-4",
		}

		for _, expectedClass := range expectedMessageAreaClasses {
			if !contains(messageAreaClass, expectedClass) {
				t.Errorf("Message area missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("header_styling", func(t *testing.T) {
		chatHeader := suite.page.Locator("[data-testid='chat-header']")
		headerClass, _ := chatHeader.GetAttribute("class")

		expectedHeaderClasses := []string{
			"flex",
			"items-center",
			"justify-between",
			"p-4",
			"border-b",
			"border-gray-200",
			"bg-white",
			"shadow-sm",
		}

		for _, expectedClass := range expectedHeaderClasses {
			if !contains(headerClass, expectedClass) {
				t.Errorf("Chat header missing Tailwind class: %s", expectedClass)
			}
		}

		// Room name styling
		roomName := suite.page.Locator("[data-testid='room-name']")
		nameClass, _ := roomName.GetAttribute("class")

		expectedNameClasses := []string{
			"text-lg",
			"font-semibold",
			"text-gray-900",
		}

		for _, expectedClass := range expectedNameClasses {
			if !contains(nameClass, expectedClass) {
				t.Errorf("Room name missing Tailwind class: %s", expectedClass)
			}
		}

		// Participant count styling
		participantCount := suite.page.Locator("[data-testid='participant-count']")
		countClass, _ := participantCount.GetAttribute("class")

		expectedCountClasses := []string{
			"text-sm",
			"text-gray-500",
		}

		for _, expectedClass := range expectedCountClasses {
			if !contains(countClass, expectedClass) {
				t.Errorf("Participant count missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("message_list_styling", func(t *testing.T) {
		messageList := suite.page.Locator("[data-testid='message-list']")
		listClass, _ := messageList.GetAttribute("class")

		expectedListClasses := []string{
			"flex-1",
			"overflow-y-auto",
			"p-4",
			"space-y-4",
		}

		for _, expectedClass := range expectedListClasses {
			if !contains(listClass, expectedClass) {
				t.Errorf("Message list missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("message_bubble_styling", func(t *testing.T) {
		for _, message := range props.Messages {
			msgSelector := fmt.Sprintf("[data-testid='message-%s']", message.ID)
			msgElement := suite.page.Locator(msgSelector)
			msgClass, _ := msgElement.GetAttribute("class")

			if message.MessageType == "system" {
				// System message styling
				expectedSystemClasses := []string{
					"flex",
					"justify-center",
					"my-2",
				}

				for _, expectedClass := range expectedSystemClasses {
					if !contains(msgClass, expectedClass) {
						t.Errorf("System message missing Tailwind class: %s", expectedClass)
					}
				}

				// System message content styling
				contentSelector := fmt.Sprintf("[data-testid='message-content-%s']", message.ID)
				contentElement := suite.page.Locator(contentSelector)
				contentClass, _ := contentElement.GetAttribute("class")

				expectedContentClasses := []string{
					"text-sm",
					"text-gray-500",
					"italic",
					"bg-gray-100",
					"px-3",
					"py-1",
					"rounded-full",
				}

				for _, expectedClass := range expectedContentClasses {
					if !contains(contentClass, expectedClass) {
						t.Errorf("System message content missing Tailwind class: %s", expectedClass)
					}
				}

			} else if message.IsCurrentUser {
				// Current user message styling
				expectedCurrentUserClasses := []string{
					"flex",
					"justify-end",
					"mb-4",
				}

				for _, expectedClass := range expectedCurrentUserClasses {
					if !contains(msgClass, expectedClass) {
						t.Errorf("Current user message missing Tailwind class: %s", expectedClass)
					}
				}

				// Message bubble styling
				contentSelector := fmt.Sprintf("[data-testid='message-content-%s']", message.ID)
				contentElement := suite.page.Locator(contentSelector)
				contentClass, _ := contentElement.GetAttribute("class")

				expectedBubbleClasses := []string{
					"bg-blue-500",
					"text-white",
					"rounded-lg",
					"px-4",
					"py-2",
					"max-w-xs",
					"md:max-w-md",
					"shadow-sm",
				}

				for _, expectedClass := range expectedBubbleClasses {
					if !contains(contentClass, expectedClass) {
						t.Errorf("Current user message bubble missing Tailwind class: %s", expectedClass)
					}
				}

			} else {
				// Other user message styling
				expectedOtherUserClasses := []string{
					"flex",
					"justify-start",
					"mb-4",
				}

				for _, expectedClass := range expectedOtherUserClasses {
					if !contains(msgClass, expectedClass) {
						t.Errorf("Other user message missing Tailwind class: %s", expectedClass)
					}
				}

				// Message bubble styling
				contentSelector := fmt.Sprintf("[data-testid='message-content-%s']", message.ID)
				contentElement := suite.page.Locator(contentSelector)
				contentClass, _ := contentElement.GetAttribute("class")

				expectedBubbleClasses := []string{
					"bg-white",
					"text-gray-900",
					"rounded-lg",
					"px-4",
					"py-2",
					"max-w-xs",
					"md:max-w-md",
					"shadow-sm",
					"border",
					"border-gray-200",
				}

				for _, expectedClass := range expectedBubbleClasses {
					if !contains(contentClass, expectedClass) {
						t.Errorf("Other user message bubble missing Tailwind class: %s", expectedClass)
					}
				}

				// Username styling
				usernameSelector := fmt.Sprintf("[data-testid='message-username-%s']", message.ID)
				usernameElement := suite.page.Locator(usernameSelector)
				usernameClass, _ := usernameElement.GetAttribute("class")

				expectedUsernameClasses := []string{
					"text-xs",
					"font-medium",
					"text-gray-700",
					"mb-1",
				}

				for _, expectedClass := range expectedUsernameClasses {
					if !contains(usernameClass, expectedClass) {
						t.Errorf("Message username missing Tailwind class: %s", expectedClass)
					}
				}
			}

			// Timestamp styling (for all non-system messages)
			if message.MessageType != "system" {
				timestampSelector := fmt.Sprintf("[data-testid='message-timestamp-%s']", message.ID)
				timestampElement := suite.page.Locator(timestampSelector)
				timestampClass, _ := timestampElement.GetAttribute("class")

				expectedTimestampClasses := []string{
					"text-xs",
					"text-gray-400",
					"mt-1",
				}

				for _, expectedClass := range expectedTimestampClasses {
					if !contains(timestampClass, expectedClass) {
						t.Errorf("Message timestamp missing Tailwind class: %s", expectedClass)
					}
				}
			}
		}
	})

	t.Run("message_input_styling", func(t *testing.T) {
		messageInput := suite.page.Locator("[data-testid='message-input']")
		inputClass, _ := messageInput.GetAttribute("class")

		expectedInputClasses := []string{
			"border-t",
			"border-gray-200",
			"bg-white",
			"p-4",
		}

		for _, expectedClass := range expectedInputClasses {
			if !contains(inputClass, expectedClass) {
				t.Errorf("Message input container missing Tailwind class: %s", expectedClass)
			}
		}

		// Text input styling
		textInput := suite.page.Locator("[data-testid='message-text-input']")
		textInputClass, _ := textInput.GetAttribute("class")

		expectedTextInputClasses := []string{
			"flex-1",
			"border",
			"border-gray-300",
			"rounded-lg",
			"px-4",
			"py-2",
			"focus:outline-none",
			"focus:ring-2",
			"focus:ring-blue-500",
			"focus:border-blue-500",
			"resize-none",
		}

		for _, expectedClass := range expectedTextInputClasses {
			if !contains(textInputClass, expectedClass) {
				t.Errorf("Message text input missing Tailwind class: %s", expectedClass)
			}
		}

		// Send button styling
		sendButton := suite.page.Locator("[data-testid='send-button']")
		buttonClass, _ := sendButton.GetAttribute("class")

		expectedButtonClasses := []string{
			"ml-2",
			"px-4",
			"py-2",
			"bg-blue-500",
			"hover:bg-blue-600",
			"text-white",
			"rounded-lg",
			"transition-colors",
			"duration-200",
			"disabled:opacity-50",
			"disabled:cursor-not-allowed",
		}

		for _, expectedClass := range expectedButtonClasses {
			if !contains(buttonClass, expectedClass) {
				t.Errorf("Send button missing Tailwind class: %s", expectedClass)
			}
		}

		// Character count styling
		charCount := suite.page.Locator("[data-testid='character-count']")
		charCountClass, _ := charCount.GetAttribute("class")

		expectedCharCountClasses := []string{
			"text-xs",
			"text-gray-400",
			"mt-1",
		}

		for _, expectedClass := range expectedCharCountClasses {
			if !contains(charCountClass, expectedClass) {
				t.Errorf("Character count missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("sidebar_styling", func(t *testing.T) {
		onlineUsersSidebar := suite.page.Locator("[data-testid='online-users-sidebar']")
		sidebarClass, _ := onlineUsersSidebar.GetAttribute("class")

		expectedSidebarClasses := []string{
			"bg-gray-100",
			"border-l",
			"border-gray-200",
			"p-4",
			"hidden",
			"md:block",
			"md:col-span-1",
		}

		for _, expectedClass := range expectedSidebarClasses {
			if !contains(sidebarClass, expectedClass) {
				t.Errorf("Online users sidebar missing Tailwind class: %s", expectedClass)
			}
		}

		// Online users header styling
		usersHeader := suite.page.Locator("[data-testid='online-users-header']")
		headerClass, _ := usersHeader.GetAttribute("class")

		expectedHeaderClasses := []string{
			"text-sm",
			"font-semibold",
			"text-gray-700",
			"mb-3",
		}

		for _, expectedClass := range expectedHeaderClasses {
			if !contains(headerClass, expectedClass) {
				t.Errorf("Online users header missing Tailwind class: %s", expectedClass)
			}
		}

		// Individual user styling
		for _, user := range props.OnlineUsers {
			userSelector := fmt.Sprintf("[data-testid='online-user-%s']", user.ID)
			userElement := suite.page.Locator(userSelector)
			userClass, _ := userElement.GetAttribute("class")

			expectedUserClasses := []string{
				"flex",
				"items-center",
				"py-2",
				"text-sm",
			}

			for _, expectedClass := range expectedUserClasses {
				if !contains(userClass, expectedClass) {
					t.Errorf("Online user missing Tailwind class: %s", expectedClass)
				}
			}

			// Current user should be highlighted
			if user.ID == props.CurrentUser.ID {
				if !contains(userClass, "font-semibold") || !contains(userClass, "text-blue-600") {
					t.Error("Current user should be highlighted in online users list")
				}
			}
		}
	})
}
