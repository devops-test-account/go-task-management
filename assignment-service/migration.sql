CREATE DATABASE task_assignment_db;
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