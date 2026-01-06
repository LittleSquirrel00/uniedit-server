# storage-quotas Specification

## Purpose

支持 Git 和 LFS 存储空间限制，防止用户超额使用存储资源。

## ADDED Requirements

### Requirement: Storage Quota Definition

The system SHALL define storage quotas in subscription plans.

#### Scenario: Plan storage limits

- **WHEN** plan is defined
- **THEN** plan includes:
  - `git_storage_mb`: Git 仓库存储限额(MB) (-1=无限)
  - `lfs_storage_mb`: LFS 对象存储限额(MB) (-1=无限)

#### Scenario: Default storage quotas

- **WHEN** system is initialized
- **THEN** plans have storage quotas:
  | Plan | Git (MB) | LFS (MB) |
  |------|----------|----------|
  | free | 100 | 500 |
  | pro | 5,000 | 10,000 |
  | team | 50,000 | 100,000 |
  | enterprise | -1 | -1 |

### Requirement: Storage Usage Tracking

The system SHALL track storage usage per user.

#### Scenario: Calculate Git storage

- **WHEN** user requests storage usage
- **THEN** system sums `size_bytes` of all user-owned repositories
- **AND** caches result for 5 minutes

#### Scenario: Calculate LFS storage

- **WHEN** user requests storage usage
- **THEN** system sums `lfs_size_bytes` of all user-owned repositories
- **AND** caches result for 5 minutes

#### Scenario: Invalidate storage cache

- **WHEN** repository size changes (push, LFS upload)
- **THEN** system invalidates storage cache for owner

### Requirement: Storage Quota Enforcement

The system SHALL enforce storage quotas on write operations.

#### Scenario: Check Git push quota

- **WHEN** git push is received
- **AND** `plan.git_storage_mb != -1`
- **THEN** estimate additional storage
- **AND** check current + additional < limit
- **IF** limit exceeded
- **THEN** reject push with 413 Payload Too Large and `git_storage_quota_exceeded`

#### Scenario: Check LFS upload quota

- **WHEN** LFS batch upload request is received
- **AND** `plan.lfs_storage_mb != -1`
- **THEN** calculate total requested size
- **AND** check current + requested < limit
- **IF** limit exceeded
- **THEN** reject with 413 Payload Too Large and `lfs_storage_quota_exceeded`

#### Scenario: Partial LFS batch

- **WHEN** LFS batch request exceeds quota
- **AND** some objects would fit
- **THEN** reject entire batch (atomic)
- **AND** include `quota_remaining` in error response

### Requirement: Storage Quota Status

The system SHALL provide storage quota status.

#### Scenario: Get storage quota status

- **WHEN** user requests quota status
- **THEN** system returns:
  - git: used_mb, limit_mb, remaining_mb
  - lfs: used_mb, limit_mb, remaining_mb

#### Scenario: Storage warning threshold

- **WHEN** storage usage > 80% of limit
- **THEN** include `warning: true` in quota response
- **AND** suggest upgrading plan
