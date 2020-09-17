package util

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	//Spotify url types
	SPOTIFYPLAYLISTURL = 1
	SPOTIFYALBUMURL    = 2
	SPOTIFYTRACKURL    = 3

	UNKNOWNURL = -1

	YOUTUBEPLAYLISTURL = 11
	YOUTUBETRACKURL    = 13

	//PATH CONSTS
	BASECOVERPATH = "cover"
	BASESONGPATH  = "song"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	//var ytPlaylistUrlRegex = `/^.*(youtu.be\/|list=)([^#\&\?]*).
	//ytPlaylistUrlRegex = `^.*(youtube.com\/playlist\?list=)([^#\&\?]*)`

	//var ytPlaylistUrlRegex = `^https?:\/\/(www.youtube.com|youtube.com)\/playlist(.*)$`
	durationRegex = `P(?P<years>\d+Y)?(?P<months>\d+M)?(?P<days>\d+D)?T?(?P<hours>\d+H)?(?P<minutes>\d+M)?(?P<seconds>\d+S)?`

	//http spotify urls regex for track, album, playlist ID's.
	spotifyHttpUrlRegex      = `^(?:https?:\/\/open.spotify.com\/(?:playlist\/|album\/|track\/)([a-zA-Z0-9]+))(?:.*)`
	spotifyHttpPlaylistRegex = `^(https:\/\/open.spotify.com\/playlist\/[[a-zA-Z0-9]{22}\?.*)$`
	spotifyHttpAlbumRegex    = `^(https:\/\/open.spotify.com\/album\/[[a-zA-Z0-9]{22}\?.*)$`
	spotifyHttpTrackRegex    = `^(https:\/\/open.spotify.com\/track\/[[a-zA-Z0-9]{22}\?.*)$`

	//youtube url regex
	ytUrlRegex = `^(?:https?\:\/\/)?(?:www\.)?(?:(?:youtube\.com\/watch\?v=)|(?:youtu.be\/))([a-zA-Z0-9\-_]{11})+.*$|^(?:https:\/\/www.youtube.com\/playlist\?list=)([a-zA-Z0-9\-_].*).*$`

	ytPlaylistUrlRegex = `^(?:https:\/\/www.youtube.com\/playlist\?list=)([a-zA-Z0-9\-_]{34}).*$`
	ytTrackUrlRegex    = `^(?:https?\:\/\/)?(?:www\.)?(?:(?:youtube\.com\/watch\?v=)|(?:youtu.be\/))([a-zA-Z0-9\-_]{11})+.*$`
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

//FormatVideoTitle formats given string appropriate for file name.
func FormatVideoTitle(videoTitle string) string {
	newTitle := strings.TrimSpace(videoTitle)

	stringReplacer := strings.NewReplacer("/", "_", "-", "_", ",", "_", " ", "", "'", "")

	newTitle = stringReplacer.Replace(newTitle)
	//videoFileFullPath := path.Join(BASESONGPATH, newTitle)

	return newTitle
}

//GetVideoPath returns formatted version of the given video title
//as full file video path.
func GetVideoPath(videoTitle string) string {
	formattedTitlePath := FormatVideoTitle(videoTitle) + ".m4a"

	formattedTitleFullPath := path.Join(BASESONGPATH, formattedTitlePath)
	return formattedTitleFullPath
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

//DeleteSoundAndCoverFile deletes given sound file and cover file.
func DeleteSoundAndCoverFile(soundFilePath, coverFilePath string) {
	err := DeleteFile(soundFilePath)
	if err != nil {
		log.Printf("Error while deleting sound file %s: %v\n", soundFilePath, err)
	} else {
		log.Printf("%s is deleted.", soundFilePath)
	}

	errCover := DeleteFile(coverFilePath)
	if errCover != nil {
		log.Printf("Error while deleting cover file %s: %v\n", coverFilePath, err)
	} else {
		log.Printf("%s is deleted.", coverFilePath)
	}
}

//IsSpotifyUrl checks given URL is a valid Spotify URL or not.
//If given URL is valid then returns true, otherwise false.
func IsSpotifyUrl(url string) bool {
	re := regexp.MustCompile(spotifyHttpUrlRegex)
	if re.MatchString(url) {
		return true
	}
	return false
}

//GetSpotifyID returns ID of playlist, album or track from the given URL.
func GetSpotifyID(url string) string {
	re := regexp.MustCompile(spotifyHttpUrlRegex)
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		return ""
	}
	return matches[1]
}

//GetSpotifyUrlType returns the type of the given url in integer.
func GetSpotifyUrlType(url string) int {
	if strings.Contains(url, "playlist") {
		return SPOTIFYPLAYLISTURL
	} else if strings.Contains(url, "album") {
		return SPOTIFYALBUMURL
	} else if strings.Contains(url, "track") {
		return SPOTIFYTRACKURL
	} else {
		return UNKNOWNURL
	}
}

//IsYoutubeUrl check given is a valid Youtube URL or not.
//If given URL is valid then returns true, otherwise false.
func IsYoutubeUrl(url string) bool {
	re := regexp.MustCompile(ytUrlRegex)
	if re.MatchString(url) {
		return true
	}
	return false
}

