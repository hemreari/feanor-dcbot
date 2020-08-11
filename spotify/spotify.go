package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	DefaultCoverUrl string = "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg"
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
				Name   string `json:"name"`
				Images []struct {
					Url string `json:"url"`
				} `json:"images"`
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

type SpotifyAlbumTracks struct {
	Name   string `json:"name"` //album name
	Images []struct {
		Url string `json:"url"` //album image url
	} `json:"images"`
	Tracks struct {
		Items []struct {
			Name    string `json:"name"` //track name
			Artists []struct {
				Name string `json:"name"` //track artist name
			} `json:"Artists"`
		} `json:"items"`
	} `json:"tracks"`
}

type SpotifyPlaylistInfo struct {
	Name  string `json:"name"`
	Owner struct {
		DisplayName string `json:"display_name"`
		ID          string `json:"id"`
	} `json:"owner"`
}

type SpotifyPlaylist struct {
	TrackName   string
	CoverUrl    string
	ArtistNames string
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

//getAPIToken returns Spotify oauth token
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

func (s *SpotifyAPI) do(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	req.Header.Add("Authorization", "Bearer "+s.BearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

//getTracksFromPlaylist sends request to get information about playlist's tracks
//to Spotify API and decodes API response to SpotifyPlaylistTracks struct.
func (s *SpotifyAPI) getTracksFromPlaylist(playlistID string) (*SpotifyPlaylistTracks, error) {
	url := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"
	resp, err := s.do("GET", url)
	if err != nil {
		return nil, fmt.Errorf("Error while getting response from playlist tracks endpoint: %v", err)
	}

	// spotify playlist
	decoder := json.NewDecoder(resp.Body)

	var spotifyPlTracks SpotifyPlaylistTracks
	err = decoder.Decode(&spotifyPlTracks)
	if err != nil {
		return nil, fmt.Errorf("Error while getting playlist track info: %v", err)
	}

	return &spotifyPlTracks, nil
}

//GetPlaylistInfo sends request to get information about playlist to
//Spotify API and decodes API response to SpotifyPlaylistInfo struct.
func (s *SpotifyAPI) getPlaylistInfo(id string) (*SpotifyPlaylistInfo, error) {
	url := "https://api.spotify.com/v1/playlists/" + playlistID

	resp, err := s.do("GET", url)
	if err != nil {
		return nil, fmt.Errorf("Error while getting playlist info: %v", err)
	}

	decoder := json.NewDecoder(resp.Body)
	var spotifyPl SpotifyPlaylistInfo
	err = decoder.Decode(&spotifyPl)
	if err != nil {
		return nil, err
	}

	return &spotifyPl, nil
}

//GetPlaylistInfo wrapper function of getPlaylistInfo function.
func (s *SpotifyAPI) GetPlaylistInfo(id string) (*SpotifyPlaylistInfo, error) {
	spotifyPlaylistInfo, err := s.getPlaylistInfo(id)
	if err != nil {
		return nil, err
	}
	return spotifyPlaylistInfo, nil
}

//GetPlaylist eliminates required information to create a download queue in the
//bot package from the getTracksFromPlaylist Function response and creates a slice
//of SpotifyPlaylist struct.
func (s *SpotifyAPI) GetSpotifyPlaylist(id string) ([]SpotifyPlaylist, error) {
	//handle playlists
	plTracks, err := s.getTracksFromPlaylist(id)
	if err != nil {
		return nil, err
	}

	spotifyPlaylistList := []SpotifyPlaylist{}

	items := plTracks.Items

	for i, value := range items {
		if i > 20 {
			break
		}

		trackName := value.Track.Name

		coverUrl := ""
		album := value.Track.Album
		if album.Images[1].Url != "" {
			coverUrl = album.Images[1].Url
		} else {
			coverUrl = DefaultCoverUrl
		}

		artistNames := ""
		artists := value.Track.Artists
		for artistIndex := range artists {
			artistNames += artists[artistIndex].Name + " "
		}

		spotifyPlaylist := SpotifyPlaylist{
			TrackName:   trackName,
			CoverUrl:    coverUrl,
			ArtistNames: artistNames,
		}

		spotifyPlaylistList = append(spotifyPlaylistList, spotifyPlaylist)
	}
	return spotifyPlaylistList, nil
}

//getAlbumTracks sends request to get information about album's tracks to
//Spotify API and decodes API Response to SpotifyAlbumTracks struct.
func (s *SpotifyAPI) getAlbumTracks(id string) (*SpotifyAlbumTracks, error) {
	url := "https://api.spotify.com/v1/albums/" + id

	resp, err := s.do("GET", url)
	if err != nil {
		return nil, fmt.Errorf("Error while getting album tracks info: %v", err)
	}

	decoder := json.NewDecoder(resp.Body)
	var spotifyAlbumTracks SpotifyAlbumTracks
	err = decoder.Decode(&spotifyAlbumTracks)
	if err != nil {
		return nil, err
	}

	return &spotifyAlbumTracks, nil
}

//GetSpotifyAlbum eliminates required information to create a download queue in
//the bot package from the getAlbumTracks function response  and creates a slice
//of SpotifyPlaylist struct.
func (s *SpotifyAPI) GetSpotifyAlbum(id string) ([]SpotifyPlaylist, error) {
	spotifyAlbumTracks, err := s.getAlbumTracks(id)
	if err != nil {
		return nil, err
	}

	spotifyPlaylistList := []SpotifyPlaylist{}

	coverUrl := spotifyAlbumTracks.Images[1].Url

	tracks := spotifyAlbumTracks.Tracks

	for i, value := range tracks.Items {
		if i > 20 {
			break
		}

		artistNames := ""
		for _, artistValue := range value.Artists {
			artistNames += artistValue.Name + " "
		}

		spotifyPlaylist := SpotifyPlaylist{
			TrackName:   value.Name,
			CoverUrl:    CoverUrl,
			ArtistNames: artistNames,
		}

		spotifyPlaylistList = append(spotifyPlaylistList, spotifyPlaylist)
	}
	return spotifyPlaylistList, nil
}
