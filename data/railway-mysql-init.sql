-- Railway MySQL Demo Data Initialization for Project Management System
-- This script creates comprehensive sample data for the SQL Graph Visualizer demo

-- Drop tables if they exist (in reverse dependency order)
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS task_dependencies;
DROP TABLE IF EXISTS project_skills;
DROP TABLE IF EXISTS user_skills;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;

-- Users table - core entity
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role ENUM('admin', 'manager', 'developer', 'analyst', 'designer') DEFAULT 'developer',
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
    category ENUM('programming', 'database', 'frontend', 'backend', 'devops', 'design', 'management') NOT NULL,
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

-- Insert sample users
INSERT INTO users (id, username, email, full_name, role, is_active) VALUES
(1, 'john_doe', 'john@company.com', 'John Doe', 'manager', TRUE),
(2, 'jane_smith', 'jane@company.com', 'Jane Smith', 'developer', TRUE),
(3, 'bob_johnson', 'bob@company.com', 'Bob Johnson', 'developer', TRUE),
(4, 'alice_brown', 'alice@company.com', 'Alice Brown', 'analyst', TRUE),
(5, 'charlie_wilson', 'charlie@company.com', 'Charlie Wilson', 'admin', TRUE),
(6, 'diana_davis', 'diana@company.com', 'Diana Davis', 'designer', TRUE),
(7, 'erik_anderson', 'erik@company.com', 'Erik Anderson', 'manager', TRUE),
(8, 'frank_miller', 'frank@company.com', 'Frank Miller', 'developer', TRUE),
(9, 'grace_lee', 'grace@company.com', 'Grace Lee', 'developer', TRUE),
(10, 'henry_taylor', 'henry@company.com', 'Henry Taylor', 'analyst', TRUE),
(11, 'irene_clark', 'irene@company.com', 'Irene Clark', 'designer', TRUE),
(12, 'jack_white', 'jack@company.com', 'Jack White', 'manager', TRUE);

-- Skills with expanded categories
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
(12, 'PostgreSQL', 'database', 'junior'),
(13, 'Kubernetes', 'devops', 'expert'),
(14, 'Vue.js', 'frontend', 'senior'),
(15, 'Django', 'backend', 'senior'),
(16, 'Project Management', 'management', 'senior'),
(17, 'Figma', 'design', 'junior'),
(18, 'TypeScript', 'programming', 'senior'),
(19, 'Redis', 'database', 'senior'),
(20, 'Jenkins', 'devops', 'senior');

-- Teams with detailed descriptions
INSERT INTO teams (id, name, description, team_lead_id) VALUES
(1, 'Backend Development', 'Server-side development, APIs, and microservices architecture', 1),
(2, 'Frontend Development', 'User interface and user experience development', 7),
(3, 'Data Analytics', 'Data analysis, visualization, and business intelligence', 4),
(4, 'DevOps & Infrastructure', 'Cloud infrastructure, deployment, and automation', 5),
(5, 'Design & UX', 'User experience design and visual design', 6),
(6, 'Mobile Development', 'iOS and Android application development', 12);

-- Expanded team memberships with various roles
INSERT INTO team_members (user_id, team_id, role) VALUES
(1, 1, 'lead'),
(2, 1, 'member'),
(3, 1, 'member'),
(8, 1, 'member'),
(4, 3, 'lead'),
(10, 3, 'member'),
(5, 4, 'lead'),
(6, 5, 'lead'),
(11, 5, 'member'),
(7, 2, 'lead'),
(9, 2, 'member'),
(2, 2, 'member'),  -- Jane is in multiple teams
(3, 4, 'member'),  -- Bob is in multiple teams
(12, 6, 'lead'),
(8, 6, 'member'),
(9, 6, 'member');

-- Comprehensive user skills mapping
INSERT INTO user_skills (user_id, skill_id, proficiency, years_experience) VALUES
-- John (Manager): Leadership + some technical
(1, 2, 'expert', 8),    -- Python
(1, 4, 'advanced', 5),  -- Node.js
(1, 5, 'advanced', 6),  -- MySQL
(1, 16, 'expert', 10),  -- Project Management

-- Jane (Developer): Full-stack with frontend focus
(2, 1, 'expert', 7),    -- JavaScript
(2, 3, 'expert', 5),    -- React
(2, 18, 'advanced', 4), -- TypeScript
(2, 2, 'intermediate', 3), -- Python
(2, 14, 'advanced', 3), -- Vue.js

-- Bob (Developer): Backend + DevOps
(3, 8, 'expert', 6),    -- Go
(3, 5, 'advanced', 5),  -- MySQL
(3, 6, 'advanced', 4),  -- Docker
(3, 13, 'intermediate', 2), -- Kubernetes
(3, 19, 'advanced', 3), -- Redis

