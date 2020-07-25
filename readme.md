dc-bot

# Commands
| Command Name | Parameter | Description |
| :----------: | :-------: | :---------: |
|    !play     | Search String or Youtube URL | If search string is given as parameter searchs the string and starts to play first found song, if Youtube URL is given plays the song in the given URL.|
| !list | Youtube Playlist URL or Spotify Playlist URL | Use to play playlist links from Youtube and Spotify. Max playable track count is 20. (Could be combined with !play command and deprecated soon.) |
| !search | Search String | Like !play command but instead of playing first found track it shows 5 different search result according to the search string that you can choose with a integer text input. |
| !skip | - | Plays the next song from play queue. |
| !stop | - | Stops playing songs and clears play queue. |
| !show | - | Prints the play queue. |

# Link Formats
These are the accepted link formats for !play and !list commands.

## Spotify
### Playlist
* spotify:playlist:76tzi26o8O920CYAvVbeYO
* https://open.spotify.com/playlist/76tzi26o8O920CYAvVbeYO?si=WKrHWhGVQTSmF7GbeqI5sw

## Youtube
### Playlist
* https://www.youtube.com/playlist?list=PLwiyx1dc3P2JR9N8gQaQN_BCvlSlap7re

# Limits
* Max number of song that can be played from a single playlist is 20 tracks for Spotify and Youtube.

# Requirements

* ffmpeg
* github.com/bwmarrin/dca
* github.com/rylio/ytdl
