# ai-task Specification

## Purpose
TBD - created by archiving change add-ai-module. Update Purpose after archive.
## Requirements
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

### Requirement: Task Execution

The system SHALL execute tasks asynchronously with progress tracking.

#### Scenario: Task execution flow

- **WHEN** task executor picks up pending task
- **THEN** update status to "running"
- **AND** execute registered executor for task type
- **AND** update progress periodically

#### Scenario: Task completion

- **WHEN** executor completes successfully
- **THEN** update status to "completed"
- **AND** store output in task record
- **AND** set completed_at timestamp

#### Scenario: Task failure

- **WHEN** executor encounters error
- **THEN** update status to "failed"
- **AND** store error details in task record

#### Scenario: External task tracking

- **WHEN** task involves external service (e.g., Runway video generation)
- **THEN** store external_task_id for status polling
- **AND** poll external service for progress updates

### Requirement: Concurrency Control

The system SHALL limit concurrent task execution.

#### Scenario: Concurrency pool

- **WHEN** multiple tasks are submitted
- **THEN** execute up to max_concurrent tasks in parallel
- **AND** queue remaining tasks

#### Scenario: Per-user limits

- **WHEN** user has reached max concurrent tasks
- **THEN** queue new tasks until slots are available

### Requirement: Task Recovery

The system SHALL recover incomplete tasks after server restart.

#### Scenario: Recover pending tasks

- **WHEN** server starts
- **THEN** query tasks with status "pending" or "running"
- **AND** re-queue for execution

#### Scenario: Stale task cleanup

- **WHEN** task has been "running" longer than timeout
- **THEN** mark as "failed" with timeout error

### Requirement: Progress Subscription

The system SHALL support real-time progress updates.

#### Scenario: Subscribe to progress

- **WHEN** client subscribes to task progress
- **THEN** receive updates via callback or SSE
- **AND** receive final status on completion

#### Scenario: Unsubscribe

- **WHEN** client unsubscribes or disconnects
- **THEN** stop sending updates for that subscription