-- Alice (Analyst): Data + some backend
(4, 2, 'expert', 6),    -- Python
(4, 5, 'expert', 7),    -- MySQL
(4, 12, 'advanced', 5), -- PostgreSQL
(4, 9, 'intermediate', 2), -- Neo4j

-- Charlie (Admin): DevOps expert
(5, 6, 'expert', 8),    -- Docker
(5, 7, 'expert', 6),    -- AWS
(5, 13, 'expert', 5),   -- Kubernetes
(5, 20, 'advanced', 4), -- Jenkins

-- Diana (Designer): Design skills
(6, 11, 'expert', 6),   -- UI/UX Design
(6, 17, 'expert', 4),   -- Figma
(6, 1, 'intermediate', 2), -- Some JavaScript for prototyping

-- Erik (Manager): Frontend + management
(7, 1, 'expert', 9),    -- JavaScript
(7, 10, 'advanced', 3), -- GraphQL
(7, 4, 'expert', 7),    -- Node.js
(7, 16, 'advanced', 6), -- Project Management

-- Frank (Developer): Backend specialist
(8, 2, 'expert', 5),    -- Python
(8, 15, 'advanced', 4), -- Django
(8, 12, 'advanced', 3), -- PostgreSQL
(8, 10, 'intermediate', 2), -- GraphQL

-- Grace (Developer): Frontend specialist
(9, 1, 'expert', 4),    -- JavaScript
(9, 3, 'expert', 3),    -- React
(9, 18, 'advanced', 2), -- TypeScript
(9, 14, 'advanced', 2), -- Vue.js

-- Henry (Analyst): Data analysis
(10, 2, 'advanced', 4), -- Python
(10, 5, 'advanced', 5), -- MySQL
(10, 12, 'intermediate', 2), -- PostgreSQL

-- Irene (Designer): Design + some frontend
(11, 11, 'expert', 5),  -- UI/UX Design
(11, 17, 'advanced', 3), -- Figma
(11, 1, 'beginner', 1), -- JavaScript

-- Jack (Manager): Mobile + management
(12, 16, 'expert', 8),  -- Project Management
(12, 1, 'advanced', 6), -- JavaScript
(12, 2, 'intermediate', 3); -- Python

-- Expanded projects with realistic budgets and timelines
INSERT INTO projects (id, name, description, status, priority, start_date, end_date, budget, team_id, created_by) VALUES
(1, 'E-commerce Platform Redesign', 'Complete overhaul of the existing e-commerce platform with microservices architecture', 'active', 'high', '2025-01-01', '2025-12-31', 450000.00, 1, 1),
(2, 'Data Analytics Dashboard', 'Interactive business intelligence dashboard for real-time analytics', 'active', 'medium', '2025-02-01', '2025-08-31', 280000.00, 3, 4),
(3, 'Mobile App Development', 'Cross-platform mobile application for customers', 'planning', 'high', '2025-03-01', '2025-11-30', 350000.00, 6, 12),
(4, 'Cloud Migration Initiative', 'Migrate entire infrastructure to AWS cloud', 'planning', 'critical', '2025-04-01', '2026-03-31', 600000.00, 4, 5),
(5, 'Design System Overhaul', 'Create comprehensive design system and component library', 'active', 'medium', '2025-01-15', '2025-07-31', 180000.00, 5, 6),
(6, 'API Gateway Implementation', 'Implement centralized API gateway for all services', 'active', 'high', '2025-02-15', '2025-09-30', 220000.00, 1, 7),
(7, 'Customer Support Portal', 'Self-service portal for customer support', 'planning', 'medium', '2025-03-15', '2025-10-31', 160000.00, 2, 7),
(8, 'Performance Optimization', 'System-wide performance improvements and monitoring', 'active', 'high', '2025-01-01', '2025-06-30', 120000.00, 4, 5);

-- Project skill requirements
INSERT INTO project_skills (project_id, skill_id, importance) VALUES
-- E-commerce Platform Redesign
(1, 1, 'critical'),    -- JavaScript
(1, 4, 'critical'),    -- Node.js
(1, 5, 'critical'),    -- MySQL
(1, 6, 'important'),   -- Docker
(1, 8, 'critical'),    -- Go
(1, 10, 'important'),  -- GraphQL

-- Data Analytics Dashboard
(2, 2, 'critical'),    -- Python
(2, 5, 'critical'),    -- MySQL
(2, 12, 'important'),  -- PostgreSQL
(2, 9, 'important'),   -- Neo4j

-- Mobile App Development
(3, 1, 'critical'),    -- JavaScript
(3, 3, 'critical'),    -- React
(3, 16, 'important'),  -- Project Management

