-- Railway MySQL Demo Data Initialization
-- This script creates sample data for the SQL Graph Visualizer demo

-- Drop tables if they exist
DROP TABLE IF EXISTS film_category;
DROP TABLE IF EXISTS film_actor;
DROP TABLE IF EXISTS actor;
DROP TABLE IF EXISTS film;  
DROP TABLE IF EXISTS category;

-- Create actors table
CREATE TABLE actor (
    actor_id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create films table  
CREATE TABLE film (
    film_id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    release_year YEAR,
    rating ENUM('G','PG','PG-13','R','NC-17') DEFAULT 'G',
    length INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create categories table
CREATE TABLE category (
    category_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create film_actor junction table
CREATE TABLE film_actor (
    film_id INT,
    actor_id INT,
    PRIMARY KEY (film_id, actor_id),
    FOREIGN KEY (film_id) REFERENCES film(film_id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES actor(actor_id) ON DELETE CASCADE
);

-- Create film_category junction table
CREATE TABLE film_category (
    film_id INT,
    category_id INT,
    PRIMARY KEY (film_id, category_id),
    FOREIGN KEY (film_id) REFERENCES film(film_id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES category(category_id) ON DELETE CASCADE
);

-- Insert sample actors
INSERT INTO actor (first_name, last_name) VALUES
('Tom', 'Hanks'),
('Brad', 'Pitt'),
('Leonardo', 'DiCaprio'),
('Will', 'Smith'),
('Johnny', 'Depp'),
('Robert', 'Downey Jr.'),
('Scarlett', 'Johansson'),
('Jennifer', 'Lawrence'),
('Emma', 'Stone'),
('Ryan', 'Gosling'),
('Christian', 'Bale'),
('Matt', 'Damon'),
('Morgan', 'Freeman'),
('Denzel', 'Washington'),
('Meryl', 'Streep'),
('Angelina', 'Jolie'),
('Julia', 'Roberts'),
('Sandra', 'Bullock'),
('George', 'Clooney'),
('Harrison', 'Ford');

-- Insert sample films
INSERT INTO film (title, description, release_year, rating, length) VALUES
('The Matrix', 'A computer programmer discovers reality is a simulation', 1999, 'R', 136),
('Inception', 'A thief who steals corporate secrets through dream-sharing technology', 2010, 'PG-13', 148),
('Titanic', 'A seventeen-year-old aristocrat falls in love with a kind but poor artist', 1997, 'PG-13', 194),
('The Shawshank Redemption', 'Two imprisoned men bond over years, finding solace and redemption', 1994, 'R', 142),
('Forrest Gump', 'The presidencies of Kennedy and Johnson through the eyes of an Alabama man', 1994, 'PG-13', 142),
('Pulp Fiction', 'The lives of two mob hitmen, a boxer, and others intertwine', 1994, 'R', 154),
('The Dark Knight', 'Batman raises the stakes in his war on crime with the Joker', 2008, 'PG-13', 152),
('Avatar', 'A paraplegic Marine dispatched to the moon Pandora on a unique mission', 2009, 'PG-13', 162),
('Avengers: Endgame', 'After Thanos, the Avengers assemble once more to reverse his actions', 2019, 'PG-13', 181),
('Iron Man', 'After being held captive, billionaire Tony Stark creates a suit of armor', 2008, 'PG-13', 126),
('Pirates of the Caribbean', 'Blacksmith Will Turner teams up with pirate Jack Sparrow', 2003, 'PG-13', 143),
('La La Land', 'A jazz musician and an aspiring actress fall in love in Los Angeles', 2016, 'PG-13', 128),
('The Wolf of Wall Street', 'Based on the true story of Jordan Belfort and his stockbrokerage firm', 2013, 'R', 180),
('Gravity', 'Two astronauts work together to survive after an accident in space', 2013, 'PG-13', 91),
('The Martian', 'An astronaut becomes stranded on Mars and must find a way to survive', 2015, 'PG-13', 144);

-- Insert sample categories
INSERT INTO category (name) VALUES
('Action'),
('Adventure'),
('Comedy'),
('Drama'),
('Horror'),
('Romance'),
('Sci-Fi'),
('Thriller'),
('Fantasy'),
('Animation');

-- Insert film-actor relationships (actors in films)
INSERT INTO film_actor (film_id, actor_id) VALUES
-- The Matrix
(1, 11), -- Christian Bale (placeholder)
-- Inception  
(2, 3),  -- Leonardo DiCaprio
-- Titanic
(3, 3),  -- Leonardo DiCaprio
-- The Shawshank Redemption
(4, 13), -- Morgan Freeman
-- Forrest Gump
(5, 1),  -- Tom Hanks
-- The Dark Knight
(7, 11), -- Christian Bale
-- Avatar
(8, 4),  -- Will Smith (placeholder)
-- Avengers: Endgame
(9, 6),  -- Robert Downey Jr.
-- Iron Man
(10, 6), -- Robert Downey Jr.
-- Pirates of the Caribbean
(11, 5), -- Johnny Depp
-- La La Land
(12, 9), -- Emma Stone
(12, 10), -- Ryan Gosling
-- The Wolf of Wall Street
(13, 3), -- Leonardo DiCaprio
-- Gravity
(14, 7), -- Scarlett Johansson (placeholder)
-- The Martian
(15, 12); -- Matt Damon

-- Insert film-category relationships
INSERT INTO film_category (film_id, category_id) VALUES
-- The Matrix
(1, 1), -- Action
(1, 7), -- Sci-Fi
-- Inception
(2, 1), -- Action
(2, 7), -- Sci-Fi
(2, 8), -- Thriller
-- Titanic
(3, 4), -- Drama
(3, 6), -- Romance
-- The Shawshank Redemption
(4, 4), -- Drama
-- Forrest Gump
(5, 4), -- Drama
(5, 6), -- Romance
-- Pulp Fiction
(6, 1), -- Action
(6, 4), -- Drama
(6, 8), -- Thriller
-- The Dark Knight
(7, 1), -- Action
(7, 4), -- Drama
(7, 8), -- Thriller
-- Avatar
(8, 1), -- Action
(8, 2), -- Adventure
(8, 7), -- Sci-Fi
-- Avengers: Endgame
(9, 1), -- Action
(9, 2), -- Adventure
(9, 7), -- Sci-Fi
-- Iron Man
(10, 1), -- Action
(10, 7), -- Sci-Fi
-- Pirates of the Caribbean
(11, 1), -- Action
(11, 2), -- Adventure
(11, 9), -- Fantasy
-- La La Land
(12, 4), -- Drama
(12, 6), -- Romance
-- The Wolf of Wall Street
(13, 4), -- Drama
-- Gravity
(14, 4), -- Drama
(14, 7), -- Sci-Fi
(14, 8), -- Thriller
-- The Martian
(15, 4), -- Drama
(15, 7); -- Sci-Fi