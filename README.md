# ForumForAll

Une application de forum simple construite avec Go et MySQL.

## Fonctionnalités

- Inscription et connexion des utilisateurs
- Création de topics
- Réponses aux topics
- Système de likes pour les topics
- Tri des messages par date et likes

## Prérequis

- Go 1.21 ou supérieur
- MySQL 5.7 ou supérieur

## Installation

1. Clonez le dépôt :
```bash
git clone https://github.com/Linijarae/ForumForAll.git
cd ForumForAll
```

2. Installez les dépendances :
```bash
go mod download
```

3. Configurez les variables d'environnement :
```bash
export DB_USER="votre_utilisateur_mysql"
export DB_PASS="votre_mot_de_passe_mysql"
export DB_NAME="nom_de_votre_base"
```

4. Créez la base de données MySQL :
```sql
CREATE DATABASE nom_de_votre_base;
```

## Démarrage

Pour démarrer l'application :
```bash
go run main.go
```

Le serveur démarrera sur le port 8001.

## API Endpoints

### Authentification
- `POST /register` - Inscription d'un nouvel utilisateur
- `POST /login` - Connexion d'un utilisateur
- `POST /logout` - Déconnexion

### Topics
- `GET /topics` - Liste tous les topics
- `GET /topics/{id}` - Récupère un topic spécifique et ses messages
- `POST /topics` - Crée un nouveau topic (authentification requise)
- `POST /api/topic/like` - Like/unlike un topic (authentification requise)
- `POST /api/topic/dislike` - Dislike/unlike un topic (authentification requise)

### Messages
- `POST /topics/{id}/messages` - Ajoute un message à un topic (authentification requise)

## Format des requêtes

### Inscription
```json
{
    "username": "utilisateur",
    "email": "utilisateur@example.com",
    "password": "motdepasse"
}
```
Le mot de passe se doit d'être de 12 caractères minimum, avec un chiffre et un caractère spécial.

### Connexion
```json
{
    "email": "utilisateur@example.com",
    "password": "motdepasse"
}
```

### Création de topic
```json
{
    "title": "Titre du topic",
    "content": "Contenu du topic"
}
```

### Création de message
```json
{
    "content": "Contenu du message"
}
``` 