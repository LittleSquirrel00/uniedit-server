-- Rollback: Order Module Tables

-- Drop trigger
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;

-- Drop indexes
DROP INDEX IF EXISTS idx_invoices_user;
DROP INDEX IF EXISTS idx_invoices_order;
DROP INDEX IF EXISTS idx_invoices_invoice_no;
DROP INDEX IF EXISTS idx_order_items_order;
DROP INDEX IF EXISTS idx_orders_user;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_order_no;
DROP INDEX IF EXISTS idx_orders_stripe_pi;
DROP INDEX IF EXISTS idx_orders_expires;

-- Drop tables in order (respecting foreign keys)
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
