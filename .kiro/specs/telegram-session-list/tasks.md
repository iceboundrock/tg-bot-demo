# Implementation Plan: Telegram Session List

## Overview

This implementation plan breaks down the Telegram Bot session list feature into discrete coding tasks. The feature enables users to view, paginate, and switch between AI conversation sessions through Telegram's Inline Keyboard interface. The implementation uses Go with the `github.com/go-telegram/bot` library and SQLite for data persistence.

## Tasks

- [x] 1. Set up project structure and dependencies
  - Create directory structure for session package and handlers package
  - Add required dependencies to go.mod: `github.com/go-telegram/bot`, `github.com/google/uuid`, `modernc.org/sqlite`
  - Define core interfaces and types in session package
  - _Requirements: 5.1, 5.2_

- [ ] 2. Implement session model and title generation
  - [x] 2.1 Create Session struct with all required fields
    - Define Session type with ID, UserID, Title, CreatedAt, UpdatedAt, LastMessage
    - Implement NewSession constructor function
    - _Requirements: 5.1, 5.2_
  
  - [x] 2.2 Implement title generation logic
    - Write generateTitle function handling empty messages, short messages, long messages
    - Handle newline replacement and whitespace normalization
    - _Requirements: 8.1, 8.2, 8.3, 8.4_
  
  - [ ]* 2.3 Write property test for title generation
    - **Property 19: Title Generation from Message**
    - **Validates: Requirements 8.1, 8.2**
  
  - [ ]* 2.4 Write property test for newline replacement
    - **Property 20: Newline Replacement in Titles**
    - **Validates: Requirements 8.4**
  
  - [ ]* 2.5 Write unit tests for title generation edge cases
    - Test empty messages, whitespace-only messages, messages with newlines
    - Test boundary cases (exactly 10 chars, exactly 30 chars)
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ] 3. Implement Store interface and SQLite implementation
  - [x] 3.1 Define Store interface
    - Define all methods: Create, Get, Update, Delete, ListByUser, CountByUser, GetActiveSession, SetActiveSession
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  
  - [x] 3.2 Implement SQLiteStore struct and initialization
    - Create NewSQLiteStore constructor with database connection
    - Implement initSchema with table creation and indexes
    - Enable WAL mode and foreign keys
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  
  - [x] 3.3 Implement CRUD operations
    - Implement Create, Get, Update, Delete methods
    - Handle error cases (not found, parse errors)
    - _Requirements: 5.1, 5.2, 5.6_
  
  - [x] 3.4 Implement query operations
    - Implement ListByUser with pagination support (offset, limit)
    - Implement CountByUser for total count
    - Ensure proper sorting by updated_at DESC
    - _Requirements: 5.3, 5.4, 5.5, 2.4_
  
  - [x] 3.5 Implement active session management
    - Implement GetActiveSession and SetActiveSession methods
    - Use UPSERT pattern for SetActiveSession
    - _Requirements: 3.1, 3.4_
  
  - [ ]* 3.6 Write property test for session data round-trip
    - **Property 15: Session Data Round-Trip**
    - **Validates: Requirements 5.2**
  
  - [ ]* 3.7 Write property test for session ID uniqueness
    - **Property 14: Session ID Uniqueness**
    - **Validates: Requirements 5.1**
  
  - [ ]* 3.8 Write property test for updated timestamp auto-update
    - **Property 16: Updated Timestamp Auto-Update**
    - **Validates: Requirements 5.6**
  
  - [ ]* 3.9 Write unit tests for store operations
    - Test error conditions (session not found, invalid UUID)
    - Test pagination edge cases (offset beyond total, limit=0)
    - Test concurrent operations
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 7.1, 7.4_

