package mym

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"temp_restya_worker/consts/strs"
)

type R_check_list struct {
	Id   string
	Rows []string
}

type R_card struct {
	Id           int
	Name         string
	Body         string
	Board_id     int
	Is_archived  int
	List_id      int
	Cards_users  []R_cards_user
	Cards_labels []R_cards_label
	Check_lists  map[string]R_check_list
}

type R_cards_label struct {
	Name     string
	Label_id int
}

type R_cards_user struct {
	Username string
	User_id  int
}

type R_board struct {
	Id    int
	Lists []R_list
}

type R_list struct {
	Id    int
	Cards []R_card
}

type R_api struct {
	Debug      bool
	Token      string
	U_login    string
	U_pwd      string
	Api_domain string
	Board_id   int
	Client     *http.Client
}

func (ra R_api) Acrh_list(list_id int, escape_label_id int) {
	cards := ra.Get_no_arch_cards_from_list(list_id)
	for _, card := range cards {
		if !card.has_label(escape_label_id) {
			ra.arch_card(list_id, card.Id)
		}
	}
}

func (ra R_api) arch_card(list_id int, card_id int) {
	const RESTYA_API_URL_PUT_BOARD_INFO string = "%s/v1/boards/%d/lists/%d/cards/%d.json?token=%s"
	const PUT_REQ_BODY_ARCH string = `{"is_archived":1}`
	ra.GetToken()
	url := fmt.Sprintf(RESTYA_API_URL_PUT_BOARD_INFO, ra.Api_domain, ra.Board_id, list_id, card_id, ra.Token)
	req, _ := http.NewRequest("PUT", url, strings.NewReader(PUT_REQ_BODY_ARCH))
	req.Header.Set("Content-Type", "application/json")
	ra.Client.Do(req)
}

func (ra R_api) Un_arch_card(list_id int, card_id int) {
	const RESTYA_API_URL_PUT_BOARD_INFO string = "%s/v1/boards/%d/lists/%d/cards/%d.json?token=%s"
	const PUT_REQ_BODY_ARCH string = `{"is_archived":0}`
	ra.GetToken()
	url := fmt.Sprintf(RESTYA_API_URL_PUT_BOARD_INFO, ra.Api_domain, ra.Board_id, list_id, card_id, ra.Token)
	req, _ := http.NewRequest("PUT", url, strings.NewReader(PUT_REQ_BODY_ARCH))
	req.Header.Set("Content-Type", "application/json")
	ra.Client.Do(req)
}

func (c R_card) has_label(label_id int) bool {
	res := false
	for _, label := range c.Cards_labels {
		if label.Label_id == label_id {
			res = true
		}
	}

	return res
}

func (c R_card) Get_bpm_id() (bpm_id string, err error) {
	//ToDo to conf
	const BPM_ID_REXEP_PTR string = "(SR[0-9]{8})|(TT[0-9]{7})"
	//ToDo to conf
	r, _ := regexp.Compile(BPM_ID_REXEP_PTR)
	res := r.FindAllString(c.Name, -1)

	if len(res) > 0 {
		return res[0], nil
	} else {
		return strs.EMPTY_STRING, errors.New("no bpm ticket")
	}

}

func (b R_board) get_list(id int) R_list {
	var temp_list R_list
	for _, list := range b.Lists {
		if list.Id == id {
			temp_list = list
		}
	}
	return temp_list
}

