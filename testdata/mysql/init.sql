-- Complex database schema for mysql-graph-visualizer demonstration
-- This schema shows various relationship types, SQL views, and custom transformations

-- Users table - core entity
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role ENUM('admin', 'manager', 'developer', 'analyst') DEFAULT 'developer',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Teams table
CREATE TABLE teams (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    team_lead_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (team_lead_id) REFERENCES users(id)
);

-- Skills table - for many-to-many relationship
CREATE TABLE skills (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL UNIQUE,
    category ENUM('programming', 'database', 'frontend', 'backend', 'devops', 'design') NOT NULL,
    level_required ENUM('junior', 'senior', 'expert') DEFAULT 'junior'
);

-- Projects table
CREATE TABLE projects (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    status ENUM('planning', 'active', 'on_hold', 'completed', 'cancelled') DEFAULT 'planning',
    priority ENUM('low', 'medium', 'high', 'critical') DEFAULT 'medium',
    start_date DATE,
    end_date DATE,
    budget DECIMAL(10,2),
    team_id INT,
    created_by INT NOT NULL,
    FOREIGN KEY (team_id) REFERENCES teams(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Tasks table - shows hierarchical relationships
CREATE TABLE tasks (
    id INT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('todo', 'in_progress', 'testing', 'done', 'blocked') DEFAULT 'todo',
    priority ENUM('low', 'medium', 'high', 'urgent') DEFAULT 'medium',
    estimated_hours INT,
    actual_hours INT,
    project_id INT NOT NULL,
    assigned_to INT,
    created_by INT NOT NULL,
    parent_task_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    due_date DATE,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (assigned_to) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
);

-- User-Team membership (many-to-many)
CREATE TABLE team_members (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    team_id INT NOT NULL,
    role ENUM('member', 'lead', 'coordinator') DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (team_id) REFERENCES teams(id),
    UNIQUE KEY unique_membership (user_id, team_id)
);

-- User-Skills mapping (many-to-many with proficiency)
CREATE TABLE user_skills (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    skill_id INT NOT NULL,
    proficiency ENUM('beginner', 'intermediate', 'advanced', 'expert') NOT NULL,
    years_experience INT DEFAULT 0,
    last_used DATE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (skill_id) REFERENCES skills(id),
    UNIQUE KEY unique_user_skill (user_id, skill_id)
);

-- Project-Skills requirements (many-to-many)
CREATE TABLE project_skills (
    id INT PRIMARY KEY AUTO_INCREMENT,
    project_id INT NOT NULL,
    skill_id INT NOT NULL,
    importance ENUM('nice_to_have', 'important', 'critical') DEFAULT 'important',
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (skill_id) REFERENCES skills(id),
    UNIQUE KEY unique_project_skill (project_id, skill_id)
);

-- Task dependencies (self-referencing many-to-many)
CREATE TABLE task_dependencies (
    id INT PRIMARY KEY AUTO_INCREMENT,
    dependent_task_id INT NOT NULL,
    prerequisite_task_id INT NOT NULL,
    dependency_type ENUM('finish_to_start', 'start_to_start', 'finish_to_finish') DEFAULT 'finish_to_start',
    FOREIGN KEY (dependent_task_id) REFERENCES tasks(id),
    FOREIGN KEY (prerequisite_task_id) REFERENCES tasks(id),
    UNIQUE KEY unique_dependency (dependent_task_id, prerequisite_task_id)
);

-- Activity log for tracking changes
CREATE TABLE activity_log (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    entity_type ENUM('project', 'task', 'team', 'user') NOT NULL,
    entity_id INT NOT NULL,
    action ENUM('created', 'updated', 'deleted', 'assigned', 'completed') NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Insert sample data

-- Users
INSERT INTO users (id, username, email, full_name, role, is_active) VALUES
(1, 'john_doe', 'john@company.com', 'John Doe', 'manager', TRUE),
(2, 'jane_smith', 'jane@company.com', 'Jane Smith', 'developer', TRUE),
(3, 'bob_johnson', 'bob@company.com', 'Bob Johnson', 'developer', TRUE),
(4, 'alice_brown', 'alice@company.com', 'Alice Brown', 'analyst', TRUE),
(5, 'charlie_wilson', 'charlie@company.com', 'Charlie Wilson', 'admin', TRUE),
(6, 'diana_davis', 'diana@company.com', 'Diana Davis', 'developer', TRUE),
(7, 'erik_anderson', 'erik@company.com', 'Erik Anderson', 'manager', TRUE);

-- Skills
INSERT INTO skills (id, name, category, level_required) VALUES
(1, 'JavaScript', 'programming', 'junior'),
(2, 'Python', 'programming', 'junior'),
(3, 'React', 'frontend', 'senior'),
(4, 'Node.js', 'backend', 'senior'),
(5, 'MySQL', 'database', 'junior'),
(6, 'Docker', 'devops', 'senior'),
(7, 'AWS', 'devops', 'expert'),
(8, 'Go', 'programming', 'senior'),
(9, 'Neo4j', 'database', 'expert'),
(10, 'GraphQL', 'backend', 'senior'),
(11, 'UI/UX Design', 'design', 'senior'),
(12, 'PostgreSQL', 'database', 'junior');

-- Teams
INSERT INTO teams (id, name, description, team_lead_id) VALUES
(1, 'Backend Development', 'Server-side development and APIs', 1),
(2, 'Frontend Development', 'User interface and experience', 7),
(3, 'Data Analytics', 'Data analysis and visualization', 4),
(4, 'DevOps', 'Infrastructure and deployment', 5);

-- Team memberships
INSERT INTO team_members (user_id, team_id, role) VALUES
(1, 1, 'lead'),
(2, 1, 'member'),
(3, 1, 'member'),
(4, 3, 'lead'),
(5, 4, 'lead'),
(6, 2, 'member'),
(7, 2, 'lead'),
(2, 2, 'member'),  -- Jane is in multiple teams
(3, 4, 'member');  -- Bob is in multiple teams

-- User skills
INSERT INTO user_skills (user_id, skill_id, proficiency, years_experience) VALUES
(1, 2, 'expert', 8),    -- John: Python
(1, 4, 'advanced', 5),  -- John: Node.js
(1, 5, 'advanced', 6),  -- John: MySQL
(2, 1, 'expert', 7),    -- Jane: JavaScript
(2, 3, 'advanced', 4),  -- Jane: React
(2, 2, 'intermediate', 3), -- Jane: Python
(3, 8, 'expert', 6),    -- Bob: Go
(3, 5, 'advanced', 5),  -- Bob: MySQL
(3, 6, 'intermediate', 2), -- Bob: Docker
(4, 2, 'advanced', 4),  -- Alice: Python
(4, 5, 'expert', 7),    -- Alice: MySQL
(4, 12, 'advanced', 5), -- Alice: PostgreSQL
(5, 6, 'expert', 8),    -- Charlie: Docker
(5, 7, 'expert', 6),    -- Charlie: AWS
(6, 1, 'advanced', 5),  -- Diana: JavaScript
(6, 3, 'expert', 6),    -- Diana: React
(6, 11, 'advanced', 4), -- Diana: UI/UX Design
(7, 1, 'expert', 9),    -- Erik: JavaScript
(7, 10, 'advanced', 3), -- Erik: GraphQL
(7, 4, 'expert', 7);    -- Erik: Node.js

-- Projects
INSERT INTO projects (id, name, description, status, priority, start_date, end_date, budget, team_id, created_by) VALUES
(1, 'E-commerce Platform', 'Modern e-commerce solution with microservices', 'active', 'high', '2025-01-01', '2025-12-31', 250000.00, 1, 1),
(2, 'Data Visualization Dashboard', 'Interactive dashboard for business analytics', 'active', 'medium', '2025-02-01', '2025-08-31', 150000.00, 3, 4),
(3, 'Mobile App Backend', 'API services for mobile application', 'planning', 'high', '2025-03-01', '2025-10-31', 180000.00, 1, 1),
(4, 'User Interface Redesign', 'Complete UI/UX overhaul of existing system', 'active', 'medium', '2025-01-15', '2025-06-30', 80000.00, 2, 7),
(5, 'Cloud Migration', 'Migrate on-premise infrastructure to AWS', 'planning', 'critical', '2025-04-01', '2025-12-31', 300000.00, 4, 5);

-- Project skill requirements
INSERT INTO project_skills (project_id, skill_id, importance) VALUES
(1, 1, 'critical'),    -- E-commerce: JavaScript
(1, 4, 'critical'),    -- E-commerce: Node.js
(1, 5, 'important'),   -- E-commerce: MySQL
(1, 6, 'important'),   -- E-commerce: Docker
(2, 2, 'critical'),    -- Dashboard: Python
(2, 5, 'critical'),    -- Dashboard: MySQL
(2, 12, 'important'),  -- Dashboard: PostgreSQL
(3, 8, 'critical'),    -- Mobile Backend: Go
(3, 9, 'important'),   -- Mobile Backend: Neo4j
(3, 10, 'important'),  -- Mobile Backend: GraphQL
(4, 1, 'critical'),    -- UI Redesign: JavaScript
(4, 3, 'critical'),    -- UI Redesign: React
(4, 11, 'critical'),   -- UI Redesign: UI/UX Design
(5, 6, 'critical'),    -- Cloud Migration: Docker
(5, 7, 'critical');    -- Cloud Migration: AWS

-- Tasks
INSERT INTO tasks (id, title, description, status, priority, estimated_hours, actual_hours, project_id, assigned_to, created_by, parent_task_id, due_date) VALUES
-- E-commerce Platform tasks
(1, 'Setup Project Architecture', 'Design microservices architecture', 'done', 'high', 40, 38, 1, 1, 1, NULL, '2025-01-15'),
(2, 'User Authentication Service', 'Implement JWT-based authentication', 'in_progress', 'high', 60, 25, 1, 2, 1, 1, '2025-02-15'),
(3, 'Product Catalog API', 'REST API for product management', 'todo', 'medium', 80, 0, 1, 3, 1, 1, '2025-03-01'),
(4, 'Payment Gateway Integration', 'Integrate Stripe payment system', 'todo', 'urgent', 100, 0, 1, 2, 1, 2, '2025-03-15'),
(5, 'Order Management System', 'Handle order lifecycle', 'todo', 'high', 120, 0, 1, 3, 1, 3, '2025-04-01'),

-- Data Visualization Dashboard tasks
(6, 'Database Schema Design', 'Design analytics database schema', 'done', 'high', 30, 32, 2, 4, 4, NULL, '2025-02-10'),
(7, 'Data ETL Pipeline', 'Extract, Transform, Load data pipeline', 'in_progress', 'high', 80, 40, 2, 4, 4, 6, '2025-03-15'),
(8, 'Frontend Dashboard Components', 'Create reusable chart components', 'todo', 'medium', 60, 0, 2, 6, 4, 7, '2025-04-01'),

-- UI Redesign tasks
(9, 'User Research', 'Conduct user interviews and surveys', 'done', 'high', 40, 45, 4, 6, 7, NULL, '2025-02-01'),
(10, 'Wireframes and Mockups', 'Create detailed UI mockups', 'in_progress', 'high', 60, 20, 4, 6, 7, 9, '2025-02-28'),
(11, 'Component Library', 'Build reusable UI components', 'todo', 'medium', 80, 0, 4, 2, 7, 10, '2025-03-31'),

-- Cloud Migration tasks
(12, 'Infrastructure Assessment', 'Evaluate current infrastructure', 'todo', 'urgent', 50, 0, 5, 5, 5, NULL, '2025-04-15'),
(13, 'AWS Environment Setup', 'Configure AWS services', 'todo', 'high', 100, 0, 5, 5, 5, 12, '2025-05-01'),
(14, 'Data Migration Plan', 'Plan database migration strategy', 'todo', 'high', 60, 0, 5, 4, 5, 12, '2025-05-15');

-- Task dependencies
INSERT INTO task_dependencies (dependent_task_id, prerequisite_task_id, dependency_type) VALUES
(2, 1, 'finish_to_start'),  -- Auth depends on Architecture
(3, 1, 'finish_to_start'),  -- Product API depends on Architecture  
(4, 2, 'finish_to_start'),  -- Payment depends on Auth
(5, 3, 'finish_to_start'),  -- Orders depend on Product API
(7, 6, 'finish_to_start'),  -- ETL depends on Schema
(8, 7, 'finish_to_start'),  -- Frontend depends on ETL
(10, 9, 'finish_to_start'), -- Mockups depend on Research
(11, 10, 'finish_to_start'), -- Components depend on Mockups
(13, 12, 'finish_to_start'), -- AWS Setup depends on Assessment
(14, 12, 'finish_to_start'); -- Migration Plan depends on Assessment

-- Activity log
INSERT INTO activity_log (user_id, entity_type, entity_id, action, details) VALUES
(1, 'project', 1, 'created', 'Created E-commerce Platform project'),
(1, 'task', 1, 'created', 'Created project architecture task'),
(1, 'task', 1, 'completed', 'Completed project architecture setup'),
(2, 'task', 2, 'assigned', 'Assigned authentication service task'),
(4, 'project', 2, 'created', 'Created Data Visualization Dashboard project'),
(4, 'task', 6, 'completed', 'Completed database schema design'),
(7, 'project', 4, 'created', 'Created UI Redesign project'),
(6, 'task', 9, 'completed', 'Completed user research phase'),
(5, 'project', 5, 'created', 'Created Cloud Migration project');
