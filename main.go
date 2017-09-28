// temp_restya_worker project main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

const (
	DEBUG              = false
	CONFIG_PATH string = `C:\goplace\src\config\temp_restya_worker\conf.ini`
)

//RESTYA_API_DOMAIN
const RESTYA_API_URL_GET_OAUTH_TOKEN string = "%s/v1/oauth.json?scope=write"

//RESTYA_API_DOMAIN token
const RESTYA_API_URL_GET_LOGIN_TOKEN string = "%s/v1/users/login.json?token=%s"

//RESTYA_API_DOMAIN board_id list_id token
const RESTYA_API_URL_POST_CREATE_CARD string = "%s/v1/boards/%d/lists/%d/cards.json?token=%s"

//RESTYA_API_DOMAIN board_id list_id card_id token
const RESTYA_API_URL_POST_ADD_LABEL_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/labels.json?token=%s"

//RESTYA_API_DOMAIN board_id list_id card_id token
const RESTYA_API_URL_POST_ADD_COMMENT_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/comments.json?token=%s"

//RESTYA_API_DOMAIN board_id list_id card_id user_id token
const RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/users/%d.json?token=%s"

//RESTYA_API_DOMAIN board_id list_id card_id token
const RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s.json?token=%s"

type settings struct {
	Ticket_folder_path string   `toml:"ticket_folder_path"`
	Api_data           api_data `toml:"api_data"`
	Board              board
}

type api_data struct {
	Restya_api_domain string
	Api_user          api_user
}
type api_user struct {
	Login    string
	Password string
}

type board struct {
	Board_id          int
	Sd_baclog_list_id int
	Members           map[string]member
}
type member struct {
	Id int
}

var client *http.Client = &http.Client{}
var token string
var member_list map[string]int
var temp_user int = 0
var temp_title string
var temp_title_data []string
var files []string
var dat []byte
var temp_arr []string
var file_path string
var val string
var dec *encoding.Decoder
var set settings

func start_load_ticket() {
	member_list = make(map[string]int)

	dec = charmap.Windows1251.NewDecoder()
	for true {
		time.Sleep(time.Second * 1)
		check_ticket()
	}
}

func main() {
	toml.DecodeFile(CONFIG_PATH, &set)
	fmt.Printf("%v\n", set)
}

func check_ticket() {
	files, _ = filepath.Glob(TICKET_FOLDER_PATH)

	if len(files) != 0 {
		token = get_token()
		temp_user = 0
		for _, file_path = range files {
			dat, _ = ioutil.ReadFile(file_path)
			dat, _ = dec.Bytes(dat)
			temp_arr = strings.Split(file_path, "\\")
			temp_title = temp_arr[len(temp_arr)-1]
			temp_title_data = strings.Fields(temp_title)
			for _, val = range temp_title_data {
				if strings.Contains(val, "#") {
					temp_title = strings.Replace(temp_title, val, "", -1)
				}
				if strings.Contains(val, "@") {
					temp_title = strings.Replace(temp_title, val, "", -1)
					temp_user = member_list[val]
				}

			}
			os.Remove(file_path)
			create_card(temp_title, strings.Replace(string(dat), "\t", " ", -1), "BPM", temp_user)

		}

	}
}

func create_card(title string, description string, label string, user_id int) {
	const JSON_BODY_PTR_ADD_CARD string = `{"board_id": %d,"list_id": %d,"name": "%s","position": 0}`
	const JSON_BODY_PTR_ADD_LABEL string = `{"name": "%s"}`
	const JSON_BODY_PTR_ADD_DESCRIPTION string = `{"description": "%s"}`
	const JSON_BODY_PTR_ADD_MEMBER string = `{"card_id":%s,"user_id":%d}`

	var temp_body string
	var objmap map[string]*json.RawMessage
	var buf *bytes.Buffer

	temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_CARD, BOARD_ID, SD_BACLOG_LIST_ID, title)
	url := fmt.Sprintf(RESTYA_API_URL_POST_CREATE_CARD, RESTYA_API_DOMAIN, BOARD_ID, SD_BACLOG_LIST_ID, token)

	resp, _ := client.Post(url, "application/json", strings.NewReader(temp_body))
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	json.Unmarshal(buf.Bytes(), &objmap)
	tmp_json_val, _ := json.Marshal(objmap["id"])
	card_id := strings.Trim(string(tmp_json_val), `"`)
	if DEBUG {
		fmt.Println(buf.String())
	}

	buf.Reset()
	if label != "" {
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_LABEL, label)
		url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_LABEL_TO_CARD, RESTYA_API_DOMAIN, BOARD_ID, SD_BACLOG_LIST_ID, card_id, token)
		client.Post(url, "application/json", strings.NewReader(temp_body))
	}
	if description != "" {
		description = strings.Replace(description, "\r\n", " ", -1)
		description = strings.Replace(description, `"`, `'`, -1)
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_DESCRIPTION, description)
		url = fmt.Sprintf(RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD, RESTYA_API_DOMAIN, BOARD_ID, SD_BACLOG_LIST_ID, card_id, token)
		req, _ := http.NewRequest("PUT", url, strings.NewReader(temp_body))
		req.Header.Set("Content-Type", "application/json")
		fmt.Println(url)
		client.Do(req)
	}
	if user_id != 0 {
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_MEMBER, card_id, user_id)
		url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD, RESTYA_API_DOMAIN, BOARD_ID, SD_BACLOG_LIST_ID, card_id, user_id, token)
		resp, _ = client.Post(url, "application/json", strings.NewReader(temp_body))
		buf.ReadFrom(resp.Body)
		if DEBUG {
			fmt.Println(buf.String())
		}
	}

}

func get_token() string {

	const JSON_BODY_REQ_TOKEN string = `{"email": "` + USER_WORK + `","password": "` + PWD_WORK + `"}`

	var access_token string = ""
	var objmap map[string]*json.RawMessage
	var tmp_json_val []byte
	var resp *http.Response
	var temp_url string

	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_OAUTH_TOKEN, RESTYA_API_DOMAIN)
	resp, _ = client.Get(temp_url)
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	resp.Body.Close()

	json.Unmarshal(buf.Bytes(), &objmap)
	buf.Reset()
	tmp_json_val, _ = json.Marshal(objmap["access_token"])
	access_token = strings.Trim(string(tmp_json_val), `"`)
	if DEBUG {
		fmt.Println(access_token)
	}
	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_LOGIN_TOKEN, RESTYA_API_DOMAIN, access_token)
	resp, _ = client.Post(temp_url, "application/json", strings.NewReader(JSON_BODY_REQ_TOKEN))

	buf.ReadFrom(resp.Body)
	json.Unmarshal(buf.Bytes(), &objmap)
	buf.Reset()
	tmp_json_val, _ = json.Marshal(objmap["access_token"])
	access_token = string(tmp_json_val)
	if DEBUG {
		fmt.Println(access_token)
	}
	return strings.Trim(access_token, `"`)

}
