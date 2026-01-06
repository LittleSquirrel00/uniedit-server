# order Specification

## Purpose
TBD - created by archiving change add-user-billing-payment. Update Purpose after archive.
## Requirements
### Requirement: Order Creation
The system SHALL create orders for subscription and top-up purchases.

#### Scenario: Create subscription order
- **WHEN** user requests to subscribe to a plan
- **THEN** system creates order with:
  - type = "subscription"
  - plan_id = selected plan
  - subtotal = plan price
  - total = subtotal - discount + tax
  - status = "pending"
  - expires_at = now + 30 minutes

#### Scenario: Create top-up order
- **WHEN** user requests to add credits
- **THEN** system creates order with:
  - type = "topup"
  - credits_amount = requested amount
  - total = credits_amount (1:1 USD cents)
  - status = "pending"

#### Scenario: Create upgrade order
- **WHEN** user upgrades subscription
- **THEN** system creates order with:
  - type = "upgrade"
  - plan_id = new plan
  - subtotal = prorated amount from Stripe
  - status = "pending"

#### Scenario: Order number generation
- **WHEN** order is created
- **THEN** system generates unique order_no in format: ORD-YYYYMMDD-XXXXX

### Requirement: Order State Machine
The system SHALL enforce valid order state transitions.

#### Scenario: Valid transitions
- **WHEN** order state changes
- **THEN** only these transitions are allowed:
  - pending → paid
  - pending → canceled
  - pending → failed
  - paid → refunded
  - failed → pending (retry)

#### Scenario: Invalid transition
- **WHEN** invalid state transition is attempted
- **THEN** system returns error "invalid_state_transition"

### Requirement: Order Fulfillment
The system SHALL fulfill orders upon successful payment.

#### Scenario: Subscription order fulfillment
- **WHEN** subscription order is marked paid
- **THEN** system:
  - Creates or updates Stripe subscription
  - Updates user subscription record
  - Sets subscription period dates

#### Scenario: Top-up order fulfillment
- **WHEN** top-up order is marked paid
- **THEN** system adds credits_amount to user balance

#### Scenario: Upgrade order fulfillment
- **WHEN** upgrade order is marked paid
- **THEN** system:
  - Updates subscription plan immediately
  - Adjusts Stripe subscription

### Requirement: Order Queries
The system SHALL provide order lookup capabilities.

#### Scenario: Get order by ID
- **WHEN** user requests order by ID
- **THEN** system returns order with items and payment info
- **IF** order belongs to different user
- **THEN** returns 403 Forbidden

#### Scenario: List user orders
- **WHEN** user requests order history
- **THEN** system returns paginated list with filters:
  - status (pending, paid, canceled, refunded)
  - type (subscription, topup, upgrade)
  - date range
- **AND** orders sorted by created_at DESC

### Requirement: Order Cancellation
The system SHALL allow order cancellation.

#### Scenario: Cancel pending order
- **WHEN** user cancels order in pending status
- **THEN** system:
  - Sets status to "canceled"
  - Sets canceled_at timestamp
  - Cancels Stripe PaymentIntent if exists

#### Scenario: Cancel non-pending order
- **WHEN** user attempts to cancel paid order
- **THEN** system returns error "order_not_cancelable"

### Requirement: Order Expiration
The system SHALL automatically expire stale orders.

#### Scenario: Expire pending order
- **WHEN** pending order passes expires_at
- **THEN** system sets status to "failed"
- **AND** cancels associated PaymentIntent

### Requirement: Invoice Generation
The system SHALL generate invoices for paid orders.

#### Scenario: Generate invoice
- **WHEN** order is marked paid
- **THEN** system creates invoice with:
  - invoice_no: INV-YYYYMMDD-XXXXX
  - amount, currency
  - status = "paid"
  - issued_at, paid_at

#### Scenario: Get invoice
- **WHEN** user requests invoice
- **THEN** system returns invoice details
- **IF** pdf_url exists, include download link

#### Scenario: List invoices
- **WHEN** user requests invoice history
- **THEN** system returns all invoices for user
- **AND** sorted by issued_at DESC

### Requirement: Order Items
The system SHALL track line items within orders.

#### Scenario: Order with items
- **WHEN** order is created
- **THEN** system creates order_items with:
  - description (e.g., "Pro Plan - Monthly")
  - quantity (default 1)
  - unit_price
  - amount = quantity * unit_price

