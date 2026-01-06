## 1. Database Schema

- [x] 1.1 Create migration `000013_create_collaboration_tables.up.sql`
  - `teams` table (id, owner_id, name, slug, description, visibility, member_limit, status, created_at, updated_at)
  - `team_members` table (team_id, user_id, role, joined_at, updated_at)
  - `team_invitations` table (id, team_id, inviter_id, invitee_email, invitee_id, role, token, status, expires_at, created_at, accepted_at)
  - Indexes: teams(owner_id, slug), team_members(user_id), team_invitations(token), team_invitations(invitee_email)
- [x] 1.2 Create migration `000013_create_collaboration_tables.down.sql`
- [ ] 1.3 Run migrations and verify schema

## 2. Core Module Structure

- [x] 2.1 Create `internal/module/collaboration/model.go`
  - Team, TeamMember, TeamInvitation structs
  - Role constants (owner, admin, member, guest)
  - Status constants (active, deleted for team; pending, accepted, rejected, revoked, expired for invitation)
- [x] 2.2 Create `internal/module/collaboration/roles.go`
  - Permission check functions (CanInvite, CanRemove, CanUpdate, etc.)
  - Role hierarchy comparison
  - Role to Git permission mapping
- [x] 2.3 Create `internal/module/collaboration/errors.go`
  - ErrTeamNotFound, ErrSlugExists, ErrMemberLimitExceeded
  - ErrInvitationNotFound, ErrInvitationExpired, ErrAlreadyMember
  - ErrInsufficientPermission, ErrCannotChangeOwner
- [x] 2.4 Create `internal/module/collaboration/dto.go`
  - CreateTeamRequest/Response, UpdateTeamRequest
  - InviteRequest/Response, AcceptInvitationResponse
  - TeamResponse, MemberResponse, InvitationResponse

## 3. Repository Layer

- [x] 3.1 Create `internal/module/collaboration/repository.go`
  - Team CRUD: Create, GetBySlug, GetByID, List, Update, Delete
  - Member CRUD: AddMember, GetMember, ListMembers, UpdateRole, RemoveMember, CountMembers
  - Invitation CRUD: Create, GetByToken, GetByID, ListByTeam, ListByEmail, UpdateStatus
- [x] 3.2 Implement transaction support for atomic operations

## 4. Service Layer

- [x] 4.1 Create `internal/module/collaboration/service.go` - Team operations
  - CreateTeam (validate, generate slug, add owner as member)
  - GetTeam, ListMyTeams
  - UpdateTeam (permission check)
  - DeleteTeam (owner only, cancel invitations)
- [x] 4.2 Implement member management in service
  - ListMembers, UpdateMemberRole, RemoveMember, LeaveTeam
  - Permission checks for each operation
- [x] 4.3 Implement invitation operations in service
  - SendInvitation (check limits, generate token, check duplicates)
  - ListTeamInvitations, ListMyInvitations
  - AcceptInvitation (validate, add member, update status)
  - RejectInvitation, RevokeInvitation

## 5. Handler Layer

- [x] 5.1 Create `internal/module/collaboration/handler.go`
  - Team routes: POST /teams, GET /teams, GET /teams/:slug, PATCH /teams/:slug, DELETE /teams/:slug
  - Member routes: GET /teams/:slug/members, PATCH /teams/:slug/members/:uid, DELETE /teams/:slug/members/:uid
  - Invitation routes: POST /teams/:slug/invitations, GET /teams/:slug/invitations
- [x] 5.2 Create invitation handler routes
  - GET /invitations (my pending)
  - POST /invitations/:token/accept
  - POST /invitations/:token/reject
  - DELETE /invitations/:id (revoke)

## 6. Module Integration

- [x] 6.1 Update `internal/app/app.go`
  - Wire collaboration module dependencies
  - Register routes under /api/v1
- [ ] 6.2 Add Billing integration interface (deferred - using default member_limit=5)
  - GetMemberLimit(planID) function
  - Check limits during invitation

## 7. Testing

- [x] 7.1 Create `internal/module/collaboration/service_test.go`
  - Role hierarchy tests
  - Permission check tests
  - Slug generation tests
  - Invitation status tests
- [ ] 7.2 Create `internal/module/collaboration/repository_test.go`
  - Database operation tests with test database

## 8. Documentation

- [ ] 8.1 Update module README with API documentation
- [ ] 8.2 Add OpenAPI spec for new endpoints
