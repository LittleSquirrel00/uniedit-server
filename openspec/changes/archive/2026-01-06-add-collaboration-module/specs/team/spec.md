# team Specification

## Purpose
Team management for collaborative access control across UniEdit resources (repositories, workflows, registry models).

## Requirements

## ADDED Requirements

### Requirement: Team Creation
The system SHALL allow users to create teams with a unique slug.

#### Scenario: Successful team creation
- **WHEN** authenticated user submits valid team name
- **THEN** system creates team with owner as the user
- **AND** generates URL-friendly slug from name
- **AND** adds owner as team member with "owner" role
- **AND** returns team details

#### Scenario: Duplicate slug
- **WHEN** user creates team with slug that already exists for that user
- **THEN** system returns 409 Conflict with "slug_already_exists"

#### Scenario: Invalid name
- **WHEN** user submits empty or whitespace-only name
- **THEN** system returns 400 Bad Request with validation error

### Requirement: Team Retrieval
The system SHALL allow users to view teams they own or belong to.

#### Scenario: Get team by slug
- **WHEN** user requests team by owner username and slug
- **AND** user is team member or team is public
- **THEN** system returns team details with member count

#### Scenario: List my teams
- **WHEN** user requests their teams
- **THEN** system returns all teams where user is a member
- **AND** includes user's role in each team

#### Scenario: Team not found
- **WHEN** user requests non-existent team
- **THEN** system returns 404 Not Found

#### Scenario: Access denied
- **WHEN** user requests private team they don't belong to
- **THEN** system returns 404 Not Found (security: hide existence)

### Requirement: Team Update
The system SHALL allow team owners and admins to update team settings.

#### Scenario: Update team details
- **WHEN** owner or admin updates team name, description, or visibility
- **THEN** system updates team
- **AND** regenerates slug if name changed
- **AND** returns updated team

#### Scenario: Insufficient permission
- **WHEN** member or guest attempts to update team
- **THEN** system returns 403 Forbidden

### Requirement: Team Deletion
The system SHALL allow team owners to delete teams.

#### Scenario: Delete team
- **WHEN** owner requests team deletion
- **THEN** system soft-deletes team (status = "deleted")
- **AND** cancels all pending invitations
- **AND** returns success

#### Scenario: Non-owner deletion attempt
- **WHEN** admin, member, or guest attempts to delete team
- **THEN** system returns 403 Forbidden with "only_owner_can_delete"

### Requirement: Member Management
The system SHALL allow team admins to manage members.

#### Scenario: List members
- **WHEN** team member requests member list
- **THEN** system returns all members with roles and join dates

#### Scenario: Update member role
- **WHEN** owner or admin updates another member's role
- **AND** target is not the owner
- **THEN** system updates role
- **AND** returns updated member

#### Scenario: Cannot change owner role
- **WHEN** anyone attempts to change owner's role
- **THEN** system returns 403 Forbidden with "cannot_change_owner_role"

#### Scenario: Admin cannot promote to owner
- **WHEN** admin attempts to set another member's role to owner
- **THEN** system returns 403 Forbidden with "only_owner_can_transfer"

#### Scenario: Remove member
- **WHEN** owner or admin removes a member
- **AND** target is not the owner
- **THEN** system removes member from team
- **AND** returns success

#### Scenario: Cannot remove owner
- **WHEN** anyone attempts to remove the owner
- **THEN** system returns 403 Forbidden with "cannot_remove_owner"

#### Scenario: Member leaves team
- **WHEN** non-owner member requests to leave team
- **THEN** system removes them from team
- **AND** returns success

### Requirement: Role-Based Access Control
The system SHALL enforce role hierarchy: owner > admin > member > guest.

#### Scenario: Role permissions
- **WHEN** checking permissions
- **THEN** owner has all permissions
- **AND** admin can invite, remove members, update team settings
- **AND** member can view team and resources
- **AND** guest has read-only access

#### Scenario: Role to Git permission mapping
- **WHEN** syncing team to repository
- **THEN** owner/admin maps to "admin" permission
- **AND** member maps to "write" permission
- **AND** guest maps to "read" permission
