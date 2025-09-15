package ui

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// RoomListTestProps represents mock data for RoomList component
type RoomListTestProps struct {
	Rooms           []RoomViewModelTest
	CurrentUser     UserContextTest
	CreateRoomState CreateRoomStateTest
}

type RoomViewModelTest struct {
	ID                  string
	Name                string
	Description         string
	ParticipantCount    int
	LastActivity        time.Time
	IsPrivate           bool
	HasUnreadMessages   bool
	LastMessage         string
}

type CreateRoomStateTest struct {
	RoomName          string
	Description       string
	IsPrivate         bool
	IsCreating        bool
	ValidationErrors  map[string]string
}

// MockRoomListHandler creates a failing mock handler for RoomList component
// This implements TDD RED phase - tests will fail initially
func MockRoomListHandler(props RoomListTestProps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Intentionally broken/minimal HTML that will cause tests to fail (TDD RED phase)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>DuckChat - Rooms</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
    <div>
        <h1>Chat Rooms</h1>
        <div>
            <div>General</div>
            <div>5 online</div>
        </div>
        <button>+</button>
    </div>
</body>
</html>`))
	})
}

func TestRoomList_Rendering(t *testing.T) {
	testCases := []struct {
		name  string
		props RoomListTestProps
	}{
		{
			name: "multiple rooms with unread messages",
			props: RoomListTestProps{
				Rooms: []RoomViewModelTest{
					{
						ID:               "room_general",
						Name:             "General Discussion",
						Description:      "Main chat room for general topics",
						ParticipantCount: 12,
						LastActivity:     time.Now().Add(-5 * time.Minute),
						IsPrivate:        false,
						HasUnreadMessages: true,
						LastMessage:      "Hey everyone! How's it going?",
					},
					{
						ID:               "room_dev",
						Name:             "Development",
						Description:      "Technical discussions and code reviews",
						ParticipantCount: 8,
						LastActivity:     time.Now().Add(-1 * time.Hour),
						IsPrivate:        false,
						HasUnreadMessages: false,
						LastMessage:      "Fixed the bug in user authentication",
					},
					{
						ID:               "room_private",
						Name:             "Project Alpha",
						Description:      "Private project discussions",
						ParticipantCount: 3,
						LastActivity:     time.Now().Add(-30 * time.Minute),
						IsPrivate:        true,
						HasUnreadMessages: true,
						LastMessage:      "Meeting at 3 PM",
					},
				},
				CurrentUser: UserContextTest{
					ID:              "user_123",
					Username:        "johndoe",
					IsAuthenticated: true,
				},
				CreateRoomState: CreateRoomStateTest{
					IsCreating: false,
				},
			},
		},
		{
			name: "empty room list",
			props: RoomListTestProps{
				Rooms: []RoomViewModelTest{},
				CurrentUser: UserContextTest{
					ID:              "user_456",
					Username:        "newuser",
					IsAuthenticated: true,
				},
				CreateRoomState: CreateRoomStateTest{
					IsCreating: false,
				},
			},
		},
		{
			name: "creating new room",
			props: RoomListTestProps{
				Rooms: []RoomViewModelTest{
					{
						ID:               "room_general",
						Name:             "General",
						ParticipantCount: 5,
						LastActivity:     time.Now(),
						IsPrivate:        false,
						HasUnreadMessages: false,
					},
				},
				CurrentUser: UserContextTest{
					ID:              "user_789",
					Username:        "admin",
					IsAuthenticated: true,
				},
				CreateRoomState: CreateRoomStateTest{
					RoomName:    "New Project Room",
					Description: "Room for the new project",
					IsPrivate:   false,
					IsCreating:  true,
				},
			},
		},
		{
			name: "room creation with validation errors",
			props: RoomListTestProps{
				Rooms: []RoomViewModelTest{
					{
						ID:               "room_existing",
						Name:             "Existing Room",
						ParticipantCount: 2,
						LastActivity:     time.Now(),
						IsPrivate:        false,
						HasUnreadMessages: false,
					},
				},
				CurrentUser: UserContextTest{
					ID:              "user_abc",
					Username:        "creator",
					IsAuthenticated: true,
				},
				CreateRoomState: CreateRoomStateTest{
					RoomName:   "Existing Room",
					IsCreating: false,
					ValidationErrors: map[string]string{
						"room_name": "Room name already exists",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := MockRoomListHandler(tc.props)
			suite := SetupBrowserTest(t, handler)
			defer suite.TeardownBrowserTest()

			if err := suite.NavigateToPath("/"); err != nil {
				t.Fatalf("Failed to navigate: %v", err)
			}

			t.Run("room_list_structure", func(t *testing.T) {
				// These will fail as mock doesn't have proper structure
				roomList := suite.page.Locator("[data-testid='room-list']")
				suite.AssertElementVisible(t, "[data-testid='room-list']")

				// Check room list header
				listHeader := suite.page.Locator("[data-testid='room-list-header']")
				suite.AssertElementVisible(t, "[data-testid='room-list-header']")

				headerText, _ := listHeader.TextContent()
				if headerText != "Chat Rooms" {
					t.Errorf("Expected header 'Chat Rooms', got '%s'", headerText)
				}
			})

			t.Run("individual_rooms", func(t *testing.T) {
				for i, room := range tc.props.Rooms {
					roomSelector := fmt.Sprintf("[data-testid='room-item-%s']", room.ID)
					roomElement := suite.page.Locator(roomSelector)
					suite.AssertElementVisible(t, roomSelector)

					// Check room name
					nameSelector := fmt.Sprintf("[data-testid='room-name-%s']", room.ID)
					nameElement := suite.page.Locator(nameSelector)
					suite.AssertElementVisible(t, nameSelector)

					nameText, _ := nameElement.TextContent()
					if nameText != room.Name {
						t.Errorf("Room %d name: expected '%s', got '%s'", i, room.Name, nameText)
					}

					// Check participant count
					countSelector := fmt.Sprintf("[data-testid='participant-count-%s']", room.ID)
					countElement := suite.page.Locator(countSelector)
					suite.AssertElementVisible(t, countSelector)

					countText, _ := countElement.TextContent()
					expectedCount := fmt.Sprintf("%d online", room.ParticipantCount)
					if countText != expectedCount {
						t.Errorf("Room %d participant count: expected '%s', got '%s'", i, expectedCount, countText)
					}

					// Check room description if present
					if room.Description != "" {
						descSelector := fmt.Sprintf("[data-testid='room-description-%s']", room.ID)
						descElement := suite.page.Locator(descSelector)
						suite.AssertElementVisible(t, descSelector)

						descText, _ := descElement.TextContent()
						if descText != room.Description {
							t.Errorf("Room %d description: expected '%s', got '%s'", i, room.Description, descText)
						}
					}

					// Check last message if present
					if room.LastMessage != "" {
						msgSelector := fmt.Sprintf("[data-testid='last-message-%s']", room.ID)
						msgElement := suite.page.Locator(msgSelector)
						suite.AssertElementVisible(t, msgSelector)

						msgText, _ := msgElement.TextContent()
						if msgText != room.LastMessage {
							t.Errorf("Room %d last message: expected '%s', got '%s'", i, room.LastMessage, msgText)
						}
					}

					// Check unread indicator
					if room.HasUnreadMessages {
						unreadSelector := fmt.Sprintf("[data-testid='unread-indicator-%s']", room.ID)
						unreadElement := suite.page.Locator(unreadSelector)
						suite.AssertElementVisible(t, unreadSelector)
					}

					// Check private room indicator
					if room.IsPrivate {
						privateSelector := fmt.Sprintf("[data-testid='private-indicator-%s']", room.ID)
						privateElement := suite.page.Locator(privateSelector)
						suite.AssertElementVisible(t, privateSelector)
					}
				}
			})

			t.Run("empty_state", func(t *testing.T) {
				if len(tc.props.Rooms) == 0 {
					// Should show empty state message
					emptyState := suite.page.Locator("[data-testid='empty-room-list']")
					suite.AssertElementVisible(t, "[data-testid='empty-room-list']")

					emptyText, _ := emptyState.TextContent()
					expectedText := "No chat rooms yet. Create your first room!"
					if emptyText != expectedText {
						t.Errorf("Expected empty state text '%s', got '%s'", expectedText, emptyText)
					}
				} else {
					// Should not show empty state when rooms exist
					suite.AssertElementNotVisible(t, "[data-testid='empty-room-list']")
				}
			})

			t.Run("create_room_button", func(t *testing.T) {
				// These will fail as mock doesn't have proper create button
				createButton := suite.page.Locator("[data-testid='create-room-button']")
				suite.AssertElementVisible(t, "[data-testid='create-room-button']")

				// Check button attributes
				buttonType, _ := createButton.GetAttribute("type")
				if buttonType != "button" {
					t.Errorf("Expected create button type 'button', got '%s'", buttonType)
				}

				// Check button text or icon
				buttonText, _ := createButton.TextContent()
				if buttonText != "+" && buttonText != "Create Room" {
					t.Errorf("Expected create button text '+' or 'Create Room', got '%s'", buttonText)
				}
			})

			t.Run("create_room_form", func(t *testing.T) {
				if tc.props.CreateRoomState.IsCreating || len(tc.props.CreateRoomState.ValidationErrors) > 0 {
					// Should show create room form
					createForm := suite.page.Locator("[data-testid='create-room-form']")
					suite.AssertElementVisible(t, "[data-testid='create-room-form']")

					// Check form inputs
					nameInput := suite.page.Locator("[data-testid='room-name-input']")
					suite.AssertElementVisible(t, "[data-testid='room-name-input']")

					descInput := suite.page.Locator("[data-testid='room-description-input']")
					suite.AssertElementVisible(t, "[data-testid='room-description-input']")

					privateToggle := suite.page.Locator("[data-testid='private-room-toggle']")
					suite.AssertElementVisible(t, "[data-testid='private-room-toggle']")

					// Check form values
					nameValue, _ := nameInput.InputValue()
					if nameValue != tc.props.CreateRoomState.RoomName {
						t.Errorf("Expected room name input '%s', got '%s'", tc.props.CreateRoomState.RoomName, nameValue)
					}

					descValue, _ := descInput.InputValue()
					if descValue != tc.props.CreateRoomState.Description {
						t.Errorf("Expected room description input '%s', got '%s'", tc.props.CreateRoomState.Description, descValue)
					}

					// Check private toggle state
					isChecked, _ := privateToggle.IsChecked()
					if isChecked != tc.props.CreateRoomState.IsPrivate {
						t.Errorf("Expected private toggle %v, got %v", tc.props.CreateRoomState.IsPrivate, isChecked)
					}

					// Check submit button state
					submitButton := suite.page.Locator("[data-testid='create-room-submit']")
					suite.AssertElementVisible(t, "[data-testid='create-room-submit']")

					if tc.props.CreateRoomState.IsCreating {
						disabled, _ := submitButton.GetAttribute("disabled")
						if disabled == "" {
							t.Error("Submit button should be disabled when creating room")
						}

						buttonText, _ := submitButton.TextContent()
						if buttonText != "Creating..." {
							t.Errorf("Expected creating button text 'Creating...', got '%s'", buttonText)
						}
					}

					// Check validation errors
					for field, message := range tc.props.CreateRoomState.ValidationErrors {
						errorSelector := "[data-testid='" + field + "-error']"
						errorElement := suite.page.Locator(errorSelector)
						suite.AssertElementVisible(t, errorSelector)

						errorText, _ := errorElement.TextContent()
						if errorText != message {
							t.Errorf("Expected %s validation error '%s', got '%s'", field, message, errorText)
						}
					}
				} else {
					// Should not show create form when not creating
					suite.AssertElementNotVisible(t, "[data-testid='create-room-form']")
				}
			})

			t.Run("room_sorting", func(t *testing.T) {
				if len(tc.props.Rooms) > 1 {
					// Check that rooms are sorted by last activity (most recent first)
					roomElements := suite.page.Locator("[data-testid^='room-item-']")
					count, _ := roomElements.Count()

					if count != len(tc.props.Rooms) {
						t.Errorf("Expected %d room elements, got %d", len(tc.props.Rooms), count)
					}

					// In a real implementation, we would verify the order matches
					// the sorted order of rooms by LastActivity
					t.Log("Room sorting verification would check actual DOM order")
				}
			})
		})
	}
}

func TestRoomList_ResponsiveDesign(t *testing.T) {
	props := RoomListTestProps{
		Rooms: []RoomViewModelTest{
			{
				ID:               "room_test1",
				Name:             "Test Room 1",
				Description:      "First test room with a longer description",
				ParticipantCount: 15,
				LastActivity:     time.Now(),
				IsPrivate:        false,
				HasUnreadMessages: true,
				LastMessage:      "This is a test message that might be quite long",
			},
			{
				ID:               "room_test2",
				Name:             "Test Room 2",
				ParticipantCount: 3,
				LastActivity:     time.Now().Add(-1 * time.Hour),
				IsPrivate:        true,
				HasUnreadMessages: false,
			},
		},
		CurrentUser: UserContextTest{
			ID:              "user_responsive",
			Username:        "testuser",
			IsAuthenticated: true,
		},
	}

	handler := MockRoomListHandler(props)
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
		roomList := suite.page.Locator("[data-testid='room-list']")
		roomListClass, _ := roomList.GetAttribute("class")

		// Check mobile-specific classes
		expectedMobileClasses := []string{"w-full", "px-4", "py-2"}
		for _, class := range expectedMobileClasses {
			if !contains(roomListClass, class) {
				t.Errorf("Room list missing mobile class: %s", class)
			}
		}

		// Check that room items stack vertically on mobile
		roomItems := suite.page.Locator("[data-testid^='room-item-']")
		firstRoom := roomItems.First()
		firstRoomClass, _ := firstRoom.GetAttribute("class")

		if !contains(firstRoomClass, "block") || !contains(firstRoomClass, "w-full") {
			t.Error("Room items should be full width blocks on mobile")
		}

		// Check that create button is appropriately sized for mobile
		createButton := suite.page.Locator("[data-testid='create-room-button']")
		createButtonClass, _ := createButton.GetAttribute("class")

		if !contains(createButtonClass, "w-full") && !contains(createButtonClass, "fixed") {
			t.Error("Create button should be full width or fixed positioned on mobile")
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
		roomList := suite.page.Locator("[data-testid='room-list']")
		roomListClass, _ := roomList.GetAttribute("class")

		// Should use grid layout on tablet
		if !contains(roomListClass, "grid") || !contains(roomListClass, "md:grid-cols-2") {
			t.Error("Room list should use grid layout on tablet")
		}
	})

	t.Run("desktop_layout", func(t *testing.T) {
		if err := suite.SetDesktopViewport(); err != nil {
			t.Fatalf("Failed to set desktop viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Check desktop-specific layout
		roomList := suite.page.Locator("[data-testid='room-list']")
		roomListClass, _ := roomList.GetAttribute("class")

		// Should use more columns on desktop
		if !contains(roomListClass, "lg:grid-cols-3") {
			t.Error("Room list should use 3 columns on desktop")
		}

		// Check that sidebar or additional panels are visible on desktop
		sidebar := suite.page.Locator("[data-testid='room-sidebar']")
		suite.AssertElementVisible(t, "[data-testid='room-sidebar']")
	})

	t.Run("text_truncation", func(t *testing.T) {
		// Test text truncation on smaller screens
		if err := suite.SetMobileViewport(); err != nil {
			t.Fatalf("Failed to set mobile viewport: %v", err)
		}

		if err := suite.NavigateToPath("/"); err != nil {
			t.Fatalf("Failed to navigate: %v", err)
		}

		// Long descriptions should be truncated
		for _, room := range props.Rooms {
			if room.Description != "" {
				descSelector := fmt.Sprintf("[data-testid='room-description-%s']", room.ID)
				descElement := suite.page.Locator(descSelector)
				descClass, _ := descElement.GetAttribute("class")

				if !contains(descClass, "truncate") && !contains(descClass, "line-clamp-2") {
					t.Errorf("Room %s description should be truncated on mobile", room.ID)
				}
			}

			// Long last messages should also be truncated
			if room.LastMessage != "" {
				msgSelector := fmt.Sprintf("[data-testid='last-message-%s']", room.ID)
				msgElement := suite.page.Locator(msgSelector)
				msgClass, _ := msgElement.GetAttribute("class")

				if !contains(msgClass, "truncate") {
					t.Errorf("Room %s last message should be truncated", room.ID)
				}
			}
		}
	})
}

func TestRoomList_Accessibility(t *testing.T) {
	props := RoomListTestProps{
		Rooms: []RoomViewModelTest{
			{
				ID:               "room_a11y",
				Name:             "Accessibility Test Room",
				Description:      "Room for testing accessibility features",
				ParticipantCount: 7,
				LastActivity:     time.Now(),
				IsPrivate:        false,
				HasUnreadMessages: true,
				LastMessage:      "Testing accessibility",
			},
		},
		CurrentUser: UserContextTest{
			ID:              "user_a11y",
			Username:        "a11yuser",
			IsAuthenticated: true,
		},
	}

	handler := MockRoomListHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("semantic_html", func(t *testing.T) {
		// These will fail as mock doesn't use semantic HTML
		roomList := suite.page.Locator("[data-testid='room-list']")
		listRole, _ := roomList.GetAttribute("role")
		if listRole != "list" && !roomList.Locator("ul").Count() != 0 {
			t.Error("Room list should use semantic list structure or role='list'")
		}

		// Check that room items have proper list item structure
		roomItems := suite.page.Locator("[data-testid^='room-item-']")
		firstRoom := roomItems.First()
		itemRole, _ := firstRoom.GetAttribute("role")
		if itemRole != "listitem" && firstRoom.Locator("li").Count() == 0 {
			t.Error("Room items should use semantic list item structure or role='listitem'")
		}
	})

	t.Run("aria_labels", func(t *testing.T) {
		// Check ARIA labels for screen readers
		roomList := suite.page.Locator("[data-testid='room-list']")
		ariaLabel, _ := roomList.GetAttribute("aria-label")
		if ariaLabel != "List of chat rooms" {
			t.Errorf("Expected room list aria-label 'List of chat rooms', got '%s'", ariaLabel)
		}

		// Check individual room accessibility
		for _, room := range props.Rooms {
			roomSelector := fmt.Sprintf("[data-testid='room-item-%s']", room.ID)
			roomElement := suite.page.Locator(roomSelector)

			ariaLabel, _ := roomElement.GetAttribute("aria-label")
			expectedLabel := fmt.Sprintf("Chat room %s, %d participants", room.Name, room.ParticipantCount)
			if room.HasUnreadMessages {
				expectedLabel += ", has unread messages"
			}
			if room.IsPrivate {
				expectedLabel += ", private room"
			}

			if ariaLabel != expectedLabel {
				t.Errorf("Expected room aria-label '%s', got '%s'", expectedLabel, ariaLabel)
			}
		}
	})

	t.Run("keyboard_navigation", func(t *testing.T) {
		// Check that room items are keyboard accessible
		roomItems := suite.page.Locator("[data-testid^='room-item-']")
		count, _ := roomItems.Count()

		for i := 0; i < count; i++ {
			roomItem := roomItems.Nth(i)
			tabIndex, _ := roomItem.GetAttribute("tabindex")
			href, _ := roomItem.GetAttribute("href")

			// Should be focusable (either link or tabindex)
			if href == "" && tabIndex == "" {
				t.Errorf("Room item %d should be keyboard accessible", i)
			}

			// Test focus styles
			if err := roomItem.Focus(); err == nil {
				// Check that focus styles are applied
				roomClass, _ := roomItem.GetAttribute("class")
				if !contains(roomClass, "focus:ring") && !contains(roomClass, "focus:outline") {
					t.Errorf("Room item %d should have focus styles", i)
				}
			}
		}

		// Test create room button accessibility
		createButton := suite.page.Locator("[data-testid='create-room-button']")
		createAriaLabel, _ := createButton.GetAttribute("aria-label")
		if createAriaLabel != "Create new chat room" {
			t.Errorf("Expected create button aria-label 'Create new chat room', got '%s'", createAriaLabel)
		}
	})

	t.Run("screen_reader_announcements", func(t *testing.T) {
		// Check live regions for dynamic updates
		if len(props.Rooms) > 0 {
			room := props.Rooms[0]
			if room.HasUnreadMessages {
				unreadSelector := fmt.Sprintf("[data-testid='unread-indicator-%s']", room.ID)
				unreadElement := suite.page.Locator(unreadSelector)

				ariaLive, _ := unreadElement.GetAttribute("aria-live")
				if ariaLive != "polite" {
					t.Error("Unread message indicator should have aria-live='polite' for screen reader announcements")
				}
			}
		}

		// Check status updates during room creation
		if props.CreateRoomState.IsCreating {
			statusRegion := suite.page.Locator("[data-testid='create-room-status']")
			suite.AssertElementVisible(t, "[data-testid='create-room-status']")

			ariaLive, _ := statusRegion.GetAttribute("aria-live")
			if ariaLive != "assertive" {
				t.Error("Room creation status should have aria-live='assertive'")
			}
		}
	})

	t.Run("color_contrast", func(t *testing.T) {
		// Basic color contrast check for unread indicators
		for _, room := range props.Rooms {
			if room.HasUnreadMessages {
				unreadSelector := fmt.Sprintf("[data-testid='unread-indicator-%s']", room.ID)
				unreadElement := suite.page.Locator(unreadSelector)

				// In a real implementation, we would:
				// 1. Get computed styles for background and text colors
				// 2. Calculate contrast ratio
				// 3. Ensure it meets WCAG guidelines
				t.Log("Color contrast would be verified with actual color calculations")

				// Check that indicator is visually distinct
				indicatorClass, _ := unreadElement.GetAttribute("class")
				if !contains(indicatorClass, "bg-") || !contains(indicatorClass, "text-") {
					t.Errorf("Unread indicator for room %s should have background and text color classes", room.ID)
				}
			}
		}
	})
}

func TestRoomList_HTMXIntegration(t *testing.T) {
	props := RoomListTestProps{
		Rooms: []RoomViewModelTest{
			{
				ID:               "room_htmx",
				Name:             "HTMX Test Room",
				ParticipantCount: 5,
				LastActivity:     time.Now(),
				IsPrivate:        false,
				HasUnreadMessages: false,
			},
		},
		CurrentUser: UserContextTest{
			ID:              "user_htmx",
			Username:        "htmxuser",
			IsAuthenticated: true,
		},
	}

	handler := MockRoomListHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("room_navigation_htmx", func(t *testing.T) {
		// These will fail as mock doesn't include HTMX attributes
		for _, room := range props.Rooms {
			roomSelector := fmt.Sprintf("[data-testid='room-item-%s']", room.ID)
			roomElement := suite.page.Locator(roomSelector)

			// Check HTMX navigation
			hxGet, _ := roomElement.GetAttribute("hx-get")
			expectedURL := fmt.Sprintf("/rooms/%s", room.ID)
			if hxGet != expectedURL {
				t.Errorf("Expected room hx-get '%s', got '%s'", expectedURL, hxGet)
			}

			// Check HTMX target
			hxTarget, _ := roomElement.GetAttribute("hx-target")
			if hxTarget != "#main-content" {
				t.Errorf("Expected room hx-target '#main-content', got '%s'", hxTarget)
			}

			// Check HTMX push URL
			hxPushUrl, _ := roomElement.GetAttribute("hx-push-url")
			if hxPushUrl != "true" {
				t.Error("Room navigation should push URL to history")
			}
		}
	})

	t.Run("create_room_htmx", func(t *testing.T) {
		// Check create room button HTMX behavior
		createButton := suite.page.Locator("[data-testid='create-room-button']")

		hxGet, _ := createButton.GetAttribute("hx-get")
		if hxGet != "/components/create-room-form" {
			t.Errorf("Expected create button hx-get '/components/create-room-form', got '%s'", hxGet)
		}

		hxTarget, _ := createButton.GetAttribute("hx-target")
		if hxTarget != "#create-room-modal" {
			t.Errorf("Expected create button hx-target '#create-room-modal', got '%s'", hxTarget)
		}

		hxSwap, _ := createButton.GetAttribute("hx-swap")
		if hxSwap != "innerHTML" {
			t.Errorf("Expected create button hx-swap 'innerHTML', got '%s'", hxSwap)
		}
	})

	t.Run("real_time_updates", func(t *testing.T) {
		// Check SSE integration for real-time room updates
		roomList := suite.page.Locator("[data-testid='room-list']")

		hxSSE, _ := roomList.GetAttribute("hx-sse")
		if hxSSE != "connect:/events" {
			t.Errorf("Expected room list hx-sse 'connect:/events', got '%s'", hxSSE)
		}

		// Check for SSE event listeners
		hxTrigger, _ := roomList.GetAttribute("hx-trigger")
		expectedTriggers := "sse:room-created,sse:room-updated,sse:user-joined,sse:user-left"
		if hxTrigger != expectedTriggers {
			t.Errorf("Expected room list hx-trigger '%s', got '%s'", expectedTriggers, hxTrigger)
		}
	})

	t.Run("loading_indicators", func(t *testing.T) {
		// Check HTMX loading indicators
		loadingIndicator := suite.page.Locator("[data-testid='room-list-loading']")
		suite.AssertElementVisible(t, "[data-testid='room-list-loading']")

		indicatorClass, _ := loadingIndicator.GetAttribute("class")
		if !contains(indicatorClass, "htmx-indicator") {
			t.Error("Loading indicator should have htmx-indicator class")
		}

		// Check that indicator is initially hidden
		if !contains(indicatorClass, "hidden") && !contains(indicatorClass, "opacity-0") {
			t.Error("Loading indicator should be initially hidden")
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Check HTMX error handling
		roomList := suite.page.Locator("[data-testid='room-list']")

		// Should have error handling attributes
		hxOnError, _ := roomList.GetAttribute("hx-on::response-error")
		if hxOnError == "" {
			t.Error("Room list should have HTMX error handling")
		}

		// Check error display element
		errorDisplay := suite.page.Locator("[data-testid='room-list-error']")
		suite.AssertElementVisible(t, "[data-testid='room-list-error']")

		// Should be initially hidden
		errorClass, _ := errorDisplay.GetAttribute("class")
		if !contains(errorClass, "hidden") {
			t.Error("Error display should be initially hidden")
		}
	})
}

func TestRoomList_TailwindStyling(t *testing.T) {
	props := RoomListTestProps{
		Rooms: []RoomViewModelTest{
			{
				ID:               "room_styling",
				Name:             "Styling Test Room",
				Description:      "Testing Tailwind CSS styling",
				ParticipantCount: 10,
				LastActivity:     time.Now(),
				IsPrivate:        false,
				HasUnreadMessages: true,
				LastMessage:      "Testing styles",
			},
		},
		CurrentUser: UserContextTest{
			ID:              "user_styling",
			Username:        "stylinguser",
			IsAuthenticated: true,
		},
	}

	handler := MockRoomListHandler(props)
	suite := SetupBrowserTest(t, handler)
	defer suite.TeardownBrowserTest()

	if err := suite.NavigateToPath("/"); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Run("container_styling", func(t *testing.T) {
		// These will fail as mock doesn't apply Tailwind classes
		roomList := suite.page.Locator("[data-testid='room-list']")
		containerClass, _ := roomList.GetAttribute("class")

		expectedClasses := []string{
			"grid",
			"gap-4",
			"p-4",
			"md:grid-cols-2",
			"lg:grid-cols-3",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(containerClass, expectedClass) {
				t.Errorf("Room list container missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("room_item_styling", func(t *testing.T) {
		room := props.Rooms[0]
		roomSelector := fmt.Sprintf("[data-testid='room-item-%s']", room.ID)
		roomElement := suite.page.Locator(roomSelector)
		roomClass, _ := roomElement.GetAttribute("class")

		expectedClasses := []string{
			"bg-white",
			"rounded-lg",
			"shadow-md",
			"p-6",
			"hover:shadow-lg",
			"transition-shadow",
			"duration-200",
			"cursor-pointer",
		}

		for _, expectedClass := range expectedClasses {
			if !contains(roomClass, expectedClass) {
				t.Errorf("Room item missing Tailwind class: %s", expectedClass)
			}
		}

		// Check unread message styling
		if room.HasUnreadMessages {
			if !contains(roomClass, "border-l-4") || !contains(roomClass, "border-blue-500") {
				t.Error("Room with unread messages should have blue left border")
			}
		}
	})

	t.Run("typography_styling", func(t *testing.T) {
		room := props.Rooms[0]

		// Room name styling
		nameSelector := fmt.Sprintf("[data-testid='room-name-%s']", room.ID)
		nameElement := suite.page.Locator(nameSelector)
		nameClass, _ := nameElement.GetAttribute("class")

		expectedNameClasses := []string{
			"text-lg",
			"font-semibold",
			"text-gray-900",
			"mb-2",
		}

		for _, expectedClass := range expectedNameClasses {
			if !contains(nameClass, expectedClass) {
				t.Errorf("Room name missing Tailwind class: %s", expectedClass)
			}
		}

		// Description styling
		if room.Description != "" {
			descSelector := fmt.Sprintf("[data-testid='room-description-%s']", room.ID)
			descElement := suite.page.Locator(descSelector)
			descClass, _ := descElement.GetAttribute("class")

			expectedDescClasses := []string{
				"text-sm",
				"text-gray-600",
				"mb-3",
				"line-clamp-2",
			}

			for _, expectedClass := range expectedDescClasses {
				if !contains(descClass, expectedClass) {
					t.Errorf("Room description missing Tailwind class: %s", expectedClass)
				}
			}
		}

		// Participant count styling
		countSelector := fmt.Sprintf("[data-testid='participant-count-%s']", room.ID)
		countElement := suite.page.Locator(countSelector)
		countClass, _ := countElement.GetAttribute("class")

		expectedCountClasses := []string{
			"text-xs",
			"text-gray-500",
			"flex",
			"items-center",
		}

		for _, expectedClass := range expectedCountClasses {
			if !contains(countClass, expectedClass) {
				t.Errorf("Participant count missing Tailwind class: %s", expectedClass)
			}
		}
	})

	t.Run("indicator_styling", func(t *testing.T) {
		room := props.Rooms[0]

		// Unread indicator styling
		if room.HasUnreadMessages {
			unreadSelector := fmt.Sprintf("[data-testid='unread-indicator-%s']", room.ID)
			unreadElement := suite.page.Locator(unreadSelector)
			unreadClass, _ := unreadElement.GetAttribute("class")

			expectedUnreadClasses := []string{
				"w-3",
				"h-3",
				"bg-blue-500",
				"rounded-full",
				"absolute",
				"top-2",
				"right-2",
			}

			for _, expectedClass := range expectedUnreadClasses {
				if !contains(unreadClass, expectedClass) {
					t.Errorf("Unread indicator missing Tailwind class: %s", expectedClass)
				}
			}
		}

		// Private indicator styling
		if room.IsPrivate {
			privateSelector := fmt.Sprintf("[data-testid='private-indicator-%s']", room.ID)
			privateElement := suite.page.Locator(privateSelector)
			privateClass, _ := privateElement.GetAttribute("class")

			expectedPrivateClasses := []string{
				"text-xs",
				"bg-yellow-100",
				"text-yellow-800",
				"px-2",
				"py-1",
				"rounded-full",
				"inline-flex",
				"items-center",
			}

			for _, expectedClass := range expectedPrivateClasses {
				if !contains(privateClass, expectedClass) {
					t.Errorf("Private indicator missing Tailwind class: %s", expectedClass)
				}
			}
		}
	})

	t.Run("create_button_styling", func(t *testing.T) {
		createButton := suite.page.Locator("[data-testid='create-room-button']")
		buttonClass, _ := createButton.GetAttribute("class")

		expectedButtonClasses := []string{
			"fixed",
			"bottom-6",
			"right-6",
			"w-14",
			"h-14",
			"bg-blue-600",
			"hover:bg-blue-700",
			"text-white",
			"rounded-full",
			"shadow-lg",
			"flex",
			"items-center",
			"justify-center",
			"text-2xl",
			"transition-colors",
			"duration-200",
		}

		for _, expectedClass := range expectedButtonClasses {
			if !contains(buttonClass, expectedClass) {
				t.Errorf("Create button missing Tailwind class: %s", expectedClass)
			}
		}
	})
}