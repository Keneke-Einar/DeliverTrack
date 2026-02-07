-- Revert location columns back to POINT (if needed)
-- Note: This will fail if data cannot be converted back to POINT format
ALTER TABLE deliveries ALTER COLUMN pickup_location TYPE POINT USING pickup_location::POINT;
ALTER TABLE deliveries ALTER COLUMN delivery_location TYPE POINT USING delivery_location::POINT;
ALTER TABLE couriers ALTER COLUMN current_location TYPE POINT USING current_location::POINT;