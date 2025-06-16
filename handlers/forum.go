package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dgrijalva/jwt-go"
)

type Topic struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	StateID     int    `json:"state_id"`
	Likes       int    `json:"likes"`
	Dislikes    int    `json:"dislikes"`
	UserLike    *bool  `json:"user_like,omitempty"`
}

type Message struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	TopicID   int    `json:"topic_id"`
}

func GetTopicsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT t.topic_id, t.title, t.description, t.tags, t.user_id, u.username, t.state_id,
				(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = TRUE) as likes,
				(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = FALSE) as dislikes
			FROM topic t
			JOIN user u ON t.user_id = u.user_id
			ORDER BY t.created_at DESC
		`)
		if err != nil {
			log.Printf("Error fetching topics: %v", err)
			http.Error(w, "Error fetching topics", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var topics []Topic
		for rows.Next() {
			var topic Topic
			err := rows.Scan(
				&topic.ID,
				&topic.Title,
				&topic.Description,
				&topic.Tags,
				&topic.UserID,
				&topic.Username,
				&topic.StateID,
				&topic.Likes,
				&topic.Dislikes,
			)
			if err != nil {
				log.Printf("Error scanning topics: %v", err)
				http.Error(w, "Error scanning topics", http.StatusInternalServerError)
				return
			}
			topics = append(topics, topic)
		}

		json.NewEncoder(w).Encode(topics)
	}
}

func GetTopicHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topicID := r.URL.Query().Get("id")
		log.Printf("Topic ID from query: %s", topicID)

		if topicID == "" {
			log.Printf("No topic ID provided")
			http.Error(w, "Topic ID is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(topicID)
		if err != nil {
			log.Printf("Invalid topic ID: %s", topicID)
			http.Error(w, "Invalid topic ID", http.StatusBadRequest)
			return
		}
		log.Printf("Fetching topic with ID: %d", id)

		// Vérifions d'abord si le topic existe
		var topicExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM topic WHERE topic_id = ?)", id).Scan(&topicExists)
		if err != nil {
			log.Printf("Error checking if topic exists: %v", err)
			http.Error(w, "Error checking topic", http.StatusInternalServerError)
			return
		}
		if !topicExists {
			log.Printf("Topic with ID %d does not exist", id)
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}

		var topic Topic
		err = db.QueryRow(`
			SELECT t.topic_id, t.title, t.description, t.tags, t.user_id, u.username, t.state_id,
				(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = TRUE) as likes,
				(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = FALSE) as dislikes
			FROM topic t
			JOIN user u ON t.user_id = u.user_id
			WHERE t.topic_id = ?
		`, id).Scan(
			&topic.ID,
			&topic.Title,
			&topic.Description,
			&topic.Tags,
			&topic.UserID,
			&topic.Username,
			&topic.StateID,
			&topic.Likes,
			&topic.Dislikes,
		)
		if err != nil {
			log.Printf("Error scanning topic row: %v", err)
			http.Error(w, "Error processing topic data", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully fetched topic: %+v", topic)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(topic)
	}
}

func CreateTopicHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var topic Topic
		if err := json.NewDecoder(r.Body).Decode(&topic); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		log.Printf("Received topic data: %+v", topic)

		// Si aucun user_id n'est fourni, utiliser l'utilisateur par défaut (ID 1)
		if topic.UserID == 0 {
			topic.UserID = 1
		}

		// Vérifier si l'utilisateur existe
		var userExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM user WHERE user_id = ?)", topic.UserID).Scan(&userExists)
		if err != nil {
			log.Printf("Error checking if user exists: %v", err)
			http.Error(w, "Error checking user", http.StatusInternalServerError)
			return
		}
		if !userExists {
			log.Printf("User with ID %d does not exist", topic.UserID)
			http.Error(w, "User does not exist", http.StatusBadRequest)
			return
		}

		result, err := db.Exec(`
			INSERT INTO topic (title, description, tags, user_id, state_id)
			VALUES (?, ?, ?, ?, ?)
		`, topic.Title, topic.Description, topic.Tags, topic.UserID, topic.StateID)
		if err != nil {
			log.Printf("Error creating topic: %v", err)
			http.Error(w, "Error creating topic", http.StatusInternalServerError)
			return
		}

		topicID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting last insert ID: %v", err)
			http.Error(w, "Error getting topic ID", http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully created topic with ID: %d", topicID)

		topic.ID = int(topicID)
		// CreatedAt et UpdatedAt ne sont plus gérés ici

		// Récupérer le nom d'utilisateur
		err = db.QueryRow("SELECT username FROM user WHERE user_id = ?", topic.UserID).Scan(&topic.Username)
		if err != nil {
			log.Printf("Error fetching username: %v", err)
			http.Error(w, "Error fetching username", http.StatusInternalServerError)
			return
		}

		// Mettre à jour le nombre de topics de l'utilisateur
		_, err = db.Exec("UPDATE user SET topic_nbr = topic_nbr + 1 WHERE user_id = ?", topic.UserID)
		if err != nil {
			log.Printf("Error updating user topic count: %v", err)
			http.Error(w, "Error updating user topic count", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully created topic: %+v", topic)
		json.NewEncoder(w).Encode(topic)
	}
}

func LikeTopicHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		topicID := r.FormValue("id")
		if topicID == "" {
			http.Error(w, "Topic ID is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(topicID)
		if err != nil {
			http.Error(w, "Invalid topic ID", http.StatusBadRequest)
			return
		}

		// Récupérer l'utilisateur depuis le token JWT
		cookie, err := r.Cookie("token_form")
		if err != nil {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("votre_clé_secrète_jwt"), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Vérifier si l'utilisateur a déjà liké/disliké ce topic
		var existingLike *bool
		err = db.QueryRow("SELECT liked FROM topic_user_like WHERE user_id = ? AND topic_id = ?", claims.UserID, id).Scan(&existingLike)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existing like: %v", err)
			http.Error(w, "Error checking like status", http.StatusInternalServerError)
			return
		}

		if existingLike != nil {
			// Si l'utilisateur a déjà liké, on supprime le like
			if *existingLike {
				_, err = db.Exec("DELETE FROM topic_user_like WHERE user_id = ? AND topic_id = ?", claims.UserID, id)
			} else {
				// Si l'utilisateur a disliké, on change en like
				_, err = db.Exec("UPDATE topic_user_like SET liked = TRUE WHERE user_id = ? AND topic_id = ?", claims.UserID, id)
			}
		} else {
			// Ajouter un nouveau like
			_, err = db.Exec("INSERT INTO topic_user_like (user_id, topic_id, liked) VALUES (?, ?, TRUE)", claims.UserID, id)
		}

		if err != nil {
			log.Printf("Error updating like: %v", err)
			http.Error(w, "Error updating like", http.StatusInternalServerError)
			return
		}

		// Rediriger vers la page précédente
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func DislikeTopicHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		topicID := r.FormValue("id")
		if topicID == "" {
			http.Error(w, "Topic ID is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(topicID)
		if err != nil {
			http.Error(w, "Invalid topic ID", http.StatusBadRequest)
			return
		}

		// Récupérer l'utilisateur depuis le token JWT
		cookie, err := r.Cookie("token_form")
		if err != nil {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("votre_clé_secrète_jwt"), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Vérifier si l'utilisateur a déjà liké/disliké ce topic
		var existingLike *bool
		err = db.QueryRow("SELECT liked FROM topic_user_like WHERE user_id = ? AND topic_id = ?", claims.UserID, id).Scan(&existingLike)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existing like: %v", err)
			http.Error(w, "Error checking like status", http.StatusInternalServerError)
			return
		}

		if existingLike != nil {
			// Si l'utilisateur a déjà disliké, on supprime le dislike
			if !*existingLike {
				_, err = db.Exec("DELETE FROM topic_user_like WHERE user_id = ? AND topic_id = ?", claims.UserID, id)
			} else {
				// Si l'utilisateur a liké, on change en dislike
				_, err = db.Exec("UPDATE topic_user_like SET liked = FALSE WHERE user_id = ? AND topic_id = ?", claims.UserID, id)
			}
		} else {
			// Ajouter un nouveau dislike
			_, err = db.Exec("INSERT INTO topic_user_like (user_id, topic_id, liked) VALUES (?, ?, FALSE)", claims.UserID, id)
		}

		if err != nil {
			log.Printf("Error updating dislike: %v", err)
			http.Error(w, "Error updating dislike", http.StatusInternalServerError)
			return
		}

		// Rediriger vers la page précédente
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func GetMessagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		topicID := r.URL.Query().Get("topic_id")
		if topicID == "" {
			http.Error(w, "Topic ID is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(topicID)
		if err != nil {
			http.Error(w, "Invalid topic ID", http.StatusBadRequest)
			return
		}

		rows, err := db.Query(`
			SELECT m.message_id, m.content, m.created_at, m.user_id, u.username, m.topic_id
			FROM message m
			JOIN user u ON m.user_id = u.user_id
			WHERE m.topic_id = ?
			ORDER BY m.created_at ASC
		`, id)
		if err != nil {
			log.Printf("Error fetching messages: %v", err)
			http.Error(w, "Error fetching messages", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []Message
		for rows.Next() {
			var message Message
			err := rows.Scan(
				&message.ID,
				&message.Content,
				&message.CreatedAt,
				&message.UserID,
				&message.Username,
				&message.TopicID,
			)
			if err != nil {
				log.Printf("Error scanning messages: %v", err)
				http.Error(w, "Error scanning messages", http.StatusInternalServerError)
				return
			}
			messages = append(messages, message)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

func CreateMessageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Récupérer l'utilisateur depuis le token JWT
		cookie, err := r.Cookie("token_form")
		if err != nil {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("votre_clé_secrète_jwt"), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Récupérer les données du formulaire
		content := r.FormValue("content")
		topicID := r.FormValue("topic_id")

		if content == "" || topicID == "" {
			http.Error(w, "Content and topic ID are required", http.StatusBadRequest)
			return
		}

		// Insérer le nouveau message
		_, err = db.Exec(`
			INSERT INTO message (content, topic_id, user_id)
			VALUES (?, ?, ?)
		`, content, topicID, claims.UserID)
		if err != nil {
			log.Printf("Error creating message: %v", err)
			http.Error(w, "Error creating message", http.StatusInternalServerError)
			return
		}

		// Rediriger vers la page du topic
		http.Redirect(w, r, fmt.Sprintf("/topic?id=%s", topicID), http.StatusSeeOther)
	}
}
