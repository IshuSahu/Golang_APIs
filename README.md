# MusicLister API

The MusicLister API is a simple Go application that allows users to register, log in, manage playlists, and perform various operations on songs within those playlists.

## Features

- **User Registration**: Users can register with their name, email, and automatically generated secret code.
- **User Login**: Users can log in using their secret code.
- **View Profile**: Users can view their profile information using their secret code.
- **Create Playlist**: Users can create playlists, and each playlist can contain multiple songs.
- **Add Song to Playlist**: Users can add a song to a specific playlist.
- **Get All Songs of Playlist**: Users can retrieve a list of all songs in a particular playlist.
- **Delete Song from Playlist**: Users can remove a song from a playlist.
- **Delete Playlist**: Users can delete a playlist.
- **Get Song Detail**: Users can retrieve details of a specific song within a playlist.

## Getting Started

### Prerequisites

- [Go](https://golang.org/) installed on your machine.

### Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/IshuSahu/Golang_Music_API
    ```

2. Run the application:

    ```bash
    go run main.go
    ```

The server will start running on `http://localhost:9090`.

## Usage

Follow these steps to interact with the MusicLister API:

1. **Register a User**:

    Send a `POST` request to `http://localhost:9090/register` with the user details in the request body. Example:

    ```json
    {
      "name": "John Doe",
      "email": "john@example.com"
    }
    ```

2. **Login**:

    Send a `POST` request to `http://localhost:9090/login` with the secret code obtained from the registration. Example:

    ```json
    {
      "secret_code": "generated-secret-code"
    }
    ```

3. **View Profile**:

    Open a browser or use a tool like cURL to access `http://localhost:9090/viewProfile?secret_code=generated-secret-code`.

4. **Create Playlist**:

    Send a `POST` request to `http://localhost:9090/createPlaylist?secret_code=generated-secret-code` with the playlist details. Example:

    ```json
    {
      "name": "My Playlist"
    }
    ```

5. **Add Song to Playlist**:

    Send a `POST` request to `http://localhost:9090/addSongToPlaylist?secret_code=generated-secret-code` with the playlist ID and song details. Example:

    ```json
    {
      "playlist_id": "1",
      "song": {
        "name": "Song Name",
        "composers": "Composer Name",
        "music_url": "https://example.com/music.mp3"
      }
    }
    ```

6. **Get All Songs of Playlist**:

    Open a browser or use a tool like cURL to access `http://localhost:9090/getAllSongsOfPlaylist?secret_code=generated-secret-code&playlist_id=1`.

7. **Delete Song from Playlist**:

    Send a `POST` request to `http://localhost:9090/deleteSongFromPlaylist?secret_code=generated-secret-code` with the playlist ID and song ID. Example:

    ```json
    {
      "playlist_id": "1",
      "song_id": "1"
    }
    ```

8. **Delete Playlist**:

    Send a `POST` request to `http://localhost:9090/deletePlaylist?secret_code=generated-secret-code` with the playlist ID. Example:

    ```json
    {
      "playlist_id": "1"
    }
    ```

9. **Get Song Detail**:

    Send a `POST` request to `http://localhost:9090/getSongDetail?secret_code=generated-secret-code` with the playlist ID and song ID. Example:

    ```json
    {
      "playlist_id": "1",
      "song_id": "1"
    }
    ```

Feel free to customize and expand the instructions based on your project's specific details.
