package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:root@tcp(localhost:3306)/pokemongame")
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	enemyPokemon, err := getEnemyPokemonByID(1)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}
	if enemyPokemon == nil {
		log.Println("No enemy Pokemon found with the provided ID.")
	} else {
		log.Printf("Enemy Pokemon: %+v\n", enemyPokemon)
	}

	Pokemon, err := getPlayerPokemonStatsByID(1)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}
	if Pokemon == nil {
		log.Println("No enemy Pokemon found with the provided ID.")
	} else {
		log.Printf("Pokemon: %+v\n", Pokemon)
	}

	http.HandleFunc("/getPlayerID", getPlayerIDHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/securityAnswer", securityAnswerHandler)
	http.HandleFunc("/checkSecurityAnswer", checkSecurityAnswerHandler)
	http.HandleFunc("/resetPassword", resetPasswordHandler)
	http.HandleFunc("/getEnemyPokemon", getEnemyPokemonByIDHandler)
	http.HandleFunc("/insertPlayerPokemon", insertPlayerPokemonHandler)
	http.HandleFunc("/insertPokemonStats", insertPokemonStatsHandler)
	http.HandleFunc("/updatePokemonStats", updatePokemonStatsHandler)
	http.HandleFunc("/getPlayerPokemonStats", getPlayerPokemonStatsHandler)
	http.HandleFunc("/getPlayerPokemonID", getPlayerPokemonIDHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func validateUserInput(playerName, password string) string {
	if playerName == "" {
		return "You need to input a username"
	}

	if len(playerName) < 3 || len(password) < 3 {
		return "Username or Password is less than 3 characters. Too short, try again!"
	}
	return ""
}

func isValidUser(playerName, password string) (string, int, error) {
	var id int
	err := db.QueryRow("SELECT playerID FROM players WHERE playerName = ?", playerName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "Username is incorrect", 0, nil
		}
		return "false", 0, err
	}

	err = db.QueryRow("SELECT playerID FROM players WHERE playerName = ? AND playerPassword = ?", playerName, password).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "Password is incorrect", 0, nil
		}
		return "false", 0, err
	}

	return "true", id, nil
}

func getPlayerIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerName := r.URL.Query().Get("username")

	if playerName == "" {
		http.Error(w, "Username needed", http.StatusBadRequest)
		return
	}

	playerID, err := getPlayerID(playerName)
	if err != nil {
		http.Error(w, "Could not retrieve playerID", http.StatusInternalServerError)
		return
	}

	if playerID == 0 {
		http.Error(w, "Cannot find player", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, playerID)
}

