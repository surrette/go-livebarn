package main

import (
	"fmt"
	"log"
	"strings"
	"strconv"
	"encoding/json"
	"os"
	"os/exec"
	"bytes"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Highlight struct {
    HighlightId   int    `json:"HighlightId"`
	GameId   int    `json:"GameId"`
    Name string `json:"Name"`
	StartTime int `json:"StartTime"`
	EndTime int `json:"EndTime"`
	Type string `json:"Type"`
	Tags string `json:"Tags"`
	GameName string `json:"GameName"`
	GameFileName string `json:"GameFileName"`
	YouTubeLink string `json:"YouTubeLink"`
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
	//***Database Connection***
	db, err := sql.Open("mysql", connectionString)
    // if there is an error opening the connection, handle it
    if err != nil {
        log.Print(err.Error())
    }
    defer db.Close()

    // SELECT highlights that have not been processed yet (NULL YouTubeLink)
    results, err := db.Query("SELECT HighlightId, h.GameId, h.Name, h.StartTime, h.EndTime, h.Type, g.Name as GameName, g.FileName as GameFileName, h.YouTubeLink FROM highlights h join games g on g.GameId=h.GameId WHERE h.YouTubeLink = 'PENDING'-- OR h.YouTubeLink='';")
    if err != nil {
        panic(err.Error()) // proper error handling instead of panic in your app
    }

    for results.Next() {
        var highlight Highlight
        // for each row, scan the result into our tag composite object
        err = results.Scan(&highlight.HighlightId, &highlight.GameId, &highlight.Name, &highlight.StartTime, &highlight.EndTime, &highlight.Type, &highlight.GameName, &highlight.GameFileName, &highlight.YouTubeLink)
        if err != nil {
            panic(err.Error()) // proper error handling instead of panic in your app
        }
        log.Printf(highlight.Name)
		
		var cutFileName = strings.Replace(highlight.GameFileName, "Merged", strings.Replace(highlight.Name, ":", "_", -1), -1)
		var out bytes.Buffer
		var stderr bytes.Buffer
		if(highlight.YouTubeLink == "PENDING"){
			//***EXECUTE ffmpeg cut***
			log.Printf("Starting ffmpeg cut...\n")
			cmd := exec.Command("ffmpeg",
				"-ss", strconv.Itoa(highlight.StartTime),
				"-t", strconv.Itoa(highlight.EndTime - highlight.StartTime),
				"-i", highlight.GameFileName,
				"-async", "1",
				cutFileName,
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
		}
		
		//***YouTube Upload***
		log.Printf("Starting YouTube upload...\n")
		var title = highlight.Name + " - " + highlight.GameName
		var cmd = exec.Command("go", "run", "upload_video.go", "errors.go", "oauth2.go",
			"--filename", "./Videos/" + cutFileName,
			"--title", title,
			"--keywords", "Beerbears Hockey")
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		//cmd.Dir = "./Videos/"
		err = cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		}
		//fmt.Printf("\CMD:\n%v\n", out)
		var YouTubeId = out.String()
		fmt.Println("Result YouTube API: " + YouTubeId)
		//*/
		
		var sqlUpdate = "UPDATE highlights SET YouTubeLink='" + YouTubeId + "', FileName='" + cutFileName + "' WHERE HighlightId = " + strconv.Itoa(highlight.HighlightId)
		fmt.Println("sqlUpdate: " + sqlUpdate)
		//***UPDATE DATABASE with video cut filename and YouTubeLink
		results, err := db.Query(sqlUpdate)
		if err != nil {
			fmt.Println(results)
			panic(err.Error()) // proper error handling instead of panic in your app
		}
	}
	db.Close()
}
