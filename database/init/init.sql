-- Create databases for each service
CREATE DATABASE IF NOT EXISTS user_db;
CREATE DATABASE IF NOT EXISTS task_db;
CREATE DATABASE IF NOT EXISTS task_assignment_db;
CREATE DATABASE IF NOT EXISTS notification_db;
CREATE DATABASE IF NOT EXISTS dashboard_db;

-- Create a common user for all services
CREATE USER IF NOT EXISTS 'taskuser'@'%' IDENTIFIED BY 'taskpassword';
GRANT ALL PRIVILEGES ON user_db.* TO 'taskuser'@'%';
GRANT ALL PRIVILEGES ON task_db.* TO 'taskuser'@'%';
GRANT ALL PRIVILEGES ON task_assignment_db.* TO 'taskuser'@'%';
GRANT ALL PRIVILEGES ON notification_db.* TO 'taskuser'@'%';
GRANT ALL PRIVILEGES ON dashboard_db.* TO 'taskuser'@'%';
FLUSH PRIVILEGES;


USE user_db;

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed initial data
INSERT INTO users (username, email, password) VALUES 
('admin', 'admin@example.com', '$2a$10$g1dHbu4wmGQbvMV9Jqo1Du5d./ix3rhdzzHObnsEBUk/snjFDxC7q');  -- password: admin 


USE task_db;

CREATE TABLE tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT,
    status ENUM('TODO', 'IN_PROGRESS', 'DONE') DEFAULT 'TODO',
    created_by INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES user_db.users(id)
);

-- Seed initial data
INSERT INTO tasks (title, description, status, created_by) VALUES 
('First Task', 'This is the first task', 'TODO', 1),
('Second Task', 'This is the second task', 'IN_PROGRESS', 1);


USE task_assignment_db;

CREATE TABLE task_assignments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    task_id INT NOT NULL,
    assigned_to INT NOT NULL,
    assigned_by INT NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED') DEFAULT 'ASSIGNED',
    FOREIGN KEY (task_id) REFERENCES task_db.tasks(id),
    FOREIGN KEY (assigned_to) REFERENCES user_db.users(id),
    FOREIGN KEY (assigned_by) REFERENCES user_db.users(id)
);

-- Seed initial data
INSERT INTO task_assignments (task_id, assigned_to, assigned_by, status) VALUES 
(1, 1, 1, 'ASSIGNED'),
(2, 1, 1, 'IN_PROGRESS');


USE notification_db;

CREATE TABLE IF NOT EXISTS notifications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    message TEXT NOT NULL,
    type ENUM('TASK_ASSIGNED', 'TASK_UPDATED', 'TASK_COMPLETED') NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user_db.users(id)
);

-- Seed initial data
INSERT INTO notifications (user_id, message, type, is_read) VALUES 
(1, 'New task assigned to you', 'TASK_ASSIGNED', FALSE),
(1, 'Task status updated', 'TASK_UPDATED', FALSE);


USE dashboard_db;

-- This database will primarily use views and joins from other databases
-- Ensure foreign key relationships with other service databases

-- Create a view to aggregate task information
CREATE VIEW user_task_summary AS
SELECT 
    u.id AS user_id,
    u.username,
    COUNT(ta.task_id) AS total_tasks,
    SUM(CASE WHEN t.status = 'TODO' THEN 1 ELSE 0 END) AS todo_tasks,
    SUM(CASE WHEN t.status = 'IN_PROGRESS' THEN 1 ELSE 0 END) AS in_progress_tasks,
    SUM(CASE WHEN t.status = 'COMPLETED' THEN 1 ELSE 0 END) AS completed_tasks
FROM 
    user_db.users u
LEFT JOIN 
    task_assignment_db.task_assignments ta ON u.id = ta.assigned_to
LEFT JOIN 
    task_db.tasks t ON ta.task_id = t.id
GROUP BY 
    u.id, u.username;