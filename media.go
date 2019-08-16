package livebarn

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

const getMediaURL = "https://livebarn.com/livebarn/api/v1.0.0/media/get?locale=en_ca"
const getMediaDownloadURL = "https://livebarn.com/livebarn/api/v1.0.0/media/download/get?locale=en_ca"

// GetMediaRequest is a struct for containing `X-LiveBarn-Data` parameters for
// the GetMedia request
type GetMediaRequest struct {
	Token      string   `json:"token"`
	UUID       string   `json:"uuid"`
	Surface    *Surface `json:"surface"`
	BeginDate  string   `json:"beginDate"`
	EndDate    string   `json:"endDate"`
	DeviceType string   `json:"deviceType"`
}

// GetMediaResponse represents the payload returned by GetMedia
type GetMediaResponse struct {
	Duration  int    `json:"duration"`
	BeginDate string `json:"beginDate"`
	RenditionId int  `json:"renditionId"`
	FeedModeId int  `json:"feedModeId"`
	URL       string `json:"url"`
}

// NullGetMediaResponse is a placeholder for an empty GetMediaResponse to be
// returned when an error occurs
var NullGetMediaResponse = &GetMediaResponse{}

// GetMediaDownloadRequest is a struct for containing `X-LiveBarn-Data`
// parameters for the GetMediaDownload request
type GetMediaDownloadRequest struct {
	Token    string `json:"token"`
	UUID     string `json:"uuid"`
	MediaURL string `json:"mediaUrl"`
}

// GetMediaDownloadResponse represents the payload returned by GetMediaDownload
type GetMediaDownloadResponse struct {
	Status int `json:"status"`
	Result struct {
		Duration int     `json:"duration"`
		Venue    Venue   `json:"venue"`
		Surface  Surface `json:"surface"`
		URL      string  `json:"url"`
	} `json:"result"`
	Timestamp int64  `json:"timestamp"`
	Date      string `json:"date"`
	Message   string `json:"message"`
}

// NullGetMediaDownloadResponse is a placeholder for an empty
// GetMediaDownloadResponse to be returned when an error occurs
var NullGetMediaDownloadResponse = &GetMediaDownloadResponse{}

// GetMedia returns video URLs for a given surface and date range
func (c *Client) GetMedia(surfaceid string, beginDate string) ([]GetMediaResponse, error) {

	//new code for API v2.0.0
	var searchURL = "https://webapi.livebarn.com/api/v2.0.0/media/surfaceid/" + surfaceid + "/begindate/" + beginDate
	//log.Printf("searchURL:\n %s \n", searchURL)
	req, err := http.NewRequest("GET", searchURL, nil)
    req.Header.Add("Authorization","Bearer " + c.Token)
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	
	//end new code
	
	if err != nil {
		log.Println("Error on httClient.\n[ERRO] -", err)
		return NullGetMediaResponse, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error on ReadAll.\n[ERRO] -", err)
		return NullGetMediaResponse, err
	}
	
	log.Printf("BODY:\n %s \n", body)

	var response []GetMediaResponse
	//var response GetMediaResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error on Unmarshal.\n[ERRO] -", err)
		return NullGetMediaResponse, err
	}
	
	return response, nil
}

// GetMediaDownload returns downloadable video URLs for a given media URL
func (c *Client) GetMediaDownload(mediaURL string) (*GetMediaDownloadResponse, error) {
	data := &GetMediaDownloadRequest{
		Token:    c.Token,
		UUID:     c.UUID,
		MediaURL: mediaURL,
	}

	resp, err := c.do(getMediaDownloadURL, data)
	if err != nil {
		return NullGetMediaDownloadResponse, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return NullGetMediaDownloadResponse, err
	}

	var response GetMediaDownloadResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return NullGetMediaDownloadResponse, err
	}

	return &response, nil
}
