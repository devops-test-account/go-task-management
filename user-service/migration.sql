CREATE DATABASE user_db;
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