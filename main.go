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

	pokemon, err := getPlayerPokemonByID(1)
	if err != nil {
		log.Fatalf("An error occurred: %v", err)
	}
	if pokemon == nil {
		log.Println("No Pokemon found with the provided ID.")
		return
	}
	log.Printf("Pokemon: %+v\n", pokemon)

	http.HandleFunc("/getPlayerID", getPlayerIDHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/securityAnswer", securityAnswerHandler)
	http.HandleFunc("/checkSecurityAnswer", checkSecurityAnswerHandler)
	http.HandleFunc("/resetPassword", resetPasswordHandler)
	http.HandleFunc("/getPlayerPokemon", getPlayerPokemonByIDHandler)
	http.HandleFunc("/getEnemyPokemon", getEnemyPokemonByIDHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func validateUserInput(playerName, password string) string {
	if playerName == "" && password == "" {
		return "now y you dont put either?!?"
	}

	if playerName == "" {
		return "y you no put Username??!"
	}
	if password == "" {
		return "y you no put Password??!"
	}

	if len(playerName) < 3 || len(password) < 3 {
		return "Username or Password is less than 3 characters. Too short, try again!"
	}
	return ""
}

func isValidUser(playerName, password string) (string, int, error) {
	var id int
	err := db.QueryRow("SELECT playerID FROM players WHERE playerName = ? AND playerPassword = ?", playerName, password).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "false", 0, nil
		}
		return "false", 0, err
	}
	return "true", id, nil
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

	if valid == "false" {
		fmt.Fprint(w, "yo tings are wrong")
		return
	}

	fmt.Fprint(w, "true")
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

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	playerName := r.FormValue("username")
	password := r.FormValue("password")

	validationError := validateUserInput(playerName, password)
	if validationError != "" {
		fmt.Fprint(w, validationError)
		return
	}

	unique, err := isPlayerNameUnique(playerName)
	if err != nil {
		fmt.Fprint(w, "Could not check playerName uniqueness")
		return
	}

	if !unique {
		fmt.Fprint(w, "PlayerName already exists!")
		return
	}

	success, err := insertPlayer(playerName, password)
	if err != nil || !success {
		fmt.Fprint(w, "Couldn't insert you into the database!")
		return
	}
	fmt.Fprint(w, "true")
}

func isPlayerNameUnique(playerName string) (bool, error) {
	var id int
	err := db.QueryRow("SELECT playerID FROM players WHERE playerName = ?", playerName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil // Player name is unique
		}
		return false, err
	}
	return false, nil // Player name already exists, not unique
}

func insertPlayer(playerName, password string) (bool, error) {
	res, err := db.Exec("INSERT INTO players (playerName, playerPassword) VALUES (?, ?)", playerName, password)
	if err != nil {
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil // If at least one row is affected, the insertion was successful.
}

func insertSecurityAnswer(playerID int, securityAnswer string) error {
	res, err := db.Exec("UPDATE players SET securityAnswers = ? WHERE playerID = ?", securityAnswer, playerID)
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
		fmt.Println("Couldn't insert you into the database!")
	}

	return nil
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
		http.Error(w, "Couldn't insert security answer into the database!", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Security answer stored successfully!")
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
	err := db.QueryRow("SELECT securityAnswers FROM players WHERE playerID = ?", playerID).Scan(&storedAnswer)
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
		http.Error(w, "Couldn't reset password!", http.StatusNotFound)
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

type Pokemon struct {
	PlayerPokemonID   int    `json:"playerPokemonID"`
	PlayerID          int    `json:"playerID"`
	PlayerPokemonName string `json:"playerPokemonName"`
	PlayerXP          int    `json:"playerXP"`
	PlayerLevel       int    `json:"playerLevel"`
	PlayerHP          int    `json:"playerHP"`
}

type EnemyPokemon struct {
	EnemyPokemonID   int    `json:"enemyPokemonID"`
	EnemyPokemonName string `json:"enemyPokemonName"`
	EnemyLevel       int    `json:"enemyLevel"`
	EnemyHp          int    `json:"enemyHp"`
	EnemySpecialMove string `json:"enemySpecialMove"`
}

func getPlayerPokemonByID(playerPokemonID int) (*Pokemon, error) {
	var pokemon Pokemon
	err := db.QueryRow("SELECT playerPokemonID, playerID, playerPokemonName, playerXP, playerLevel, playerHP FROM playerPokemons WHERE playerPokemonID = ?", playerPokemonID).
		Scan(&pokemon.PlayerPokemonID, &pokemon.PlayerID, &pokemon.PlayerPokemonName, &pokemon.PlayerXP, &pokemon.PlayerLevel, &pokemon.PlayerHP)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pokemon, nil
}

func getPlayerPokemonByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	pokemonID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid Player Pokemon ID", http.StatusBadRequest)
		return
	}

	pokemon, err := getPlayerPokemonByID(pokemonID)
	if err != nil {
		http.Error(w, "Could not retrieve Pokemon", http.StatusInternalServerError)
		return
	}

	if pokemon == nil {
		http.Error(w, "Cannot find Pokemon", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pokemon)
}
func getEnemyPokemonByID(enemyPokemonID int) (*EnemyPokemon, error) {
	var enemyPokemon EnemyPokemon
	err := db.QueryRow("SELECT enemyPokemonID, enemyPokemonName, enemyLevel, enemyHp, enemySpecialMove FROM enemypokemons WHERE enemyPokemonID = ?", enemyPokemonID).Scan(&enemyPokemon.EnemyPokemonID, &enemyPokemon.EnemyPokemonName, &enemyPokemon.EnemyLevel, &enemyPokemon.EnemyHp, &enemyPokemon.EnemySpecialMove)
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
