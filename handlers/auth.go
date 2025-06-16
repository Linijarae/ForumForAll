package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("votre_clé_secrète_jwt") // À changer en production

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

func generateToken(userID int, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req LoginRequest
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")

		if req.Username == "" || req.Password == "" {
			http.Error(w, "Tous les champs sont requis", http.StatusBadRequest)
			return
		}

		fmt.Println(req)

		var userID int
		var username string
		var hashedPassword string
		err := db.QueryRow("SELECT user_id, username, password FROM user WHERE username = ?", req.Username).Scan(&userID, &username, &hashedPassword)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Nom d'utilisateur ou mot de passe incorrect"})
			return
		}

		fmt.Println(userID)
		fmt.Println(username)
		fmt.Println(hashedPassword)

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Nom d'utilisateur ou mot de passe incorrect"})
			return
		}

		token, err := generateToken(userID, username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "Erreur lors de la génération du token"})
			return
		}
		fmt.Println(token)
		// Mettre à jour la dernière connexion
		_, err = db.Exec("UPDATE user SET last_connection = NOW() WHERE user_id = ?", userID)
		if err != nil {
			println("Error updating last connection:", err.Error())
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "token_form",
			Value:   token,
			Path:    "/",
			Expires: time.Now().Add(24 * time.Hour),
		})
		http.Redirect(w, r, "/index", http.StatusSeeOther)
	}
}

func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		req.Username = r.FormValue("username")
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")
		req.ConfirmPassword = r.FormValue("confirmPassword")

		// Fonction utilitaire pour afficher la page avec une erreur
		displayError := func(msg string) {
			tmpl, _ := template.ParseFiles("templates/register.html")
			tmpl.Execute(w, struct{ Error string }{Error: msg})
		}

		if req.Username == "" || req.Email == "" || req.Password == "" || req.ConfirmPassword == "" {
			displayError("Tous les champs sont requis")
			return
		}

		if req.Password != req.ConfirmPassword {
			displayError("Les mots de passe ne correspondent pas")
			return
		}

		if len(req.Password) < 12 {
			displayError("Le mot de passe doit contenir au moins 12 caractères")
			return
		}

		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM user WHERE username = ? OR mail = ?)", req.Username, req.Email).Scan(&exists)
		if err != nil {
			displayError("Erreur lors de la vérification de l'utilisateur")
			return
		}
		if exists {
			displayError("Ce nom d'utilisateur ou cet email est déjà utilisé")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			displayError("Erreur lors du hachage du mot de passe")
			return
		}

		_, err = db.Exec("INSERT INTO user (username, mail, password, role_id) VALUES (?, ?, ?, 1)", req.Username, req.Email, string(hashedPassword))
		if err != nil {
			displayError("Erreur lors de la création de l'utilisateur")
			return
		}

		http.Redirect(w, r, "/register", http.StatusSeeOther)
	}
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, tokenErr := r.Cookie("token_form")
		if tokenString.Value == "" || tokenErr != nil {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}

		if !token.Valid {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
}
