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
	"time"

	"github.com/hemreari/feanor-dcbot/config"
	"github.com/hemreari/feanor-dcbot/spotify"
	"github.com/hemreari/feanor-dcbot/util"
	"github.com/hemreari/feanor-dcbot/youtube"

	"github.com/bwmarrin/discordgo"
	"github.com/hemreari/go-datastructures/queue"
	"layeh.com/gopus"
)

const (
	channels         int    = 2                   // 1 for mono, 2 for stereo
	frameRate        int    = 48000               // audio sampling rate
	frameSize        int    = 960                 // uint16 size of each audio frame
	maxBytes         int    = (frameSize * 2) * 2 // max size of opus data
	youtubeUrlPrefix string = "https://www.youtube.com/watch?v="
	DefaultCoverPath string = "https://github.com/golang/go/blob/master/doc/gopher/fiveyears.jpg"
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

type SongInstance struct {
	title     string
	artist    string
	songPath  string
	coverUrl  string
	coverPath string
	videoID   string
	duration  string
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

func initSpotifyAPI() *spotify.SpotifyAPI {
	spotifyAPI := spotify.NewSpotifyAPI(cfg.Spotify.ClientID, cfg.Spotify.ClientSecretID)
	return spotifyAPI
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
		query := strings.Trim(m.Content, "!play ")
		if query == "" {
			return
		}
		if util.ValidateYtUrl(query) {
			vi.prepYtUrl(query, s, m)
		}
		vi.prepQuery(query, s, m)
		//vi.prepPlay(query, s, m)
	}

	if strings.HasPrefix(m.Content, "!list") {
		link := strings.Trim(m.Content, "!list ")
		if strings.Contains(link, "spotify") {
			playlistID := util.GetSpotifyPlaylistID(link)
			vi.prepSpotifyPlaylist(playlistID, s, m)
		}
	}

	//search commands searchs query on yt and if it's
	//finds anything related plays.
	if strings.HasPrefix(m.Content, "!search") {
		query := strings.Trim(m.Content, "!search ")
		if query == "" {
			vi.sendMessageToChannel(m.ChannelID, "Unsufficient query. Try again, with query.")
			return
		}
		vi.searchOnYoutube(query, s, m)
	}

	//skip commands plays next song
	if strings.Compare(m.Content, "!skip") == 0 {
		vi.skipSong(m)
	}

	//stop commands stops playing song
	if strings.Compare(m.Content, "!stop") == 0 {
		vi.stopSong(m)
	}

	if strings.Compare(m.Content, "!show") == 0 {
		vi.showPlayQueue(m)
	}

	if strings.Compare(m.Content, "!testreddit") == 0 {
	}
}

func (vi *VoiceInstance) validateMessage(s *discordgo.Session, m *discordgo.MessageCreate) (*discordgo.Guild, error) {
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return nil, fmt.Errorf("Couldn't find channel: %v\n", err)
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		// Could not find guild.
		return nil, fmt.Errorf("Couldn't find guild: %v\n", err)
	}

	return g, nil

	/*
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				dgv, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
				vi.dgv = dgv
				if err != nil {
					fmt.Printf("Couldn't join the voice channel: %v\n", err)
					return false
				}
				return true
			}
		}
		return false
	*/
}

func (vi *VoiceInstance) channelVoiceJoin(g *discordgo.Guild, s *discordgo.Session, m *discordgo.MessageCreate) bool {
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			dgv, err := s.ChannelVoiceJoin(g.ID, vs.ChannelID, false, true)
			if err != nil {
				fmt.Printf("Couldn't join the voice channel: %v\n", err)
				return false
			}
			vi.dgv = dgv
			return true
		}
	}
	return false
}

