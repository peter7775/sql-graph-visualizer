#!/bin/sh
# Measures server-side latency of the key demo queries (same SQL and literal values as in
# config/demo-config.yml) directly in the demo MySQL container. Useful to verify the
# before/after-index story without running the whole benchmark UI.
#
# Usage:
#   scripts/demo-query-timing.sh            # time queries with the current index state
#   scripts/demo-query-timing.sh apply      # create the demo indexes, then time
#   scripts/demo-query-timing.sh revert     # drop the demo indexes, then time
#   scripts/demo-query-timing.sh explain    # print EXPLAIN for each query
#
# Environment: MYSQL_CONTAINER (default mysql-demo), MYSQL_ROOT_PASSWORD (default demopass123),
#              MYSQL_DATABASE (default ecommerce_demo), RUNS (default 5)
set -eu

CONTAINER="${MYSQL_CONTAINER:-mysql-demo}"
PASSWORD="${MYSQL_ROOT_PASSWORD:-demopass123}"
DATABASE="${MYSQL_DATABASE:-ecommerce_demo}"
RUNS="${RUNS:-5}"

run_sql() {
  docker exec "$CONTAINER" mysql -uroot -p"$PASSWORD" -h127.0.0.1 -N -B "$DATABASE" -e "$1" 2>/dev/null
}

ACTION="${1:-time}"

case "$ACTION" in
  apply)
    run_sql "CREATE INDEX idx_orders_customer_total ON orders (customer_id, total_amount); CREATE INDEX idx_order_items_product_qty ON order_items (product_id, quantity); ANALYZE TABLE orders, order_items;" >/dev/null
    echo "Indexes created."
    ;;
  revert)
    run_sql "DROP INDEX idx_orders_customer_total ON orders; DROP INDEX idx_order_items_product_qty ON order_items; ANALYZE TABLE orders, order_items;" >/dev/null
    echo "Indexes dropped."
    ;;
esac

# name|query
QUERIES="orders_by_customer|SELECT o.id, o.status, o.total_amount, o.created_at, c.email FROM orders o JOIN customers c ON c.id = o.customer_id WHERE o.customer_id = 4242 ORDER BY o.id DESC LIMIT 10
order_items_by_order|SELECT oi.id, oi.quantity, oi.unit_price, p.name FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = 100000
product_units_sold|SELECT SUM(oi.quantity) AS units_sold FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.product_id = 1234
customer_profile|SELECT c.id, c.email, c.country, COUNT(o.id) AS orders_count FROM customers c LEFT JOIN orders o ON o.customer_id = c.id WHERE c.id = 4242 GROUP BY c.id, c.email, c.country
top_products_in_category|SELECT /*+ JOIN_ORDER(p, oi) */ p.id, p.name, SUM(oi.quantity) AS units FROM products p JOIN order_items oi ON oi.product_id = p.id WHERE p.category_id = 7 GROUP BY p.id, p.name ORDER BY units DESC LIMIT 10
top_customers_in_city|SELECT /*+ JOIN_ORDER(c, o) */ c.id, c.email, COUNT(o.id) AS orders_count, SUM(o.total_amount) AS spent FROM customers c JOIN orders o ON o.customer_id = c.id WHERE c.country = 'CZ' AND c.city = 'Prague' GROUP BY c.id, c.email ORDER BY spent DESC LIMIT 10
category_sales_summary|SELECT /*+ JOIN_ORDER(c, p, oi) */ c.name, COUNT(*) AS line_items, SUM(oi.quantity) AS units FROM categories c JOIN products p ON p.category_id = c.id JOIN order_items oi ON oi.product_id = p.id WHERE c.id = 7 GROUP BY c.id, c.name"

if [ "$ACTION" = "explain" ]; then
  echo "$QUERIES" | while IFS='|' read -r name query; do
    echo "== $name"
    run_sql "EXPLAIN $query"
  done
  exit 0
fi

echo "Server-side latency in ms, $RUNS runs per query (first run may include cold cache):"
echo "$QUERIES" | while IFS='|' read -r name query; do
  vals=""
  i=0
  while [ "$i" -lt "$RUNS" ]; do
    # The last output line is the elapsed time of the statement that ran before it.
    ms=$(run_sql "SET @t0 = NOW(6); $query; SELECT ROUND(TIMESTAMPDIFF(MICROSECOND, @t0, NOW(6)) / 1000, 1);" | tail -n 1)
    vals="$vals $ms"
    i=$((i + 1))
  done
  printf '%-26s%s\n' "$name" "$vals"
done