- [x] 4. Checkpoint - Ensure storage layer tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Implement Session Manager business logic
  - [x] 5.1 Create Manager struct and constructor
    - Define Manager type with Store dependency
    - Implement NewManager constructor
    - _Requirements: 1.1, 2.1, 3.1_
  
  - [x] 5.2 Implement ListSessions method
    - Call store.ListByUser with pagination parameters
    - Calculate hasMore flag based on total count
    - Return sessions and hasMore indicator
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.3, 2.4_
  
  - [x] 5.3 Implement SwitchSession method
    - Verify session ownership before switching
    - Call store.SetActiveSession on success
    - Return appropriate errors for unauthorized access
    - _Requirements: 3.1, 6.1, 6.2_
  
  - [x] 5.4 Implement CreateSession method
    - Create new session using NewSession
    - Store session in database
    - Set as active session automatically
    - _Requirements: 3.4, 5.1, 5.2_
  
  - [x] 5.5 Implement GetOrCreateActiveSession method
    - Try to get active session first
    - Create new session if none exists
    - _Requirements: 3.3, 3.4_
  
  - [ ]* 5.6 Write property test for session list filtering
    - **Property 1: Session List Returns User's Sessions Only**
    - **Validates: Requirements 1.1, 5.3, 6.3**
  
  - [ ]* 5.7 Write property test for session display limit
    - **Property 2: Session Display Limit**
    - **Validates: Requirements 1.2**
  
  - [ ]* 5.8 Write property test for pagination correctness
    - **Property 5: Pagination Returns Correct Subset**
    - **Validates: Requirements 2.1, 5.5**
  
  - [ ]* 5.9 Write property test for session sorting
    - **Property 6: Sessions Sorted by Update Time**
    - **Validates: Requirements 2.4, 5.4**
  
  - [ ]* 5.10 Write property test for active session update
    - **Property 7: Active Session Update**
    - **Validates: Requirements 3.1**
  
  - [ ]* 5.11 Write property test for session ownership verification
    - **Property 17: Session Ownership Verification**
    - **Validates: Requirements 6.1, 6.2**
  
  - [ ]* 5.12 Write property test for auto-create session
    - **Property 10: Auto-Create Session When None Active**
    - **Validates: Requirements 3.4**
  
  - [ ]* 5.13 Write unit tests for manager methods
    - Test error handling for unauthorized access
    - Test empty session list case
    - Test boundary cases for pagination
    - _Requirements: 1.1, 1.4, 2.1, 3.1, 6.1, 6.2, 7.1_

- [x] 6. Checkpoint - Ensure business logic tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Implement helper functions for formatting
  - [x] 7.1 Implement formatTimeAgo function
    - Convert timestamps to relative time strings (just now, Xm ago, Xh ago, Xd ago, Jan 2)
    - _Requirements: 1.5_
  
  - [x] 7.2 Implement truncate function
    - Truncate strings to max length with ellipsis
    - Handle UTF-8 runes correctly
    - _Requirements: 1.5_
  
  - [x] 7.3 Implement formatSessionButton function
    - Format session title and time for button display
    - _Requirements: 1.5_
  
  - [ ]* 7.4 Write unit tests for formatting functions
    - Test various time durations for formatTimeAgo
    - Test truncation at boundaries
    - Test UTF-8 handling
    - _Requirements: 1.5_

