-- Testdata tables for the mysql-graph-visualizer
CREATE TABLE testdata_uzly (
    id INT PRIMARY KEY AUTO_INCREMENT,
    id_typu INT NOT NULL,
    infix VARCHAR(255),
    nazev VARCHAR(255),
    prefix VARCHAR(500)
);

CREATE TABLE testdata_uzly_php_action (
    id INT PRIMARY KEY AUTO_INCREMENT,
    id_node INT NOT NULL,
    php_code TEXT,
    FOREIGN KEY (id_node) REFERENCES testdata_uzly(id)
);

-- Sample test data
INSERT INTO testdata_uzly (id, id_typu, infix, nazev, prefix) VALUES 
(1, 17, 'test_action_1', 'Test PHP Action 1', '/sys/actions/test1'),
(2, 17, 'test_action_2', 'Test PHP Action 2', '/sys/actions/test2'),
(3, 17, 'test_action_3', 'Test PHP Action 3', '/sys/actions/test3'),
(4, 15, 'other_node', 'Other Node Type', '/sys/other/node1');

INSERT INTO testdata_uzly_php_action (id, id_node, php_code) VALUES 
(1, 1, '<?php echo "Hello from Action 1"; ?>'),
(2, 2, '<?php echo "Hello from Action 2"; ?>'),
(3, 3, '<?php echo "Hello from Action 3"; ?>');
