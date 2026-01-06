# billing-quotas Specification

## Purpose

支持按 AI 任务类型的差异化配额管理（Chat、Image、Video、Embedding）。

## MODIFIED Requirements

### Requirement: Plan Management

The system SHALL define subscription plans with task-specific quotas.

#### Scenario: Task-specific quotas

- **WHEN** plan is defined
- **THEN** plan includes:
  - `monthly_chat_tokens`: Chat/Completion Token 限额 (-1=无限, 0=使用 monthly_tokens)
  - `monthly_image_credits`: 图片生成次数 (-1=无限)
  - `monthly_video_minutes`: 视频生成时长(分钟) (-1=无限)
  - `monthly_embedding_tokens`: Embedding Token 限额 (-1=无限)

#### Scenario: Backward compatibility

- **WHEN** plan has `monthly_chat_tokens = 0`
- **THEN** chat requests use `monthly_tokens` for quota check (original behavior)

#### Scenario: Updated default plans

- **WHEN** system is initialized
- **THEN** plans have task-specific quotas:
  | Plan | Chat | Image | Video(min) | Embedding |
  |------|------|-------|------------|-----------|
  | free | 10K | 10 | 5 | 10K |
  | pro | 500K | 200 | 60 | 500K |
  | team | 2M | 1,000 | 300 | 2M |
  | enterprise | -1 | -1 | -1 | -1 |

### Requirement: Quota Management

The system SHALL enforce task-specific quotas.

#### Scenario: Check chat quota

- **WHEN** chat/completion request is received
- **AND** `plan.monthly_chat_tokens > 0`
- **THEN** check monthly chat token usage < plan limit
- **IF** limit exceeded
- **THEN** return 402 Payment Required with `chat_quota_exceeded`

#### Scenario: Check image quota

- **WHEN** image generation request is received
- **AND** `plan.monthly_image_credits != -1`
- **THEN** check monthly image count < plan limit
- **IF** limit exceeded
- **THEN** return 402 Payment Required with `image_quota_exceeded`

#### Scenario: Check video quota

- **WHEN** video generation request is received
- **AND** `plan.monthly_video_minutes != -1`
- **THEN** check monthly video minutes < plan limit
- **IF** limit exceeded
- **THEN** return 402 Payment Required with `video_quota_exceeded`

#### Scenario: Check embedding quota

- **WHEN** embedding request is received
- **AND** `plan.monthly_embedding_tokens != -1`
- **THEN** check monthly embedding tokens < plan limit
- **IF** limit exceeded
- **THEN** return 402 Payment Required with `embedding_quota_exceeded`

#### Scenario: Get task-specific quota status

- **WHEN** user requests quota status
- **THEN** system returns for each task type:
  - used, limit, remaining
- **AND** includes `reset_at` timestamp

### Requirement: Usage Recording

The system SHALL record usage by task type.

#### Scenario: Record chat usage

- **WHEN** chat request completes successfully
- **THEN** increment chat token counter in Redis
- **AND** record to usage_records with task_type="chat"

#### Scenario: Record image usage

- **WHEN** image generation completes successfully
- **THEN** increment image credit counter in Redis
- **AND** record to usage_records with task_type="image"

#### Scenario: Record video usage

- **WHEN** video generation completes successfully
- **THEN** increment video minutes counter in Redis
- **AND** record to usage_records with task_type="video"
- **AND** calculate duration from task metadata

#### Scenario: Record embedding usage

- **WHEN** embedding request completes successfully
- **THEN** increment embedding token counter in Redis
- **AND** record to usage_records with task_type="embedding"

#### Scenario: Usage aggregation by task type

- **WHEN** user requests usage statistics
- **THEN** system aggregates by task_type
- **AND** returns breakdown for chat, image, video, embedding
