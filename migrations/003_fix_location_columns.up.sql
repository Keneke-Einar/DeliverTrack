-- Fix location columns from POINT to TEXT
ALTER TABLE deliveries ALTER COLUMN pickup_location TYPE TEXT;
ALTER TABLE deliveries ALTER COLUMN delivery_location TYPE TEXT;
ALTER TABLE couriers ALTER COLUMN current_location TYPE TEXT;