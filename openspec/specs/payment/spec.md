# payment Specification

## Purpose
TBD - created by archiving change add-user-billing-payment. Update Purpose after archive.
## Requirements
### Requirement: Payment Provider Interface
The system SHALL abstract payment operations behind a provider interface.

#### Scenario: Provider abstraction
- **WHEN** payment operation is needed
- **THEN** system uses PaymentProvider interface with methods:
  - CreateCustomer(user) → customerID
  - CreatePaymentIntent(amount, currency, metadata) → intentID
  - CreateSubscription(customerID, priceID) → subscriptionID
  - CancelSubscription(subscriptionID, immediately)
  - CreateRefund(chargeID, amount) → refundID

#### Scenario: Stripe provider
- **WHEN** Stripe is configured
- **THEN** system registers StripeProvider implementation

### Requirement: Customer Management
The system SHALL create Stripe customers for users.

#### Scenario: Create customer on first payment
- **WHEN** user initiates first payment
- **AND** subscription.stripe_customer_id is empty
- **THEN** system creates Stripe customer with user email
- **AND** stores customer ID in subscription

#### Scenario: Use existing customer
- **WHEN** user has stripe_customer_id
- **THEN** system uses existing customer

### Requirement: Payment Intent Creation
The system SHALL create PaymentIntents for one-time charges.

#### Scenario: Create payment intent
- **WHEN** order is ready for payment
- **THEN** system creates Stripe PaymentIntent with:
  - amount = order.total
  - currency = order.currency
  - metadata.order_id = order.id
- **AND** returns client_secret for frontend

#### Scenario: Payment intent response
- **WHEN** payment intent is created
- **THEN** system returns:
  - payment_intent_id
  - client_secret
  - amount, currency

### Requirement: Subscription Creation
The system SHALL create Stripe subscriptions for recurring billing.

#### Scenario: Create subscription
- **WHEN** user subscribes to paid plan
- **THEN** system creates Stripe subscription with:
  - customer = stripe_customer_id
  - price = plan.stripe_price_id
- **AND** stores stripe_subscription_id

#### Scenario: Subscription with trial
- **WHEN** plan has trial period configured
- **THEN** system creates subscription with trial_end

### Requirement: Payment Method Management
The system SHALL manage user payment methods.

#### Scenario: Attach payment method
- **WHEN** user adds payment method
- **THEN** system attaches to Stripe customer
- **AND** optionally sets as default

#### Scenario: List payment methods
- **WHEN** user requests payment methods
- **THEN** system returns list with:
  - id, type (card)
  - last4, brand, exp_month, exp_year

#### Scenario: Detach payment method
- **WHEN** user removes payment method
- **THEN** system detaches from Stripe customer

### Requirement: Webhook Processing
The system SHALL process Stripe webhook events.

#### Scenario: Webhook verification
- **WHEN** webhook request is received
- **THEN** system verifies Stripe-Signature header
- **IF** invalid signature
- **THEN** returns 400 Bad Request

#### Scenario: Idempotent processing
- **WHEN** webhook event is received
- **THEN** system checks if event_id already processed
- **IF** already processed
- **THEN** returns 200 OK without re-processing

#### Scenario: Event recording
- **WHEN** webhook event is received
- **THEN** system stores event in stripe_webhook_events table
- **AND** marks as processed after handling

### Requirement: Payment Success Handling
The system SHALL handle successful payments.

#### Scenario: payment_intent.succeeded
- **WHEN** payment_intent.succeeded webhook received
- **THEN** system:
  - Finds order by metadata.order_id
  - Creates payment record with status "succeeded"
  - Marks order as paid
  - Triggers order fulfillment

#### Scenario: invoice.paid
- **WHEN** invoice.paid webhook received (subscription renewal)
- **THEN** system:
  - Updates subscription period dates
  - Resets monthly quota counters

### Requirement: Payment Failure Handling
The system SHALL handle failed payments.

#### Scenario: payment_intent.payment_failed
- **WHEN** payment_intent.payment_failed webhook received
- **THEN** system:
  - Creates payment record with status "failed"
  - Records failure_code and failure_message
  - Keeps order in pending status for retry

#### Scenario: invoice.payment_failed
- **WHEN** invoice.payment_failed webhook received
- **THEN** system:
  - Updates subscription status to past_due
  - Sends notification to user (future)

### Requirement: Subscription Updates
The system SHALL handle subscription status changes.

#### Scenario: customer.subscription.updated
- **WHEN** subscription.updated webhook received
- **THEN** system:
  - Updates subscription status
  - Updates period dates
  - Updates cancel_at_period_end flag

#### Scenario: customer.subscription.deleted
- **WHEN** subscription.deleted webhook received
- **THEN** system:
  - Sets subscription status to canceled
  - Reverts user to free plan
  - Clears stripe_subscription_id

### Requirement: Refund Processing
The system SHALL process refunds.

#### Scenario: Create refund
- **WHEN** admin initiates refund for paid order
- **THEN** system:
  - Creates Stripe refund via charge_id
  - Updates payment.refunded_amount
  - If full refund, sets order status to "refunded"

#### Scenario: Partial refund
- **WHEN** admin initiates partial refund
- **THEN** system:
  - Creates partial Stripe refund
  - Updates payment.refunded_amount
  - Order status remains "paid"

#### Scenario: Refund with credits reversal
- **WHEN** top-up order is refunded
- **THEN** system deducts credits from balance
- **IF** balance would go negative
- **THEN** returns error "insufficient_balance_for_refund"

### Requirement: Payment Record
The system SHALL maintain payment history.

#### Scenario: Payment record creation
- **WHEN** payment status changes
- **THEN** system creates/updates payment record with:
  - order_id, user_id
  - amount, currency, method
  - status (pending, processing, succeeded, failed, canceled, refunded)
  - provider = "stripe"
  - stripe_payment_intent_id
  - stripe_charge_id
  - timestamps (succeeded_at, failed_at)

#### Scenario: Get payment
- **WHEN** user requests payment details
- **THEN** system returns payment with order info

