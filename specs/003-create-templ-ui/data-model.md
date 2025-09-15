# Data Model: Templ UI Components

## UI State Entities

### LoginFormState
**Purpose**: Manages login form presentation and validation state
**Fields**:
- `Username`: string - entered username
- `Password`: string - entered password (never logged)
- `IsSubmitting`: bool - form submission state
- `ValidationErrors`: map[string]string - field-specific errors
- `GeneralError`: string - login failure message

**Validation Rules**:
- Username: required, 3-50 characters, alphanumeric + underscore
- Password: required, minimum 6 characters
- Rate limiting: max 5 attempts per IP per 5 minutes

### ChatPageState
**Purpose**: Represents the complete chat interface state
**Fields**:
- `CurrentUser`: User - authenticated user info
- `CurrentRoom`: ChatRoom - active chat room
- `Messages`: []Message - ordered message history
- `OnlineUsers`: []User - currently connected users
- `IsConnected`: bool - SSE connection status
- `UnreadCount`: int - unread message counter

**Relationships**:
- Has one CurrentUser (from JWT token)
- Has one CurrentRoom (from URL parameter)
- Has many Messages (from Redis stream)
- Has many OnlineUsers (from Redis set)

### MessageInputState
**Purpose**: Manages message composition interface
**Fields**:
- `Content`: string - message being typed
- `IsSending`: bool - message send state
- `CharacterCount`: int - current message length
- `ValidationError`: string - input validation error

**Validation Rules**:
- Content: required, maximum 1000 characters
- Content: no empty/whitespace-only messages
- Rate limiting: max 10 messages per user per minute

### RoomCreationState
**Purpose**: Handles new chat room creation interface
**Fields**:
- `RoomName`: string - proposed room name
- `Description`: string - optional room description
- `IsPrivate`: bool - room privacy setting
- `IsCreating`: bool - creation in progress
- `ValidationErrors`: map[string]string - field errors

**Validation Rules**:
- RoomName: required, 3-30 characters, alphanumeric + spaces + hyphens
- Description: optional, maximum 200 characters
- RoomName: must be unique across all rooms

### ResponsiveLayoutState
**Purpose**: Tracks current responsive design state
**Fields**:
- `ViewportWidth`: int - current screen width
- `DeviceType`: enum[mobile, tablet] - current layout mode
- `Orientation`: enum[portrait, landscape] - screen orientation
- `IsTouchDevice`: bool - touch capability detection

**State Transitions**:
- mobile (0-767px) ↔ tablet (768px+)
- portrait ↔ landscape (orientation change)
- Server-side rendering defaults to mobile-first

## Template Data Structures

### LoginPageData
```go
type LoginPageData struct {
    FormState LoginFormState
    CSRFToken string
    RedirectURL string
    ErrorMessage string
}
```

### ChatPageData
```go
type ChatPageData struct {
    PageState ChatPageState
    Messages []MessageViewModel
    InputState MessageInputState
    LayoutState ResponsiveLayoutState
    SSEEndpoint string
}
```

### MessageViewModel
```go
type MessageViewModel struct {
    ID string
    Content string
    Username string
    Timestamp time.Time
    IsCurrentUser bool
    AvatarURL string
}
```

### RoomListData
```go
type RoomListData struct {
    Rooms []RoomViewModel
    CreateState RoomCreationState
    CurrentUser User
}
```

### RoomViewModel
```go
type RoomViewModel struct {
    ID string
    Name string
    Description string
    ParticipantCount int
    LastActivity time.Time
    IsPrivate bool
    HasUnreadMessages bool
}
```

## Template Component Hierarchy

```
BaseLayout (responsive container)
├── LoginPage (authentication)
│   └── LoginForm (form component)
├── DashboardPage (room selection)
│   ├── RoomList (room display)
│   ├── CreateRoomButton (room creation)
│   └── UserProfile (user info)
└── ChatPage (messaging interface)
    ├── ChatHeader (room info)
    ├── MessageList (scrollable messages)
    │   └── MessageBubble (individual messages)
    ├── OnlineUsers (participant list)
    └── MessageInput (compose interface)
```

## Data Flow Patterns

### Authentication Flow
1. **GET /login** → LoginPageData → login.templ
2. **POST /login** → validation → JWT cookie → redirect
3. **Middleware** → JWT validation → User context

### Chat Room Flow
1. **GET /rooms** → RoomListData → rooms.templ
2. **POST /rooms** → create room → redirect to room
3. **GET /chat/{roomId}** → ChatPageData → chat.templ
4. **SSE /events/{roomId}** → real-time message stream

### Message Flow
1. **POST /messages** → MessageInputState → validation → Redis
2. **SSE event** → broadcast to room participants
3. **HTMX update** → MessageList re-render with new message

### Responsive Flow
1. **Server-side** → ResponsiveLayoutState → mobile-first rendering
2. **Client-side** → Tailwind breakpoints → layout adaptation
3. **HTMX** → viewport change events → server updates (optional)

## Validation & Error Handling

### Client-Side (HTMX)
- Form validation on submit
- Real-time character counting
- Visual error state updates
- Optimistic UI updates

### Server-Side (Go)
- Input sanitization and validation
- Business rule enforcement
- Rate limiting per user/IP
- Structured error responses

### Error State Presentation
- Field-level validation errors
- Form-level error messages
- Connection status indicators
- Graceful degradation for SSE failures