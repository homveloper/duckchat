# Quickstart: Templ UI Components Testing Guide

## Prerequisites

**System Requirements**:
- Go 1.21+ installed
- Redis server running (for backend integration)
- Modern web browser (Chrome, Firefox, Safari, Edge)
- Terminal/command line access

**Dependencies**:
```bash
# Install Templ CLI
go install github.com/a-h/templ/cmd/templ@latest

# Install Tailwind CSS CLI (for development)
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
chmod +x tailwindcss-linux-x64
sudo mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss
```

**Environment Setup**:
```bash
# Clone and navigate to project
cd duckchat

# Install Go dependencies
go mod tidy

# Generate Templ templates
templ generate

# Build Tailwind CSS
tailwindcss -i web/static/input.css -o web/static/output.css --watch
```

## User Story Validation Tests

### Story 1: Mobile Login Experience
**Goal**: Verify mobile-optimized login form renders correctly

**Steps**:
1. **Start the server**:
   ```bash
   go run cmd/server/main.go
   ```

2. **Test mobile viewport**:
   - Open browser to `http://localhost:8080/login`
   - Set browser developer tools to mobile view (375x812 - iPhone X)
   - **Expected**: Login form fits screen without horizontal scrolling
   - **Expected**: Input fields are appropriately sized for touch
   - **Expected**: Submit button is easily tappable (minimum 44px height)

3. **Test form validation**:
   - Leave username empty, enter password, submit
   - **Expected**: Username validation error displays clearly
   - Enter short username (<3 chars), submit
   - **Expected**: Username length error displays
   - Enter valid credentials
   - **Expected**: Successful login redirects to `/rooms`

4. **Test error states**:
   - Enter invalid credentials, submit
   - **Expected**: Error message displays without breaking layout
   - **Expected**: Form remains usable after error

**Success Criteria**: ✅ All mobile login interactions work smoothly without layout issues

### Story 2: Room Creation Interface
**Goal**: Verify room creation button and interface work on mobile

**Steps**:
1. **Navigate to rooms page** (after login):
   ```
   http://localhost:8080/rooms
   ```

2. **Test create room button**:
   - **Expected**: "Create Room" button visible and accessible
   - **Expected**: Button follows mobile-friendly design (min 44px height)
   - Tap create room button
   - **Expected**: Room creation form appears or modal opens

3. **Test room creation form**:
   - Enter room name: "Test Mobile Room"
   - Optionally add description
   - Submit form
   - **Expected**: New room appears in room list
   - **Expected**: Can navigate to newly created room

4. **Test validation**:
   - Try creating room with empty name
   - **Expected**: Validation error displays
   - Try creating room with duplicate name
   - **Expected**: Conflict error displays

**Success Criteria**: ✅ Room creation works seamlessly on mobile interface

### Story 3: Chat Interface Usability
**Goal**: Verify chat page displays messages and provides input functionality

**Steps**:
1. **Navigate to chat room**:
   - Click on any room from the rooms list
   - **Expected**: Chat interface loads at `/chat/{roomId}`

2. **Test message display**:
   - **Expected**: Message history displays in scrollable container
   - **Expected**: Messages are readable on mobile screen
   - **Expected**: User's own messages align differently from others
   - **Expected**: Timestamps are visible but not intrusive

3. **Test message input**:
   - **Expected**: Message input field fixed at bottom of screen
   - **Expected**: Input field expands appropriately for longer messages
   - Type test message: "Hello from mobile test!"
   - Submit message
   - **Expected**: Message appears in chat immediately
   - **Expected**: Input field clears after submission

4. **Test character counting**:
   - Type message approaching limit (900+ characters)
   - **Expected**: Character count displays
   - **Expected**: Warning appears near limit
   - Try to exceed limit
   - **Expected**: Prevented from submitting over-limit message

**Success Criteria**: ✅ Chat interface provides smooth messaging experience on mobile

### Story 4: Responsive Layout Transition
**Goal**: Verify automatic adaptation from mobile to tablet layout

**Steps**:
1. **Start in mobile view**:
   - Open chat room in mobile viewport (375px width)
   - **Expected**: Mobile layout active (single column, bottom input)

2. **Transition to tablet**:
   - Gradually expand browser width to 768px
   - **Expected**: Layout smoothly transitions to tablet mode
   - **Expected**: Additional space utilized effectively
   - **Expected**: No content overflow or layout breaks

3. **Test tablet-specific features**:
   - **Expected**: Online users list may become visible
   - **Expected**: Message input may show additional features
   - **Expected**: Room list may show more information per room

4. **Test orientation changes** (if on device):
   - Rotate device from portrait to landscape
   - **Expected**: Layout adapts to new dimensions
   - **Expected**: No content cutoff or unusable interface

**Success Criteria**: ✅ Responsive design transitions work without manual refresh

### Story 5: Real-time Message Updates
**Goal**: Verify smooth real-time messaging experience

**Steps**:
1. **Open multiple browser tabs**:
   - Tab 1: Login as User A, join room
   - Tab 2: Login as User B, join same room

