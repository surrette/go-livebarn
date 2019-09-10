package main

import (
	"fmt"
	"log"
	"github.com/bmorton/go-livebarn"
	"strings"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"os"
	"os/exec"
	"bytes"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Game struct {
    GameId   int    `json:"GameId"`
    Name string `json:"Name"`
	StartTime time.Time `json:"StartTime"`
	Surface string `json:"Surface"`
	Auth string `json:"Auth"`
	FileName string  `json:"FileName"`
}

type Configuration struct {
    Server    string
    Database   string
	User    string
    Password   string
	LiveBarnUUID    string
}

func main() {
	file, _ := os.Open("config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err := decoder.Decode(&config)
	if err != nil {
	  fmt.Println("error:", err)
	}
	
	//***Database Connection***
	var connectionString = config.User + ":" + config.Password + "@tcp(" + config.Server + ":3306)/" + config.Database + "?parseTime=true"
	log.Println(connectionString)
	db, err := sql.Open("mysql", connectionString)
    // if there is an error opening the connection, handle it
    if err != nil {
        log.Print(err.Error())
    }
    defer db.Close()
	
	//***SURFACE MAPPING San Jose Solar For America***
	//{"id":546,"name":"North Rink","uuid":"206faaace81944b793f4f0d2dfc3f3b4","orderIndex":1,"surfaceStatusId":8,"venueId":369,"sportId":1,"closedFrom":null,"closedUntil":null,"comingSoon":false,"online":false}
	//{"id":547,"name":"South Rink","uuid":"5c8a3cdcd5a943e7b30212ff388a7043","orderIndex":1,"surfaceStatusId":3,"venueId":369,"sportId":1,"closedFrom":null,"closedUntil":null,"comingSoon":false,"online":true}]
	//{"id":548,"name":"East Rink","uuid":"8c3b213305454370ae427d166f4f1d53","orderIndex":3,"surfaceStatusId":3,"venueId":369,"sportId":1,"closedFrom":null,"closedUntil":null,"comingSoon":false,"online":true}
	//{"id":549,"name":"Center Rink","uuid":"50612242ea7f429591dbb81280f8e19c","orderIndex":2,"surfaceStatusId":8,"venueId":369,"sportId":1,"closedFrom":null,"closedUntil":null,"comingSoon":false,"online":false}
	var m  = make(map[string]string)
	m["San Jose North"] = "546"
	m["San Jose South"] = "547"
	m["San Jose East"] = "548"
	m["San Jose Center"] = "549"

    // SELECT games that have not been processed yet (NULL YouTubeLink)
    results, err := db.Query("SELECT GameId, Name, StartTime, Surface, Auth, IFNULL(FileName, '') as FileName FROM games WHERE YouTubeLink = '' OR YouTubeLink is NULL;")
    if err != nil {
        panic(err.Error()) // proper error handling instead of panic in your app
    }

	var mergedFileName string
	var out bytes.Buffer
	var stderr bytes.Buffer
    for results.Next() {
        var game Game
        // for each row, scan the result into our tag composite object
        err = results.Scan(&game.GameId, &game.Name, &game.StartTime, &game.Surface, &game.Auth, &game.FileName)
        if err != nil {
            panic(err.Error()) // proper error handling instead of panic in your app
        }
        log.Printf(game.Name)

		var surfaceid = m[game.Surface] //| 546=North | 547=South | 548=East | 549=Center
		var name = game.Name
		var beginDateTimeString = game.StartTime.Format("2006-01-02T15:04")
		var beginDateString = game.StartTime.Format("2006-01-02")
		var surface = strings.Replace(game.Surface, " ", "_", -1)

		if(game.FileName == ""){ //don't download video again if file already exists. Useful for YouTube API failures (daily quota hit for example)
			var uuid = config.LiveBarnUUID //same per account
			var token = game.Auth //changes each login		
			client := livebarn.New(token, uuid) // (token.access_token, lb_uuid_key) Sign into Livebarn and get variables. In Chrome hit F12 (Developer) -> Application tab -> token.access_token, lb_uuid_key
			client.DebugMode = true
			
			var videoDuration=0 
			var targetVideoDuration=80*60*1000 //80 minutes in milliseconds
			//***SEARCH LIVEBARN***
			resp, _ := client.GetMedia(surfaceid, beginDateTimeString)	
			
			//***LOOP OVER FILES***
			var mergeFiles = ""
			var i=0
			for videoDuration < targetVideoDuration {
				part := resp[i]
				var dateTimeString = part.BeginDate[0:16]
				videoDuration += part.Duration
				dateTimeString =  strings.Replace(dateTimeString, ":", "_", -1)
				//Videos file for .mp4
				filename := fmt.Sprintf("%s-%s-Part%d.mp4", dateTimeString, surface, i+1)
				mergeFiles+="file " + filename + "\n"
				log.Printf("\nDownloading %s ...\n", filename)
				
				//***Download video file from LiveBarn**
				url, _ := client.GetMediaDownload(part.URL)
				err := livebarn.DownloadFile(filename, url.Result.URL)
				if err != nil {
					panic(err)
				}
				i++
			}
			
			//***WRITE ffmpeg-merge.txt - LIST OF MERGE FILE***
			log.Printf("Starting merge file write...\n")
			err := ioutil.WriteFile("./Videos/ffmpeg-merge.txt", []byte(mergeFiles), 0644)
			if err != nil {
				log.Println(err)
			}
			
			//***EXECUTE ffmpeg MERGE***
			mergedFileName = fmt.Sprintf("%s-%s-Merged.mp4", strings.Replace(beginDateTimeString, ":", "_", -1), surface)
			log.Printf("Starting ffmpeg merge...\n")
			cmd := exec.Command("ffmpeg",
				"-f", "concat",
				"-i", "ffmpeg-merge.txt",
				"-c", "copy",
				mergedFileName,
				"-y",
				)
			cmd.Dir = "./Videos/"
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			}
			fmt.Println("Result: " + out.String())
		} else{ //if merged file already created
			mergedFileName = game.FileName
		}
		
		//***YouTube Upload***
		log.Printf("Starting YouTube upload...\n")
		var title = name + " (" + beginDateString + ")"
		cmd := exec.Command("go", "run", "upload_video.go", "errors.go", "oauth2.go",
			"--filename", "./Videos/" + mergedFileName,
			"--title", title,
			"--keywords", "Beerbears Hockey")
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		}
		var YouTubeId = out.String()
		
		//***UPDATE DATABASE with server merged filename and YouTubeLink
		var sqlUpdate = "UPDATE games SET YouTubeLink='" + YouTubeId + "', FileName='" + mergedFileName + "' WHERE GameId = " + strconv.Itoa(game.GameId)
		fmt.Println("sqlUpdate: " + sqlUpdate)
		results, err := db.Query(sqlUpdate)
		if err != nil {
			fmt.Println(results)
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		
	}
	db.Close()
}
