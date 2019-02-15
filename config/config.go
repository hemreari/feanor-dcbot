package Config

type Config struct {
	Spotify SpotifyConfig `json:"spotify"`
	Discord DiscordConfig `json:"discord"`
	MySQL   MySQLConfig   `json:"mysql"`
}

type SpotifyConfig struct {
	ClientID       string `json"clientID"`
	ClientSecretID string `json:"clientSecretID"`
}

type Discord struct {
	Token string `json:"token"`
}

type MySQL struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Host     string `json:"host"`
	DBName   string `json:"dbName"`
}
