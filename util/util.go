package util

import (
	"fmt"
	"os"
	"strings"
)

func FormatVideoTitle(videoTitle string) string {
	newTitle := strings.Trim(videoTitle, " ")
	newTitle = strings.ReplaceAll(newTitle, " ", "_")

	return newTitle
}

//GetWorkingDirPath returns working path
func GetWorkingDirPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Error while getting working dir: %v", err)
	}
	return dir, nil
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Error while deleting %s file: %v", path, err)
	}
	return nil
}

//GetSpotifyPlaylistID returns playlist ID
//from the given url
//TODO: Error checking
func GetSpotifyPlaylistID(url string) string {
	//format 1: spotify:playlist:76tzi26o8O920CYAvVbeYO
	if strings.HasPrefix(url, "spotify") {
		return strings.Trim(strings.Split(url, ":")[2], " ")
	}

	//format 2: https://open.spotify.com/playlist/76tzi26o8O920CYAvVbeYO?si=WKrHWhGVQTSmF7GbeqI5sw
	if strings.HasPrefix(url, "https") {
		trimmed := strings.TrimPrefix(url, "https://open.spotify.com/")
		return strings.Trim(strings.Split(strings.Split(trimmed, "/")[1], "?")[0], " ")
	}

	return ""

	/*
		seperated := strings.Split(url, "?")[1]
		playlistID := strings.Trim(seperated, "si=")
		return playlistID
	*/
}
