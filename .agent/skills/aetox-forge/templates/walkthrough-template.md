# Walkthrough: [TICKET-ID] - [Ticket Title]

- **Ticket**: `docs/specs/[feature-slug]/tickets/[ticket-file].md`
- **Status**: [Completed | Partial / Blocked]
- **Date**: YYYY-MM-DD

---

## 1. What Was Implemented
[Summary of the exact capabilities added or modified to fulfill the ticket's acceptance criteria.]

---

## 2. Changes Made

### Files Created
- `path/to/new_file.go`: [Summary of additions]

### Files Modified
- `path/to/modified_file.go`: [Summary of changes]

---

## 3. Test & Verification Results

### Failing Test (Red Phase)
- **Command**: `[Test command, e.g. go test, npm test, pytest -v]`
- **Initial Failure**:
  ```
  [Paste failing test output proving the missing capability]
  ```

### Passing Test (Green Phase)
- **Command**: `[Test command]`
- **Output**:
  ```
  [Paste passing test output]
  ```

### Full Regression Suite
- **Command**: `[Project test command]`
- **Result**: [PASS - All existing tests pass without regressions]

---

## 4. Next Ticket
- **Next Unblocked Ticket**: `[TICKET-ID+1] - [Title]`
- **Context Boundary**: Recommend clearing conversation context before loading next ticket.
