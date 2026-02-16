-- Database: braccet_tournament
-- Note: organizer_id references Auth Service user, validated via API call
CREATE TABLE tournaments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    organizer_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    game VARCHAR(100),
    format ENUM('single_elimination', 'double_elimination') NOT NULL,
    status ENUM('draft', 'registration', 'in_progress', 'completed', 'cancelled') DEFAULT 'draft',
    max_participants INT UNSIGNED,
    registration_open BOOLEAN DEFAULT FALSE,
    settings JSON,
    starts_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_status (status),
    INDEX idx_organizer (organizer_id)
);
