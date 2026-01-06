## Context

UniEdit Server needs team collaboration capabilities for Git repositories, workflows, and registry models. The current Git module has basic collaborator support (direct assignment), but lacks invitation workflows and team abstractions.

### Constraints
- Must integrate with existing Billing module for member limits
- Must support both registered and unregistered user invitations
- Must be reusable across Git, Workflow, and Registry modules
- Must follow existing module patterns (handler/service/repository layers)

### Stakeholders
- End users: Create teams, invite collaborators
- Git module: Sync team permissions to repositories
- Billing module: Enforce plan-based member limits

## Goals / Non-Goals

### Goals
- Provide team management with role-based access control
- Implement invitation flow with email support
- Enable team-to-resource permission synchronization
- Support member limits based on billing plans

### Non-Goals
- Organization hierarchy (multi-team management) - P2
- SSO/SAML integration - P2
- Audit logging - P2
- Branch protection rules - separate feature

## Decisions

### Decision 1: Separate `collaboration` module vs extending `user` module

**Choice**: Create new `internal/module/collaboration/` module

**Rationale**:
- User module handles individual identity; collaboration handles group relationships
- Keeps modules focused (Single Responsibility Principle)
- Collaboration is cross-cutting (used by Git, Workflow, Registry)
- Easier to test and maintain independently

**Alternatives considered**:
- Extend User module: Would bloat user module, mix concerns
- Extend Git module: Not reusable for Workflow/Registry

### Decision 2: Permission model

**Choice**: Role-based with 4 levels: `owner`, `admin`, `member`, `guest`

**Rationale**:
- Simple hierarchy covers most use cases
- Maps cleanly to Git permissions (admin→admin, member→write, guest→read)
- Easy to understand and implement

**Alternatives considered**:
- Fine-grained permissions: More complex, not needed initially
- Just 3 roles: `guest` needed for read-only collaborators

### Decision 3: Invitation token storage

**Choice**: Store tokens in database with expiration

**Rationale**:
- Simple implementation
- Supports revocation
- Can track invitation history

**Alternatives considered**:
- JWT tokens: Harder to revoke, no history
- Redis-only: Loses history on restart

### Decision 4: Team-to-resource binding

**Choice**: Teams reference resources through a linking table, sync permissions on team membership changes

**Rationale**:
- Decouples team from specific resource types
- Allows lazy sync (on access) or eager sync (on change)
- Git module keeps its existing collaborator model

## Data Model

```
┌─────────────┐       ┌──────────────────┐       ┌───────────────────┐
│   teams     │       │  team_members    │       │ team_invitations  │
├─────────────┤       ├──────────────────┤       ├───────────────────┤
│ id          │◄──────│ team_id          │       │ id                │
│ owner_id    │       │ user_id          │       │ team_id           │
│ name        │       │ role             │       │ inviter_id        │
│ slug        │       │ joined_at        │       │ invitee_email     │
│ description │       │ updated_at       │       │ invitee_id (opt)  │
│ visibility  │       └──────────────────┘       │ role              │
│ member_limit│                                  │ token             │
│ created_at  │                                  │ status            │
│ updated_at  │                                  │ expires_at        │
└─────────────┘                                  │ created_at        │
                                                 │ accepted_at       │
                                                 └───────────────────┘
```

## Module Structure

```
internal/module/collaboration/
├── model.go            # Team, TeamMember, TeamInvitation
├── roles.go            # Role constants and permission checks
├── dto.go              # Request/Response DTOs
├── errors.go           # Module-specific errors
├── handler.go          # HTTP handlers
├── service.go          # Business logic
├── repository.go       # Data access
└── service_test.go     # Unit tests
```

## API Design

```
# Teams
POST   /api/v1/teams                    Create team
GET    /api/v1/teams                    List my teams
GET    /api/v1/teams/:slug              Get team
PATCH  /api/v1/teams/:slug              Update team
DELETE /api/v1/teams/:slug              Delete team

# Members
GET    /api/v1/teams/:slug/members      List members
PATCH  /api/v1/teams/:slug/members/:uid Update member role
DELETE /api/v1/teams/:slug/members/:uid Remove member

# Invitations (team context)
POST   /api/v1/teams/:slug/invitations  Send invitation
GET    /api/v1/teams/:slug/invitations  List team invitations

# Invitations (user context)
GET    /api/v1/invitations              My pending invitations
POST   /api/v1/invitations/:token/accept  Accept invitation
POST   /api/v1/invitations/:token/reject  Reject invitation
DELETE /api/v1/invitations/:id          Revoke invitation (inviter)
```

## Risks / Trade-offs

### Risk 1: Team-Git sync complexity
- **Risk**: Keeping team members in sync with repo collaborators adds complexity
- **Mitigation**: Start with manual sync API, add automatic sync later

### Risk 2: Invitation email delivery
- **Risk**: Email delivery is out of scope, but invitations need notifications
- **Mitigation**: Return invitation token/link in API; email integration is separate concern

### Risk 3: Member limit enforcement race conditions
- **Risk**: Concurrent invitation acceptances could exceed limits
- **Mitigation**: Use database transaction with member count check

## Migration Plan

1. Create database tables (teams, team_members, team_invitations)
2. Implement collaboration module with full CRUD
3. Add integration point for Git module (optional sync)
4. Update Billing module to expose member limits

**Rollback**: Drop new tables, no existing data affected

## Open Questions

1. Should team deletion cascade to invitations or soft-delete?
   - **Proposed**: Soft-delete team, cancel pending invitations
2. Should we support bulk invitations?
   - **Proposed**: Not in v1, single invite per request
