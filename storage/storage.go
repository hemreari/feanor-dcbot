package storage

import (
	"database/sql"

	"../config"
	_ "github.com/go-sql-driver/mysql"
)

type StorageClient struct {
	Client *sql.DB
}

func NewStorageClient(cfg *config.Config) (*StorageClient, error) {
	dbStr := cfg.MySQL.UserName + ":" + cfg.MySQL.Password + "@" + "tcp(" + cfg.MySQL.Host + ")" + "/" + cfg.MySQL.DBName
	driver := cfg.MySQL.Driver

	client, err := sql.Open(driver, dbStr)
	if err != nil {
		return nil, err
	}
	client.SetMaxOpenConns(99)
	return &StorageClient{
		Client: client,
	}, nil
}

func (s *StorageClient) InsertTrackData(playlistID, artistName, trackName, youtubeID string) error {
	_, err := s.Client.Exec("INSERT INTO music(spotify_playlist_id, spotify_artist_name, spotify_track_name, youtube_url) VALUES (?, ?, ?, ?)", playlistID, artistName, trackName, youtubeID)

	if err != nil {
		return err
	}
	return nil
}

func (s *StorageClient) InsertLocation(path, youtubeID string) error {
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
func (s *StorageClient) RowExists(columnName, fieldData string) (bool, error) {
	query := "SELECT exists(SELECT ID FROM music WHERE " + columnName + "=\"" + fieldData + "\")"
	var exists bool

	err := s.Client.QueryRow(query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}