func (ra R_api) Create_card(card R_card) {
	const JSON_BODY_PTR_ADD_CARD string = `{"board_id": %d,"list_id": %d,"name": "%s","position": 0}`
	const JSON_BODY_PTR_ADD_LABEL string = `{"name": "%s"}`
	const JSON_BODY_PTR_ADD_DESCRIPTION string = `{"description": "%s"}`
	const JSON_BODY_PTR_ADD_MEMBER string = `{"card_id":%d,"user_id":%d}`
	const JSON_BODY_PTR_ADD_CHEACK_LIST string = `{"name":"%s","position":1}`
	const JSON_BODY_PTR_ADD_ROW_TOO_CHEACK_LIST string = `{"name":"%s"}`
	const RESTYA_API_URL_POST_CREATE_CARD string = "%s/v1/boards/%d/lists/%d/cards.json?token=%s"
	const RESTYA_API_URL_POST_ADD_LABEL_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%d/labels.json?token=%s"
	const RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%d.json?token=%s"
	const RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%d/users/%d.json?token=%s"
	const RESTYA_API_URL_POST_ADD_CHEACK_LIST_TOO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%d/checklists.json?token=%s"
	const RESTYA_API_URL_POST_ADD_ROW_TOO_CHEACK_LIST string = "%s/v1/boards/%d/lists/%d/cards/%d/checklists/%s/items.json?token=%s"

	var temp_body string
	var buf *bytes.Buffer
	restya_card := struct{ Id string }{}

	//	var objmap map[string]*json.RawMessage

	ra.GetToken()
	temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_CARD, ra.Board_id, card.List_id, card.Name)
	url := fmt.Sprintf(RESTYA_API_URL_POST_CREATE_CARD, ra.Api_domain, ra.Board_id, card.List_id, ra.Token)

	resp, _ := ra.Client.Post(url, "application/json", strings.NewReader(temp_body))
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	json.Unmarshal(buf.Bytes(), &restya_card)
	card.Id, _ = strconv.Atoi(restya_card.Id)

	buf.Reset()

	if len(card.Cards_labels) > 0 {
		for _, label := range card.Cards_labels {
			temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_LABEL, label.Name)
			url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_LABEL_TO_CARD, ra.Api_domain, ra.Board_id, card.List_id, card.Id, ra.Token)
			ra.Client.Post(url, "application/json", strings.NewReader(temp_body))
		}

	}
	if card.Body != "" {
		card.Body = strings.Replace(card.Body, strs.EMPTY_STRING, strs.SPACE, -1)
		card.Body = strings.Replace(card.Body, strs.DOUBLE_QUOTES, strs.QUOTATION, -1)
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_DESCRIPTION, card.Body)
		url = fmt.Sprintf(RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD, ra.Api_domain, ra.Board_id, card.List_id, card.Id, ra.Token)
		req, _ := http.NewRequest("PUT", url, strings.NewReader(temp_body))
		req.Header.Set("Content-Type", "application/json")
		ra.Client.Do(req)
	}
	if len(card.Cards_users) > 0 {
		for _, user := range card.Cards_users {
			temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_MEMBER, card.Id, user.User_id)
			url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD, ra.Api_domain, ra.Board_id, card.List_id, card.Id, user.User_id, ra.Token)
			resp, _ = ra.Client.Post(url, "application/json", strings.NewReader(temp_body))
			buf.ReadFrom(resp.Body)
			if ra.Debug {
				fmt.Println(buf.String())
			}
			buf.Reset()
		}

	}
	if len(card.Check_lists) > 0 {
		for list_name, list := range card.Check_lists {
			temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_CHEACK_LIST, list_name)
			url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_CHEACK_LIST_TOO_CARD, ra.Api_domain, ra.Board_id, card.List_id, card.Id, ra.Token)
			resp, _ = ra.Client.Post(url, "application/json", strings.NewReader(temp_body))
			buf.ReadFrom(resp.Body)
			json.Unmarshal(buf.Bytes(), &list)
			for _, row := range list.Rows {
				temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_ROW_TOO_CHEACK_LIST, row)
				url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_ROW_TOO_CHEACK_LIST, ra.Api_domain, ra.Board_id, card.List_id, card.Id, list.Id, ra.Token)
				ra.Client.Post(url, "application/json", strings.NewReader(temp_body))
				fmt.Println("Name - ", list_name, " Row - ", row)
			}
			buf.Reset()

		}
	}

}

func (ra R_api) Get_no_arch_cards_from_list(list_id int) []R_card {
	const RESTYA_API_URL_GET_BOARD_INFO string = "%s/v1/boards/%d.json?token=%s"
	var cards []R_card
	board := R_board{}
	ra.GetToken()
	url := fmt.Sprintf(RESTYA_API_URL_GET_BOARD_INFO, ra.Api_domain, ra.Board_id, ra.Token)
	resp, _ := ra.Client.Get(url)
	dec := json.NewDecoder(resp.Body)
	dec.More()
	dec.Decode(&board)
	list := board.get_list(list_id)
	for _, card := range list.Cards {
		if card.Is_archived == 0 {
			cards = append(cards, card)
		}
	}

	return cards

}

func (ra *R_api) GetToken() {
	//RESTYA_API_DOMAIN
	const RESTYA_API_URL_GET_OAUTH_TOKEN string = "%s/v1/oauth.json?scope=write"
	//RESTYA_API_DOMAIN token
	const RESTYA_API_URL_GET_LOGIN_TOKEN string = "%s/v1/users/login.json?token=%s"

	var JSON_BODY_REQ_TOKEN string = `{"email": "` + ra.U_login + `","password": "` + ra.U_pwd + `"}`

	var access_token string = ""
	var objmap map[string]*json.RawMessage
	var tmp_json_val []byte
	var resp *http.Response
	var temp_url string

	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_OAUTH_TOKEN, ra.Api_domain)
	resp, _ = ra.Client.Get(temp_url)
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	resp.Body.Close()

	json.Unmarshal(buf.Bytes(), &objmap)
	buf.Reset()
	tmp_json_val, _ = json.Marshal(objmap["access_token"])
	access_token = strings.Trim(string(tmp_json_val), `"`)
	if ra.Debug {
		fmt.Println(access_token)
	}

	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_LOGIN_TOKEN, ra.Api_domain, access_token)
	resp, _ = ra.Client.Post(temp_url, "application/json", strings.NewReader(JSON_BODY_REQ_TOKEN))

	buf.ReadFrom(resp.Body)
	json.Unmarshal(buf.Bytes(), &objmap)
	buf.Reset()
	tmp_json_val, _ = json.Marshal(objmap["access_token"])
	access_token = string(tmp_json_val)
	if ra.Debug {
		fmt.Println(access_token)
	}
	ra.Token = strings.Trim(access_token, strs.DOUBLE_QUOTES)

}
