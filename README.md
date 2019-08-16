# go-livebarn

`go-livebarn` is an attempt at creating a client library for interacting with
[LiveBarn](https://livebarn.com) through inspecting its browser-based
interactions with the server.

This library is a work-in-progress.

## Example

The example provided in `examples/fetch.go` demonstrates how to fetch URLs to
download video segments from an ice arena at a certain time.

```go
func main() {
	client := livebarn.New("my-token-here", "my-uuid-here")
	client.DebugMode = true

	oaklandIceNHL := &livebarn.Surface{UUID: "dff0fc40649943109e4ddab3118f3da2"}

	resp, _ := client.GetMedia(
		oaklandIceNHL,
		&livebarn.DateRange{
			Start: time.Date(2018, 03, 17, 12, 00, 00, 00, time.Local),
			End:   time.Date(2018, 03, 17, 14, 00, 00, 00, time.Local),
		},
	)

	for i, part := range resp.Result {
		filename := fmt.Sprintf("Part %d.mp4", i+1)
		log.Printf("Downloading %s...\n", filename)
		url, _ := client.GetMediaDownload(part.URL)

		err := livebarn.DownloadFile(filename, url.Result.URL)
		if err != nil {
			panic(err)
		}
	}
}
```

## `ffmpeg` Bonus

This little snippet will concatenate multiple video files into one:

```
$ ffmpeg -i "concat:input1.mp4|input2.mp4|input3.mp4" -c copy output.mp4
```

## surrette fork additions August 2019
- Updated media.go to use v2 of the LiveBarn API. v1 was failing for get videos.
- Added ffmpeg merge directly into fetch.go using exec.Command
- Added YouTube upload into fetch.go using https://github.com/youtube/api-samples/tree/master/go. NOTE: you must add a client_secret.json file to the directory for oauth2.go to work. Credentials can be created at https://developers.google.com/console.
- highlight.go creates a highlight video using ffmpeg to slice the merged video and then upload to YouTube

// (token.access_token, lb_uuid_key) Sign into Livebarn and get variables. In Chrome hit F12 (Developer) -> Application tab ->

You'll notice that game variables are pulled from a MySQL database. You may want to rewrite the code to input these variables a different way, but here are MySQL tables if you want to recreate.

CREATE TABLE `games` (
  `GameId` varchar(45) NOT NULL,
  `HomeTeam` varchar(200) DEFAULT NULL,
  `AwayTeam` varchar(200) DEFAULT NULL,
  `Name` varchar(1000) DEFAULT NULL,
  `StartTime` datetime DEFAULT NULL,
  `Surface` varchar(50) DEFAULT NULL,
  `Auth` varchar(45) DEFAULT NULL,
  `FileName` varchar(200) DEFAULT NULL,
  `YouTubeLink` varchar(200) DEFAULT NULL,
  PRIMARY KEY (`GameId`),
  UNIQUE KEY `GameId_UNIQUE` (`GameId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `highlights` (
  `HighlightId` int(11) NOT NULL AUTO_INCREMENT,
  `GameId` int(11) DEFAULT NULL,
  `Team` varchar(200) DEFAULT NULL,
  `Name` varchar(1000) DEFAULT NULL,
  `Type` varchar(45) DEFAULT NULL,
  `SubType` varchar(45) DEFAULT NULL,
  `StartTime` int(11) DEFAULT NULL,
  `EndTime` int(11) DEFAULT NULL,
  `FileName` varchar(200) DEFAULT NULL,
  `YouTubeLink` varchar(200) DEFAULT NULL,
  `Period` int(11) DEFAULT NULL,
  `GameTime` varchar(45) DEFAULT NULL,
  `GoalNum` int(11) DEFAULT NULL,
  `Verified` tinyint(4) DEFAULT '0',
  PRIMARY KEY (`HighlightId`),
  UNIQUE KEY `unique_highlight` (`GameId`,`Team`,`Type`,`SubType`,`Period`,`GameTime`,`GoalNum`)
) ENGINE=InnoDB AUTO_INCREMENT=623 DEFAULT CHARSET=utf8;
