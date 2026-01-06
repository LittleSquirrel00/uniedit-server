## ADDED Requirements

### Requirement: Task Manager Interface

The system SHALL provide a generic task management interface in shared infrastructure.

#### Scenario: Submit task

- **WHEN** module submits a task with type and payload
- **THEN** create task record with status "pending"
- **AND** return task ID immediately

#### Scenario: Get task status

- **WHEN** module queries task by ID
- **THEN** return current status, progress, and output if completed

#### Scenario: List tasks by owner

- **WHEN** module lists tasks with owner filter
- **THEN** return paginated task list ordered by created_at DESC

#### Scenario: Cancel task

- **WHEN** module cancels a running task
- **THEN** set status to "cancelled"
- **AND** attempt to cancel external task if applicable

### Requirement: Task Executor Registration

The system SHALL support pluggable task executors via registration pattern.

#### Scenario: Register executor

- **WHEN** module registers an executor for a task type
- **THEN** executor is invoked for tasks of that type
- **AND** multiple modules can register different executors

#### Scenario: Execute task

- **WHEN** task manager picks up pending task
- **THEN** find registered executor for task type
- **AND** invoke executor with task context

#### Scenario: Unknown task type

- **WHEN** task has no registered executor
- **THEN** mark task as failed with "unknown task type" error

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

- **WHEN** task involves external service (e.g., video generation)
- **THEN** store external_task_id for status polling
- **AND** poll external service for progress updates

### Requirement: External Task Poller

The system SHALL provide a generic external task poller for async operations.

#### Scenario: Register poller

- **WHEN** module registers an external task poller
- **THEN** poller is invoked for tasks with external_task_id

#### Scenario: Poll external status

- **WHEN** poller checks external service
- **THEN** update task progress based on external status
- **AND** complete task when external task completes

#### Scenario: Polling failure

- **WHEN** external service is unreachable
- **THEN** retry with exponential backoff
- **AND** fail task after max retries

### Requirement: Concurrency Control

The system SHALL limit concurrent task execution.

#### Scenario: Concurrency pool

- **WHEN** multiple tasks are submitted
- **THEN** execute up to max_concurrent tasks in parallel
- **AND** queue remaining tasks

#### Scenario: Per-owner limits

- **WHEN** owner has reached max concurrent tasks
- **THEN** queue new tasks until slots are available

#### Scenario: Configurable limits

- **WHEN** system configures concurrency limits
- **THEN** apply limits at both global and per-owner level

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
- **THEN** receive updates via callback
- **AND** receive final status on completion

#### Scenario: Unsubscribe

- **WHEN** client unsubscribes
- **THEN** stop sending updates for that subscription
