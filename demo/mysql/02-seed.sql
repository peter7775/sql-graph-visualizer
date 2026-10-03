-- SQL Graph Visualizer - live benchmark demo: bulk seed.
--
-- Row counts (deterministic, no RAND(), so every demo start looks the same):
--   categories   20
--   customers    20 000
--   products      2 500   (125 per category)
--   orders      200 000   (exactly 10 per customer)
--   order_items ~600 000  (1..5 per order, best sellers skewed to low product ids)
--
-- Sized so that a query WITHOUT the demo indexes (full scan of orders / order_items)
-- takes tens to a few hundred milliseconds on a laptop-class machine, while the same
-- query WITH the index takes about a millisecond. Increase the numbers (and the
-- demo_seq upper bound) for a bigger, slower dataset.
--
-- Runs inside the MySQL docker-entrypoint init phase (executed once, on an empty
-- data volume). Plain INSERT ... SELECT over small helper tables, in batches of
-- 100k orders to keep transactions small.

USE ecommerce_demo;

SET SESSION sql_log_bin = 0;
SET SESSION unique_checks = 0;
SET SESSION foreign_key_checks = 0;

-- ---------------------------------------------------------------------------
-- Helper number tables (dropped at the end)
-- ---------------------------------------------------------------------------
CREATE TABLE demo_digits (d TINYINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB;
INSERT INTO demo_digits (d) VALUES (0),(1),(2),(3),(4),(5),(6),(7),(8),(9);

CREATE TABLE demo_seq (n INT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB;
INSERT INTO demo_seq (n)
SELECT n FROM (
    SELECT a.d + b.d * 10 + c.d * 100 + d4.d * 1000 + e.d * 10000 + f.d * 100000 AS n
    FROM demo_digits a, demo_digits b, demo_digits c, demo_digits d4, demo_digits e, demo_digits f
) x
WHERE n BETWEEN 1 AND 200000;

CREATE TABLE demo_k (k TINYINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB;
INSERT INTO demo_k (k) VALUES (1),(2),(3),(4),(5);

-- ---------------------------------------------------------------------------
-- categories (20)
-- ---------------------------------------------------------------------------
INSERT INTO categories (id, name, description)
SELECT n,
       ELT(n, 'Laptops', 'Phones', 'Tablets', 'Monitors', 'Keyboards',
              'Mice', 'Headphones', 'Speakers', 'Cameras', 'Printers',
              'Networking', 'Storage', 'Gaming', 'Wearables', 'Smart Home',
              'Cables', 'Chargers', 'Components', 'Software', 'Accessories'),
       CONCAT('Demo category #', n)
FROM demo_seq
WHERE n <= 20;

-- ---------------------------------------------------------------------------
-- customers (20 000)
-- ---------------------------------------------------------------------------
INSERT INTO customers (id, email, first_name, last_name, country, city, is_active, created_at, last_login_at)
SELECT n,
       CONCAT('customer', n, '@example.com'),
       ELT(1 + n % 10, 'Anna', 'Petr', 'Jana', 'Martin', 'Eva', 'Tomas', 'Lucie', 'Jakub', 'Marie', 'David'),
       ELT(1 + (n DIV 10) % 10, 'Novak', 'Svoboda', 'Dvorak', 'Cerny', 'Prochazka', 'Kucera', 'Vesely', 'Horak', 'Nemec', 'Marek'),
       ELT(1 + n % 8, 'CZ', 'SK', 'DE', 'PL', 'AT', 'FR', 'US', 'GB'),
       ELT(1 + (n DIV 8) % 8, 'Prague', 'Brno', 'Ostrava', 'Plzen', 'Liberec', 'Olomouc', 'Kosice', 'Vienna'),
       IF(n % 25 = 0, 0, 1),
       TIMESTAMPADD(DAY, n % 700, '2023-01-01 08:00:00'),
       NULL
FROM demo_seq
WHERE n <= 20000;

-- ---------------------------------------------------------------------------
-- products (2 500; 125 per category)
-- ---------------------------------------------------------------------------
INSERT INTO products (id, category_id, sku, name, price, stock_quantity, is_active, rating, created_at)
SELECT n,
       1 + n % 20,
       CONCAT('SKU-', LPAD(n, 6, '0')),
       CONCAT('Product ', n),
       ROUND(5 + (n * 37 % 49500) / 100, 2),
       100 + n % 900,
       IF(n % 40 = 0, 0, 1),
       ROUND(3 + (n % 21) / 10, 2),
       TIMESTAMPADD(DAY, n % 500, '2023-01-01 08:00:00')
FROM demo_seq
WHERE n <= 2500;

-- ---------------------------------------------------------------------------
-- orders (200 000) in 2 batches; customer_id = 1 + (n * 7919) % 20000 gives
-- every customer exactly 10 orders (7919 is coprime with 20000).
-- ---------------------------------------------------------------------------
INSERT INTO orders (id, customer_id, status, total_amount, created_at)
SELECT n,
       1 + (n * 7919) % 20000,
       ELT(1 + (n * 13) % 10, 'delivered', 'delivered', 'delivered', 'delivered', 'delivered',
                              'delivered', 'shipped', 'processing', 'pending', 'cancelled'),
       ROUND(10 + (n * 53 % 49000) / 100, 2),
       TIMESTAMPADD(SECOND, (n * 173) % 63072000, '2023-10-01 00:00:00')
FROM demo_seq WHERE n BETWEEN 1 AND 100000;

INSERT INTO orders (id, customer_id, status, total_amount, created_at)
SELECT n,
       1 + (n * 7919) % 20000,
       ELT(1 + (n * 13) % 10, 'delivered', 'delivered', 'delivered', 'delivered', 'delivered',
                              'delivered', 'shipped', 'processing', 'pending', 'cancelled'),
       ROUND(10 + (n * 53 % 49000) / 100, 2),
       TIMESTAMPADD(SECOND, (n * 173) % 63072000, '2023-10-01 00:00:00')
FROM demo_seq WHERE n BETWEEN 100001 AND 200000;

-- ---------------------------------------------------------------------------
-- order_items (~600k): order n gets 1 + n % 5 items. Product choice is skewed
-- (squared uniform) so low product ids are "best sellers".
-- ---------------------------------------------------------------------------
INSERT INTO order_items (order_id, product_id, quantity, unit_price)
SELECT o.id,
       1 + FLOOR(2500 * POW(((o.id * 31 + k.k * 977) % 10007) / 10007, 2)),
       1 + (o.id + k.k) % 4,
       ROUND(5 + ((o.id * 31 + k.k * 977) % 49500) / 100, 2)
FROM orders o JOIN demo_k k ON k.k <= 1 + o.id % 5
WHERE o.id BETWEEN 1 AND 100000
ORDER BY o.id, k.k;

INSERT INTO order_items (order_id, product_id, quantity, unit_price)
SELECT o.id,
       1 + FLOOR(2500 * POW(((o.id * 31 + k.k * 977) % 10007) / 10007, 2)),
       1 + (o.id + k.k) % 4,
       ROUND(5 + ((o.id * 31 + k.k * 977) % 49500) / 100, 2)
FROM orders o JOIN demo_k k ON k.k <= 1 + o.id % 5
WHERE o.id BETWEEN 100001 AND 200000
ORDER BY o.id, k.k;

-- ---------------------------------------------------------------------------
-- Cleanup + fresh optimizer statistics
-- ---------------------------------------------------------------------------
DROP TABLE demo_k;
DROP TABLE demo_seq;
DROP TABLE demo_digits;

ANALYZE TABLE categories, customers, products, orders, order_items;
