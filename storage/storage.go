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

func (s *StorageClient) InsertTrackData(artistName, trackName, youtubeID string) error {
	_, err := s.Client.Exec("INSERT INTO music(spotify_artist_name, spotify_track_name, youtube_url) VALUES (?, ?, ?)", artistName, trackName, youtubeID)

	if err != nil {
		return err
	}
	return nil
}

func (s *StorageClient) InsertLocation(path string) error {
	_, err := s.Client.Exec("INSERT INTO music(location) VALUES (?)", path)
	if err != nil {
		return nil
	}
	return nil
}

func (s *StorageClient) RowExists(query string) (bool, error) {
	var exists bool

	err := s.Client.QueryRow(query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}
