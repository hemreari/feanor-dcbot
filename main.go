package main

import (
	//"encoding/binary"
	"encoding/json"
	//"flag"
	"fmt"
	"io/ioutil"
	"log"
	//"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	//"strings"
	"syscall"
	"time"

	"./audio"
	"./config"
	"./spotify"
	"./storage"
	"./youtube"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
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

var buffer = make([][]byte, 0)

func main() {
	banner()
	readConfig(&cfg, "config.json")
	log.Println("Starting Feanor. Hope everything works well.")

	musicDBClient, err := storage.NewMusicDBClient(&cfg)
	if err != nil {
		log.Println(err)
	}

	clientID := cfg.Spotify.ClientID
	clientSecretID := cfg.Spotify.ClientSecretID

	spotifyAPI := spotify.NewSpotifyAPI(clientID, clientSecretID)
	youtubeAPI := youtube.NewYoutubeAPI(cfg.Youtube.ApiKey)

	token, err := spotifyAPI.GetAPIToken()
	if err != nil {
		log.Println(err)
	}

	playlistID := cfg.PlaylistID.Shame
	spotifyPl, err := spotifyAPI.GetTrackFromPlaylist(token.AccessToken, playlistID)
	if err != nil {
		log.Println(err)
	}

	// parse the spotify playlist to artist and track name.
	items := spotifyPl.Items
	for index := range items {
		trackName := items[index].Track.Name

		artistsName := ""
		artists := items[index].Track.Artists
		for artistIndex := range artists {
			artistsName += artists[artistIndex].Name
		}

		exists, err := musicDBClient.RowExists("spotify_artist_name", artistsName)
		if err != nil {
			log.Println(err)
		}
		if exists {
			continue
		}

		youtubeQueryStr := artistsName + trackName
		youtubeID := youtubeAPI.Search(youtubeQueryStr)

		err = musicDBClient.InsertTrackData(playlistID, artistsName, trackName, youtubeID)
		if err != nil {
			log.Println(err)
		}
	}

	// get youtube video id's from db
	var ytDbId string
	rows, err := musicDBClient.Client.Query("SELECT youtube_url FROM music")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&ytDbId)
		if err != nil {
			log.Println(err)
		}
		//download the video
		youtubeURL := "https://www.youtube.com/watch?v=" + ytDbId

		mp3Path, err := audio.DownloadYTVideo(youtubeURL, &cfg)
		if err != nil {
			log.Println(err)
		}
		err = musicDBClient.InsertLocation(mp3Path, ytDbId)
		if err != nil {
			log.Println(err)
		}
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!airhorn")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.Contains(m.Content, "!feanor") {
		// insert user info to db
	}

	if strings.Contains(m.Content, "secret word") {
		s.ChannelMessageSend(m.ChannelID, "dont use that word")
	}

	if strings.Contains(m.Content, "playmusic") {
		//serverID := m.ChannelID.GuildID
		//go CreateVoiceInstance("https://www.youtube.com/watch?v=9U2CSiklIpo", serverID)
	}

	// check if the message is "!airhorn"
	if strings.HasPrefix(m.Content, "!korna") {

		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID)
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			log.Println(event.Members)
			_, _ = s.ChannelMessageSend(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channel to play a sound.")
			return
		}
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string) (err error) {

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(100 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	fmt.Println("Reading Folder: ", "musics/")
	files, _ := ioutil.ReadDir("musics/")
	for _, f := range files {
		fmt.Println("PlayAudioFile:", f.Name())
		dgvoice.PlayAudioFile(vc, fmt.Sprintf("%s/%s", "musics/", f.Name()), make(chan bool))
	}

	// Send the buffer data.
	for _, buff := range buffer {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