-- Cloud Migration Initiative
(4, 6, 'critical'),    -- Docker
(4, 7, 'critical'),    -- AWS
(4, 13, 'critical'),   -- Kubernetes
(4, 20, 'important'),  -- Jenkins

-- Design System Overhaul
(5, 11, 'critical'),   -- UI/UX Design
(5, 17, 'critical'),   -- Figma
(5, 1, 'important'),   -- JavaScript for prototyping

-- API Gateway Implementation
(6, 8, 'critical'),    -- Go
(6, 10, 'critical'),   -- GraphQL
(6, 4, 'important'),   -- Node.js
(6, 6, 'important'),   -- Docker

-- Customer Support Portal
(7, 1, 'critical'),    -- JavaScript
(7, 3, 'critical'),    -- React
(7, 11, 'important'),  -- UI/UX Design

-- Performance Optimization
(8, 6, 'critical'),    -- Docker
(8, 13, 'important'),  -- Kubernetes
(8, 19, 'critical'),   -- Redis
(8, 7, 'important');   -- AWS

-- Comprehensive tasks with realistic hierarchy
INSERT INTO tasks (id, title, description, status, priority, estimated_hours, actual_hours, project_id, assigned_to, created_by, parent_task_id, due_date) VALUES
-- E-commerce Platform Redesign tasks
(1, 'Architecture Planning', 'Design microservices architecture and system overview', 'done', 'high', 80, 75, 1, 1, 1, NULL, '2025-01-31'),
(2, 'User Authentication Service', 'Implement OAuth2 and JWT-based authentication', 'in_progress', 'high', 120, 60, 1, 2, 1, 1, '2025-03-15'),
(3, 'Product Catalog API', 'RESTful API for product management', 'todo', 'medium', 100, 0, 1, 3, 1, 1, '2025-04-30'),
(4, 'Payment Gateway Integration', 'Integrate multiple payment providers', 'todo', 'urgent', 150, 0, 1, 8, 1, 2, '2025-05-31'),
(5, 'Order Management System', 'Complete order lifecycle management', 'todo', 'high', 180, 0, 1, 3, 1, 3, '2025-06-30'),

-- Data Analytics Dashboard tasks
(6, 'Data Pipeline Architecture', 'Design ETL pipeline for analytics data', 'done', 'high', 60, 58, 2, 4, 4, NULL, '2025-02-28'),
(7, 'Real-time Data Processing', 'Implement streaming data processing', 'in_progress', 'high', 120, 70, 2, 10, 4, 6, '2025-04-30'),
(8, 'Dashboard Frontend', 'React-based interactive dashboard', 'todo', 'medium', 100, 0, 2, 9, 4, 7, '2025-06-30'),
(9, 'Performance Metrics API', 'API for dashboard data and metrics', 'in_progress', 'medium', 80, 30, 2, 4, 4, 6, '2025-05-31'),

-- Mobile App Development tasks
(10, 'Mobile App Requirements', 'Gather and document mobile app requirements', 'done', 'high', 40, 42, 3, 12, 12, NULL, '2025-03-31'),
(11, 'UI/UX Design for Mobile', 'Create mobile-first design system', 'in_progress', 'high', 80, 25, 3, 11, 12, 10, '2025-05-31'),
(12, 'Cross-platform Framework Setup', 'Setup React Native development environment', 'todo', 'medium', 60, 0, 3, 8, 12, 10, '2025-04-30'),

-- Cloud Migration Initiative tasks
(13, 'Infrastructure Assessment', 'Evaluate current infrastructure for cloud migration', 'done', 'urgent', 100, 95, 4, 5, 5, NULL, '2025-04-30'),
(14, 'AWS Environment Setup', 'Configure production AWS environment', 'in_progress', 'high', 150, 80, 4, 5, 5, 13, '2025-06-30'),
(15, 'Migration Strategy Planning', 'Detailed migration plan with rollback procedures', 'in_progress', 'high', 80, 40, 4, 3, 5, 13, '2025-05-31'),

-- Design System Overhaul tasks
(16, 'Design Audit', 'Audit existing design components and patterns', 'done', 'medium', 60, 58, 5, 6, 6, NULL, '2025-02-28'),
(17, 'Component Library Creation', 'Build reusable component library in Figma', 'in_progress', 'high', 120, 70, 5, 6, 6, 16, '2025-05-31'),
(18, 'Documentation Website', 'Create documentation website for design system', 'todo', 'medium', 80, 0, 5, 11, 6, 17, '2025-07-31'),

