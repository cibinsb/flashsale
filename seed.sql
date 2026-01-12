-- seed.sql (CORRECTED)

-- Drop table if it exists
DROP TABLE IF EXISTS orders;

-- Create the table
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    item_sku VARCHAR(50) NOT NULL,
    quantity INTEGER NOT NULL,
    total_price NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert 10 seed orders with VALID UUIDs
INSERT INTO orders (id, user_id, item_sku, quantity, total_price) VALUES
    ('1e1d2c60-0a2b-4c3e-9f1d-1e1d2c600a2b', 'user_1001', 'ITEM-A-FLASH', 1, 99.99),
    ('2f2e3d71-1b3c-5d4f-a02e-2f2e3d711b3c', 'user_1002', 'ITEM-B-HOT', 2, 19.50),
    ('3a3f4e82-2c4d-6e5a-b13f-3a3f4e822c4d', 'user_1003', 'ITEM-C-RARE', 1, 500.00), -- Corrected from '3g3f4e82...'
    ('4b4a5f93-3d5e-7f6b-c24a-4b4a5f933d5e', 'user_1001', 'ITEM-D-COMMON', 5, 25.00), -- Corrected from '4h4g5f93...'
    ('5c5b6a04-4e6f-8a7c-d35b-5c5b6a044e6f', 'user_1004', 'ITEM-E-BIG', 1, 1999.99), -- Corrected from '5i5h6g04...'
    ('6d6c7b15-5f7a-9b8d-e46c-6d6c7b155f7a', 'user_1005', 'ITEM-A-FLASH', 3, 299.97), -- Corrected from '6j6i7h15...'
    ('7e7d8c26-6a8b-a9c0-f57d-7e7d8c266a8b', 'user_1006', 'ITEM-F-LUX', 1, 10000.00), -- Corrected from '7k7j8i26...'
    ('8f8e9d37-7b9c-b0d1-a68e-8f8e9d377b9c', 'user_1007', 'ITEM-B-HOT', 4, 39.00), -- Corrected from '8l8k9j37...'
    ('9a9f0e48-8c0d-c1e2-b79f-9a9f0e488c0d', 'user_1008', 'ITEM-A-FLASH', 1, 99.99), -- Corrected from '9m9l0k48...'
    ('0b0a1f59-9d1e-d2f3-c80a-0b0a1f599d1e', 'user_1009', 'ITEM-C-RARE', 2, 1000.00); -- Corrected from '0n0m1l59...'