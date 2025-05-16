CREATE DATABASE task_db;
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