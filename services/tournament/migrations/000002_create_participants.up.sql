-- Database: braccet_tournament
-- Note: user_id references Auth Service user, validated via API call
CREATE TABLE participants (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tournament_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    seed INT UNSIGNED,
    status ENUM('registered', 'checked_in', 'active', 'eliminated', 'disqualified') DEFAULT 'registered',
    checked_in_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    UNIQUE KEY unique_participant (tournament_id, user_id),
    INDEX idx_tournament_seed (tournament_id, seed)
);
