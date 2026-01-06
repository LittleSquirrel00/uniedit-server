## MODIFIED Requirements

### Requirement: Task Manager

The system SHALL manage async AI tasks with database persistence via compatibility wrapper over shared/task.

**Note**: This requirement wraps `shared/task` infrastructure. The `ai/task` package provides:
- Type aliases for backward compatibility (`Task`, `Status`, `Error`)
- Conversion functions (`ToSharedTask`, `FromSharedTask`)
- AI-specific fields mapped to shared task Metadata

**Migration Path**:
- New code SHOULD import from `internal/shared/task` directly
- Existing `ai/task` imports continue to work via wrapper

#### Scenario: Submit task

- **WHEN** user submits a task with type and payload
- **THEN** create task record with status "pending"
- **AND** return task ID immediately

#### Scenario: Get task status

- **WHEN** user queries task by ID
- **THEN** return current status, progress, and output if completed
