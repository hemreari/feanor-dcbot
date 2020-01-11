package audio

import (
//"fmt"
//"bytes"
//"log"
//"bytes"
//"os"
//"os/exec"
//"strings"
//"../config"
//"github.com/jonas747/dca"
//"github.com/rylio/ytdl"
)

/*
/* Downloads given youtube url to main directory,
 * converts the downloaded mp4 file to mp3 and returns
 * the mp3 file path or err.
 * using github.com/rylio/ytdl.
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
		_ = os.Remove(videoPath)
		return mp3Path, nil
	}

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		file, err := os.Create(videoPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		video.Download(video.Formats[0], file)
	}
	log.Printf("Video Downloaded: %s\n", videoPath)

	mp3Path, err = ConvertMP4ToMp3(videoPath, mp3Path)
	if err != nil {
		return "", err
	}
	return mp3Path, nil
}
*/

/*
//not working
func ConvertVideoToDca(videoTitle string) error {
	log.Printf("Starting to convert %s to dca.\n", videoTitle)

	cmd := "ffmpeg -i " + "Rammstein_-_Deutschland_(Official_Video).m4a" + "-f s16le -ar 48000 -ac 2 pipe:1 | dca > " + "out.dca"
	exec.Command("bash", "-c", cmd)
	if err != nil {
		return err
	}
	_ = cmd.Run()

	/*
		dcaFile, err := os.Create("output.dca")
		if err != nil {
			return fmt.Errorf("Error while converting file: %v", err)
		}

		cmdFfmpegArgs := []string{
			"-i",
			"Rammstein_-_Deutschland_(Official_Video).m4a",
			"-f",
			"s16le",
			"-ar",
			"48000",
			"-ac",
			"2",
			"test.pcm",
		}

		cmdFfmpeg := exec.Command("ffmpeg", cmdFfmpegArgs...)
		cmdDca := exec.Command("dca")

		//cmdFfmpeg.Stdout = os.Stdout
		cmdFfmpeg.Stderr = os.Stderr

		cmdDca.Stdin, _ = cmdFfmpeg.StdoutPipe()
		cmdDca.Stdout = dcaFile
		cmdDca.Start()
		cmdFfmpeg.Run()
		cmdDca.Wait()
		dcaFile.Close()
*/

/*
		stdin, err := cmdFfmpeg.StdinPipe()
		if err != nil {
			return fmt.Errorf("Error1 while converting %s to dca: %v", videoTitle, err)
		}

		file, err := os.Open("test.pcm")
		if err != nil {
			return fmt.Errorf("Error2 while converting %s to dca: %v", videoTitle, err)
		}

		io.Copy(file, stdin)
		stdin.Close()

		err = cmdDca.Run()
		if err != nil {
			return fmt.Errorf("Error3 while converting %s to dca: %v", videoTitle, err)
		}

	return nil
}

/*
/* Converts given mp3 file to DCA file using
 * github.com/bwmarrin/dca.
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

			}
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
} */
