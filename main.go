package main

import (
	"database/sql"
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
	defer db.Close()

	http.HandleFunc("/getPlayerID", getPlayerIDHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/securityAnswer", securityAnswerHandler)

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
		if err == sql.ErrNoRows {
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
		if err == sql.ErrNoRows {
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
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	playerID, err := getPlayerID(playerName)
	if err != nil {
		http.Error(w, "An error occurred while retrieving the player ID.", http.StatusInternalServerError)
		return
	}

	if playerID == 0 {
		http.Error(w, "Player not found", http.StatusNotFound)
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
		fmt.Fprint(w, "An error occurred while checking the user.")
		return
	}

	if !unique {
		fmt.Fprint(w, "Username already exists")
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
		if err == sql.ErrNoRows {
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
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
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
