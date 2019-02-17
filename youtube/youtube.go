package youtube

import (
	//"io/ioutil"
	"fmt"
	"log"
	"net/http"

	//"golang.org/x/oauth2"
	//"golang.org/x/oauth2/clientcredentials"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

type YoutubeAPI struct {
	DeveloperKey string
}

func NewYoutubeAPI(developerKey string) *YoutubeAPI {
	return &YoutubeAPI{
		DeveloperKey: developerKey,
	}
}

func (y *YoutubeAPI) Search(query string) string {
	developerKey := y.DeveloperKey

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	// Make the API call to YouTube.
	call := service.Search.List("id,snippet").
		Q(query).
		MaxResults(1)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
	}

	// Group video, channel, and playlist results in separate lists.
	//videos := make(map[string]string)
	//channels := make(map[string]string)
	//playlists := make(map[string]string)

	// Iterate through each item and add it to the correct list.
	for _, item := range response.Items {
		switch item.Id.Kind {
		case "youtube#video":
			return item.Id.VideoId
			//videos[item.Id.VideoId] = item.Snippet.Title
		default:
			return ""
		}
	}

	return ""

	/*
		printIDs("Videos", videos)
		printIDs("Channels", channels)
		printIDs("Playlists", playlists)
	*/
}

// Print the ID and title of each result in a list as well as a name that
// identifies the list. For example, print the word section name "Videos"
// above a list of video search results, followed by the video ID and title
// of each matching video.
func printIDs(sectionName string, matches map[string]string) {
	fmt.Printf("%v:\n", sectionName)
	for id, title := range matches {
		fmt.Printf("[%v] %v\n", id, title)
	}
	fmt.Printf("\n\n")
}