func getPlayerID(playerName string) (int, error) {
	var id int
	err := db.QueryRow("SELECT playerID FROM players WHERE playerName = ?", playerName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerName := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	validationError := validateUserInput(playerName, password)
	if validationError != "" {
		fmt.Fprint(w, validationError)
		return
	}

	valid, _, err := isValidUser(playerName, password)
	if err != nil {
		fmt.Fprint(w, "An error occurred while checking the user.")
		return
	}

	if valid != "true" {
		fmt.Fprint(w, valid)
		return
	}

	fmt.Fprint(w, "true")
}

func isPlayerNameUnique(playerName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM players WHERE playerName = ?", playerName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func insertPlayer(playerName, password string) (bool, error) {
	_, err := db.Exec("INSERT INTO players (playerName, playerPassword) VALUES (?, ?)", playerName, password)
	if err != nil {
		return false, err
	}
	return true, nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerName := r.FormValue("username")
	password := r.FormValue("password")

	validationError := validateUserInput(playerName, password)
	if validationError != "" {
		http.Error(w, validationError, http.StatusBadRequest)
		return
	}

	unique, err := isPlayerNameUnique(playerName)
	if err != nil {
		http.Error(w, "Could not check if player name is unique", http.StatusInternalServerError)
		return
	}

	if !unique {
		http.Error(w, "PlayerName already exists!", http.StatusConflict)
		return
	}

	success, err := insertPlayer(playerName, password)
	if err != nil || !success {
		http.Error(w, "Could not insert you into the database!", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "true")
}

func securityAnswerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerID, err := strconv.Atoi(r.FormValue("playerID"))
	if err != nil {
		http.Error(w, "Invalid playerID", http.StatusBadRequest)
		return
	}

	securityAnswer := r.FormValue("securityAnswer")

	err = insertSecurityAnswer(playerID, securityAnswer)
	if err != nil {
		http.Error(w, "Could not insert security answer into the database!", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Security answer stored successfully!")
}

func insertSecurityAnswer(playerID int, securityAnswer string) error {
	res, err := db.Exec("INSERT INTO playersecurity (playerID, securityAnswers) VALUES (?, ?) ON DUPLICATE KEY UPDATE securityAnswers = VALUES(securityAnswers)", playerID, securityAnswer)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		fmt.Println("Security answer stored successfully!")
	} else {
		fmt.Println("Could not insert you into the database!")
	}

	return nil
}

func checkSecurityAnswerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerID, err := strconv.Atoi(r.URL.Query().Get("playerID"))
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	securityAnswer := r.URL.Query().Get("securityAnswer")

	matches, err := doesSecurityAnswerMatch(playerID, securityAnswer)
	if err != nil {
		http.Error(w, "An error occurred while checking the security answer.", http.StatusInternalServerError)
		return
	}

	if !matches {
		http.Error(w, "Security answers dont match!", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, "true")
}

func doesSecurityAnswerMatch(playerID int, securityAnswer string) (bool, error) {
	var storedAnswer string
	err := db.QueryRow("SELECT securityAnswers FROM playersecurity WHERE playerID = ?", playerID).Scan(&storedAnswer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // Player does not exist
		}
		return false, err
	}
	return securityAnswer == storedAnswer, nil // Check if the security answer matches
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerID, err := strconv.Atoi(r.URL.Query().Get("playerID"))
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	newPassword := r.URL.Query().Get("newPassword")

	success, err := resetPassword(playerID, newPassword)
	if err != nil {
		http.Error(w, "An error occurred while resetting the password.", http.StatusInternalServerError)
		return
	}

	if !success {
		http.Error(w, "Could not reset password!", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, "true")
}

func resetPassword(playerID int, newPassword string) (bool, error) {
	res, err := db.Exec("UPDATE players SET playerPassword = ? WHERE playerID = ?", newPassword, playerID)
	if err != nil {
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil // If at least one row is affected, the password reset was successful.
}

func getPlayerPokemonID(playerID int) (int, error) {
	var playerPokemonID int
	err := db.QueryRow("SELECT playerPokemonID FROM playerpokemons WHERE playerID = ?", playerID).Scan(&playerPokemonID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return playerPokemonID, nil
}

func getPlayerPokemonIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerID, err := strconv.Atoi(r.URL.Query().Get("playerID"))
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	playerPokemonID, err := getPlayerPokemonID(playerID)
	if err != nil {
		http.Error(w, "Could not retrieve player Pokemon ID", http.StatusInternalServerError)
		return
	}

	if playerPokemonID == 0 {
		http.Error(w, "No player Pokemon found for the provided player ID", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, playerPokemonID)
}

type Pokemon struct {
	PlayerPokemonID   int    `json:"playerPokemonID"`
	PlayerPokemonName string `json:"playerPokemonName"`
	PlayerXP          int    `json:"playerXP"`
	PlayerLevel       int    `json:"playerLevel"`
	PlayerHP          int    `json:"playerHP"`
	Evolution         int    `json:"evolution"`
}

func getPlayerPokemonStatsByID(playerPokemonID int) (*Pokemon, error) {
	var pokemon Pokemon
	err := db.QueryRow("SELECT playerPokemonID, playerXP, playerLevel, playerHP, playerPokemonName, evolution FROM playerpokemonstats WHERE playerPokemonID = ?", playerPokemonID).
		Scan(&pokemon.PlayerPokemonID, &pokemon.PlayerXP, &pokemon.PlayerLevel, &pokemon.PlayerHP, &pokemon.PlayerPokemonName, &pokemon.Evolution)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pokemon, nil
}

func getPlayerPokemonStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerPokemonID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid Player Pokemon ID", http.StatusBadRequest)
		return
	}

	pokemon, err := getPlayerPokemonStatsByID(playerPokemonID)
	if err != nil {
		http.Error(w, "Could not retrieve Pokemon stats", http.StatusInternalServerError)
		return
	}

	if pokemon == nil {
		http.Error(w, "Cannot find Pokemon stats", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pokemon)
}

func insertPlayerPokemon(playerID int) (int, error) {
	result, err := db.Exec("INSERT INTO playerpokemons (playerID) VALUES (?)", playerID)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func insertPlayerPokemonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerID, err := strconv.Atoi(r.FormValue("playerID"))
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	playerPokemonID, err := insertPlayerPokemon(playerID)
	if err != nil {
		http.Error(w, "Could not insert player Pokemon", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, playerPokemonID)
}

func insertPokemonStats(playerPokemonID, playerXP, playerLevel, playerHP int, playerPokemonName string, evolution int) error {
	_, err := db.Exec("INSERT INTO playerpokemonstats (playerPokemonID, playerXP, playerLevel, playerHP, playerPokemonName, evolution) VALUES (?, ?, ?, ?, ?, ?)", playerPokemonID, playerXP, playerLevel, playerHP, playerPokemonName, evolution)
	return err
}

func insertPokemonStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerPokemonID, err := strconv.Atoi(r.FormValue("playerPokemonID"))
	if err != nil {
		http.Error(w, "Invalid player Pokemon ID", http.StatusBadRequest)
		return
	}

	playerXP, err := strconv.Atoi(r.FormValue("playerXP"))
	if err != nil {
		http.Error(w, "Invalid player XP", http.StatusBadRequest)
		return
	}

	playerLevel, err := strconv.Atoi(r.FormValue("playerLevel"))
	if err != nil {
		http.Error(w, "Invalid player level", http.StatusBadRequest)
		return
	}

	playerHP, err := strconv.Atoi(r.FormValue("playerHP"))
	if err != nil {
		http.Error(w, "Invalid player HP", http.StatusBadRequest)
		return
	}
	evolution, err := strconv.Atoi(r.FormValue("evolution"))
	if err != nil {
		http.Error(w, "Invalid evolution", http.StatusBadRequest)
		return
	}

	playerPokemonName := r.FormValue("playerPokemonName")

	err = insertPokemonStats(playerPokemonID, playerXP, playerLevel, playerHP, playerPokemonName, evolution)
	if err != nil {
		http.Error(w, "Could not insert Pokemon stats", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Pokemon stats inserted successfully!")
}

func updatePokemonStats(playerPokemonID, playerXP, playerLevel, playerHP int, evolution int, playerPokemonName string) error {
	_, err := db.Exec("UPDATE playerpokemonstats SET playerXP = ?, playerLevel = ?, playerHP = ?, playerPokemonName = ?, evolution = ? WHERE playerPokemonID = ?", playerXP, playerLevel, playerHP, playerPokemonName, evolution, playerPokemonID)
	return err
}
func updatePokemonStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerPokemonID, err := strconv.Atoi(r.FormValue("playerPokemonID"))
	if err != nil {
		http.Error(w, "Invalid player Pokemon ID", http.StatusBadRequest)
		return
	}

	playerXP, err := strconv.Atoi(r.FormValue("playerXP"))
	if err != nil {
		http.Error(w, "Invalid player XP", http.StatusBadRequest)
		return
	}

	playerLevel, err := strconv.Atoi(r.FormValue("playerLevel"))
	if err != nil {
		http.Error(w, "Invalid player level", http.StatusBadRequest)
		return
	}

	playerHP, err := strconv.Atoi(r.FormValue("playerHP"))
	if err != nil {
		http.Error(w, "Invalid player HP", http.StatusBadRequest)
		return
	}
	evolution, err := strconv.Atoi(r.FormValue("evolution"))
	if err != nil {
		http.Error(w, "Invalid evolution", http.StatusBadRequest)
		return
	}

	playerPokemonName := r.FormValue("playerPokemonName")

	err = updatePokemonStats(playerPokemonID, playerXP, playerLevel, playerHP, evolution, playerPokemonName)
	if err != nil {
		http.Error(w, "Could not update Pokemon stats", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Pokemon stats updated successfully!")
}

type EnemyPokemon struct {
	EnemyPokemonID   int    `json:"enemyPokemonID"`
	EnemyPokemonName string `json:"enemyPokemonName"`
	EnemyLevel       int    `json:"enemyLevel"`
	EnemyHp          int    `json:"enemyHp"`
}

func getEnemyPokemonByID(enemyPokemonID int) (*EnemyPokemon, error) {
	var enemyPokemon EnemyPokemon
	err := db.QueryRow("SELECT enemypokemons.enemyPokemonID, enemypokemons.enemyPokemonName, enemypokemonstats.enemyHp, enemypokemonstats.enemyLevel FROM enemypokemons, enemypokemonstats WHERE enemypokemons.enemyPokemonID = enemypokemonstats.enemyPokemonID AND enemypokemons.enemyPokemonID = ?", enemyPokemonID).
		Scan(&enemyPokemon.EnemyPokemonID, &enemyPokemon.EnemyPokemonName, &enemyPokemon.EnemyHp, &enemyPokemon.EnemyLevel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &enemyPokemon, nil
}

func getEnemyPokemonByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	enemyPokemonID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid Enemy Pokemon ID", http.StatusBadRequest)
		return
	}

	enemyPokemon, err := getEnemyPokemonByID(enemyPokemonID)
	if err != nil {
		http.Error(w, "Could not retrieve Enemy Pokemon", http.StatusInternalServerError)
		return
	}

	if enemyPokemon == nil {
		http.Error(w, "Cannot find Enemy Pokemon", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(enemyPokemon)
}
