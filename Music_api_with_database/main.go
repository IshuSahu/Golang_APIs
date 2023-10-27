package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Song struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Composers string `json:"composers"`
	MusicURL  string `json:"music_url"`
}

type Playlist struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Songs []Song `json:"songs"`
}

type User struct {
	ID         string     `json:"id"`
	SecretCode string     `json:"secret_code"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Playlists  []Playlist `json:"playlists"`
}

type MusicListerHandlers struct {
	sync.Mutex
	users map[string]User
	db    *sql.DB
}

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "BackendMusicAPI"
)

func connectDB() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func createTables(db *sql.DB) {

	createUserTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		secret_code TEXT,
		name TEXT,
		email TEXT
	);
	`
	createPlaylistTable := `
		CREATE TABLE IF NOT EXISTS playlists (
			id SERIAL PRIMARY KEY,
			user_id INTEGER,
			name TEXT
		);
	`
	createSongTable := `
		CREATE TABLE IF NOT EXISTS songs (
			id SERIAL PRIMARY KEY,
			playlist_id INTEGER,
			name TEXT,
			composers TEXT,
			music_url TEXT
		);
	`

	_, err := db.Exec(createUserTable)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(createPlaylistTable)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(createSongTable)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTables(db)

	musicListerHandlers := &MusicListerHandlers{
		users: make(map[string]User),
		db:    db,
	}

	http.HandleFunc("/register", musicListerHandlers.register)
	http.HandleFunc("/login", musicListerHandlers.login)
	http.HandleFunc("/viewProfile", musicListerHandlers.viewProfile)
	// http.HandleFunc("/createPlaylist", musicListerHandlers.createPlaylist)
	// http.HandleFunc("/addSongToPlaylist", musicListerHandlers.addSongToPlaylist)
	// http.HandleFunc("/getAllSongsOfPlaylist", musicListerHandlers.getAllSongsOfPlaylist)
	// http.HandleFunc("/deleteSongFromPlaylist", musicListerHandlers.deleteSongFromPlaylist)
	// http.HandleFunc("/deletePlaylist", musicListerHandlers.deletePlaylist)
	// http.HandleFunc("/getSongDetail", musicListerHandlers.getSongDetail)

	fmt.Println("Server is running on :9090...")
	http.ListenAndServe(":9090", nil)
}

// Rest of your handlers remain the same, and you can interact with the database as needed in those handlers.
func generateUniqueSecretCode(db *sql.DB) (string, error) {
	_, err := db.Exec("CREATE SEQUENCE IF NOT EXISTS user_sequence")
	if err != nil {
		return "", err
	}

	var id int
	err = db.QueryRow("SELECT nextval('user_sequence')").Scan(&id)
	if err != nil {
		return "", err
	}

	currentTime := time.Now().Unix()
	secretCode := fmt.Sprintf("%d-%d", currentTime, id)

	return secretCode, nil
}
func (h *MusicListerHandlers) register(w http.ResponseWriter, r *http.Request) {
	var newUser User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var existingEmail string
	err := h.db.QueryRow("SELECT email FROM users WHERE email = $1", newUser.Email).Scan(&existingEmail)
	if err == nil {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "Email already registered")
		return
	} else if err != sql.ErrNoRows {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Database error: %v", err)
		return
	}

	secretCode, err := generateUniqueSecretCode(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec("INSERT INTO users (secret_code, name, email) VALUES ($1, $2, $3)",
		secretCode, newUser.Name, newUser.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Lock()
	h.users[secretCode] = newUser
	h.Unlock()

	response := "User registered successfully"
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MusicListerHandlers) viewProfile(w http.ResponseWriter, r *http.Request) {
	secretCode := r.URL.Query().Get("secret_code")
	if secretCode == "" {
		http.Error(w, "Secret code is missing in the request", http.StatusBadRequest)
		return
	}

	var user User
	err := h.db.QueryRow("SELECT id, secret_code, name, email FROM users WHERE secret_code = $1", secretCode).
		Scan(&user.ID, &user.SecretCode, &user.Name, &user.Email)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "User not found")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Database error: %v", err)
		return
	}

	playlists, err := h.getUserPlaylists(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to retrieve playlists: %v", err)
		return
	}

	user.Playlists = playlists

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *MusicListerHandlers) getUserPlaylists(userID string) ([]Playlist, error) {
	rows, err := h.db.Query("SELECT p.id, p.name, s.id, s.name, s.composers, s.music_url "+
		"FROM playlists AS p "+
		"LEFT JOIN songs AS s ON p.id = s.playlist_id "+
		"WHERE p.user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make(map[string]*Playlist)
	for rows.Next() {
		var playlistID, playlistName, songID, songName, composers, musicURL string
		if err := rows.Scan(&playlistID, &playlistName, &songID, &songName, &composers, &musicURL); err != nil {
			return nil, err
		}

		if playlist, ok := playlists[playlistID]; ok {
			// Playlist already exists, add the song
			playlist.Songs = append(playlist.Songs, Song{
				ID:        songID,
				Name:      songName,
				Composers: composers,
				MusicURL:  musicURL,
			})
		} else {
			// Playlist doesn't exist, create it and add the song
			playlists[playlistID] = &Playlist{
				ID:   playlistID,
				Name: playlistName,
				Songs: []Song{{
					ID:        songID,
					Name:      songName,
					Composers: composers,
					MusicURL:  musicURL,
				}},
			}
		}
	}

	// Convert the map of playlists to a slice
	result := make([]Playlist, 0, len(playlists))
	for _, playlist := range playlists {
		result = append(result, *playlist)
	}

	return result, nil
}
func (h *MusicListerHandlers) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SecretCode string `json:"secret_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	var userProfile User
	err := h.db.QueryRow("SELECT id, secret_code, name, email FROM users WHERE secret_code = $1", input.SecretCode).
		Scan(&userProfile.ID, &userProfile.SecretCode, &userProfile.Name, &userProfile.Email)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := "welcome to the Music API"
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
