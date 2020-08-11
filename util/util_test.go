package util

import (
	"strings"
	"testing"
)

var (
	spotifyPlUrl1    = "https://open.spotify.com/playlist/37i9dQZF1DZ06evO1fLAgU?si=5y39c0TYQVKyMbyErnaYIA"
	spotifyAlbumUrl1 = "https://open.spotify.com/album/4HklP3MTUYViTMiNdj43R3?si=OZ98hcJ3Rkqz0CxG5ejexA"
	spotifyTrackUrl1 = "https://open.spotify.com/track/2az3iTNyJ1M1JJnsU2Gq6H?si=EHR4hOb_ThuDYHveInMmcg"
)

func TestGetSpotifyID(t *testing.T) {
	tables := []struct {
		url string
		id  string
	}{
		{spotifyPlUrl1, "37i9dQZF1DZ06evO1fLAgU"},
		{spotifyAlbumUrl1, "4HklP3MTUYViTMiNdj43R3"},
		{spotifyTrackUrl1, "2az3iTNyJ1M1JJnsU2Gq6H"},
	}

	for _, table := range tables {

		id := GetSpotifyID(table.url)

		if strings.Compare(id, table.id) != 0 {
			t.Errorf("id is incorrect, got: %s, want: %s", id, table.id)
		}
	}
}
