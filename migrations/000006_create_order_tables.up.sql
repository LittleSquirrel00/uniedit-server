-- Order Module Tables
-- Orders, order items, and invoices

-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_no VARCHAR(50) NOT NULL UNIQUE,  -- ORD-YYYYMMDD-XXXXX
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,  -- subscription, topup, upgrade
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, paid, canceled, refunded, failed
    subtotal BIGINT NOT NULL DEFAULT 0,  -- in cents
    discount BIGINT NOT NULL DEFAULT 0,
    tax BIGINT NOT NULL DEFAULT 0,
    total BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) DEFAULT 'usd',
    plan_id VARCHAR(50) REFERENCES plans(id),  -- for subscription orders
    credits_amount BIGINT DEFAULT 0,  -- for topup orders
    stripe_payment_intent_id VARCHAR(255),
    stripe_invoice_id VARCHAR(255),
    paid_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,  -- payment expiration
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_order_no ON orders(order_no);
CREATE INDEX idx_orders_stripe_pi ON orders(stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;
CREATE INDEX idx_orders_expires ON orders(expires_at) WHERE status = 'pending';

-- Order items table
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity INT DEFAULT 1,
    unit_price BIGINT NOT NULL,  -- in cents
    amount BIGINT NOT NULL       -- quantity * unit_price
);

CREATE INDEX idx_order_items_order ON order_items(order_id);

-- Invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_no VARCHAR(50) NOT NULL UNIQUE,  -- INV-YYYYMMDD-XXXXX
    order_id UUID NOT NULL REFERENCES orders(id),
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(10) DEFAULT 'usd',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',  -- draft, finalized, paid, void
    pdf_url TEXT,
    stripe_invoice_id VARCHAR(255),
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_at TIMESTAMP WITH TIME ZONE NOT NULL,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_user ON invoices(user_id, issued_at DESC);
CREATE INDEX idx_invoices_order ON invoices(order_id);
CREATE INDEX idx_invoices_invoice_no ON invoices(invoice_no);

-- Trigger for orders updated_at
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE orders IS 'Purchase orders for subscriptions and top-ups';
COMMENT ON TABLE order_items IS 'Line items within an order';
COMMENT ON TABLE invoices IS 'Payment invoices generated from orders';
COMMENT ON COLUMN orders.order_no IS 'Human-readable order number: ORD-YYYYMMDD-XXXXX';
COMMENT ON COLUMN orders.type IS 'Order type: subscription, topup, upgrade';
COMMENT ON COLUMN orders.status IS 'Order status: pending, paid, canceled, refunded, failed';