-- API Gateway Implementation tasks
(19, 'Gateway Architecture Design', 'Design API gateway architecture and routing', 'in_progress', 'high', 80, 40, 6, 7, 7, NULL, '2025-04-30'),
(20, 'Authentication & Authorization', 'Implement gateway-level auth', 'todo', 'high', 100, 0, 6, 8, 7, 19, '2025-06-30'),
(21, 'Rate Limiting & Monitoring', 'Implement rate limiting and monitoring', 'todo', 'medium', 60, 0, 6, 3, 7, 19, '2025-07-31'),

-- Customer Support Portal tasks
(22, 'Portal Requirements Analysis', 'Analyze requirements for support portal', 'done', 'medium', 40, 38, 7, 7, 7, NULL, '2025-04-15'),
(23, 'Knowledge Base Integration', 'Integrate with existing knowledge base', 'todo', 'medium', 80, 0, 7, 9, 7, 22, '2025-07-31'),
(24, 'Ticket System Implementation', 'Build support ticket management system', 'todo', 'high', 120, 0, 7, 2, 7, 22, '2025-08-31'),

-- Performance Optimization tasks
(25, 'Performance Baseline', 'Establish current performance metrics', 'done', 'high', 40, 42, 8, 5, 5, NULL, '2025-02-28'),
(26, 'Database Query Optimization', 'Optimize slow database queries', 'in_progress', 'high', 80, 35, 8, 3, 5, 25, '2025-05-31'),
(27, 'Caching Layer Implementation', 'Implement Redis caching layer', 'todo', 'high', 100, 0, 8, 3, 5, 25, '2025-06-30'),
(28, 'Monitoring Dashboard Setup', 'Setup comprehensive monitoring', 'in_progress', 'medium', 60, 20, 8, 5, 5, 25, '2025-04-30');

-- Task dependencies with realistic workflow
INSERT INTO task_dependencies (dependent_task_id, prerequisite_task_id, dependency_type) VALUES
-- E-commerce dependencies
(2, 1, 'finish_to_start'),  -- Auth depends on Architecture
(3, 1, 'finish_to_start'),  -- Product API depends on Architecture
(4, 2, 'finish_to_start'),  -- Payment depends on Auth
(5, 3, 'finish_to_start'),  -- Orders depend on Product API

-- Data Analytics dependencies
(7, 6, 'finish_to_start'),  -- Real-time processing depends on Pipeline
(8, 7, 'finish_to_start'),  -- Frontend depends on Data Processing
(9, 6, 'finish_to_start'),  -- Metrics API depends on Pipeline

-- Mobile App dependencies
(11, 10, 'finish_to_start'), -- Design depends on Requirements
(12, 10, 'finish_to_start'), -- Framework setup depends on Requirements

-- Cloud Migration dependencies
(14, 13, 'finish_to_start'), -- AWS Setup depends on Assessment
(15, 13, 'finish_to_start'), -- Migration Plan depends on Assessment

-- Design System dependencies
(17, 16, 'finish_to_start'), -- Component Library depends on Audit
(18, 17, 'finish_to_start'), -- Documentation depends on Components

-- API Gateway dependencies
(20, 19, 'finish_to_start'), -- Auth depends on Architecture
(21, 19, 'finish_to_start'), -- Rate limiting depends on Architecture

-- Support Portal dependencies
(23, 22, 'finish_to_start'), -- Knowledge Base depends on Requirements
(24, 22, 'finish_to_start'), -- Ticket System depends on Requirements

-- Performance Optimization dependencies
(26, 25, 'finish_to_start'), -- Query optimization depends on Baseline
(27, 25, 'finish_to_start'), -- Caching depends on Baseline
(28, 25, 'finish_to_start'); -- Monitoring depends on Baseline

-- Comprehensive activity log
INSERT INTO activity_log (user_id, entity_type, entity_id, action, details) VALUES
(1, 'project', 1, 'created', 'Created E-commerce Platform Redesign project with $450K budget'),
(1, 'task', 1, 'created', 'Created architecture planning task'),
(1, 'task', 1, 'completed', 'Completed architecture planning on schedule'),
(2, 'task', 2, 'assigned', 'Assigned authentication service development'),
(4, 'project', 2, 'created', 'Created Data Analytics Dashboard project'),
(4, 'task', 6, 'completed', 'Completed data pipeline architecture design'),
(5, 'project', 4, 'created', 'Created Cloud Migration Initiative - critical priority'),
(5, 'task', 13, 'completed', 'Infrastructure assessment completed - ready for migration'),
(6, 'project', 5, 'created', 'Created Design System Overhaul project'),
(6, 'task', 16, 'completed', 'Design audit completed - found 47 inconsistencies'),
(7, 'project', 6, 'created', 'Created API Gateway Implementation project'),
(7, 'project', 7, 'created', 'Created Customer Support Portal project'),
(12, 'project', 3, 'created', 'Created Mobile App Development project'),
(12, 'task', 10, 'completed', 'Mobile app requirements gathering completed');