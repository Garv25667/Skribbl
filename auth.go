package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func issueJWT(playerID int32) (string, error) {
	claims := jwt.MapClaims{
		"player_id": playerID,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

func verifyJWT(tokenStr string) (int32, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	playerID := int32(claims["player_id"].(float64))
	return playerID, nil
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username is required", 400)
		return
	}

	// check if player already exists
	player, err := s.DB.GetPlayerByUsername(context.Background(), username)
	if err != nil {
		// doesnt exist, create one
		player, err = s.DB.CreatePlayer(context.Background(), username)
		if err != nil {
			http.Error(w, "could not create player", 500)
			return
		}
	}

	token, err := issueJWT(player.ID)
	if err != nil {
		http.Error(w, "could not issue token", 500)
		return
	}

	w.Write([]byte(token))
}
