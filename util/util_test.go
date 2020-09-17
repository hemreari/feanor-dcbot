package util

import (
	"fmt"
	"strings"
	"testing"
)

var (
	spotifyPlUrl1      = "https://open.spotify.com/playlist/37i9dQZF1DZ06evO1fLAgU?si=5y39c0TYQVKyMbyErnaYIA"
	spotifyAlbumUrl1   = "https://open.spotify.com/album/4HklP3MTUYViTMiNdj43R3?si=OZ98hcJ3Rkqz0CxG5ejexA"
	spotifyTrackUrl1   = "https://open.spotify.com/track/2az3iTNyJ1M1JJnsU2Gq6H?si=EHR4hOb_ThuDYHveInMmcg"
	spotifyUnknownUrl1 = "https://google.com"

	ytPlUrl1      = "https://www.youtube.com/playlist?list=PL4o29bINVT4EG_y-k5jGoOu3-Am8Nvi10"
	ytTrackUrl1   = "https://www.youtube.com/watch?v=B9v8jLBrvug"
	ytTrackUrl2   = "https://youtu.be/SlPhMPnQ58k"
	ytUnknownUrl1 = "https://google.com"
)

func TestGetSpotifyID(t *testing.T) {
	tables := []struct {
		url string
		id  string
	}{
		{spotifyPlUrl1, "37i9dQZF1DZ06evO1fLAgU"},
		{spotifyAlbumUrl1, "4HklP3MTUYViTMiNdj43R3"},
		{spotifyTrackUrl1, "2az3iTNyJ1M1JJnsU2Gq6H"},
		{spotifyUnknownUrl1, ""},
	}

	for _, table := range tables {
		id := GetSpotifyID(table.url)

		if strings.Compare(id, table.id) != 0 {
			t.Errorf("id is incorrect, got: %s, want: %s", id, table.id)
		}
	}
}

func TestGetSpotifyUrlType(t *testing.T) {
	tables := []struct {
		url     string
		urlType int
	}{
		{spotifyPlUrl1, 1},
		{spotifyAlbumUrl1, 2},
		{spotifyTrackUrl1, 3},
		{spotifyUnknownUrl1, -1},
	}

	for _, table := range tables {
		urlType := GetSpotifyUrlType(table.url)

		if urlType != table.urlType {
			t.Errorf("url type is incorrect, got: %d, want: %d", urlType, table.urlType)
		}
	}
}

func TestIsYoutubeURL(t *testing.T) {
	tables := []struct {
		url   string
		valid bool
	}{
		{ytPlUrl1, true},
		{ytTrackUrl1, true},
		{ytTrackUrl2, true},
		{ytUnknownUrl1, false},
	}

	for _, table := range tables {
		isValid := IsYoutubeUrl(table.url)

		if isValid != table.valid {
			t.Errorf("IsYoutubeURL is failed, url: %s", table.url)
		}
	}
}

func TestGetYoutubeID(t *testing.T) {
	tables := []struct {
		url string
		id  string
	}{
		{ytPlUrl1, "PL4o29bINVT4EG_y-k5jGoOu3-Am8Nvi10"},
		{ytTrackUrl1, "B9v8jLBrvug"},
		{ytTrackUrl2, "SlPhMPnQ58k"},
		{ytUnknownUrl1, ""},
	}

	for _, table := range tables {
		id := GetYoutubeID(table.url)

		if strings.Compare(id, table.id) != 0 {
			t.Errorf("id is incorrect, got: %s, want: %s", id, table.id)
		}
	}
}

func TestYoutubeUrlType(t *testing.T) {
	tables := []struct {
		url     string
		urlType int
	}{
		{ytPlUrl1, YOUTUBEPLAYLISTURL},
		{ytTrackUrl1, YOUTUBETRACKURL},
		{ytTrackUrl2, YOUTUBETRACKURL},
		{ytUnknownUrl1, YOUTUBETRACKURL},
	}

	for _, table := range tables {
		urlType := GetYoutubeUrlType(table.url)

		if urlType != table.urlType {
			t.Errorf("Url type is false, got: %d, want %d", table.urlType, urlType)
		}
	}
}

func TestGetCoverImage(t *testing.T) {
	imgUrl := "https://hemreari.com/assets/img/coming_soon_homepage.jpg"

	coverPath, err := GetCoverImage(imgUrl)
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	fmt.Println(coverPath)
}

func TestFormatVideoTitle(t *testing.T) {
	tables := []struct {
		title          string
		formattedTitle string
	}{
		{
			"AC/DC - Thunderstruck (Official Video)",
			"AC_DC_Thunderstruck(OfficialVideo)",
		},
		{
			"The Rolling Stones - Paint It, Black (Official Lyric Video)",
			"TheRollingStones_PaintIt_Black(OfficialLyricVideo)",
		},
		{
			"Help I'm Alive by Metric",
			"HelpImAlivebyMetric",
		},
	}

	for _, table := range tables {
		newTitle := FormatVideoTitle(table.title)

		if strings.Compare(newTitle, table.formattedTitle) != 0 {
			t.Errorf("Formatted video title is not match, got: %s, want: %s", newTitle, table.formattedTitle)
		}
	}
}

func TestGetVideoPath(t *testing.T) {
	tables := []struct {
		title          string
		formattedTitle string
	}{
		{
			"AC/DC - Thunderstruck (Official Video)",
			"song/AC_DC_Thunderstruck(OfficialVideo).m4a",
		},
		{
			"The Rolling Stones - Paint It, Black (Official Lyric Video)",
			"song/TheRollingStones_PaintIt_Black(OfficialLyricVideo).m4a",
		},
		{
			"Help I'm Alive by Metric",
			"song/HelpImAlivebyMetric.m4a",
		},
	}

	for _, table := range tables {
		newTitle := GetVideoPath(table.title)

		if strings.Compare(newTitle, table.formattedTitle) != 0 {
			t.Errorf("Formatted video title is not match, got: %s, want: %s", newTitle, table.formattedTitle)
		}
	}
}
