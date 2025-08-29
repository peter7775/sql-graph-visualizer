CREATE TABLE users (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    email VARCHAR(255),
    role VARCHAR(100)
);

CREATE TABLE departments (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    code VARCHAR(50)
);

CREATE TABLE user_departments (
    user_id INT,
    department_id INT,
    start_date DATE,
    role VARCHAR(100),
    PRIMARY KEY (user_id, department_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (department_id) REFERENCES departments(id)
);

CREATE TABLE department_collaborations (
    department_id_1 INT,
    department_id_2 INT,
    start_date DATE,
    num_projects INT,
    PRIMARY KEY (department_id_1, department_id_2),
    FOREIGN KEY (department_id_1) REFERENCES departments(id),
    FOREIGN KEY (department_id_2) REFERENCES departments(id)
);

-- test data
INSERT INTO users VALUES 
(1, 'John Doe', 'john@example.com', 'Developer'),
(2, 'Jane Smith', 'jane@example.com', 'Manager');

INSERT INTO departments VALUES
(1, 'IT Department', 'IT001'),
(2, 'HR Department', 'HR001');

INSERT INTO user_departments VALUES
(1, 1, '2024-01-01', 'Senior Developer'),
(2, 2, '2024-01-01', 'HR Manager');

INSERT INTO department_collaborations VALUES
(1, 2, '2024-01-01', 3); 