# invitation Specification

## Purpose
Invitation system for adding users to teams with acceptance workflow.

## Requirements

## ADDED Requirements

### Requirement: Send Invitation
The system SHALL allow team admins to invite users by email.

#### Scenario: Invite registered user
- **WHEN** admin sends invitation to email of registered user
- **THEN** system creates invitation with status "pending"
- **AND** links invitation to user ID
- **AND** generates secure token (32 bytes, URL-safe)
- **AND** sets expiration to 7 days
- **AND** returns invitation details with accept URL

#### Scenario: Invite unregistered user
- **WHEN** admin sends invitation to email not in system
- **THEN** system creates invitation with status "pending"
- **AND** invitee_id is null
- **AND** generates secure token
- **AND** returns invitation details

#### Scenario: Duplicate pending invitation
- **WHEN** admin invites same email that has pending invitation
- **THEN** system returns 409 Conflict with "invitation_already_pending"

#### Scenario: User already member
- **WHEN** admin invites email of existing team member
- **THEN** system returns 409 Conflict with "user_already_member"

#### Scenario: Insufficient permission
- **WHEN** member or guest attempts to invite
- **THEN** system returns 403 Forbidden

#### Scenario: Member limit exceeded
- **WHEN** team member count + pending invitations >= member_limit
- **THEN** system returns 403 Forbidden with "member_limit_exceeded"

### Requirement: List Invitations
The system SHALL allow viewing invitations in different contexts.

#### Scenario: List team invitations (admin view)
- **WHEN** team admin requests team's invitations
- **THEN** system returns all invitations with status, invitee, inviter, created_at

#### Scenario: List my pending invitations (user view)
- **WHEN** authenticated user requests their invitations
- **THEN** system returns pending invitations where invitee_email matches user's email
- **OR** invitee_id matches user's ID

### Requirement: Accept Invitation
The system SHALL allow invited users to accept invitations.

#### Scenario: Accept valid invitation
- **WHEN** user accepts invitation with valid token
- **AND** invitation status is "pending"
- **AND** invitation is not expired
- **THEN** system adds user to team with specified role
- **AND** sets invitation status to "accepted"
- **AND** sets accepted_at timestamp
- **AND** links invitee_id if not already set
- **AND** returns team details

#### Scenario: Accept as different user
- **WHEN** logged-in user accepts invitation sent to different email
- **THEN** system returns 403 Forbidden with "invitation_not_for_you"

#### Scenario: Accept expired invitation
- **WHEN** user accepts invitation past expires_at
- **THEN** system returns 410 Gone with "invitation_expired"

#### Scenario: Accept already processed invitation
- **WHEN** user accepts invitation with status "accepted" or "rejected"
- **THEN** system returns 410 Gone with "invitation_already_processed"

#### Scenario: Invalid token
- **WHEN** user submits non-existent token
- **THEN** system returns 404 Not Found

#### Scenario: Member limit check on accept
- **WHEN** user accepts invitation but team is at member limit
- **THEN** system returns 403 Forbidden with "member_limit_exceeded"

### Requirement: Reject Invitation
The system SHALL allow invited users to reject invitations.

#### Scenario: Reject valid invitation
- **WHEN** user rejects invitation with valid token
- **AND** invitation status is "pending"
- **THEN** system sets invitation status to "rejected"
- **AND** returns success

#### Scenario: Reject as different user
- **WHEN** logged-in user rejects invitation sent to different email
- **THEN** system returns 403 Forbidden with "invitation_not_for_you"

### Requirement: Revoke Invitation
The system SHALL allow team admins to revoke pending invitations.

#### Scenario: Revoke pending invitation
- **WHEN** admin revokes invitation by ID
- **AND** invitation status is "pending"
- **THEN** system sets invitation status to "revoked"
- **AND** returns success

#### Scenario: Revoke non-pending invitation
- **WHEN** admin revokes accepted or rejected invitation
- **THEN** system returns 400 Bad Request with "cannot_revoke_processed_invitation"

#### Scenario: Insufficient permission to revoke
- **WHEN** member or guest attempts to revoke
- **THEN** system returns 403 Forbidden

### Requirement: Invitation Expiration
The system SHALL handle expired invitations.

#### Scenario: Expiration check
- **WHEN** invitation is accessed after expires_at
- **THEN** system treats it as invalid
- **AND** returns appropriate error

#### Scenario: Expired invitation cleanup (background)
- **WHEN** background job runs
- **THEN** system marks expired pending invitations as "expired"

### Requirement: Invitation Security
The system SHALL protect invitation tokens.

#### Scenario: Token generation
- **WHEN** creating invitation
- **THEN** system generates cryptographically secure random token
- **AND** token is 32 bytes, base64url encoded
- **AND** token is unique

#### Scenario: Token single use
- **WHEN** invitation is accepted, rejected, or revoked
- **THEN** token cannot be reused
