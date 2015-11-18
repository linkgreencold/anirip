package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type SessionParameters struct {
	AccountStatus   string
	SearchTerm      string
	DesiredSeasons  string
	DesiredEpisodes string
	DesiredQuality  string
	DesiredLanguage string
	Cookies         CrunchyCookie
}

type CRError struct {
	Message string
	Err     error
}

func (e CRError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf(">>> Error : %v : %v", e.Message, e.Err)
	}
	return fmt.Sprintf(">>> Error : %v.", e.Message)
}

// Gets user input from the user and unmarshalls it into the input
func getStandardUserInput(prefixText string, input *string) error {
	fmt.Printf(prefixText)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		*input = scanner.Text()
		break
	}
	if err := scanner.Err(); err != nil {
		return CRError{"There was an error getting standard user input", err}
	}
	return nil
}

func getEpisodeFileName(showTitle string, seasonNumber string, episodeNumber string, episodeDesc string) string {
	return cleanFileName(showTitle + " - S0" + seasonNumber + "E0" + episodeNumber + " - " + episodeDesc)
}

// Cleans up the given filename so it can be written without any issues
func cleanFileName(fileName string) string {
	illegalCharacters := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	newFileName := fileName
	for _, illegalChar := range illegalCharacters {
		newFileName = strings.Replace(newFileName, illegalChar, " ", -1)
	}
	return newFileName
}

// Gets XML data for the requested request type and episode
func getXML(req string, episode Episode, params SessionParameters) (string, error) {
	xmlUrl := "http://www.crunchyroll.com/xml/?"

	// formdata to indicate the source page
	formData := url.Values{
		"current_page": {episode.URL},
	}

	// Constructs a queryString for user set settings
	queryString := url.Values{}
	if req == "RpcApiSubtitle_GetXml" {
		queryString = url.Values{
			"req":                {"RpcApiSubtitle_GetXml"},
			"subtitle_script_id": {strconv.Itoa(episode.Subtitle.ID)},
		}
	} else if req == "RpcApiVideoPlayer_GetStandardConfig" {
		queryString = url.Values{
			"req":           {"RpcApiVideoPlayer_GetStandardConfig"},
			"media_id":      {strconv.Itoa(episode.ID)},
			"video_format":  {"108"},
			"video_quality": {"80"},
			"auto_play":     {"1"},
			"aff":           {"crunchyroll-website"},
			"show_pop_out_controls":   {"1"},
			"pop_out_disable_message": {""},
			"click_through":           {"0"},
		}
	} else {
		queryString = url.Values{
			"req":                  {req},
			"media_id":             {strconv.Itoa(episode.ID)},
			"video_format":         {"108"},
			"video_encode_quality": {"80"},
		}
	}

	// Constructs a client and request that will get the xml we're asking for
	client := &http.Client{}
	xmlReq, err := http.NewRequest("POST", xmlUrl+queryString.Encode(), bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return "", CRError{"There was an error creating our getXML request", err}
	}
	xmlReq.Header.Add("Host", "www.crunchyroll.com")
	xmlReq.Header.Add("Origin", "http://static.ak.crunchyroll.com")
	xmlReq.Header.Add("Content-type", "application/x-www-form-urlencoded")
	xmlReq.Header.Add("Referer", "http://static.ak.crunchyroll.com/versioned_assets/StandardVideoPlayer.fb2c7182.swf")
	xmlReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.86 Safari/537.36")
	xmlReq.Header.Add("X-Requested-With", "ShockwaveFlash/19.0.0.245")
	for i := 0; i < len(params.Cookies.Cookies); i++ {
		xmlReq.AddCookie(params.Cookies.Cookies[i])
	}

	// Executes request and returns the result as a string
	resp, err := client.Do(xmlReq)
	if err != nil {
		return "", CRError{"There was an error executing our getXML request", err}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", CRError{"There was an error reading our getXML response", err}
	}
	return string(body), nil
}

// Splits up the FLV file so we can handle all peices with mergemkv
func splitMergeAndClean(episodeFileName string, episode Episode) error {
	err := splitEpisodeFLV(episodeFileName)
	if err != nil {
		return CRError{"There was an error while trying to split the FLV", err}
	}

	// Merges all the files together to create a single solid MKV
	err = mergeEpisodeMKV(episodeFileName)
	if err != nil {
		return CRError{"There was an issue while trying to merge the MKV", err}
	}

	// Attempts to remove all the files we're done with
	os.Remove("temp\\" + episodeFileName + ".ass")
	os.Remove("temp\\" + episodeFileName + ".264")
	os.Remove("temp\\" + episodeFileName + ".txt")
	os.Remove("temp\\" + episodeFileName + ".aac")
	os.Remove("temp\\" + episodeFileName + ".flv")
	return nil
}