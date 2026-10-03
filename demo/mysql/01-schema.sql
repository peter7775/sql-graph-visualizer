-- SQL Graph Visualizer - live benchmark demo: e-shop schema.
--
-- IMPORTANT (demo story): there is deliberately NO index and NO foreign key on
--   orders.customer_id   and   order_items.product_id
-- MySQL/InnoDB creates an index automatically for every FOREIGN KEY, which
-- would remove the "slow without index" part of the demo. All relations in this
-- schema are therefore logical only. The demo optimization
-- (config/demo-config.yml -> live_demo.optimization) creates exactly these two
-- indexes ("Apply suggested index") and drops them again ("Revert").
--
-- Ordinary indexes (primary keys, lookups that are not part of the story) are
-- kept, e.g. order_items(order_id), orders(created_at), products(category_id).

USE ecommerce_demo;

CREATE TABLE categories (
    id          INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name        VARCHAR(100) NOT NULL,
    description VARCHAR(255) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_categories_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE customers (
    id            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    email         VARCHAR(120) NOT NULL,
    first_name    VARCHAR(50)  NOT NULL,
    last_name     VARCHAR(50)  NOT NULL,
    country       CHAR(2)      NOT NULL,
    city          VARCHAR(60)  NOT NULL,
    is_active     TINYINT(1)   NOT NULL DEFAULT 1,
    created_at    DATETIME     NOT NULL,
    last_login_at DATETIME     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_customers_email (email),
    KEY idx_customers_country_city (country, city)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE products (
    id             INT UNSIGNED  NOT NULL AUTO_INCREMENT,
    category_id    INT UNSIGNED  NOT NULL,
    sku            VARCHAR(20)   NOT NULL,
    name           VARCHAR(120)  NOT NULL,
    price          DECIMAL(10,2) NOT NULL,
    stock_quantity INT           NOT NULL DEFAULT 0,
    is_active      TINYINT(1)    NOT NULL DEFAULT 1,
    rating         DECIMAL(3,2)  NOT NULL DEFAULT 0.00,
    created_at     DATETIME      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_products_sku (sku),
    KEY idx_products_category (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- NOTE: no index on customer_id (logical reference to customers.id).
CREATE TABLE orders (
    id           INT UNSIGNED  NOT NULL AUTO_INCREMENT,
    customer_id  INT UNSIGNED  NOT NULL,
    status       VARCHAR(16)   NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    created_at   DATETIME      NOT NULL,
    PRIMARY KEY (id),
    KEY idx_orders_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- NOTE: no index on product_id (logical reference to products.id).
CREATE TABLE order_items (
    id         INT UNSIGNED  NOT NULL AUTO_INCREMENT,
    order_id   INT UNSIGNED  NOT NULL,
    product_id INT UNSIGNED  NOT NULL,
    quantity   INT           NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_order_items_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
