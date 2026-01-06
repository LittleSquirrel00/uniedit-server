# user-management Specification

## Purpose
TBD - created by archiving change add-user-billing-payment. Update Purpose after archive.
## Requirements
### Requirement: Email Registration
The system SHALL allow users to register with email and password.

#### Scenario: Successful registration
- **WHEN** user submits valid email, password (min 8 chars), and name
- **THEN** system creates user with status "pending"
- **AND** sends verification email with token
- **AND** returns user info without tokens

#### Scenario: Duplicate email
- **WHEN** user submits email that already exists
- **THEN** system returns 409 Conflict with error "email_already_registered"

#### Scenario: Invalid password
- **WHEN** user submits password shorter than 8 characters
- **THEN** system returns 400 Bad Request with validation error

### Requirement: Email Verification
The system SHALL verify user email addresses before allowing login.

#### Scenario: Successful verification
- **WHEN** user clicks verification link with valid token
- **THEN** system sets email_verified to true
- **AND** changes user status from "pending" to "active"
- **AND** returns success message

#### Scenario: Expired token
- **WHEN** verification token is older than 24 hours
- **THEN** system returns 400 Bad Request with "token_expired"

#### Scenario: Invalid token
- **WHEN** verification token does not exist
- **THEN** system returns 400 Bad Request with "invalid_token"

#### Scenario: Resend verification
- **WHEN** user requests new verification email
- **THEN** system invalidates previous tokens
- **AND** sends new verification email

### Requirement: Password Login
The system SHALL authenticate users with email and password.

#### Scenario: Successful login
- **WHEN** user submits correct email and password
- **AND** user status is "active"
- **AND** email is verified
- **THEN** system returns access token and refresh token

#### Scenario: Unverified email
- **WHEN** user submits correct credentials but email not verified
- **THEN** system returns 403 Forbidden with "email_not_verified"

#### Scenario: Suspended user
- **WHEN** user submits correct credentials but status is "suspended"
- **THEN** system returns 403 Forbidden with "account_suspended"

#### Scenario: Invalid credentials
- **WHEN** user submits incorrect email or password
- **THEN** system returns 401 Unauthorized with "invalid_credentials"

### Requirement: Password Reset
The system SHALL allow users to reset forgotten passwords.

#### Scenario: Request password reset
- **WHEN** user requests password reset for existing email
- **THEN** system sends reset email with token (valid 1 hour)
- **AND** returns success regardless of email existence (security)

#### Scenario: Complete password reset
- **WHEN** user submits valid reset token and new password
- **THEN** system updates password hash
- **AND** invalidates all existing refresh tokens
- **AND** invalidates reset token

### Requirement: Change Password
The system SHALL allow authenticated users to change their password.

#### Scenario: Successful password change
- **WHEN** user provides correct current password and new password
- **THEN** system updates password hash
- **AND** invalidates all refresh tokens except current session

#### Scenario: Incorrect current password
- **WHEN** user provides incorrect current password
- **THEN** system returns 401 Unauthorized with "incorrect_password"

### Requirement: User Status Management
The system SHALL track user status through lifecycle states.

#### Scenario: Status transitions
- **WHEN** user is created via email registration
- **THEN** initial status is "pending"
- **WHEN** email is verified
- **THEN** status changes to "active"
- **WHEN** admin suspends user
- **THEN** status changes to "suspended"
- **WHEN** user deletes account
- **THEN** status changes to "deleted" (soft delete)

#### Scenario: Status validation
- **WHEN** request is made with user in non-active status
- **THEN** system returns appropriate error based on status

### Requirement: Account Deletion
The system SHALL allow users to delete their own accounts.

#### Scenario: Self-deletion with password
- **WHEN** user requests account deletion with correct password
- **THEN** system performs soft delete (status = "deleted")
- **AND** revokes all tokens
- **AND** cancels active subscription (if any)
- **AND** retains data for 30 days before hard delete

#### Scenario: OAuth user deletion
- **WHEN** OAuth user requests account deletion
- **THEN** system performs soft delete without password requirement

### Requirement: Admin User Management
The system SHALL allow administrators to manage users.

#### Scenario: List users
- **WHEN** admin requests user list with optional filters
- **THEN** system returns paginated user list with status, email, created_at

#### Scenario: Suspend user
- **WHEN** admin suspends a user with reason
- **THEN** system sets status to "suspended"
- **AND** records suspend_reason and suspended_at
- **AND** revokes all user tokens

#### Scenario: Reactivate user
- **WHEN** admin reactivates a suspended user
- **THEN** system sets status to "active"
- **AND** clears suspend fields

#### Scenario: Admin permission check
- **WHEN** non-admin user attempts admin operation
- **THEN** system returns 403 Forbidden

