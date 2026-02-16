-- Database: braccet_bracket
-- Note: tournament_id and participant IDs are external references
CREATE TABLE matches (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tournament_id BIGINT UNSIGNED NOT NULL,
    bracket_type ENUM('winners', 'losers', 'grand_final') DEFAULT 'winners',
    round INT NOT NULL,
    position INT NOT NULL,
    participant1_id BIGINT UNSIGNED,
    participant2_id BIGINT UNSIGNED,
    participant1_name VARCHAR(100),
    participant2_name VARCHAR(100),
    winner_id BIGINT UNSIGNED,
    participant1_score INT,
    participant2_score INT,
    status ENUM('pending', 'ready', 'in_progress', 'completed') DEFAULT 'pending',
    scheduled_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    next_match_id BIGINT UNSIGNED,
    loser_match_id BIGINT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (next_match_id) REFERENCES matches(id) ON DELETE SET NULL,
    FOREIGN KEY (loser_match_id) REFERENCES matches(id) ON DELETE SET NULL,
    INDEX idx_tournament_round (tournament_id, bracket_type, round),
    UNIQUE KEY unique_match_position (tournament_id, bracket_type, round, position)
);
