package youtube

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	//"strings"

	"github.com/hemreari/feanor-dcbot/util"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const (
	DefaultPlaylistItemCount int64 = 20
)

type YoutubeAPI struct {
	DeveloperKey string
}

type SearchResult struct {
	VideoID    string
	VideoTitle string
	Duration   string
	VideoPath  string
	CoverPath  string
}

func NewYoutubeAPI(developerKey string) *YoutubeAPI {
	return &YoutubeAPI{
		DeveloperKey: developerKey,
	}
}

//GetVideoID searches given query on the youtube and returns
//first video's ID and Title.
//!!!! SOME SEARCH RESULTS ON YT DOESN'T RETURN ID. HANDLE ERROR.
//FOR NOW I'M DOING IT ON func DownloadVideo.
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
				Duration:   y.GetDurationByID(item.Id.VideoId),
				CoverPath:  "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg",
			}
		default:
			return &SearchResult{}
		}
	}

	return &SearchResult{}
}

func (y *YoutubeAPI) GetVideoResults(query string) *[]SearchResult {
	developerKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error while creating new YouTube client: %v", err)
	}

	var results []SearchResult

	call := service.Search.List("id,snippet").Q(query)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
	}

	for _, item := range response.Items {
		searchResult := SearchResult{}
		switch item.Id.Kind {
		case "youtube#video":
			if item.Snippet.Title == "" {
				continue
			}
			searchResult.VideoID = item.Id.VideoId
			searchResult.VideoTitle = item.Snippet.Title
			searchResult.Duration = y.GetDurationByID(item.Id.VideoId)
			searchResult.CoverPath = "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg"
			results = append(results, searchResult)
		default:
			results = append(results, searchResult)
		}
	}
	return &results
}

func (y *YoutubeAPI) GetDurationByID(id string) string {
	devKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error while creating new Youtube client: %v", err)
	}

	call := service.Videos.List("id,contentDetails").Id(id)
	response, err := call.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range response.Items {
		return util.ParseISO8601(item.ContentDetails.Duration)
	}
	return ""
}

func (y *YoutubeAPI) DownloadVideo(searchResult *SearchResult) (string, error) {
	videoPath := util.FormatVideoTitle(searchResult.VideoTitle) + ".m4a"

	if searchResult.VideoID == "" {
		return "", fmt.Errorf("Coulnd't get a video ID.")
	}

	log.Printf("Starting to download: %s\n", videoPath)

	ytdlArgs := []string{
		"--force-ipv4",
		"-f",
		"'bestaudio[ext=m4a]",
		"-o",
		videoPath,
		searchResult.VideoID,
	}

	cmd := exec.Command("youtube-dl", ytdlArgs...)
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error while downloading %s: %v", videoPath, err)
	}

	return videoPath, nil
}

//DownloadVideo downloads video and returns video path.
func DownloadVideo(searchResult *SearchResult) (string, error) {
	videoTitle := searchResult.VideoTitle
	videoPath := videoTitle + ".m4a"

	if searchResult.VideoID == "" {
		return "", fmt.Errorf("Couldn't find the video ID.")
	}

	log.Printf("Starting to download: %s\n", videoTitle)

	ytdlArgs := []string{
		"--force-ipv4",
		"-f",
		"'bestaudio[ext=m4a]",
		"-o",
		videoPath,
		searchResult.VideoID,
	}

	cmd := exec.Command("youtube-dl", ytdlArgs...)
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error while downloading %s: %v", videoTitle, err)
	}

	return videoPath, nil
}

//SearchDownload combines abilities of GetVideoID and
//DownloadVideo func's as a standalone function
func (y *YoutubeAPI) SearchDownload(query string) (*SearchResult, error) {
	searchRes := y.GetVideoID(query)

	path, err := DownloadVideo(searchRes)
	if err != nil {
		return nil, err
	}
	searchRes.VideoPath = path
	return searchRes, nil
}

//GetInfoByID returns video information about the given video id.
func (y *YoutubeAPI) GetInfoByID(id string) *SearchResult {
	devKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error while creating new YouTube client: %v", err)
	}

	call := service.Videos.List("id,contentDetails,snippet").Id(id)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
	}

	for _, item := range response.Items {
		switch item.Kind {
		case "youtube#video":
			newTitle := util.FormatVideoTitle(item.Snippet.Title)
			return &SearchResult{
				VideoID:    item.Id,
				VideoTitle: newTitle,
				Duration:   util.ParseISO8601(item.ContentDetails.Duration),
			}
		default:
			return &SearchResult{}
		}
	}
	return nil
}

//GetYoutubePlaylistByID makes the api request to Youtube Data API to
//get information about the given playlist ID.
func (y *YoutubeAPI) GetYoutubePlaylistByID(playlistID string) (*youtube.PlaylistItemListResponse, error) {
	devKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, fmt.Errorf("Error while creating new Youtube client: %v", err)
	}

	call := service.PlaylistItems.List("snippet").PlaylistId(playlistID).MaxResults(DefaultPlaylistItemCount)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("Error while doing get Youtube playlist by ID request: %v", err)
	}

	return response, nil
}
