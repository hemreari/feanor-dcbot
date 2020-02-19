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
	"../spotify"
	"../util"
	"../youtube"

	"github.com/bwmarrin/discordgo"
	"github.com/hemreari/go-datastructures/queue"
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

type SongInstance struct {
	title     string
	artist    string
	songPath  string
	coverUrl  string
	coverPath string
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

	//stop commands stops playing song
	if strings.Compare(m.Content, "!stop") == 0 {
		vi.stopSong(m)
	}

	if strings.Compare(m.Content, "!show") == 0 {
		vi.showPlayQueue(m)
		log.Println("Play Queue: ", vi.playQueue)
	}

	if strings.HasPrefix(m.Content, "!list") {
		link := strings.Trim(m.Content, "!list ")
		if strings.Contains(link, "spotify") {
			playlistID := util.GetSpotifyPlaylistID(link)
			vi.prepSpotifyPlaylist(playlistID, s, m)
		}
	}
}

func (vi *VoiceInstance) validateMessage(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		log.Printf("Couldn't find channel: %v\n", err)
		// Could not find channel.
		return false
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		log.Printf("Couldn't find guild: %v\n", err)
		// Could not find guild.
		return false
	}

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
}

//prepSpotifyPlaylist gets songs from the given Spotify playlist ID
//and parses them as youtube queries to add to the download queue.
//finally starts the play process.
func (vi *VoiceInstance) prepSpotifyPlaylist(playlistID string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if !vi.validateMessage(s, m) {
		log.Println("message is not valid to join the voice channel.")
		return
	}

	//initialise spotify api.
	spotifyAPI := initSpotifyAPI()

	//get playlist info from spotify api.
	sptfyPlaylistInfo, err := spotifyAPI.GetPlaylist(playlistID)
	if err != nil {
		log.Printf("Error while getting Spotify playlist information: %v", err)
		vi.sendMessageToChannel(m.ChannelID, "Unexpected thing is happened. Please, Try again.")
		return
	}

	log.Println("Playlist Name: ", sptfyPlaylistInfo.Name)
	log.Println("Playlist Owner: ", sptfyPlaylistInfo.Owner.ID)

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

	//parse playlist tracks to artist and track name.
	items := plTracks.Items
	for index := range items {
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
			continue
		}
		vi.downloadQueue.Put(&songInstance)
	}

	//start the play process.
	vi.playQueueFunc(m.ChannelID)
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
			go vi.processDownloadQueue(channelID)
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
		log.Println("Download queue is empty. Closing the channel")
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

	songPath, err := yt.SearchDownload(query)
	if err != nil {
		log.Println(err)
		log.Printf("Putting %s to the error queue.", query)
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		vi.errQueue.Put(query)
		return
	}

	var coverPath string
	coverPath, err = util.GetCoverImage(songInstance.coverUrl)
	if err != nil {
		log.Println(err)
		coverPath = "default.jpg"
	}

	songInstance.songPath = songPath
	songInstance.coverPath = coverPath

	vi.playQueue.Put(songInstance)

	return
}

//vide supra processDownloadQueue function comment.
func (vi *VoiceInstance) downloadQuery(songInstance *SongInstance, channelID string) error {
	query := songInstance.artist + songInstance.title
	songPath, err := yt.SearchDownload(query)
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
		coverPath = "default.jpg"
	}

	songInstance.songPath = songPath
	songInstance.coverPath = coverPath

	vi.playQueue.Put(songInstance)
	return nil
}

func (vi *VoiceInstance) downloadPlayQuery(query, channelID string) error {
	songPath, err := yt.SearchDownload(query)
	if err != nil {
		vi.sendMessageToChannel(channelID, "Query is insufficient to find a result. Try again.")
		log.Printf("Putting %s to the error queue.", query)
		vi.errQueue.Put(query)
		return err
	}

	songInstance := &SongInstance{
		title:     songPath,
		songPath:  songPath,
		coverPath: "default.jpg",
	}

	vi.playQueue.Put(songInstance)
	return nil
}

func (vi *VoiceInstance) processPlayQueue(playStat chan<- int, messageChannelID string) {
	stop := make(chan int)
	nextItem, err := vi.playQueue.Get(1)
	if err != nil {
		log.Printf("Error while getting item from playlist: %v", err)
		return
	}

	songInstance := getSongInstanceFromInterface(nextItem[0])
	if songInstance == nil {
		log.Println("Error while converting interface {} to SongInstance{}.")
		return
	}

	songPath := songInstance.songPath
	playingMsg := songInstance.title + " " + songInstance.artist
	vi.sendFileWithMessage(messageChannelID, playingMsg, songInstance.coverPath)
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
//TODO: when downloading songs in the download queue vi.playQueue is empty
//this cause in showPlayQueue func to empty queue error.
func (vi *VoiceInstance) showPlayQueue(m *discordgo.MessageCreate) {
	items, err := vi.playQueue.PeekAll()
	if err != nil {
		log.Println(err)
		vi.sendErrorMessageToChannel(m.ChannelID)
		return
	}

	displayQueueText := "Next Songs: \n"

	for index := range items {
		instance := getSongInstanceFromInterface(items[index])
		if instance == nil {
			log.Println("Error while converting interface {} to SongInstance{}.")
			vi.sendErrorMessageToChannel(m.ChannelID)
			return
		}

		songTitle := instance.title + " " + instance.artist
		displayQueueText += songTitle + "\n"
	}

	vi.sendMessageToChannel(m.ChannelID, displayQueueText)
	return
}

//sendMessageToChannel sends the text to channel that given id.
func (vi *VoiceInstance) sendMessageToChannel(channelID, text string) {
	_, err := vi.session.ChannelMessageSend(channelID, text)
	if err != nil {
		log.Printf("Error while sending message to channel: %v", err)
	}
	return
}

//sendErrorMessageToChannel sends error message to given channel ID.
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
		}
		return &newInstance
	}
	return nil
}
