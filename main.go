package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"forum/config"
	"forum/handlers"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func initDB() {
	var err error
	// Configuration pour WampServer par défaut
	dsn := "root:@tcp(localhost:3306)/forum"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Récupérer le paramètre de tri
	sortBy := r.URL.Query().Get("sort")

	// Récupérer le tag sélectionné
	selectedTag := r.URL.Query().Get("tags")

	// Construire la requête SQL de base
	baseQuery := `
		SELECT t.topic_id, t.title, t.description, t.tags, t.user_id, u.username, t.state_id,
			(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = TRUE) as likes,
			(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = FALSE) as dislikes
		FROM topic t
		JOIN user u ON t.user_id = u.user_id
	`

	// Ajouter le filtre par tag si un tag est sélectionné
	var whereClause string
	if selectedTag != "" {
		whereClause = " WHERE t.tags LIKE ? OR t.tags LIKE ? OR t.tags LIKE ? OR t.tags = ?"
	}

	// Ajouter l'ordre de tri approprié
	var orderBy string
	switch sortBy {
	case "likes":
		orderBy = "ORDER BY likes DESC"
	case "dislikes":
		orderBy = "ORDER BY dislikes DESC"
	default:
		orderBy = "ORDER BY t.created_at DESC"
	}

	// Construire la requête finale
	query := baseQuery
	if whereClause != "" {
		query += whereClause
	}
	query += " " + orderBy

	// Exécuter la requête
	var rows *sql.Rows
	var err error
	if selectedTag != "" {
		// Préparer les paramètres pour la recherche de tag
		// Format: "tag, ", ", tag, ", ", tag", "tag"
		tagPatterns := []string{
			selectedTag + ", %",         // tag au début
			"%, " + selectedTag + ", %", // tag au milieu
			"%, " + selectedTag,         // tag à la fin
			selectedTag,                 // tag unique
		}
		rows, err = db.Query(query, tagPatterns[0], tagPatterns[1], tagPatterns[2], tagPatterns[3])
	} else {
		rows, err = db.Query(query)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var topics []struct {
		ID          int
		Title       string
		Description string
		Tags        string
		UserID      int
		Username    string
		StateID     int
		Likes       int
		Dislikes    int
	}

	for rows.Next() {
		var topic struct {
			ID          int
			Title       string
			Description string
			Tags        string
			UserID      int
			Username    string
			StateID     int
			Likes       int
			Dislikes    int
		}
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		topics = append(topics, topic)
	}

	// Récupérer l'utilisateur connecté
	cookie, _ := r.Cookie("token_form")
	var username string
	if cookie != nil {
		claims := &handlers.Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("votre_clé_secrète_jwt"), nil
		})
		if err == nil && token.Valid {
			username = claims.Username
		}
	}

	// Préparer les thèmes avec leur état de sélection
	themes := make([]struct {
		ID       string
		Label    string
		Selected bool
	}, len(config.Themes))
	for i, theme := range config.Themes {
		themes[i] = struct {
			ID       string
			Label    string
			Selected bool
		}{
			ID:       theme.ID,
			Label:    theme.Label,
			Selected: theme.ID == selectedTag,
		}
	}

	// Préparer les données pour le template
	data := struct {
		Topics []struct {
			ID          int
			Title       string
			Description string
			Tags        string
			UserID      int
			Username    string
			StateID     int
			Likes       int
			Dislikes    int
		}
		Username string
		Themes   []struct {
			ID       string
			Label    string
			Selected bool
		}
		SortBy       string
		SelectedTags string
	}{
		Topics:       topics,
		Username:     username,
		Themes:       themes,
		SortBy:       sortBy,
		SelectedTags: selectedTag,
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

// Fonction utilitaire pour vérifier si une chaîne est dans un tableau
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func topicPageHandler(w http.ResponseWriter, r *http.Request) {
	topicID := r.URL.Query().Get("id")
	if topicID == "" {
		http.Error(w, "Topic ID is required", http.StatusBadRequest)
		return
	}

	// Récupérer les détails du topic
	var topic struct {
		ID          int
		Title       string
		Description string
		Tags        string
		UserID      int
		Username    string
		StateID     int
		Likes       int
		Dislikes    int
	}

	err := db.QueryRow(`
		SELECT t.topic_id, t.title, t.description, t.tags, t.user_id, u.username, t.state_id,
			(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = TRUE) as likes,
			(SELECT COUNT(*) FROM topic_user_like WHERE topic_id = t.topic_id AND liked = FALSE) as dislikes
		FROM topic t
		JOIN user u ON t.user_id = u.user_id
		WHERE t.topic_id = ?
	`, topicID).Scan(
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Récupérer les messages du topic
	rows, err := db.Query(`
		SELECT m.message_id, m.content, m.created_at, m.user_id, u.username
		FROM message m
		JOIN user u ON m.user_id = u.user_id
		WHERE m.topic_id = ?
		ORDER BY m.created_at ASC
	`, topicID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []struct {
		ID        int
		Content   string
		CreatedAt string
		UserID    int
		Username  string
	}

	for rows.Next() {
		var message struct {
			ID        int
			Content   string
			CreatedAt string
			UserID    int
			Username  string
		}
		err := rows.Scan(
			&message.ID,
			&message.Content,
			&message.CreatedAt,
			&message.UserID,
			&message.Username,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, message)
	}

	// Récupérer l'utilisateur connecté
	cookie, _ := r.Cookie("token_form")
	var username string
	if cookie != nil {
		claims := &handlers.Claims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("votre_clé_secrète_jwt"), nil
		})
		if err == nil && token.Valid {
			username = claims.Username
		}
	}

	// Préparer les données pour le template
	data := struct {
		Topic struct {
			ID          int
			Title       string
			Description string
			Tags        string
			UserID      int
			Username    string
			StateID     int
			Likes       int
			Dislikes    int
		}
		Messages []struct {
			ID        int
			Content   string
			CreatedAt string
			UserID    int
			Username  string
		}
		Username string
	}{
		Topic:    topic,
		Messages: messages,
		Username: username,
	}

	tmpl, err := template.ParseFiles("templates/topic.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func createTopicHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Récupérer le token
	cookie, err := r.Cookie("token_form")
	if err != nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Parser le token JWT
	claims := &handlers.Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("votre_clé_secrète_jwt"), nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Récupérer les données du formulaire
	title := r.FormValue("title")
	description := r.FormValue("description")
	tags := r.Form["tags"]                 // Récupère tous les tags sélectionnés
	tagsString := strings.Join(tags, ", ") // Convertit le tableau en chaîne séparée par des virgules

	// Insérer le nouveau topic
	_, err = db.Exec(`
		INSERT INTO topic (title, description, tags, user_id, state_id)
		VALUES (?, ?, ?, ?, ?)
	`, title, description, tagsString, claims.UserID, 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mettre à jour le nombre de topics de l'utilisateur
	_, err = db.Exec("UPDATE user SET topic_nbr = topic_nbr + 1 WHERE user_id = ?", claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Rediriger vers la page d'accueil
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func createMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Récupérer le token
	cookie, err := r.Cookie("token_form")
	if err != nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Parser le token JWT
	claims := &handlers.Claims{}
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

	// Insérer le nouveau message
	_, err = db.Exec(`
		INSERT INTO message (content, topic_id, user_id)
		VALUES (?, ?, ?)
	`, content, topicID, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Rediriger vers la page du topic
	http.Redirect(w, r, fmt.Sprintf("/topic?id=%s", topicID), http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Supprimer le cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "token_form",
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
	})

	// Rediriger vers la page de connexion
	http.Redirect(w, r, "/register", http.StatusSeeOther)
}

func main() {
	initDB()
	// Servir les fichiers statiques
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes publiques
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token_form")
		if err != nil {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}
		fmt.Println(cookie.Value)
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/index", http.StatusSeeOther)
			return
		}
		http.NotFound(w, r)
	})
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/api/auth/login", handlers.LoginHandler(db))
	http.HandleFunc("/api/auth/register", handlers.RegisterHandler(db))

	// Routes protégées
	http.HandleFunc("/index", handlers.AuthMiddleware(indexHandler))
	http.HandleFunc("/topic", handlers.AuthMiddleware(topicPageHandler))
	http.HandleFunc("/topics", handlers.AuthMiddleware(createTopicHandler))
	http.HandleFunc("/api/messages", handlers.AuthMiddleware(createMessageHandler))
	http.HandleFunc("/logout", handlers.AuthMiddleware(logoutHandler))

	http.HandleFunc("/api/topic", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetTopicHandler(db)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/topic/like", handlers.AuthMiddleware(handlers.LikeTopicHandler(db)))
	http.HandleFunc("/api/topic/dislike", handlers.AuthMiddleware(handlers.DislikeTopicHandler(db)))

	// Démarrage du serveur
	log.Println("Serveur démarré sur :8001")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
