package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
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
	// playlist map[string]Playlist
}

func generateUniqueSecretCode() string {
	currentTime := time.Now().Unix()
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	secretCode := fmt.Sprintf("%d-%s", currentTime, s)

	return secretCode
}
func (h *MusicListerHandlers) register(w http.ResponseWriter, r *http.Request) {

	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed the request body: %v", err)
		return
	}

	newUser.ID = fmt.Sprintf("%d", len(h.users)+1)
	newUser.SecretCode = generateUniqueSecretCode()

	for _, existingUser := range h.users {
		if existingUser.Email == newUser.Email {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "Email already registered")
			return
		}
	}
	h.users[newUser.ID] = newUser

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

func (h *MusicListerHandlers) login(w http.ResponseWriter, r *http.Request) {

	var input struct {
		SecretCode string `json:"secret_code"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	for _, user := range h.users {
		if user.SecretCode == input.SecretCode {
			response := "welcome to the Music API"
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "User not found")
}

func (h *MusicListerHandlers) viewProfile(w http.ResponseWriter, r *http.Request) {
	secretCode := r.URL.Query().Get("secret_code")
	if secretCode == "" {
		http.Error(w, "Secret code is missing in the request", http.StatusBadRequest)
		return
	}

	var userProfile User
	var found bool
	for _, user := range h.users {
		if user.SecretCode == secretCode {
			userProfile = user
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userProfile)
}

func (h *MusicListerHandlers) createPlaylist(w http.ResponseWriter, r *http.Request) {
	var newPlaylist Playlist
	err := json.NewDecoder(r.Body).Decode(&newPlaylist)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	userSecretCode := r.URL.Query().Get("secret_code")
	var user User
	found := false

	h.Lock()
	for _, u := range h.users {
		if u.SecretCode == userSecretCode {
			user = u
			found = true
			break
		}
	}
	h.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	newPlaylist.ID = fmt.Sprintf("%d", len(user.Playlists)+1)
	user.Playlists = append(user.Playlists, newPlaylist)

	h.Lock()
	h.users[user.ID] = user
	h.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPlaylist)
}
func (h *MusicListerHandlers) addSongToPlaylist(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SecretCode string `json:"secret_code"`
		PlaylistID string `json:"playlist_id"`
		Song       Song   `json:"song"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	h.Lock()
	defer h.Unlock()

	// Find the user with the provided SecretCode
	var user User
	found := false
	for _, u := range h.users {
		if u.SecretCode == request.SecretCode {
			user = u
			found = true
			break
		}
	}

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	var playlist *Playlist
	for i := range user.Playlists {
		if user.Playlists[i].ID == request.PlaylistID {
			playlist = &user.Playlists[i]
			break
		}
	}

	if playlist == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Playlist not found")
		return
	}

	request.Song.ID = fmt.Sprintf("%d", len(playlist.Songs)+1)
	playlist.Songs = append(playlist.Songs, request.Song)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(playlist)
}

func (h *MusicListerHandlers) getAllSongsOfPlaylist(w http.ResponseWriter, r *http.Request) {
	secretCode := r.URL.Query().Get("secret_code")
	playlistID := r.URL.Query().Get("playlist_id")

	var user User
	found := false

	h.Lock()
	for _, u := range h.users {
		if u.SecretCode == secretCode {
			user = u
			found = true
			break
		}
	}
	h.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	var targetPlaylist *Playlist
	for _, p := range user.Playlists {
		if p.ID == playlistID {
			targetPlaylist = &p
			break
		}
	}

	if targetPlaylist == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Playlist not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(targetPlaylist.Songs)
}

func (h *MusicListerHandlers) deleteSongFromPlaylist(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SecretCode string `json:"secret_code"`
		PlaylistID string `json:"playlist_id"`
		SongID     string `json:"song_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	var user User
	found := false

	h.Lock()
	for _, u := range h.users {
		if u.SecretCode == request.SecretCode {
			user = u
			found = true
			break
		}
	}
	h.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	var targetPlaylist *Playlist
	for i, p := range user.Playlists {
		if p.ID == request.PlaylistID {
			targetPlaylist = &user.Playlists[i]
			user.Playlists[i] = *targetPlaylist
			break
		}
	}

	if targetPlaylist == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Playlist not found")
		return
	}

	var deletedSong *Song
	for i, song := range targetPlaylist.Songs {
		if song.ID == request.SongID {
			deletedSong = &targetPlaylist.Songs[i]
			targetPlaylist.Songs = append(targetPlaylist.Songs[:i], targetPlaylist.Songs[i+1:]...)
			break
		}
	}

	if deletedSong == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Song not found in the playlist")
		return
	}

	h.Lock()
	h.users[user.SecretCode] = user
	h.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(deletedSong)
}

func (h *MusicListerHandlers) deletePlaylist(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SecretCode string `json:"secret_code"`
		PlaylistID string `json:"playlist_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	var user User
	found := false

	h.Lock()
	for i, u := range h.users {
		if u.SecretCode == request.SecretCode {
			user = u
			found = true

			for j, playlist := range user.Playlists {
				if playlist.ID == request.PlaylistID {
					user.Playlists = append(user.Playlists[:j], user.Playlists[j+1:]...)
					break
				}
			}
			h.users[i] = user
			break
		}
	}
	h.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Playlist deleted successfully")
}

func (h *MusicListerHandlers) getSongDetail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SecretCode string `json:"secret_code"`
		PlaylistID string `json:"playlist_id"`
		SongID     string `json:"song_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	var user User
	found := false

	h.Lock()
	for _, u := range h.users {
		if u.SecretCode == request.SecretCode {
			user = u
			found = true
			break
		}
	}
	h.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "User not authorized")
		return
	}

	var targetPlaylist *Playlist
	for _, p := range user.Playlists {
		if p.ID == request.PlaylistID {
			targetPlaylist = &p
			break
		}
	}

	if targetPlaylist == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Playlist not found")
		return
	}

	var songDetails *Song
	for _, song := range targetPlaylist.Songs {
		if song.ID == request.SongID {
			songDetails = &song
			break
		}
	}

	if songDetails == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Song not found in the playlist")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(songDetails)
}

func main() {
	musicListerHandlers := &MusicListerHandlers{
		users: make(map[string]User),
	}

	http.HandleFunc("/register", musicListerHandlers.register)
	http.HandleFunc("/login", musicListerHandlers.login)
	http.HandleFunc("/viewProfile", musicListerHandlers.viewProfile)
	http.HandleFunc("/createPlaylist", musicListerHandlers.createPlaylist)
	http.HandleFunc("/addSongToPlaylist", musicListerHandlers.addSongToPlaylist)
	http.HandleFunc("/getAllSongsOfPlaylist", musicListerHandlers.getAllSongsOfPlaylist)
	http.HandleFunc("/deleteSongFromPlaylist", musicListerHandlers.deleteSongFromPlaylist)
	http.HandleFunc("/deletePlaylist", musicListerHandlers.deletePlaylist)
	http.HandleFunc("/getSongDetail", musicListerHandlers.getSongDetail)

	fmt.Println("Server is running on :9090...")
	http.ListenAndServe(":9090", nil)
}

/*
	--> add User one by one
	--> add Secrete code on Url : http://localhost:9090/viewProfile?secret_code=code
*/
