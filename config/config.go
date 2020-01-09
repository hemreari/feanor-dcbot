package config

type Config struct {
	Spotify    SpotifyConfig    `json:"spotify"`
	Youtube    YoutubeConfig    `json:"youtube"`
	Discord    DiscordConfig    `json:"discord"`
	PlaylistID PlaylistIDConfig `json:"playlistIDs"`
	MusicDir   MusicDirectory   `json:"musicDirectory"`
}

type SpotifyConfig struct {
	ClientID       string `json:"clientID"`
	ClientSecretID string `json:"clientSecretID"`
}

type YoutubeConfig struct {
	ClientID       string `json:"clientID"`
	ClientSecretID string `json:"clientSecretID"`
	ApiKey         string `json:"apiKey"`
}

type DiscordConfig struct {
	Token string `json:"token"`
}

type PlaylistIDConfig struct {
	Shame string `json:"shame"`
}

type MusicDirectory struct {
	DownloadPath string `json"downlaodPath"`
}