func (vi *VoiceInstance) searchOnYoutube(query string, s *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := vi.validateMessage(s, m)
	if err != nil {
		log.Println(err)
		return
	}

	resultsMap := make(map[int]youtube.SearchResult)

	results := yt.GetVideoResults(query)
	resultCounter := 1

	for _, value := range *results {
		resultsMap[resultCounter] = value
		resultCounter++
	}

	messageID := vi.sendSearchResultMessageToChannel(m.ChannelID, results)

	s.AddHandlerOnce(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		userResponseInt, err := strconv.Atoi(m.Content)
		if err != nil {
			log.Println(err)
			vi.sendMessageToChannel(m.ChannelID, "I accept only numbers.")
		}
		if userResponseInt < 1 && userResponseInt > resultCounter {
			vi.sendMessageToChannel(m.ChannelID, "Not an available option.")
		}
		if strings.HasPrefix(m.Content, "!done") {
			return
		}
		result := resultsMap[userResponseInt]
		vi.sentMessageEditEmbed(m.ChannelID, messageID, &result)
		vi.prepSearchSelectionPlay(&result, s, m)
		return
	})
}

//prepSpotifyPlaylist gets songs from the given Spotify playlist ID
//and parses them as youtube queries to add to the download queue.
//finally starts the play process.
func (vi *VoiceInstance) prepSpotifyPlaylist(playlistID string, s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := vi.validateMessage(s, m)
	if err != nil {
		log.Println(err)
		return
	}

	if !vi.channelVoiceJoin(guild, s, m) {
		return
	}

	//initialise spotify api.
	spotifyAPI := initSpotifyAPI()

	//get playlist tracks
	plTracks, err := spotifyAPI.GetTracksFromPlaylist(playlistID)
	if err != nil {
		log.Printf("Error while getting Spotify playlist tracks: %v", err)
		vi.sendMessageToChannel(m.ChannelID, "Unexpected thing is happened. Please, Try again.")
		return
	}

	//when !list command is received stop the play process
	//if bot has on going play job.
	if vi.isPlaying == true {
		vi.stopSong(m)
	}

	itemCounter := 0

	//parse playlist tracks to artist and track name.
	items := plTracks.Items
	for index := range items {
		//for a single playlist, we are processing maximum 100 songs.
		//It consumes %10.05 of daily quota limit.
		if itemCounter > 20 {
			break
		}
		trackName := items[index].Track.Name

		var coverUrl string
		album := items[index].Track.Album
		if album.Images[1].Url != "" {
			coverUrl = album.Images[1].Url
		}

		var artistsName string
		artists := items[index].Track.Artists
		for artistIndex := range artists {
			artistsName += artists[artistIndex].Name + " "
		}

		songInstance := SongInstance{
			title:    trackName,
			artist:   artistsName,
			coverUrl: coverUrl,
		}

		//first query is not putting in to the download query
		//it's directly downloading.
		if vi.playQueue.Empty() {
			_ = vi.downloadQuery(&songInstance, m.ChannelID)
			itemCounter++
			continue
		}
		vi.downloadQueue.Put(&songInstance)
		itemCounter++
	}

	//start the play process.
	vi.playQueueFunc(m.ChannelID)
}

