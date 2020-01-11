package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"./bot"
	"./config"
	//"./spotify"
	"./youtube"
)

var cfg config.Config

func readConfig(cfg *config.Config, configFileName string) {
	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatal("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func banner() {
	b, err := ioutil.ReadFile("asciiart.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func main() {
	banner()
	readConfig(&cfg, "config.json")
	log.Println("Starting Feanor.")

	//clientID := cfg.Spotify.ClientID
	//clientSecretID := cfg.Spotify.ClientSecretID

	//make api connections
	//spotifyAPI := spotify.NewSpotifyAPI(clientID, clientSecretID)
	youtubeAPI := youtube.NewYoutubeAPI(cfg.Youtube.ApiKey)

	/*
		token, err := spotifyAPI.GetAPIToken()
		if err != nil {
			log.Println(err)
		}
	*/
	/*
		//get spotify playlist
		playlistID := cfg.PlaylistID.Shame
		spotifyPlTracks, err := spotifyAPI.GetTrackFromPlaylist(token.AccessToken, playlistID)
		if err != nil {
			log.Println(err)
		}

		//get spotify playlist tracks
		spotifyPl, err := spotifyAPI.GetPlaylist(token.AccessToken, playlistID)
		if err != nil {
			log.Println(err)
		}

		//playlistName := spotifyPl.Name
		log.Println("playlistName: ", spotifyPl.Name)

		//playlistOwner := spotifyPl.Owner.ID
		log.Println("playlistOwner: ", spotifyPl.Owner.ID)

		// parse the spotify playlist to artist and track name.
		items := spotifyPlTracks.Items
		for index := range items {
			trackName := items[index].Track.Name

			artistsName := ""
			artists := items[index].Track.Artists
			for artistIndex := range artists {
				artistsName += artists[artistIndex].Name
			}

			youtubeQueryStr := artistsName + trackName

			ytSearchResult := youtubeAPI.GetVideoID(youtubeQueryStr)

			_, err := youtube.DownloadVideo(ytSearchResult, &cfg)
			if err != nil {
				log.Println(err)
			}
		}
	*/

	err := bot.InitBot(cfg.Discord.Token, youtubeAPI, &cfg)
	if err != nil {
		log.Println(err)
	}
}
