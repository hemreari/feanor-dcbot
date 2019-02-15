package spotify

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyAPI struct {
	ClientID       string
	ClientSecretID string
}

type SpotifyAccessToken struct {
	AccessToken string
}

type SpotifyPlaylist struct {
	Name   string `json:"name"`
	Tracks struct {
		Items []struct {
			Track struct {
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
	} `json:"tracks"`
}

func NewSpotifyAPI(clientID, clientSecretID string) *SpotifyAPI {
	return &SpotifyAPI{
		ClientID:       clientID,
		ClientSecretID: clientSecretID,
	}
}

func (s *SpotifyAPI) GetAPIToken() *oauth2.Token {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecretID,
		TokenURL:     "https://accounts.spotify.com/api/token",
	}

	tok, err := conf.Token(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return tok
}

func (s *SpotifyAPI) GetTrackFromPlaylist(token, playlistID string) SpotifyPlaylist {
	url := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"
	bearerToken := "Bearer " + token

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	/*
		fmt.Println(string([]byte(body)))
	*/
	// spotify playlist
	body, _ := ioutil.ReadAll(resp.Body)
	var spotifyPl SpotifyPlaylist
	err = json.Unmarshal([]byte(body), &spotifyPl)
	if err != nil {
		log.Println(err)
	}
	return spotifyPl
}
