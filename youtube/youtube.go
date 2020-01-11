package youtube

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"../util"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

type YoutubeAPI struct {
	DeveloperKey string
}

type SearchResult struct {
	VideoID    string
	VideoTitle string
}

func NewYoutubeAPI(developerKey string) *YoutubeAPI {
	return &YoutubeAPI{
		DeveloperKey: developerKey,
	}
}

//GetVideoID searches given query on the youtube and returns
//first video's ID and Title.
func (y *YoutubeAPI) GetVideoID(query string) *SearchResult {
	developerKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error while creating new YouTube client: %v", err)
	}

	// Make the API call to YouTube.
	call := service.Search.List("id,snippet").
		Q(query).
		MaxResults(1)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
	}

	// Iterate through each item and add it to the correct list.
	for _, item := range response.Items {
		switch item.Id.Kind {
		case "youtube#video":
			newTitle := util.FormatVideoTitle(item.Snippet.Title)
			return &SearchResult{
				VideoID:    item.Id.VideoId,
				VideoTitle: newTitle,
			}
		default:
			return &SearchResult{}
		}
	}

	return &SearchResult{}
}

func DownloadVideo(searchResult *SearchResult) (string, error) {
	videoTitle := searchResult.VideoTitle
	videoPath := videoTitle + ".m4a"

	log.Printf("Starting to download: %s\n", videoTitle)

	ytdlArgs := []string{
		"-f",
		"'bestaudio[ext=m4a]",
		"-o",
		videoPath,
		searchResult.VideoID,
	}

	cmd := exec.Command("youtube-dl", ytdlArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error while downloading %s: %v", videoTitle, err)
	}

	return videoPath, nil
}

//SearchDownload combines abilities of GetVideoID and
//DownloadVideo func's as a standalone function
func (y *YoutubeAPI) SearchDownload(query string) (string, error) {
	searchRes := y.GetVideoID(query)

	path, err := DownloadVideo(searchRes)
	if err != nil {
		return "", err
	}
	return path, nil
}
