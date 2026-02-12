# Requirements Document

## Introduction

This document defines the requirements for the Telegram Bot session list feature. This feature allows users to view, manage, and switch between their AI conversation sessions through the Telegram interface, providing quick access via Inline Keyboard.

## Glossary

- **Session**: A conversation session between a user and AI, containing session ID, title, creation time, update time, and last message
- **Bot**: Telegram Bot webhook server implemented in Go
- **Inline_Keyboard**: Telegram's inline keyboard UI component displayed below messages
- **Callback_Query**: Callback event triggered when a user clicks an Inline Keyboard button
- **Session_Store**: Session data storage layer responsible for persisting session data
- **Active_Session**: The session context currently being used by a user

## Requirements

### Requirement 1: Session List Query

**User Story:** As a user, I want to view my session list through a command, so that I can quickly see what conversation sessions I have.

#### Acceptance Criteria

1. WHEN a user sends the /sessions command, THE Bot SHALL return a session list message for that user
2. WHEN returning a session list message, THE Bot SHALL display up to 6 sessions using Inline_Keyboard
3. WHEN the number of sessions exceeds 6, THE Bot SHALL add a "More" button at the bottom of the Inline_Keyboard
4. WHEN a user has no sessions, THE Bot SHALL return a message indicating no sessions exist
5. WHEN displaying sessions, THE Bot SHALL show the session title and last update time

### Requirement 2: Session Pagination

**User Story:** As a user, I want to browse all sessions, so that I can find the session I need when there are many sessions.

#### Acceptance Criteria

1. WHEN a user clicks the "More" button, THE Bot SHALL load the next page of sessions (the next 6 sessions)
2. WHEN loading the next page of sessions, THE Bot SHALL update the Inline_Keyboard to display the new session list
3. WHEN on the last page, THE Bot SHALL not display the "More" button
4. WHEN paginating sessions, THE Bot SHALL sort sessions by update time in descending order

### Requirement 3: Session Switching

**User Story:** As a user, I want to click a session button to switch to that session, so that I can continue a previous conversation.

#### Acceptance Criteria

1. WHEN a user clicks a session button, THE Bot SHALL set that session as the user's Active_Session
2. WHEN session switching succeeds, THE Bot SHALL return a confirmation message displaying the current session title
3. WHEN a user sends a new message, THE Bot SHALL process that message in the Active_Session context
4. WHEN a user has no Active_Session, THE Bot SHALL automatically create a new session

### Requirement 4: Callback Data Format

**User Story:** As a system, I need standardized callback data formats, so that I can correctly handle user button click operations.

#### Acceptance Criteria

1. WHEN creating a session button, THE Bot SHALL use the format "open_s_{sessionID}" as callback_data
2. WHEN creating a pagination button, THE Bot SHALL use the format "more_sessions_{offset}" as callback_data
3. THE Bot SHALL ensure callback_data length does not exceed 64 bytes

### Requirement 5: Session Data Storage

**User Story:** As a system, I need to persist session data, so that users can access their sessions at different times.

#### Acceptance Criteria

1. WHEN creating a new session, THE Session_Store SHALL generate a unique UUID as the session ID
2. WHEN storing a session, THE Session_Store SHALL save user_id, title, created_at, updated_at, and last_message
3. WHEN querying sessions, THE Session_Store SHALL support filtering by user_id
4. WHEN querying sessions, THE Session_Store SHALL support sorting by updated_at in descending order
5. WHEN querying sessions, THE Session_Store SHALL support paginated queries (offset and limit)
6. WHEN updating a session, THE Session_Store SHALL automatically update the updated_at timestamp

### Requirement 6: Session Ownership Verification

**User Story:** As a system, I need to verify that users can only access their own sessions, so that user privacy and data security are protected.

#### Acceptance Criteria

1. WHEN a user clicks a session button, THE Bot SHALL verify that the session's user_id matches the operating user's ID
2. IF session ownership verification fails, THEN THE Bot SHALL return an error message and reject the operation
3. WHEN querying the session list, THE Bot SHALL only return sessions belonging to the current user

### Requirement 7: Error Handling

**User Story:** As a user, I want the system to handle error situations gracefully, so that I know what went wrong.

#### Acceptance Criteria

1. WHEN a session ID does not exist, THE Bot SHALL return a friendly error message
2. WHEN database connection fails, THE Bot SHALL return a system error message and log the error
3. WHEN callback_data format is invalid, THE Bot SHALL ignore the callback and log a warning
4. WHEN concurrent operations cause conflicts, THE Bot SHALL use a last-write-wins strategy

### Requirement 8: Session Title Generation

**User Story:** As a user, I want new sessions to have meaningful titles, so that I can quickly identify session content.

#### Acceptance Criteria

1. WHEN creating a new session, THE Bot SHALL use the first 30 characters of the first user message as the title
2. WHEN the first message is less than 10 characters, THE Bot SHALL use the complete message as the title
3. WHEN the first message is empty or contains only whitespace, THE Bot SHALL use the default title "New Session" plus timestamp
4. WHEN the title contains newline characters, THE Bot SHALL replace newlines with spaces

### Requirement 9: Performance Requirements

**User Story:** As a user, I want the session list to load quickly, so that I have a smooth user experience.

#### Acceptance Criteria

1. WHEN querying the session list, THE Bot SHALL return a response within 500 milliseconds
2. WHEN switching sessions, THE Bot SHALL complete the state update within 200 milliseconds
3. THE Session_Store SHALL support at least 1000 concurrent query requests