- [ ] 8. Implement command handler for /sessions
  - [x] 8.1 Create SessionsCommandHandler function
    - Extract user ID from update
    - Call sessionMgr.ListSessions with offset=0, limit=6
    - Handle empty sessions case with appropriate message
    - Build inline keyboard and send message
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_
  
  - [x] 8.2 Implement buildSessionKeyboard function
    - Create inline keyboard rows for each session
    - Add "More" button if hasMore is true
    - Format callback_data correctly
    - _Requirements: 1.2, 1.3, 2.3, 4.1, 4.2_
  
  - [ ]* 8.3 Write property test for more button presence
    - **Property 3: More Button Presence**
    - **Validates: Requirements 1.3, 2.3**
  
  - [ ]* 8.4 Write property test for session display format
    - **Property 4: Session Display Format**
    - **Validates: Requirements 1.5**
  
  - [ ]* 8.5 Write property test for session button callback format
    - **Property 11: Session Button Callback Format**
    - **Validates: Requirements 4.1**
  
  - [ ]* 8.6 Write property test for pagination button callback format
    - **Property 12: Pagination Button Callback Format**
    - **Validates: Requirements 4.2**
  
  - [ ]* 8.7 Write property test for callback data length constraint
    - **Property 13: Callback Data Length Constraint**
    - **Validates: Requirements 4.3**
  
  - [ ]* 8.8 Write unit tests for command handler
    - Test empty sessions case
    - Test exactly 6 sessions (boundary)
    - Test more than 6 sessions
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [ ] 9. Implement callback handler for inline keyboard
  - [x] 9.1 Create CallbackQueryHandler function
    - Answer callback query immediately
    - Route based on callback data prefix (open_s_, more_sessions_)
    - Handle invalid callback data gracefully
    - _Requirements: 3.1, 2.1, 7.3_
  
  - [x] 9.2 Implement handleOpenSession function
    - Parse session ID from callback data
    - Call sessionMgr.SwitchSession
    - Handle errors (not found, unauthorized, invalid UUID)
    - Send confirmation message with session title
    - _Requirements: 3.1, 3.2, 6.1, 6.2, 7.1, 7.2_
  
  - [x] 9.3 Implement handleMoreSessions function
    - Parse offset from callback data
    - Call sessionMgr.ListSessions with new offset
    - Update message reply markup with new keyboard
    - _Requirements: 2.1, 2.2, 2.3_
  
  - [ ]* 9.4 Write property test for switch confirmation contains title
    - **Property 8: Switch Confirmation Contains Title**
    - **Validates: Requirements 3.2**
  
  - [ ]* 9.5 Write unit tests for callback handlers
    - Test invalid session ID format
    - Test unauthorized access attempt
    - Test invalid callback data format
    - Test pagination with various offsets
    - _Requirements: 3.1, 3.2, 6.1, 6.2, 7.1, 7.2, 7.3_

- [ ] 10. Implement message handler for active session routing
  - [x] 10.1 Create message handler function
    - Extract user ID and message text
    - Call sessionMgr.GetOrCreateActiveSession
    - Route message to active session context
    - _Requirements: 3.3, 3.4_
  
  - [ ]* 10.2 Write property test for messages route to active session
    - **Property 9: Messages Route to Active Session**
    - **Validates: Requirements 3.3**
  
  - [ ]* 10.3 Write unit tests for message handler
    - Test with existing active session
    - Test with no active session (auto-create)
    - _Requirements: 3.3, 3.4_

- [ ] 11. Wire all components together
  - [x] 11.1 Create main bot initialization function
    - Initialize SQLite store with database path
    - Create session manager with store
    - Register command handler for /sessions
    - Register callback query handler
    - Register message handler
    - _Requirements: All_
  
  - [x] 11.2 Add configuration struct and loading
    - Define Config struct with SessionsPerPage, DatabasePath, etc.
    - Implement config loading from environment or file
    - _Requirements: All_
  
  - [x] 11.3 Add error handling and logging
    - Implement error response helpers
    - Add structured logging for key operations
    - _Requirements: 7.1, 7.2, 7.3, 7.4_
  
  - [ ]* 11.4 Write integration tests
    - Test complete flow: /sessions → click session → verify active
    - Test complete flow: /sessions → click More → verify pagination
    - Test complete flow: send message → verify session creation
    - _Requirements: 1.1, 2.1, 3.1, 3.4_

- [ ] 12. Final checkpoint - Ensure all tests pass
  - Run all unit tests and property tests
  - Verify test coverage meets goals (>80% for unit tests, all 20 properties)
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties with minimum 100 iterations
- Unit tests validate specific examples and edge cases
- The implementation follows a bottom-up approach: data layer → business logic → handlers → integration
- SQLite is used for persistence with WAL mode for better concurrency
- All callback_data must stay within 64-byte limit
- Sessions are always sorted by updated_at in descending order
