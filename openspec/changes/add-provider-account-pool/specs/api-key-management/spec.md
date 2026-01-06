# api-key-management Specification

## Purpose

Enhanced system API key management with IP whitelisting, audit logging, and auto-rotation.

## ADDED Requirements

### Requirement: IP Whitelist

The system SHALL support IP address restrictions for system API keys.

#### Scenario: Set IP whitelist

- **WHEN** admin updates key with allowed_ips list
- **THEN** the whitelist is stored
- **AND** takes effect immediately

#### Scenario: Validate request IP

- **WHEN** API request arrives with system API key
- **AND** key has non-empty allowed_ips
- **THEN** check request IP against whitelist
- **IF** IP not in whitelist
- **THEN** return 403 Forbidden with "ip_not_allowed"

#### Scenario: Empty whitelist

- **WHEN** key has empty allowed_ips array
- **THEN** allow requests from any IP

#### Scenario: IPv4 and IPv6 support

- **WHEN** allowed_ips contains CIDR notation (e.g., "192.168.1.0/24")
- **THEN** match against CIDR range
- **WHEN** allowed_ips contains single IP
- **THEN** exact match required

### Requirement: Audit Logging

The system SHALL maintain audit logs for API key operations.

#### Scenario: Log key creation

- **WHEN** user creates new API key
- **THEN** log action="created" with key_id, user_id, timestamp, IP

#### Scenario: Log key usage

- **WHEN** API key is used for request
- **THEN** log action="used" with request metadata
- **BUT** aggregate high-frequency logs (1 entry per minute per key)

#### Scenario: Log key rotation

- **WHEN** API key is rotated
- **THEN** log action="rotated" with old_prefix, new_prefix

#### Scenario: Log key deletion

- **WHEN** API key is deleted
- **THEN** log action="deleted" with deletion reason

#### Scenario: Query audit logs

- **WHEN** user requests audit logs for a key
- **THEN** return paginated logs with filters:
  - Time range
  - Action type
  - IP address

### Requirement: Auto-Rotation

The system SHALL support scheduled automatic key rotation.

#### Scenario: Schedule rotation

- **WHEN** user sets rotate_after_days for a key
- **THEN** store rotation schedule
- **AND** system checks daily for keys due rotation

#### Scenario: Execute rotation

- **WHEN** key's last_rotated_at + rotate_after_days < now
- **THEN** generate new key value
- **AND** update key_hash and key_prefix
- **AND** set last_rotated_at to now
- **AND** log rotation event

#### Scenario: Rotation notification

- **WHEN** key rotation is executed
- **THEN** notification is available via API
- **AND** old key remains valid for 24 hours grace period

#### Scenario: Cancel rotation

- **WHEN** user sets rotate_after_days to null
- **THEN** disable auto-rotation for this key
