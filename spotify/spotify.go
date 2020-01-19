package spotify

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyAPI struct {
	ClientID       string
	ClientSecretID string
	BearerToken    string
}

type SpotifyPlaylistTracks struct {
	Items []struct {
		Track struct {
			Album struct {
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Name string `json:"name"`
			} `json:"album"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Name string `json:"name"`
		} `json:"track"`
	} `json:"items"`
	Limit    int         `json:"limit"`
	Next     string      `json:"next"`
	Offset   int         `json:"offset"`
	Previous interface{} `json:"previous"`
	Total    int         `json:"total"`
}

type SpotifyPlaylistInfo struct {
	Name  string `json:"name"`
	Owner struct {
		DisplayName string `json:"display_name"`
		ID          string `json:"id"`
	} `json:"owner"`
}

//!!!! Spotify api endpoints requires bearer token, clientID and
//clientSecretID needed only for acquiring the bearer token.
//endpoint functions could be call by a different struct that
//contains only acquired bearer token.
func NewSpotifyAPI(clientID, clientSecretID string) *SpotifyAPI {
	spoAPI := &SpotifyAPI{
		ClientID:       clientID,
		ClientSecretID: clientSecretID,
	}

	bearerToken, err := spoAPI.getAPIToken()
	if err != nil {
		log.Printf("Error while getting Spotify OAUTH Token: %v", err)
	}

	return &SpotifyAPI{
		ClientID:       clientID,
		ClientSecretID: clientSecretID,
		BearerToken:    bearerToken.AccessToken,
	}
}

//getAPIToken returns spotify oauth token
func (s *SpotifyAPI) getAPIToken() (*oauth2.Token, error) {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecretID,
		TokenURL:     "https://accounts.spotify.com/api/token",
	}

	tok, err := conf.Token(ctx)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

//GetTracksFromPlaylist returns track information from the given
//spotify playlist ID.
func (s *SpotifyAPI) GetTracksFromPlaylist(playlistID string) (*SpotifyPlaylistTracks, error) {
	url := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"
	bearerToken := "Bearer " + s.BearerToken

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// spotify playlist
	decoder := json.NewDecoder(resp.Body)

	var spotifyPlTracks SpotifyPlaylistTracks
	err = decoder.Decode(&spotifyPlTracks)
	if err != nil {
		return nil, err
	}

	return &spotifyPlTracks, nil
}

func (s *SpotifyAPI) GetPlaylist(playlistID string) (*SpotifyPlaylistInfo, error) {
	url := "https://api.spotify.com/v1/playlists/" + playlistID
	bearerToken := "Bearer " + s.BearerToken

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// spotify playlist
	decoder := json.NewDecoder(resp.Body)

	var spotifyPl SpotifyPlaylistInfo
	err = decoder.Decode(&spotifyPl)
	if err != nil {
		return nil, err
	}

	return &spotifyPl, nil
}
