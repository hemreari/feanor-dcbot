package audio

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"../config"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

/* Downloads given youtube url to main directory
 * using github.com/rylio/ytdl.
 * TODO: add download destination path to config.json
 */
func DownloadYTVideo(url string, cfg *config.Config) (string, error) {
	if _, err := os.Stat(cfg.MusicDir.DownloadPath); os.IsNotExist(err) {
		err := os.Mkdir(cfg.MusicDir.DownloadPath, 0777)
		if err != nil {
			return "", err
		}
	}

	video, err := ytdl.GetVideoInfo(url)
	if err != nil {
		return "", err
	}

	videoPath := cfg.MusicDir.DownloadPath + video.Title + ".mp4"
	mp3Path := strings.TrimSuffix(videoPath, ".mp4") + ".mp3"

	if _, err := os.Stat(mp3Path); err == nil {
		log.Printf("%s is already converted to mp3. Skipping\n", videoPath)
		return videoPath, nil
	}

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		file, err := os.Create(videoPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		video.Download(video.Formats[0], file)
		log.Println("Video Downloaded: " + videoPath)
		return videoPath, nil
	}
	log.Println("Video Downloaded: " + videoPath)
	return videoPath, nil
}

/* Converts given mp4 file to mp3 file with ffmpeg. */
func ConvertMP4ToMp3(sourcePath, destPath string) (string, error) {
	// destination path exists so mp4 file already converted.
	// skip that file.
	if _, err := os.Stat(destPath); err == nil {
		log.Printf("Didn't converted, %s already exists.\n", sourcePath)
		if _, err := os.Stat(sourcePath); err == nil {
			//delete mp4 file
			delFileCmd := exec.Command("rm", sourcePath)
			err := delFileCmd.Run()
			if err != nil {
				return "", err
			}
			log.Printf("Deleted %s.\n", sourcePath)
			return "", nil
		}
		return destPath, nil
	}
	cmd := exec.Command("ffmpeg", "-i", sourcePath, destPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return destPath, nil
}

/* Converts given mp3 file to DCA file using
 * github.com/bwmarrin/dca. */
func ConvertMP3ToDCA(sourcePath, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		log.Printf("Didn't converted, %s aldready exists.\n", destPath)
		/*
			if _, err := os.Stat(sourcePath); err == nil {
				delFileCmd := exec.Command("rm", sourcePath)
				err := delFileCmd.Run()
				if err != nil {
					return err
				}
				return nil

			}*/
		return nil
	}

	encodeSession, err := dca.EncodeFile(sourcePath, dca.StdEncodeOptions)
	if err != nil {
		return err
	}
	defer encodeSession.Cleanup()
	output, err := os.Create(destPath)
	if err != nil {
		return err
	}
	io.Copy(output, encodeSession)
	return nil

	/*
		cmd := "ffmpeg -i " + sourcePath + "-f s16le -ar 48000 -ac 2 pipe:1 | dca > " + destPath
		_, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return err
		}
		return nil
	*/
}
