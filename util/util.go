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
	return strings.Split(url, ":")[2]
	/*
		seperated := strings.Split(url, "?")[1]
		playlistID := strings.Trim(seperated, "si=")
		return playlistID
	*/
}
