-- Table des rôles
CREATE TABLE role (
    role_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL
);

-- Table des états
CREATE TABLE state (
    state_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL
);

-- Table des utilisateurs
CREATE TABLE user (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    mail VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    bio TEXT,
    last_connection DATETIME,
    topic_nbr INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    profil_img_path VARCHAR(255),
    role_id INT,
    FOREIGN KEY (role_id) REFERENCES role(role_id)
);

-- Table des topics
CREATE TABLE topic (
    topic_id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    tags VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    state_id INT,
    user_id INT,
    FOREIGN KEY (state_id) REFERENCES state(state_id),
    FOREIGN KEY (user_id) REFERENCES user(user_id)
);

-- Table des messages
CREATE TABLE message (
    message_id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    topic_id INT,
    user_id INT,
    FOREIGN KEY (topic_id) REFERENCES topic(topic_id),
    FOREIGN KEY (user_id) REFERENCES user(user_id)
);

-- Table des réponses
CREATE TABLE response (
    response_id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    message_id INT,
    user_id INT,
    FOREIGN KEY (message_id) REFERENCES message(message_id),
    FOREIGN KEY (user_id) REFERENCES user(user_id)
);

-- Table des likes sur les réponses
CREATE TABLE response_user_like (
    user_id INT,
    response_id INT,
    liked BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (user_id, response_id),
    FOREIGN KEY (user_id) REFERENCES user(user_id),
    FOREIGN KEY (response_id) REFERENCES response(response_id)
);

-- Table des likes sur les topics
CREATE TABLE topic_user_like (
    user_id INT,
    topic_id INT,
    liked BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (user_id, topic_id),
    FOREIGN KEY (user_id) REFERENCES user(user_id),
    FOREIGN KEY (topic_id) REFERENCES topic(topic_id)
);

-- Insertion des données par défaut
INSERT INTO role (name) VALUES ('user');
INSERT INTO state (name) VALUES ('ouvert');
INSERT INTO user (username, mail, role_id) VALUES ('default_user', 'default@example.com', 1);

-- Ajouter la colonne tags à la table topic
ALTER TABLE topic ADD COLUMN tags VARCHAR(255);

SHOW COLUMNS FROM topic LIKE 'tags';

