package config

type Config struct {
	Spotify    SpotifyConfig    `json:"spotify"`
	Youtube    YoutubeConfig    `json:"youtube"`
	Discord    DiscordConfig    `json:"discord"`
	MySQL      MySQLConfig      `json:"mysql"`
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

type MySQLConfig struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Host     string `json:"host"`
	DBName   string `json:"dbName"`
	Driver   string `json:"driver"`
}

type PlaylistIDConfig struct {
	MusicOfAinur  string `json:"musicOfAinur"`
	Erebor        string `json:"erebor"`
	Mordor        string `json:"mordor"`
	MakamIstirasi string `json:"makamIstirasi"`
}

type MusicDirectory struct {
	DownloadPath string `json"downlaodPath"`
}
