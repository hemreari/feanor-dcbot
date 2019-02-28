package storage

import (
	"database/sql"

	"../config"
	_ "github.com/go-sql-driver/mysql"
)

type MusicDBClient struct {
	Client *sql.DB
}

type UserDBClient struct {
	Client *sql.DB
}

func NewMusicDBClient(cfg *config.Config) (*MusicDBClient, error) {
	dbStr := cfg.MySQLMusic.UserName + ":" + cfg.MySQLMusic.Password + "@" + "tcp(" + cfg.MySQLMusic.Host + ")" + "/" + cfg.MySQLMusic.DBName
	driver := cfg.MySQLMusic.Driver

	client, err := sql.Open(driver, dbStr)
	if err != nil {
		return nil, err
	}
	client.SetMaxOpenConns(99)
	return &MusicDBClient{
		Client: client,
	}, nil
}

func NewUserDBClient(cfg *config.Config) (*UserDBClient, error) {
	dbStr := cfg.MySQLUser.UserName + ":" + cfg.MySQLUser.Password + "@" + "tcp(" + cfg.MySQLUser.Host + ")" + "/" + cfg.MySQLUser.DBName
	driver := cfg.MySQLUser.Driver

	client, err := sql.Open(driver, dbStr)
	if err != nil {
		return nil, err
	}
	client.SetMaxOpenConns(99)
	return &UserDBClient{
		Client: client,
	}, nil
}

func (s *MusicDBClient) InsertTrackData(playlistName, playlistOwner, playlistID, artistName, trackName, youtubeID string) error {
	_, err := s.Client.Exec("INSERT INTO music(spotify_playlist_id, spotify_playlist_owner_id, spotify_playlist_owner_name, spotify_artist_name, spotify_track_name, youtube_url) VALUES (?, ?, ?, ?, ?, ?)", playlistID, playlistOwner, playlistName, artistName, trackName, youtubeID)

	if err != nil {
		return err
	}
	return nil
}

func (s *MusicDBClient) InsertLocation(path, youtubeID string) error {
	var id string
	idQuery := "SELECT ID from music where youtube_url=?"
	err := s.Client.QueryRow(idQuery, youtubeID).Scan(&id)
	if err != nil {
		return err
	}

	query := "update music set location=? where youtube_url=?"
	_, err = s.Client.Exec(query, path, youtubeID)
	if err != nil {
		return err
	}
	return nil
}

/* checks the given column field data exists */
func (s *MusicDBClient) RowExists(columnName, fieldData string) (bool, error) {
	query := "SELECT exists(SELECT ID FROM music WHERE " + columnName + "=\"" + fieldData + "\")"
	var exists bool

	err := s.Client.QueryRow(query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}
