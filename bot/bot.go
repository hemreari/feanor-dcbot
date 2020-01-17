package bot

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"../config"
	"../util"
	"../youtube"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

type VoiceInstance struct {
	dgv       *discordgo.VoiceConnection
	session   *discordgo.Session
	stop      bool
	skip      bool
	isPlaying bool
	playlist  *queue.Queue
}

var (
	speakers    map[uint32]*gopus.Decoder
	opusEncoder *gopus.Encoder
	mu          sync.Mutex
	yt          *youtube.YoutubeAPI
	cfg         *config.Config
	vi          *VoiceInstance
)

func InitBot(botToken string, ytAPI *youtube.YoutubeAPI, config *config.Config) error {
	yt = ytAPI
	cfg = config
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return fmt.Errorf("Error while creating discord session: %v", err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	err = dg.Open()
	if err != nil {
		return fmt.Errorf("Error while opening discord session: %v", err)
	}

	playlist := queue.New(20)

	vi = &VoiceInstance{
		session:   dg,
		dgv:       nil,
		stop:      false,
		skip:      false,
		isPlaying: false,
		playlist:  playlist,
	}

	log.Println("Feanor is running. Press Ctrl-c to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()

	return nil
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "Valinor'dan sevgiler.")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	/*
		if strings.Contains(m.Content, "!feanor") {
		}
	*/

	//show command prints current playlist.
	if strings.HasPrefix(m.Content, "!show") {
		vi.displayPlaylist(m)
	}

	//skip commands plays next song
	if strings.Compare(m.Content, "!skip") == 0 {
		vi.SkipSong()
	}

	//play commands searchs after !play command
	//and plays the first result.
	if strings.HasPrefix(m.Content, "!play") {
		query := strings.Trim(m.Content, "!play")
		if query == "" {
			return
		}

		videoPath, err := yt.SearchDownload(query)
		if err != nil {
			log.Println(err)
			return
		}

		if vi.isPlaying == true && !vi.playlist.Empty() {
			vi.playlist.Put(videoPath)
			return
		}

		if vi.isPlaying == true && vi.playlist.Empty() {
			vi.playlist.Put(videoPath)
			return
		}

		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			log.Printf("Couldn't find channel: %v\n", err)
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			log.Printf("Couldn't find guild: %v\n", err)
			// Could not find guild.
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				dgv, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
				vi.dgv = dgv
				if err != nil {
					fmt.Printf("Couldn't join the voice channel: %v\n", err)
					return
				}
				//if playlist queue is empty, put first
				//item and play the playlist
				if vi.playlist.Empty() && vi.isPlaying == false {
					vi.playlist.Put(videoPath)
					vi.PlayQueue(make(chan bool))
				}
				return
			}
		}
	}

	//stop commands stops music if any music is playing
	if strings.Compare(m.Content, "!stop") == 0 {
		if vi.isPlaying == false {
			return
		}

		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			log.Printf("Couldn't find channel: %v\n", err)
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			log.Printf("Couldn't find guild: %v\n", err)
			// Could not find guild.
			return
		}

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				vi.StopMusic()
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
			_, _ = s.ChannelMessageSend(channel.ID, "Welcome")
			return
		}
	}
}

func (vi *VoiceInstance) StopMusic() {
	vi.stop = true
	vi.isPlaying = false
}

func (vi *VoiceInstance) SkipSong() {
	vi.skip = true
}

func (vi *VoiceInstance) PlayQueue(stop <-chan bool) {
	//check playlist queue empty or not
	if vi.playlist.Empty() {
		log.Printf("Playlist is empty, quiting.\n")
		return
	}
	// Send "speaking" packet over the voice websocket
	err := vi.dgv.Speaking(true)
	if err != nil {
		log.Println("Couldn't set speaking", err)
	}

	defer func() {
		err := vi.dgv.Speaking(false)
		if err != nil {
			log.Println("Couldn't stop speaking", err)
		}
		vi.dgv.Disconnect()
		log.Printf("Bot disconnected from the voice channel.\n")
		vi.stop = false
		vi.isPlaying = false
	}()

	//if playlist is not empty, retrieve first item
	//from the queue and play it.
	vi.isPlaying = true
	for !vi.playlist.Empty() {
		nextItem, err := vi.playlist.Get(1)
		if err != nil {
			log.Printf("Error while getting item from playlist: %v", err)
			return
		}

		nextItemPath := strings.Trim(fmt.Sprintf("%v", nextItem), "[]")
		log.Println("Next item: ", nextItemPath)
		vi.PlayAudioFile(nextItemPath, stop)
	}
}

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.
func (vi *VoiceInstance) PlayAudioFile(filename string, stop <-chan bool) {
	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar",
		strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		log.Println("StdoutPipe Error:", err)
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		log.Println("RunStart Error", err)
		return
	}

	go func() {
		<-stop
		err = run.Process.Kill()
	}()

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		err := util.DeleteFile(filename)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("%s is deleted\n", filename)
		}
	}()

	send := make(chan []int16, 2)
	defer close(send)

	close := make(chan bool)
	go func() {
		SendPCM(vi.dgv, send)
		close <- true
	}()

	log.Printf("Now playing %s.\n", filename)

	for {
		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			log.Println("error reading from ffmpeg stdout", err)
			return
		}

		if vi.skip == true {
			err = run.Process.Kill()
			if err != nil {
				log.Printf("Error while killing process: %v", err)
			}
			vi.skip = false
			return
		}

		if vi.stop == true {
			vi.disconnectBot()
			return
		}

		// Send received PCM to the sendPCM channel
		select {
		case send <- audiobuf:
		case <-close:
			return
		}
	}
}

//disconnectBot removes bot from the voice channel
//and clears playlist queue.
func (vi *VoiceInstance) disconnectBot() {
	//delete all items in the playlist queue
	for !vi.playlist.Empty() {
		nextItem, err := vi.playlist.Get(1)
		if err != nil {
			log.Printf("Error while getting item from playlist: %v", err)
			return
		}

		nextItemPath := strings.Trim(fmt.Sprintf("%v", nextItem), "[]")
		util.DeleteFile(nextItemPath)
	}

	vi.playlist.Dispose()

	log.Println("All files has been deleted and queue is disposed.")

	err := vi.dgv.Speaking(false)
	if err != nil {
		log.Println("Couldn't stop speaking", err)
	}

	vi.dgv.Disconnect()
	log.Printf("Bot disconnected from the voice channel.\n")
	vi.stop = false
	vi.isPlaying = false
	return
}

func (vi *VoiceInstance) displayPlaylist(m *discordgo.MessageCreate) error {
	log.Println("playlist:", vi.playlist)
	return nil
}

// SendPCM will receive on the provied channel encode
// received PCM data into Opus then send that to Discordgo
func SendPCM(v *discordgo.VoiceConnection, pcm <-chan []int16) {
	if pcm == nil {
		return
	}

	var err error

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		log.Println("NewEncoder Error", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recv, ok := <-pcm
		if !ok {
			log.Println("PCM channel closed")
			return
		}

		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			log.Println("Encoding Error", err)
			return
		}

		if v.Ready == false || v.OpusSend == nil {
			// OnError(fmt.Sprintf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
			// Sending errors here might not be suited
			return
		}
		// send encoded opus data to the sendOpus channel
		v.OpusSend <- opus
	}
}
