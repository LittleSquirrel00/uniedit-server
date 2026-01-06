# team-quotas Specification

## Purpose

支持团队成员数量配额管理，与订阅套餐关联。

## MODIFIED Requirements

### Requirement: Team Member Quota Definition

The system SHALL define team member quotas in subscription plans.

#### Scenario: Plan member limits

- **WHEN** plan is defined
- **THEN** plan includes:
  - `max_team_members`: 团队成员上限 (-1=无限)

#### Scenario: Default member quotas

- **WHEN** system is initialized
- **THEN** plans have member quotas:
  | Plan | Max Members |
  |------|-------------|
  | free | 1 (owner only) |
  | pro | 5 |
  | team | 50 |
  | enterprise | -1 |

### Requirement: Team Quota Enforcement

The system SHALL enforce team member quotas.

#### Scenario: Check quota before invite

- **WHEN** team owner/admin invites new member
- **AND** `plan.max_team_members != -1`
- **THEN** check current_members + pending_invites < limit
- **IF** limit reached
- **THEN** return 402 Payment Required with `team_member_quota_exceeded`

#### Scenario: Owner subscription determines quota

- **WHEN** checking team quota
- **THEN** use team owner's subscription plan
- **NOT** the inviter's plan (if inviter is admin)

#### Scenario: Quota applies to all teams

- **WHEN** user owns multiple teams
- **THEN** quota is per-team, not cumulative

### Requirement: Team Quota Status

The system SHALL provide team quota status.

#### Scenario: Get team quota status

- **WHEN** user requests quota status
- **THEN** for each owned team, return:
  - current_members: actual member count
  - pending_invites: pending invitation count
  - limit: plan's max_team_members
  - remaining: limit - current - pending

#### Scenario: Include quota in team response

- **WHEN** team details are requested
- **AND** requester is owner or admin
- **THEN** include quota info in response

### Requirement: Quota Downgrade Handling

The system SHALL handle quota when plan downgrades.

#### Scenario: Downgrade with excess members

- **WHEN** user downgrades to plan with lower member limit
- **AND** current members > new limit
- **THEN** existing members remain (no automatic removal)
- **BUT** new invitations are blocked until under limit

#### Scenario: Grace period message

- **WHEN** team is over quota after downgrade
- **THEN** include `over_quota: true` in team response
- **AND** suggest removing members or upgrading
