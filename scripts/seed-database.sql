-- Ride Hailing Database Seed Script
-- This script creates sample data for local development and testing

-- Note: Passwords are hashed using bcrypt for 'password123'
-- Hash: $2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G

BEGIN;

-- Clear existing data (in reverse order of dependencies)
TRUNCATE TABLE payments, rides, wallets, drivers, users CASCADE;

-- Insert sample users (riders, drivers, and admin)
INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
-- Riders
('11111111-1111-1111-1111-111111111111', 'alice@example.com', '+1234567001', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Alice', 'Johnson', 'rider', true, true, NOW() - INTERVAL '90 days'),
('11111111-1111-1111-1111-111111111112', 'bob@example.com', '+1234567002', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Bob', 'Smith', 'rider', true, true, NOW() - INTERVAL '60 days'),
('11111111-1111-1111-1111-111111111113', 'carol@example.com', '+1234567003', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Carol', 'Williams', 'rider', true, true, NOW() - INTERVAL '45 days'),
('11111111-1111-1111-1111-111111111114', 'david@example.com', '+1234567004', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'David', 'Brown', 'rider', true, true, NOW() - INTERVAL '30 days'),
('11111111-1111-1111-1111-111111111115', 'eve@example.com', '+1234567005', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Eve', 'Davis', 'rider', true, false, NOW() - INTERVAL '7 days'),

-- Drivers
('22222222-2222-2222-2222-222222222221', 'driver1@example.com', '+1234567101', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Frank', 'Miller', 'driver', true, true, NOW() - INTERVAL '120 days'),
('22222222-2222-2222-2222-222222222222', 'driver2@example.com', '+1234567102', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Grace', 'Wilson', 'driver', true, true, NOW() - INTERVAL '100 days'),
('22222222-2222-2222-2222-222222222223', 'driver3@example.com', '+1234567103', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Henry', 'Moore', 'driver', true, true, NOW() - INTERVAL '80 days'),
('22222222-2222-2222-2222-222222222224', 'driver4@example.com', '+1234567104', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Ivy', 'Taylor', 'driver', true, true, NOW() - INTERVAL '60 days'),
('22222222-2222-2222-2222-222222222225', 'driver5@example.com', '+1234567105', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'Jack', 'Anderson', 'driver', true, true, NOW() - INTERVAL '40 days'),

-- Admin
('99999999-9999-9999-9999-999999999999', 'admin@example.com', '+1234567999', '$2a$10$FszDRKMgYDRCo2eOoA02iOyr.3NMpwr4owhVZ3rRJWGYU/KO/Sc0G', 'System', 'Admin', 'admin', true, true, NOW() - INTERVAL '365 days');

-- Insert driver details
INSERT INTO drivers (id, user_id, license_number, vehicle_model, vehicle_plate, vehicle_color, vehicle_year, is_available, is_online, rating, total_rides, current_latitude, current_longitude, last_location_update) VALUES
('d1111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'DL1234567', 'Toyota Camry', 'ABC123', 'Silver', 2020, true, true, 4.8, 245, 37.7749, -122.4194, NOW()),
('d1111111-1111-1111-1111-111111111112', '22222222-2222-2222-2222-222222222222', 'DL2345678', 'Honda Accord', 'XYZ789', 'Black', 2021, true, true, 4.9, 312, 37.7849, -122.4094, NOW()),
('d1111111-1111-1111-1111-111111111113', '22222222-2222-2222-2222-222222222223', 'DL3456789', 'Tesla Model 3', 'TES001', 'White', 2022, false, false, 4.7, 189, 37.7649, -122.4294, NOW() - INTERVAL '2 hours'),
('d1111111-1111-1111-1111-111111111114', '22222222-2222-2222-2222-222222222224', 'DL4567890', 'Chevrolet Malibu', 'CHV456', 'Blue', 2019, true, true, 4.6, 421, 37.7549, -122.4394, NOW()),
('d1111111-1111-1111-1111-111111111115', '22222222-2222-2222-2222-222222222225', 'DL5678901', 'Ford Fusion', 'FRD789', 'Red', 2021, true, false, 4.5, 156, 37.7949, -122.3994, NOW() - INTERVAL '1 hour');

-- Insert wallets for all users
INSERT INTO wallets (id, user_id, balance, currency) VALUES
-- Riders
('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 150.00, 'USD'),
('a1111111-1111-1111-1111-111111111112', '11111111-1111-1111-1111-111111111112', 75.50, 'USD'),
('a1111111-1111-1111-1111-111111111113', '11111111-1111-1111-1111-111111111113', 200.00, 'USD'),
('a1111111-1111-1111-1111-111111111114', '11111111-1111-1111-1111-111111111114', 50.00, 'USD'),
('a1111111-1111-1111-1111-111111111115', '11111111-1111-1111-1111-111111111115', 100.00, 'USD'),
-- Drivers
('a2222222-2222-2222-2222-222222222221', '22222222-2222-2222-2222-222222222221', 1250.00, 'USD'),
('a2222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 1580.00, 'USD'),
('a2222222-2222-2222-2222-222222222223', '22222222-2222-2222-2222-222222222223', 980.00, 'USD'),
('a2222222-2222-2222-2222-222222222224', '22222222-2222-2222-2222-222222222224', 2100.00, 'USD'),
('a2222222-2222-2222-2222-222222222225', '22222222-2222-2222-2222-222222222225', 750.00, 'USD');

-- Insert sample rides (various statuses)
INSERT INTO rides (id, rider_id, driver_id, status, pickup_latitude, pickup_longitude, pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address, estimated_distance, estimated_duration, estimated_fare, actual_distance, actual_duration, final_fare, surge_multiplier, requested_at, accepted_at, started_at, completed_at, rating, feedback) VALUES
-- Completed rides
('10000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 'completed', 37.7749, -122.4194, '123 Market St, San Francisco, CA', 37.7849, -122.4094, '456 Mission St, San Francisco, CA', 2.5, 12, 15.00, 2.3, 11, 14.50, 1.00, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days' + INTERVAL '2 minutes', NOW() - INTERVAL '5 days' + INTERVAL '5 minutes', NOW() - INTERVAL '5 days' + INTERVAL '16 minutes', 5, 'Great driver!'),
('10000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111112', '22222222-2222-2222-2222-222222222222', 'completed', 37.7849, -122.4094, '789 Howard St, San Francisco, CA', 37.7649, -122.4294, '321 Folsom St, San Francisco, CA', 3.2, 15, 18.00, 3.5, 17, 19.50, 1.20, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days' + INTERVAL '1 minute', NOW() - INTERVAL '4 days' + INTERVAL '4 minutes', NOW() - INTERVAL '4 days' + INTERVAL '21 minutes', 4, 'Good ride'),
('10000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111113', '22222222-2222-2222-2222-222222222221', 'completed', 37.7649, -122.4294, '555 California St, San Francisco, CA', 37.7949, -122.3994, '888 Montgomery St, San Francisco, CA', 5.1, 25, 28.00, 5.0, 24, 27.50, 1.00, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days' + INTERVAL '3 minutes', NOW() - INTERVAL '3 days' + INTERVAL '6 minutes', NOW() - INTERVAL '3 days' + INTERVAL '30 minutes', 5, 'Excellent service'),
('10000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111114', '22222222-2222-2222-2222-222222222224', 'completed', 37.7549, -122.4394, '111 Pine St, San Francisco, CA', 37.7749, -122.4194, '222 Bush St, San Francisco, CA', 1.8, 10, 12.00, 1.9, 11, 12.50, 1.00, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '2 minutes', NOW() - INTERVAL '2 days' + INTERVAL '5 minutes', NOW() - INTERVAL '2 days' + INTERVAL '16 minutes', 4, 'On time'),
('10000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'completed', 37.7949, -122.3994, '999 Broadway, San Francisco, CA', 37.7749, -122.4194, '777 Geary St, San Francisco, CA', 4.5, 20, 24.00, 4.7, 22, 25.00, 1.00, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '1 minute', NOW() - INTERVAL '1 day' + INTERVAL '4 minutes', NOW() - INTERVAL '1 day' + INTERVAL '26 minutes', 5, 'Perfect!'),

-- In-progress ride
('10000001-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111112', '22222222-2222-2222-2222-222222222221', 'in_progress', 37.7849, -122.4094, '333 Kearny St, San Francisco, CA', 37.7549, -122.4394, '444 Grant Ave, San Francisco, CA', 2.8, 14, 16.00, NULL, NULL, NULL, 1.00, NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '8 minutes', NOW() - INTERVAL '5 minutes', NULL, NULL, NULL),

-- Accepted ride (driver on the way)
('10000001-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111113', '22222222-2222-2222-2222-222222222224', 'accepted', 37.7649, -122.4294, '666 Van Ness Ave, San Francisco, CA', 37.7949, -122.3994, '555 Polk St, San Francisco, CA', 3.5, 18, 20.00, NULL, NULL, NULL, 1.00, NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '3 minutes', NULL, NULL, NULL, NULL),

-- Requested ride (waiting for driver)
('10000001-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111114', NULL, 'requested', 37.7749, -122.4194, '100 First St, San Francisco, CA', 37.7849, -122.4094, '200 Second St, San Francisco, CA', 1.5, 8, 10.00, NULL, NULL, NULL, 1.50, NOW() - INTERVAL '2 minutes', NULL, NULL, NULL, NULL, NULL),

-- Cancelled ride
('10000001-0000-0000-0000-000000000009', '11111111-1111-1111-1111-111111111115', '22222222-2222-2222-2222-222222222225', 'cancelled', 37.7949, -122.3994, '300 Third St, San Francisco, CA', 37.7549, -122.4394, '400 Fourth St, San Francisco, CA', 4.0, 20, 22.00, NULL, NULL, NULL, 1.00, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '58 minutes', NULL, NULL, NULL, NULL);

-- Insert payments for completed rides
INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, commission, driver_earnings, method, status, transaction_id, processed_at) VALUES
('20000001-0000-0000-0000-000000000001', '10000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222221', 14.50, 3.63, 10.87, 'card', 'completed', 'stripe_ch_1abc123', NOW() - INTERVAL '5 days' + INTERVAL '20 minutes'),
('20000001-0000-0000-0000-000000000002', '10000001-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111112', '22222222-2222-2222-2222-222222222222', 19.50, 4.88, 14.62, 'wallet', 'completed', 'wallet_tx_2xyz456', NOW() - INTERVAL '4 days' + INTERVAL '25 minutes'),
('20000001-0000-0000-0000-000000000003', '10000001-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111113', '22222222-2222-2222-2222-222222222221', 27.50, 6.88, 20.62, 'card', 'completed', 'stripe_ch_3def789', NOW() - INTERVAL '3 days' + INTERVAL '35 minutes'),
('20000001-0000-0000-0000-000000000004', '10000001-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111114', '22222222-2222-2222-2222-222222222224', 12.50, 3.13, 9.37, 'cash', 'completed', NULL, NOW() - INTERVAL '2 days' + INTERVAL '20 minutes'),
('20000001-0000-0000-0000-000000000005', '10000001-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 25.00, 6.25, 18.75, 'card', 'completed', 'stripe_ch_4ghi012', NOW() - INTERVAL '1 day' + INTERVAL '30 minutes');

COMMIT;

-- Verify data was inserted
SELECT 'Users:' as table_name, COUNT(*) as count FROM users
UNION ALL
SELECT 'Drivers:', COUNT(*) FROM drivers
UNION ALL
SELECT 'Wallets:', COUNT(*) FROM wallets
UNION ALL
SELECT 'Rides:', COUNT(*) FROM rides
UNION ALL
SELECT 'Payments:', COUNT(*) FROM payments;
