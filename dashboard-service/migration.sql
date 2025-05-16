CREATE DATABASE dashboard_db;
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