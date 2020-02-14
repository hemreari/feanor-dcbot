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
	//"../spotify"
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
	dgv           *discordgo.VoiceConnection
	session       *discordgo.Session
	stop          bool
	skip          bool
	isPlaying     bool
	playQueue     *queue.Queue
	downloadQueue *queue.Queue
	errQueue      *queue.Queue
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

	playQueue := createNewQueue()
	downloadQueue := createNewQueue()
	errQueue := createNewQueue()

	vi = &VoiceInstance{
		session:       dg,
		dgv:           nil,
		stop:          false,
		skip:          false,
		isPlaying:     false,
		playQueue:     playQueue,
		downloadQueue: downloadQueue,
		errQueue:      errQueue,
	}

	log.Println("Feanor is running. Press Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
	return nil
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "Valinor'dan sevgiler.")
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

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	//play commands searchs after !play command
	//and plays the first result.
	if strings.HasPrefix(m.Content, "!play") {
		query := strings.Trim(m.Content, "!play")
		if query == "" {
			return
		}
		vi.prepPlay(query, s, m)
	}

	//skip commands plays next song
	if strings.Compare(m.Content, "!skip") == 0 {
		vi.skipSong(m)
	}

	if strings.Compare(m.Content, "!show") == 0 {
		log.Println("Download Queue: ", vi.downloadQueue)
		log.Println("Play Queue: ", vi.playQueue)
	}
}

func (vi *VoiceInstance) prepPlay(query string, s *discordgo.Session, m *discordgo.MessageCreate) {
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

	if vi.isPlaying == false {
		err = vi.downloadQuery(query, m.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
	}

	if vi.isPlaying == true {
		vi.downloadQueue.Put(query)
		chanDownloadStat := make(chan int)
		go vi.processDownloadQueue(chanDownloadStat, m.ChannelID)
		return
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			dgv, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
			vi.dgv = dgv
			if err != nil {
				fmt.Printf("Couldn't join the voice channel: %v\n", err)
				return
			}
			vi.playQueueFunc(m.ChannelID)
			return
		}
	}
}

func (vi *VoiceInstance) playQueueFunc(messageChannelID string) {
	err := vi.dgv.Speaking(true)
	if err != nil {
		log.Println("Couldn't set speaking", err)
	}

	defer func() {
		vi.disconnectBot()
	}()

	chanDownloadStat := make(chan int)
	chanPlayStat := make(chan int)
	for {
		if !vi.downloadQueue.Empty() {
			go vi.processDownloadQueue(chanDownloadStat, messageChannelID)
			downloadStat := <-chanDownloadStat
			//handle downloadstat -1 and -2 status.
			log.Println("chanDownloadStat: ", downloadStat)
			if downloadStat == -2 || downloadStat == -1 {
				continue
			}
		}

		//first song download is finished. we can proceed to
		//play part
		vi.isPlaying = true
		go vi.processPlayQueue(chanPlayStat, messageChannelID)
		if vi.downloadQueue.Empty() {
			playStat := <-chanPlayStat
			log.Println("playStat:", playStat)
			if playStat == 0 {
				return
			}
		} else {
			select {
			case playStat := <-chanPlayStat:
				log.Println("chanPlayStat: ", playStat)
			default:
				continue
			}
		}
	}
}

func (vi *VoiceInstance) processDownloadQueue(r chan<- int, channelID string) {
	if vi.downloadQueue.Empty() {
		log.Println("Download queue is empty. Closing the channel")
		r <- 1
		return
	}

	nextQuery, err := vi.downloadQueue.Get(1)
	if err != nil {
		log.Println(err)
		r <- -2
		return
	}

	query := strings.Trim(fmt.Sprintf("%v", nextQuery), "[]")

	songPath, err := yt.SearchDownload(query)
	if err != nil {
		log.Println(err)
		log.Printf("Putting %s to the error queue.", query)
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		vi.errQueue.Put(nextQuery)
		r <- -1
		return
	}

	vi.playQueue.Put(songPath)
	r <- 1
	return
}

func (vi *VoiceInstance) downloadQuery(query, channelID string) error {
	songPath, err := yt.SearchDownload(query)
	if err != nil {
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		log.Printf("Putting %s to the error queue.", query)
		vi.errQueue.Put(query)
		return err
	}

	vi.playQueue.Put(songPath)
	return nil
}

