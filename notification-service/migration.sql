CREATE DATABASE IF NOT EXISTS notification_db;
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