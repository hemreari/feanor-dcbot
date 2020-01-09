package youtube

import (
	"log"
	"net/http"
	"os/exec"

	//"net/url"
	"os"

	"../config"

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

	// Iterate through each item and add it to the correct list.
	for _, item := range response.Items {
		switch item.Id.Kind {
		case "youtube#video":
			return item.Id.VideoId
		default:
			return ""
		}
	}

	return ""
}

/*
func DownloadMP4(urlStr string, cfg *config.Config) (string, error) {
	log.Println("url: ", urlStr)

	if _, err := os.Stat(cfg.MusicDir.DownloadPath); os.IsNotExist(err) {
		err := os.Mkdir(cfg.MusicDir.DownloadPath, 0777)
		if err != nil {
			return "", err
		}
	}

	info, err := ytdl.GetVideoInfo(urlStr)
	if err != nil {
		return "", err
	}

	format := info.Formats.Worst(ytdl.FormatResolutionKey)[0]

	videoPath := cfg.MusicDir.DownloadPath + info.Title + ".mp4"

	//check video if is download already or not
	//if video is alread downloaded don't download.
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		file, err := os.Create(videoPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		err = info.Download(format, file)
		if err != nil {
			return "", err
		}
	}

	return videoPath, nil
}
*/

func DownloadMP4(urlStr string, cfg *config.Config) error {
	log.Println("Downloading: ", urlStr)

	//check download path is exist given in the config
	//if doesn't exist create folder.
	if _, err := os.Stat(cfg.MusicDir.DownloadPath); os.IsNotExist(err) {
		err := os.Mkdir(cfg.MusicDir.DownloadPath, 0777)
		if err != nil {
			return err
		}
	}

	ytdlArgs := []string{
		"-f",
		"'bestaudio[ext=m4a]",
		urlStr,
	}

	cmd := exec.Command("youtube-dl", ytdlArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	/*
		tmpMp4Path := "Rammstein - Deutschland (Official Video)-NeQM1c-XCDc.m4a"

		convertMP4toDca(tmpMp4Path)
	*/

	return nil
}

/*
func convertMP4toDca(mp4Path string) {
	log.Println("Converting mp4 to dca")
}
*/
