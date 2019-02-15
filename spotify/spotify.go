package spotify

import (
	"context"
	"encoding/json"
	//"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyAPI struct {
	ClientID       string
	ClientSecretID string
}

type SpotifyPlaylist struct {
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

func NewSpotifyAPI(clientID, clientSecretID string) *SpotifyAPI {
	return &SpotifyAPI{
		ClientID:       clientID,
		ClientSecretID: clientSecretID,
	}
}

func (s *SpotifyAPI) GetAPIToken() (*oauth2.Token, error) {
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

func (s *SpotifyAPI) GetTrackFromPlaylist(token, playlistID string) (*SpotifyPlaylist, error) {
	url := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"
	bearerToken := "Bearer " + token

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// spotify playlist
	decoder := json.NewDecoder(resp.Body)
	var spotifyPl SpotifyPlaylist
	err = decoder.Decode(&spotifyPl)
	if err != nil {
		return nil, err
	}

	return &spotifyPl, nil
}