2. **Test message delivery**:
   - User A sends message: "Testing real-time sync"
   - **Expected**: Message appears immediately in User B's chat
   - User B replies: "Message received!"
   - **Expected**: Reply appears in User A's chat

3. **Test connection indicators**:
   - **Expected**: Connection status indicator shows "connected"
   - Temporarily disconnect internet
   - **Expected**: Status shows "disconnected" or similar
   - Reconnect internet
   - **Expected**: Status returns to "connected"
   - **Expected**: Any missed messages load automatically

4. **Test typing indicators** (if implemented):
   - User A starts typing
   - **Expected**: User B sees typing indicator
   - User A stops typing without sending
   - **Expected**: Typing indicator disappears

**Success Criteria**: ✅ Real-time features work reliably across multiple clients

## Performance Validation

### Page Load Speed Test
```bash
# Test server response times
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8080/login"
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8080/rooms"
```

**Expected Results**:
- Login page: < 100ms server response
- Rooms page: < 150ms server response
- Chat page: < 200ms server response (including message history)

### Template Rendering Performance
```bash
# Generate templates and measure build time
time templ generate
```

**Expected Results**:
- Template generation: < 5 seconds for all components
- CSS build: < 3 seconds for complete stylesheet
- No template compilation errors

### Mobile Performance Test
1. Open browser dev tools → Performance tab
2. Set CPU throttling to "4x slowdown" (simulate slower mobile)
3. Record performance while navigating: login → rooms → chat
4. **Expected**: Page interactions remain responsive
5. **Expected**: No JavaScript errors (zero JS approach)

## Accessibility Validation

### Keyboard Navigation Test
1. **Navigate entire app using only keyboard**:
   - Tab through all interactive elements
   - **Expected**: All buttons and inputs focusable
   - **Expected**: Focus indicators clearly visible
   - **Expected**: Logical tab order maintained

2. **Test form interactions**:
   - Navigate to login form using tab
   - Fill and submit using keyboard only
   - **Expected**: All actions possible without mouse

### Screen Reader Compatibility
1. **Enable screen reader** (VoiceOver on Mac, NVDA on Windows)
2. **Navigate login form**:
   - **Expected**: Form fields properly labeled
   - **Expected**: Validation errors announced
   - **Expected**: Submit button purpose clear

3. **Navigate chat interface**:
   - **Expected**: Messages read in chronological order
   - **Expected**: Message authors clearly identified
   - **Expected**: Input field purpose announced

## Error Scenario Testing

### Network Failure Handling
1. **Simulate network issues**:
   - Disconnect internet during message send
   - **Expected**: Graceful error message displays
   - **Expected**: Message queued for retry (optional)
   - Reconnect internet
   - **Expected**: Normal functionality resumes

### Server Error Responses
1. **Test various HTTP error codes**:
   - 401 Unauthorized: **Expected** redirect to login
   - 403 Forbidden: **Expected** access denied message
   - 404 Not Found: **Expected** room not found error
   - 500 Server Error: **Expected** generic error page

### Browser Compatibility
1. **Test in multiple browsers**:
   - Chrome (latest)
   - Firefox (latest)
   - Safari (if on Mac)
   - Edge (if on Windows)
   - **Expected**: Consistent behavior across all browsers
   - **Expected**: No browser-specific layout issues

## Completion Checklist

**Basic Functionality**:
- [ ] Login form renders and works on mobile
- [ ] Room list displays and allows room creation
- [ ] Chat interface shows messages and accepts input
- [ ] Responsive design transitions mobile ↔ tablet
- [ ] Real-time messages work between multiple users

**Performance**:
- [ ] Page load times under target thresholds
- [ ] Template generation completes without errors
- [ ] Mobile performance acceptable under throttling

**Accessibility**:
- [ ] Full keyboard navigation possible
- [ ] Screen reader compatibility verified
- [ ] Focus indicators visible and logical

**Error Handling**:
- [ ] Network failures handled gracefully
- [ ] Server errors display appropriate messages
- [ ] Cross-browser compatibility confirmed

**Final Validation**:
- [ ] All user stories pass acceptance criteria
- [ ] No critical bugs discovered during testing
- [ ] Ready for production deployment

## Troubleshooting

**Common Issues**:

1. **Templates not generating**:
   ```bash
   # Check Templ installation
   templ version
   # Reinstall if needed
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. **CSS not updating**:
   ```bash
   # Rebuild Tailwind CSS
   tailwindcss -i web/static/input.css -o web/static/output.css --watch
   ```

3. **Server won't start**:
   ```bash
   # Check Redis connection
   redis-cli ping
   # Verify Go dependencies
   go mod tidy && go build
   ```

4. **HTMX not working**:
   - Check browser console for JavaScript errors
   - Verify HTMX library loaded correctly
   - Check CSRF tokens in forms

5. **Responsive layout issues**:
   - Clear browser cache
   - Check Tailwind CSS classes in templates
   - Verify meta viewport tag in base layout

**Getting Help**:
- Check server logs: `go run cmd/server/main.go 2>&1 | tee server.log`
- Browser dev tools → Console for client-side errors
- Redis logs: `redis-cli monitor` to see backend operations