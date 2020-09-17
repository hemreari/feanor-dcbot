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
	CoverUrl   string
	CoverPath  string
}

func NewYoutubeAPI(developerKey string) *YoutubeAPI {
	return &YoutubeAPI{
		DeveloperKey: developerKey,
	}
}

func (y *YoutubeAPI) GetYoutubePlaylist(id string, urlType int) ([]SearchResult, error) {
	switch urlType {
	case util.YOUTUBEPLAYLISTURL:
		playlist, err := y.HandleYoutubePlaylist(id)
		if err != nil {
			return nil, err
		}
		return playlist, err
	case util.YOUTUBETRACKURL:
		playlist, err := y.HandleYoutubeTrack(id)
		if err != nil {
			return nil, err
		}
		return playlist, err
	}
	return nil, nil
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
				CoverUrl:   "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg",
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
			searchResult.CoverUrl = "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg"
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

//DownloadVideo downloads video with ytdl, returns downloaded video file's path.
func DownloadVideo(videoTitle, videoID string) (string, error) {
	if videoID == "" {
		return "", fmt.Errorf("Coulnd't get a video ID.")
	}

	videoFullPath, err := ytdlExecute(videoTitle, videoID)
	if err != nil {
		return "", err
	}

	return videoFullPath, nil
}

//ytdlExecute executes ytdl command on the OS with proper
//arguments to download a video then returns downloaded
//video file's path.
func ytdlExecute(videoTitle, videoID string) (string, error) {
	videoFullPath := util.GetVideoPath(videoTitle)

	log.Printf("Starting to download: %s\n", videoTitle)

	ytdlArgs := []string{
		"--force-ipv4",
		"-f",
		"'bestaudio[ext=m4a]",
		"-o",
		videoFullPath,
		videoID,
	}

	cmd := exec.Command("youtube-dl", ytdlArgs...)
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error while downloading %s: %v", videoTitle, err)
	}
	return videoFullPath, nil
}

//SearchDownload combines abilities of GetVideoID and
//DownloadVideo func's as a standalone function
func (y *YoutubeAPI) SearchDownload(query string) (*SearchResult, error) {
	searchRes := y.GetVideoID(query)

	path, err := DownloadVideo(searchRes.VideoTitle, searchRes.VideoID)
	if err != nil {
		return nil, err
	}
	searchRes.VideoPath = path
	return searchRes, nil
}

//GetInfoByID returns video information about the given video id.
func (y *YoutubeAPI) GetInfoByID(id string) (*SearchResult, error) {
	devKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, fmt.Errorf("Error while creating new YouTube client: %v", err)
	}

	call := service.Videos.List("id,contentDetails,snippet").Id(id)
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("Error while making call: %v", err)
	}

	for _, item := range response.Items {
		switch item.Kind {
		case "youtube#video":
			snippet := item.Snippet
			//newTitle := util.FormatVideoTitle(snippet.Title)
			return &SearchResult{
				VideoID:    item.Id,
				VideoTitle: snippet.Title,
				Duration:   util.ParseISO8601(item.ContentDetails.Duration),
				CoverUrl:   snippet.Thumbnails.High.Url,
			}, nil
		default:
			return &SearchResult{}, nil
		}
	}
	return nil, nil
}

func (y *YoutubeAPI) HandleYoutubePlaylist(id string) ([]SearchResult, error) {
	plTracks, err := y.getYoutubePlaylistById(id)
	if err != nil {
		return nil, err
	}

	playlist := []SearchResult{}

	for i, playlistItem := range plTracks.Items {
		if i > 20 {
			break
		}

		videoID := playlistItem.Snippet.ResourceId.VideoId
		thumbnailUrl := playlistItem.Snippet.Thumbnails.High.Url
		videoTitle := playlistItem.Snippet.Title

		track := SearchResult{
			VideoID:    videoID,
			VideoTitle: videoTitle,
			CoverUrl:   thumbnailUrl,
			Duration:   y.GetDurationByID(videoID),
		}
		playlist = append(playlist, track)
	}
	return playlist, nil
}

//getYoutubePlaylistById makes the api request to Youtube Data API to
//get information about the given playlist ID.
func (y *YoutubeAPI) getYoutubePlaylistById(id string) (*youtube.PlaylistItemListResponse, error) {
	devKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, fmt.Errorf("Error while creating new Youtube client: %v", err)
	}

	call := service.PlaylistItems.List("snippet").PlaylistId(id).MaxResults(DefaultPlaylistItemCount)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("Error while doing get Youtube playlist by ID request: %v", err)
	}

	return response, nil
}

func (y *YoutubeAPI) HandleYoutubeTrack(id string) ([]SearchResult, error) {
	trackInfo, err := y.GetInfoByID(id)
	if err != nil {
		return nil, err
	}

	playlist := []SearchResult{}
	playlist = append(playlist, *trackInfo)

	return playlist, nil
}
