# Change: Add Collaboration Module

## Why

The Git module currently supports direct collaborator assignment (owner adds user by ID), but lacks:
1. **Invitation system** - Users cannot accept/reject collaboration requests
2. **Team management** - No way to manage groups of users with shared permissions
3. **Cross-module reusability** - Workflow and Registry modules will need the same collaboration patterns

A dedicated collaboration module provides a unified, reusable solution for team-based access control across all resource types.

## What Changes

### New Capability: `team`
- Team entity with owner, members, and visibility
- Role-based access control (owner/admin/member/guest)
- Member limits tied to billing plans
- Team CRUD operations

### New Capability: `invitation`
- Email-based invitation flow (supports unregistered users)
- Token-based acceptance/rejection
- Invitation expiration and revocation
- Notification integration ready

### Integration Points
- **Git Module**: Sync team members to repository collaborators
- **Billing Module**: Enforce member limits based on subscription plan
- **User Module**: Resolve user by email for invitations

## Impact

- **Affected specs**: None (new capabilities)
- **Affected code**:
  - New: `internal/module/collaboration/`
  - Modified: `internal/app/app.go` (wire up module)
  - New migrations: `migrations/000010_create_collaboration_tables.*.sql`
- **Database**: 3 new tables (`teams`, `team_members`, `team_invitations`)
- **API**: New `/api/v1/teams` and `/api/v1/invitations` endpoints