//GetYoutubeID returns ID of playlist or track from the given URL.
func GetYoutubeID(url string) string {
	var re *regexp.Regexp

	if strings.Contains(url, "playlist") {
		re = regexp.MustCompile(ytPlaylistUrlRegex)
	} else {
		re = regexp.MustCompile(ytTrackUrlRegex)
	}
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		return ""
	}
	return matches[1]
}

//GetYoutubeUrlType returns the type of the given url in integer.
func GetYoutubeUrlType(url string) int {
	//Youtube has lots of different URL formats. So this makes
	//hard to detect URL formats. If a given Youtbe URL is not a
	//playlist URL then YOUTUBETRACKURL code will be returned.

	//There is also Mixes that we think of as playlist.
	//I haven't decide whether Mixes treated as playlist
	//or not(even bot don't accept Mix URL). So until Mix
	//and playlist concept is decided, Mix URLs will not be accepted.
	if strings.Contains(url, "playlist") {
		return YOUTUBEPLAYLISTURL
	} else {
		return YOUTUBETRACKURL
	}
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

//GetYtVideoID trims video id from the given url.
func GetYtVideoID(url string) string {
	//format: https://www.youtube.com/watch?v=qT6XCvDUUsU
	return strings.TrimPrefix(url, "https://www.youtube.com/watch?v=")
}

//GetYoutubePlaylistID trims youtube playlist id from the url.
//Checks given url if it is not playlist url return empty string ("")
//otherwise trims playlist id and returns.
func GetYoutubePlaylistID(url string) string {
	if !ValidateYoutubePlaylistUrl(url) {
		return ""
	}
	return strings.TrimPrefix(url, "https://www.youtube.com/playlist?list=")
}

//GetCoverImage downloads album cover image from the
//given url and returns its path.
func GetCoverImage(coverUrl string) (string, error) {
	resp, err := http.Get(coverUrl)
	if err != nil {
		return "", fmt.Errorf("Error while getting cover image: %v", err)
	}
	defer resp.Body.Close()

	imgFileName := RandStringRunes(15) + ".jpg"

	imgFileFullPath := path.Join(BASECOVERPATH, imgFileName)

	imgFile, err := os.Create(imgFileFullPath)
	if err != nil {
		return "", fmt.Errorf("Error while creating cover image file: %v", err)
	}
	defer imgFile.Close()

	_, err = io.Copy(imgFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error while getting cover image file: %v", err)
	}
	return imgFileFullPath, nil
}

//ValiadateYoutubeUrl validates whether given url is a youtube url or not.
func ValidateYoutubeUrl(url string) bool {
	r, err := regexp.Compile(ytUrlRegex)
	if err != nil {
		log.Println(err)
		return false
	}
	return r.MatchString(url)
}

//ValidateYoutubePlaylistUrl validates whether given url is a youtube playlist url or not.
func ValidateYoutubePlaylistUrl(url string) bool {
	r, err := regexp.Compile(ytPlaylistUrlRegex)
	if err != nil {
		log.Println(err)
		return false
	}

	return r.MatchString(url)
}

//ParseISO8601 takes a duration in format ISO8601 and parses to
//MM:SS format.
func ParseISO8601(duration string) string {
	r, err := regexp.Compile(durationRegex)
	if err != nil {
		log.Println(err)
		return ""
	}

	matches := r.FindStringSubmatch(duration)

	years := parseInt64(matches[1])
	months := parseInt64(matches[2])
	days := parseInt64(matches[3])
	hours := parseInt64(matches[4])
	minutes := parseInt64(matches[5])
	seconds := parseInt64(matches[6])

	hour := int64(time.Hour)
	minute := int64(time.Minute)
	second := int64(time.Second)

	return time.Duration(years*24*365*hour +
		months*30*24*hour + days*24*hour +
		hours*hour + minutes*minute + seconds*second).String()
}

func parseInt64(value string) int64 {
	if len(value) == 0 {
		return 0
	}

	parsed, err := strconv.Atoi(value[:len(value)-1])
	if err != nil {
		return 0
	}

	return int64(parsed)
}

//CreateCoverFolder creates a folder called cover
//if it's not already exists.
func CreateCoverFolder() error {
	_, err := os.Stat(BASECOVERPATH)
	if !os.IsNotExist(err) {
		return nil
	} else {
		mkdirErr := os.Mkdir(BASECOVERPATH, 0755)
		if mkdirErr != nil {
			return fmt.Errorf("Couldn't create cover folder: %v", err)
		}
		return nil
	}
}

//CreateSongFolder creates a folder called song
//if it's not already exists.
func CreateSongFolder() error {
	_, err := os.Stat(BASESONGPATH)
	if !os.IsNotExist(err) {
		return nil
	} else {
		mkdirErr := os.Mkdir(BASESONGPATH, 0755)
		if mkdirErr != nil {
			return fmt.Errorf("Couldn't create song folder: %v", err)
		}
		return nil
	}
}