//prepQuery preperas simple queries like "rammstein deutschland" to play.
func (vi *VoiceInstance) prepQuery(query string, s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := vi.validateMessage(s, m)
	if err != nil {
		log.Println(err)
		return
	}

	if vi.isPlaying == false {
		err = vi.downloadPlayQuery(query, m.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
	}

	if vi.isPlaying == true {
		go vi.downloadPlayQuery(query, m.ChannelID)
		return
	}

	if !vi.channelVoiceJoin(guild, s, m) {
		return
	}
	vi.playQueueFunc(m.ChannelID)
	return
}

//prepYtUrl prepares given yt urls like "https://www.youtube.com/watch?v=6MyAOqrPACY" to play.
func (vi *VoiceInstance) prepYtUrl(url string, s *discordgo.Session, m *discordgo.MessageCreate) {
	videoID := util.GetYtVideoID(url)

	yt.GetInfoByID(videoID)
	return
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
		err = vi.downloadPlayQuery(query, m.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
	}

	if vi.isPlaying == true {
		go vi.downloadPlayQuery(query, m.ChannelID)
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

func (vi *VoiceInstance) prepSearchSelectionPlay(searchResult *youtube.SearchResult, s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, err := vi.validateMessage(s, m)
	if err != nil {
		return
	}

	if vi.isPlaying == false {
		err = vi.downloadSelection(searchResult, m.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
	}

	if vi.isPlaying == true {
		vi.stopSong(m)
		vi.downloadSelection(searchResult, m.ChannelID)
	}

	if !vi.channelVoiceJoin(guild, s, m) {
		return
	}
	vi.playQueueFunc(m.ChannelID)
}

func (vi *VoiceInstance) playQueueFunc(channelID string) {
	err := vi.dgv.Speaking(true)
	if err != nil {
		log.Println("Couldn't set speaking", err)
	}

	defer func() {
		vi.disconnectBot()
	}()

	chanPlayStat := make(chan int)
	for {
		if !vi.downloadQueue.Empty() {
			maxNumberofGoroutines := 2

			conGoroutines := make(chan struct{}, maxNumberofGoroutines)

			for i := 0; i < maxNumberofGoroutines; i++ {
				conGoroutines <- struct{}{}
			}

			doneLimitChan := make(chan bool)
			waitForAllJobs := make(chan bool)

			go func() {
				for i := 0; i < int(vi.downloadQueue.Len()); i++ {
					<-doneLimitChan
					conGoroutines <- struct{}{}
				}
				waitForAllJobs <- true
			}()

			for i := 1; i <= int(vi.downloadQueue.Len()); i++ {
				<-conGoroutines
				go func(id int) {
					vi.processDownloadQueue(channelID)
					doneLimitChan <- true
				}(i)
			}

			<-waitForAllJobs
		}

		//isPlaying is false means that bot is not  playing any song
		//at the moment and play queue is empty so we can call processPlayQueue
		//function to play a song from the play queue.
		//I added isPlaying condition to prevent calling other processPlayQueue
		//goroutinues that causes playing multiple song simultaneously.
		if vi.isPlaying == false && !vi.playQueue.Empty() {
			vi.isPlaying = true
			go vi.processPlayQueue(chanPlayStat, channelID)
		}
		//if download queue is empty, there is no other jobs to run
		//we can just wait to end the play job.
		if vi.downloadQueue.Empty() {
			playStat := <-chanPlayStat
			log.Println("playStat:", playStat)
			if playStat == 0 {
				vi.sendMessageToChannel(channelID, "See you later.")
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

//processDownloadQueue had channel to keep track of the download status
//of songs but I think this not necessary anymore so I removed channel.
//With the latest changes processDownloadQueue and downloadQuery functions
//seems like doing the same jobs. I am planning to combine this two functions
//in to one.
func (vi *VoiceInstance) processDownloadQueue(channelID string) {
	if vi.downloadQueue.Empty() {
		return
	}

	nextItem, err := vi.downloadQueue.Get(1)
	if err != nil {
		log.Println(err)
		return
	}

	songInstance := getSongInstanceFromInterface(nextItem[0])
	if songInstance == nil {
		log.Println("Error while converting interface {} to SongInstance{}.")
		return
	}

	query := songInstance.artist + songInstance.title

	searchResult, err := yt.SearchDownload(query)
	if err != nil {
		log.Println(err)
		log.Printf("Putting %s to the error queue.", query)
		//vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		vi.errQueue.Put(query)
		return
	}

	var coverPath string
	coverPath, err = util.GetCoverImage(songInstance.coverUrl)
	if err != nil {
		log.Println(err)
		coverPath = DefaultCoverPath
	}

	songInstance.songPath = searchResult.VideoPath
	songInstance.coverPath = coverPath

	vi.playQueue.Put(songInstance)
	return
}

func (vi *VoiceInstance) downloadSelection(searchResult *youtube.SearchResult, channelID string) error {
	songPath, err := yt.DownloadVideo(searchResult)
	if err != nil {
		vi.sendMessageToChannel(channelID, "Unexpected thing happend. Try again.")
		return err
	}

	coverPath := DefaultCoverPath

	songInstance := &SongInstance{
		songPath:  songPath,
		coverPath: coverPath,
	}

	vi.playQueue.Put(songInstance)
	return nil
}

//vide supra processDownloadQueue function comment.
func (vi *VoiceInstance) downloadQuery(songInstance *SongInstance, channelID string) error {
	query := songInstance.artist + songInstance.title
	searchResult, err := yt.SearchDownload(query)
	if err != nil {
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		log.Printf("Putting %s to the error queue.", query)
		vi.errQueue.Put(query)
		return err
	}

	//get cover image
	var coverPath string
	coverPath, err = util.GetCoverImage(songInstance.coverUrl)
	if err != nil {
		log.Println(err)
		coverPath = DefaultCoverPath
	}

	songInstance.songPath = searchResult.VideoPath
	songInstance.coverPath = coverPath
	songInstance.duration = searchResult.Duration
	songInstance.videoID = searchResult.VideoID

	vi.playQueue.Put(songInstance)
	return nil
}

func (vi *VoiceInstance) downloadPlayQuery(query, channelID string) error {
	searchResult, err := yt.SearchDownload(query)
	if err != nil {
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		log.Printf("Putting %s to the error queue.", query)
		vi.errQueue.Put(query)
		return err
	}

	songInstance := &SongInstance{
		title:     searchResult.VideoTitle,
		songPath:  searchResult.VideoPath,
		coverPath: DefaultCoverPath,
		videoID:   searchResult.VideoID,
	}

	vi.playQueue.Put(songInstance)
	return nil
}

func (vi *VoiceInstance) processPlayQueue(playStat chan<- int, messageChannelID string) {
	stop := make(chan int)
	nextItem, err := vi.playQueue.Get(1)
	if err != nil {
		log.Printf("Error while getting item from playlist queue: %v", err)
		return
	}

	songInstance := getSongInstanceFromInterface(nextItem[0])
	if songInstance == nil {
		log.Println("Error while converting interface {} to SongInstance{}.")
		return
	}

	songPath := songInstance.songPath
	_, err = vi.sendEmbedNowPlayingMessage(messageChannelID, songInstance)
	if err != nil {
		log.Println(err)
	}
	//vi.sendFileWithMessage(messageChannelID, playingMsg, songInstance.coverPath)
	go vi.playAudioFile(songPath, stop)
	stat := <-stop

	if stat == 0 || stat == 1 {
		errDeleteFile := util.DeleteFile(songPath)
		if err != nil {
			log.Println(errDeleteFile)
		} else {
			log.Printf("%s is deleted.", songPath)
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
			return
		}

		//EOF received and play queue is empty means that
		//song is played and there is left no song to play.
		//we can end the play process by sending 0(int) to channel.
		if (err == io.EOF || err == io.ErrUnexpectedEOF) && vi.playQueue.Empty() {
			log.Println("EOF received and play queue is empty. Ending play process.")
			vi.isPlaying = false
			err = run.Process.Kill()
			stop <- 0
			return
		}

		if (err != io.EOF && err != io.ErrUnexpectedEOF) && err != nil {
			log.Println("Error reading from ffmpeg stdout: ", err)
			stop <- -1
			return
		}

		//handle !skip
		if vi.skip == true {
			//if playqueue is not empty send 1(int) to the channel
			//to play next song on the queue.
			if vi.isPlaying == true && !vi.playQueue.Empty() {
				vi.isPlaying = false
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

		//handle !stop
		if vi.stop == true {
			vi.stop = false
			err = run.Process.Kill()
			vi.downloadQueue = createNewQueue()
			vi.playQueue = clearPlaylistQueue(vi.playQueue)
			stop <- 0
			return
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

func (vi *VoiceInstance) stopSong(m *discordgo.MessageCreate) {
	//if currently no song is playing, no need to stop it.
	if vi.isPlaying == false {
		log.Println("No song is playing. stop returning.")
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
			vi.stop = true
			return
		}
	}
}

//showPlayQueue sends the songs in the play queue to given channel ID.
func (vi *VoiceInstance) showPlayQueue(m *discordgo.MessageCreate) {
	if vi.playQueue.Empty() {
		vi.sendMessageToChannel(m.ChannelID, "Play queue is empty.")
		return
	}

	_, err := vi.sendEmbedPlayQueueMessage(m.ChannelID)
	if err != nil {
		vi.sendErrorMessageToChannel(m.ChannelID)
		return
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

//sendErrorMessageToChannel sends error message to given channel.
func (vi *VoiceInstance) sendErrorMessageToChannel(channelID string) {
	messageText := "Error while handling request. Please Try again."
	vi.sendMessageToChannel(channelID, messageText)
	return
}

//sendNowPlayingToChannel sends the playing song name message to channel.
func (vi *VoiceInstance) sendNowPlayingToChannel(channelID, songTitle string) {
	messageText := "Now Playing " + songTitle
	vi.sendMessageToChannel(channelID, messageText)
	return
}

func (vi *VoiceInstance) sendFileWithMessage(channelID, text, coverPath string) {
	messageText := "Now Playing " + text
	artCoverFile, _ := os.Open(coverPath)
	_, err := vi.session.ChannelFileSendWithMessage(channelID, messageText, coverPath, artCoverFile)
	if err != nil {
		log.Printf("Error while sending message to channel: %v", err)
	}
	artCoverFile.Close()
}

//sentMessageEditEmbed edits last sent message(in our case this message is sendSearchResult)
//to show now playing message.
func (vi *VoiceInstance) sentMessageEditEmbed(channelID, messageID string, searchResult *youtube.SearchResult) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{},
		Color:  0x26e232,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name: "Now Playing: ",
				Value: formatEmbededLinkText(searchResult.VideoTitle,
					searchResult.Duration,
					searchResult.VideoID),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Image: &discordgo.MessageEmbedImage{
			URL: searchResult.CoverPath,
		},
	}
	_, err := vi.session.ChannelMessageEditEmbed(channelID, messageID, embed)
	if err != nil {
		log.Printf("Error while editing embeded message: %v", err)
		return
	}
	return
}

func (vi *VoiceInstance) sendEmbedNowPlayingMessage(channelID string, songInstance *SongInstance) (string, error) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{},
		Color:  0x26e232,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name: "Now Playing",
				Value: formatEmbededLinkText(songInstance.title,
					songInstance.duration,
					songInstance.videoID),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Image: &discordgo.MessageEmbedImage{
			URL: songInstance.coverUrl,
		},
	}

	message, err := vi.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return "", fmt.Errorf("Error while sending now playing embed message: %v", err)
	}

	return message.ID, nil
}

//sendEmbedPlayQueueMessage sends an embeded message that contains
//next songs in the playlist to the given channel ID.
func (vi *VoiceInstance) sendEmbedPlayQueueMessage(channelID string) (string, error) {
	fields, err := createMessageEmbedFieldsPlayQueue()
	if err != nil {
		vi.sendErrorMessageToChannel(channelID)
		return "", fmt.Errorf("Error while creating message embed play queue: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Author:    &discordgo.MessageEmbedAuthor{},
		Color:     0xff5733,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	message, err := vi.sendEmbeddedMessageToChannel(channelID, embed)
	if err != nil {
		return "", fmt.Errorf("Error while sending embedded play queue message to channel: %v", err)
	}
	return message.ID, nil
}

//sendEmbeddedMessageToChannel sends embedded message to given channel.
//returns sent message struct.
func (vi *VoiceInstance) sendEmbeddedMessageToChannel(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	message, err := vi.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return nil, fmt.Errorf("Error while sending embedded message to channel: %v", err)
	}
	return message, nil
}

//sendSearchResultMessageToChannel sends results of !saerch commands to given channel.
//returns sent message ID.
func (vi *VoiceInstance) sendSearchResultMessageToChannel(channelID string, searchResults *[]youtube.SearchResult) string {
	embed := &discordgo.MessageEmbed{
		Author:    &discordgo.MessageEmbedAuthor{},
		Color:     0xfd0057,
		Fields:    createMessageEmbedFields(searchResults),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	message, err := vi.sendEmbeddedMessageToChannel(channelID, embed)
	if err != nil {
		log.Println(err)
		return ""
	}
	return message.ID
}

//createMessageEmbedFields is a helper function to create MessageEmbedField array.
func createMessageEmbedFields(searchResults *[]youtube.SearchResult) []*discordgo.MessageEmbedField {
	messageEmbedFields := []*discordgo.MessageEmbedField{}
	resultCounter := 1
	for _, element := range *searchResults {
		embedField := &discordgo.MessageEmbedField{
			Name:   strconv.Itoa(resultCounter) + ")",
			Value:  formatEmbededLinkText(element.VideoTitle, element.Duration, element.VideoID),
			Inline: false,
		}
		messageEmbedFields = append(messageEmbedFields, embedField)
		resultCounter++
	}

	return messageEmbedFields
}

//createMessageEmbedFieldsPlayQueue is a helper function to sendEmbedPlayQueueMessage func
//to create MessageEmbedField array.
func createMessageEmbedFieldsPlayQueue() ([]*discordgo.MessageEmbedField, error) {
	messageEmbedFields := []*discordgo.MessageEmbedField{}

	items, err := vi.playQueue.PeekAll()
	if err != nil {
		return nil, err
	}
	for _, element := range items {
		instance := getSongInstanceFromInterface(element)
		if instance == nil {
			continue
		}
		embedField := &discordgo.MessageEmbedField{
			Name:   "Play Queue:",
			Value:  formatEmbededLinkText(instance.title, instance.duration, instance.videoID),
			Inline: false,
		}
		messageEmbedFields = append(messageEmbedFields, embedField)
	}
	return messageEmbedFields, nil
}

//formatEmbededLinkText is a helper function to create embeded link text.
func formatEmbededLinkText(title, duration, id string) string {
	return "[" + title + "(" + duration + ")" + "](" + youtubeUrlPrefix + id + ")"
}

//createNewQueue creates new queue and
//returns newly created queue.
func createNewQueue() *queue.Queue {
	newQueue := queue.New(20)
	return newQueue
}

//clearPlaylistQueue deletes all the song files that are present in
//the play queue.
func clearPlaylistQueue(playlistQueue *queue.Queue) *queue.Queue {
	for !playlistQueue.Empty() {
		nextItem, err := playlistQueue.Get(1)
		if err != nil {
			log.Printf("Error while getting item from playlist queue: %v", err)
			return createNewQueue()
		}

		songInstance := getSongInstanceFromInterface(nextItem[0])
		if songInstance == nil {
			log.Println("Error while converting interface {} to SongInstance{}.")
			return createNewQueue()
		}

		util.DeleteFile(songInstance.songPath)
		util.DeleteFile(songInstance.coverPath)
	}
	if playlistQueue.Empty() {
		log.Println("All files has been deleted and Play Queue cleared.")
		return playlistQueue
	} else {
		log.Printf("Creating new playlist queue.")
		return createNewQueue()
	}
}

//getSongInstanceFromInterface takes interface{} argument
//and returns SongInstance
func getSongInstanceFromInterface(object interface{}) *SongInstance {
	inst, ok := object.(*SongInstance)

	if ok {
		newInstance := SongInstance{
			title:     inst.title,
			artist:    inst.artist,
			songPath:  inst.songPath,
			coverUrl:  inst.coverUrl,
			coverPath: inst.coverPath,
			videoID:   inst.videoID,
			duration:  inst.duration,
		}
		return &newInstance
	}
	return nil
}
