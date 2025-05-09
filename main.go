package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type Game struct {
	ID            string       `json:"id"`
	Board         [3][3]string `json:"board"`
	CurrentPlayer string       `json:"currentPlayer"`
	Status        string       `json:"status"` // playing won or draw
	Winner        string       `json:"winner"`
}

type Move struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

var (
	// index => value
	// string => *Game
	games     = make(map[string]*Game)
	mutex     sync.Mutex
	idCounter uint64
)

func generateID() string {
	return strconv.FormatUint(atomic.AddUint64(&idCounter, 1), 10)
}

func createNewGame(w http.ResponseWriter, r *http.Request) {

	// if req is POST, invalidate it
	if r.Method != http.MethodPost {

		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return

	}

	id := generateID()

	// create a game now
	game := &Game{
		ID:            id,
		Board:         [3][3]string{},
		CurrentPlayer: "X",
		Status:        "Playing",
	}

	// lock the mutex to hold other operation of games

	mutex.Lock()

	// add that game into zlist of games
	games[id] = game

	mutex.Unlock()

	//  now we add in the header, that the content sent is json explicitly

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(game); err != nil {
		return
	}

}

func GetGame(w http.ResponseWriter, r *http.Request, id string) {

	// return json
	mutex.Lock()

	game, ok := games[id]

	mutex.Unlock()

	if !ok {
		http.Error(w, "Game Not Found", http.StatusNotFound)
		return
	}

	// setting content type as json
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(game); err != nil {
		return
	}

}

func MakeMove(w http.ResponseWriter, r *http.Request, id string) {

	// check if POST or not
	// mutex lock, load the game from games, unlock
	// if game.status != playing, game ended, cant move
	// decode the json

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// load game
	mutex.Lock()
	// map gives the values, and boolean in return which
	// true -> exists false-> does not exist
	game, ok := games[id]
	mutex.Unlock()

	if !ok {
		http.Error(w, "Game does not exist", http.StatusNotFound)
		return

	}

	// game in playing or already played
	if game.Status != "playing" {
		http.Error(w, "Game already finished", http.StatusBadRequest)
		return
	}

	//? DECODE THE JSON

	var move Move
	if err := json.NewDecoder(r.Body).Decode(&move); err != nil {
		http.Error(w, "Invalid Move/JSON", http.StatusBadRequest)
		return
	}

	// check if valid move or not
	if move.Row < 0 || move.Row > 2 || move.Col < 0 || move.Col > 2 {
		http.Error(w, "Invalid move position", http.StatusBadRequest)
		return
	}

	// check if position is already took or not

	if game.Board[move.Row][move.Col] != "" {
		http.Error(w, "Position already took", http.StatusBadRequest)
		return
	}

	// place the move currentplayer ko, update current player
	game.Board[move.Row][move.Col] = game.CurrentPlayer

	// check if Win or Draw
	if checkWin(game.Board, game.CurrentPlayer) {
		game.Status = "Won"
		game.Winner = game.CurrentPlayer

	} else if checkDraw(game.Board) {
		game.Status = "Draw"

	} else {
		// make the move
		if game.CurrentPlayer == "X" {
			game.CurrentPlayer = "O"
		} else {
			game.CurrentPlayer = "X"
		}

	}

	// return json

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(game); err != nil {
		return
	}

}

func checkWin(board [3][3]string, player string) bool {

	n := 4

	for i := range n {
		if board[0][i] == player && board[1][i] == player && board[2][i] == player {
			return true
		}
		if board[i][0] == player && board[i][1] == player && board[i][2] == player {
			return true
		}
	}

	// check diagnoals
	if board[0][0] == player && board[1][1] == player && board[2][2] == player {
		return true
	}

	// another diagnoal
	if board[0][2] == player && board[1][1] == player && board[2][0] == player {
		return true
	}

	return false
}

func checkDraw(board [3][3]string) bool {

	n := 3

	for i := range n {
		for j := range n {
			if board[i][j] == "" {

				//  means not draw
				return false
			}
		}
	}

	//  means draw as no empty "" in board
	return true

}

func UrlHandler(w http.ResponseWriter, r *http.Request) {

	// trim the prefix /game/
	path := strings.TrimPrefix(r.URL.Path, "/game/")

	// divide into parts by / / / /
	parts := strings.Split(path, "/")

	id := parts[0]

	if len(parts) == 1 {

		// if GET, route to getGame, else invalid
		if r.Method == http.MethodGet {

			GetGame(w, r, id)

		} else {

			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		}

	} else if len(parts) == 2 && parts[1] == "move" {

		MakeMove(w, r, id)

	} else {

		http.Error(w, "Not Found", http.StatusNotFound)
	}

}

func main() {

	PORT := ":8080"

	// create a mux to handle everything
	mux := http.NewServeMux()

	// handle /game
	mux.HandleFunc("/game", createNewGame)

	// handle anything aru
	mux.HandleFunc("/game/", UrlHandler)

	// add aru static files to be accessable
	mux.Handle("/", http.FileServer(http.Dir(".")))

	if err := http.ListenAndServe(PORT, mux); err != nil {
		fmt.Printf("Error : %s", err)
		return
	}

}