func (vi *VoiceInstance) processPlayQueue(playStat chan<- int, messageChannelID string) {
	stop := make(chan int)
	nextItem, err := vi.playQueue.Get(1)
	if err != nil {
		log.Printf("Error while getting item from playlist: %v", err)
		return
	}

	nextItemPath := strings.Trim(fmt.Sprintf("%v", nextItem), "[]")
	vi.sendNowPlayingToChannel(messageChannelID, nextItemPath)
	go vi.playAudioFile(nextItemPath, stop)
	stat := <-stop

	if stat == 0 || stat == 1 {
		errDeleteFile := util.DeleteFile(nextItemPath)
		if err != nil {
			log.Println(errDeleteFile)
		} else {
			log.Printf("%s is deleted.", nextItemPath)
		}
	}
	playStat <- stat
}

func (vi *VoiceInstance) playAudioFile(filename string, stop chan<- int) {
	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar",
		strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		log.Println("StdoutPipe Error:", err)
		stop <- -1
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)
	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		log.Println("RunStart Error", err)
		stop <- -1
		return
	}

	send := make(chan []int16, 2)

	go func() {
		SendPCM(vi.dgv, send)
	}()

	for {
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		//song is played and there is still song to play in the play queue
		if (err == io.EOF || err == io.ErrUnexpectedEOF) && !vi.playQueue.Empty() {
			vi.isPlaying = false
			stop <- 1
		}

		//EOF received and play queue is empty means that
		//song is played and there is left no song to play.
		//we can end the play process by sending 0(int) to channel.
		if (err == io.EOF || err == io.ErrUnexpectedEOF) && vi.playQueue.Empty() {
			log.Println("EOF received and play queue is empty. Ending play process.")
			err = run.Process.Kill()
			stop <- 0
			return
		}

		if (err != io.EOF && err != io.ErrUnexpectedEOF) && err != nil {
			log.Println("Error reading from ffmpeg stdout: ", err)
			stop <- -1
			return
		}

		//handle skip
		if vi.skip == true {
			//if playqueue is not empty send 1(int) to the channel
			//to play next song on the queue.
			if vi.isPlaying == true && !vi.playQueue.Empty() {
				vi.skip = false
				err = run.Process.Kill()
				stop <- 1
				return
			}

			//if the song that now playing is last song on the queue
			//means that there is left no song to play next so stop the
			//play process by sending 0(int) to the channel.
			if vi.isPlaying == true && vi.playQueue.Empty() {
				vi.skip = false
				stop <- 0
				return
			}
		}

		select {
		case send <- audiobuf:
		}
	}
}

//disconnectBot disconnects bot from the voice channel
func (vi *VoiceInstance) disconnectBot() {
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

func (vi *VoiceInstance) skipSong(m *discordgo.MessageCreate) {
	//if currently no song is playing, no need to skip it.
	if vi.isPlaying == false {
		log.Println("No song is playing. skip returning.")
		return
	}

	c, err := vi.session.State.Channel(m.ChannelID)
	if err != nil {
		log.Printf("Couldn't find channel: %v\n", err)
		// Could not find channel.
		return
	}

	g, err := vi.session.Guild(c.GuildID)
	if err != nil {
		log.Printf("Couldn't find guild: %v\n", err)
		// Could not find guild.
		return
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			vi.skip = true
			return
		}
	}
}

//sendMessageToChannel sends the text to channel that given id.
func (vi *VoiceInstance) sendMessageToChannel(channelID, text string) {
	_, err := vi.session.ChannelMessageSend(channelID, text)
	if err != nil {
		log.Printf("Error while sending message to channel: %v", err)
	}
	return
}

//sendNowPlayingToChannel sends the playing song name message to channel.
func (vi *VoiceInstance) sendNowPlayingToChannel(channelID, songTitle string) {
	messageText := "Now Playing " + songTitle
	vi.sendMessageToChannel(channelID, messageText)
	return
}

//createNewQueue create new playlist queue and
//returns newly created queue.
func createNewQueue() *queue.Queue {
	newQueue := queue.New(20)
	return newQueue
}
