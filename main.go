// temp_restya_worker project main.go
package main

import (
	"bytes"
	"crypto/md5"
	"elk_conn/rest_db_conn"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	tc "tfs_client/client"

	cron "gopkg.in/robfig/cron.v2"

	"temp_restya_worker/consts/strs"
	"temp_restya_worker/mym"
	//	"temp_restya_worker/temp"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

const (
	DEBUG               = false
	MAIN_CONF    string = `L:\DIGITAL\restya\conf\`
	CONFIG_PATH  string = MAIN_CONF + `conf.ini`
	CRON_PATH    string = MAIN_CONF + `cron`
	PATH_TO_TPLS string = MAIN_CONF + `tpl\`
	//RESTYA_API_DOMAIN
	RESTYA_API_URL_GET_OAUTH_TOKEN string = "%s/v1/oauth.json?scope=write"

	//RESTYA_API_DOMAIN token
	RESTYA_API_URL_GET_LOGIN_TOKEN string = "%s/v1/users/login.json?token=%s"

	//RESTYA_API_DOMAIN board_id token search_name
	RESTYA_API_URL_GET_SEARCH_CARD string = "%s/v1/boards/%d/cards/search.json?token=%s&q=%s"

	//RESTYA_API_DOMAIN board_id list_id token
	RESTYA_API_URL_POST_CREATE_CARD string = "%s/v1/boards/%d/lists/%d/cards.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_POST_ADD_LABEL_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/labels.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_POST_ADD_COMMENT_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/comments.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_POST_СOPY_CARD string = "%s/v1/boards/%d/lists/%d/cards/%d/copy.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id user_id token
	RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s/users/%d.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD string = "%s/v1/boards/%d/lists/%d/cards/%s.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_PUT_MOVE_CARD_TO_LIST string = "%s/v1/boards/%d/lists/{listId}/cards.json?token=%s"

	//RESTYA_API_DOMAIN board_id list_id card_id token
	RESTYA_API_URL_PUT_ADD_DUE_DATE string = "%s/v1/boards/%d/lists/%d/cards/%s.json?token=%s"
)

type settings struct {
	Ticket_folder_path string     `toml:"ticket_folder_path"`
	Api_data           api_data   `toml:"api_data"`
	Tfs_data           tfs_data   `toml:"tfs_data"`
	Slack_data         slack_data `toml:"slack_data"`
	Restya_db_data     restya_db_data
	Board              board
	Test_string        string
	Path_to_ref        string
	Our_proxy          string
}

type restya_db_data struct {
	Elk_addres    string
	Db_host       string
	Db_port       int
	Db_user       string
	Db_pwd        string
	Db_name       string
	T_user        string
	T_pwd         string
	T_local_ip    string
	T_local_port  int
	T_server_ip   string
	T_server_port int
	T_remote_ip   string
	T_remote_port int
}

type slack_data struct {
	Web_hooks_url             string
	Sql_ptr_select_user_names string
	Sql_ptr_select_activeties string
	Table_title               string
}

type tfs_data struct {
	User        string
	Pwd         string
	Domain      string
	Protocol    string
	Api_version string
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
	Id                int
	Sd_baclog_list_id int
	Members           map[string]member
}
type member struct {
	Id           int
	Outlook_name string
	Tfs_name     string
}

type restya_card struct {
	Id       int
	Name     string
	Board_id int
}

//ToDo DISAPPEAR THINGS
type refinement struct {
	bpm_id      string
	add         time.Time
	member_name string
}

type refinements struct {
	refs []refinement
}

//ToDo DISAPPEAR THINGS

var client *http.Client = &http.Client{}
var token string
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
var cron_body_md5 string = ""
var func_map map[string]func(map[string]string)
var tfs_client tc.TFS_api_client

func start_load_ticket() {

	for true {
		time.Sleep(time.Second * 1)
		check_ticket()
	}

}

func check_ref() {
	ra := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
	cards := ra.Get_no_arch_cards_from_list(422)
	refs := get_ref()
	enc := charmap.Windows1251.NewEncoder()
	for _, card := range cards {
		if bpm_id, err := card.Get_bpm_id(); err == nil {
			if len(card.Cards_users) > 0 {
				refs.add_ref(bpm_id, fmt.Sprintf("%d", time.Now().Unix()), set.Board.Members[fmt.Sprint("@", card.Cards_users[0].Username)].Outlook_name)
			}

		}

	}

	str_to_file, _ := enc.String(refs.str_write_to_file())
	ref_file, _ := os.Create(set.Path_to_ref)
	ref_file.Write([]byte(str_to_file))
	ref_file.Close()
}

func cron_chat_write(msg string) {
	fmt.Println(msg)
}

func cron_arch_list(lists []string, esc int) {
	var temp_list_id int
	ra := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
	//	ra.Un_arch_card(419, 3076)

	for _, list_id := range lists {
		temp_list_id, _ = strconv.Atoi(list_id)
		ra.Acrh_list(temp_list_id, esc)
	}
}

func reopen_card(card_id int, list_id int) {
	toml.DecodeFile(CONFIG_PATH, &set)
	ra := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
	ra.Un_arch_card(list_id, card_id)
}

func start_cron() {

	var c *cron.Cron = cron.New()
	var cron_str string
	var is_new bool

	dec = charmap.Windows1251.NewDecoder()
	toml.DecodeFile(CONFIG_PATH, &set)
	tfs_client = tc.TFS_api_client{Login: set.Tfs_data.User,
		Pwd:         set.Tfs_data.Pwd,
		Domain:      set.Tfs_data.Domain,
		Protocol:    set.Tfs_data.Protocol,
		Api_version: set.Tfs_data.Api_version}
	c.Start()
	for true {
		time.Sleep(time.Second * 1)

		if cron_str, is_new = check_is_new_cron(); is_new {
			c.Stop()
			c = test_c(cron_str)
			c.Start()
		}
	}
}

func main() {
	start_cron()
	//	un_arch_card()
	//	copy_restya_list()
	//	_, err := toml.DecodeFile(CONFIG_PATH, &set)
	//	if err != nil {
	//		fmt.Println(err.Error())
	//	}

}
func send_report_to_slack(msg string) {

	const WEB_HOOKS_CONTENT_TYPE string = "application/json"

	var slack_client *http.Client
	type body_struct struct {
		Text string `json:"text"`
	}
	body_json := body_struct{Text: msg}

	body_bytes, err_json := json.Marshal(body_json)
	if err_json != nil {
		fmt.Println(err_json)
	}
	proxyURL, err := url.Parse(set.Our_proxy)
	if err != nil {
		fmt.Println(err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	slack_client = &http.Client{
		Transport: transport,
	}
	slack_client.Post(set.Slack_data.Web_hooks_url, WEB_HOOKS_CONTENT_TYPE, bytes.NewReader(body_bytes))
}

func list_act_per_usr(users_list_int []int, format_row string) string {

	const (
		PTR_START_DAY   string = "%s 00:00:00"
		PTR_END_DAY     string = "%s 23:59:59"
		PTR_DATE_FORMAT string = "%d-%02d-%02d"
		START_WORK_WEEK string = ""
	)

	var users_list []string
	var names_and_actions map[string]int
	var temp_name string
	var temp_act_count int
	var start_date_str string
	var end_date_str string
	var result_table string
	var end_date time.Time
	temp_time := time.Now()

	if temp_time.Weekday() == time.Monday {
		temp_time = temp_time.AddDate(0, 0, -3)
	} else {
		temp_time = temp_time.AddDate(0, 0, -1)
	}
	end_date = temp_time
	for temp_time.Weekday() != time.Sunday {
		temp_time = temp_time.AddDate(0, 0, -1)
	}
	start_date_str = fmt.Sprintf(PTR_DATE_FORMAT, temp_time.Year(), temp_time.Month(), temp_time.Day())
	end_date_str = fmt.Sprintf(PTR_DATE_FORMAT, end_date.Year(), end_date.Month(), end_date.Day())
	names_and_actions = make(map[string]int)
	for _, val := range users_list_int {
		users_list = append(users_list, strconv.Itoa(val))
	}
	conn := rest_db_conn.Init(set.Restya_db_data.T_user, set.Restya_db_data.T_pwd, &rest_db_conn.Endpoint{Host: set.Restya_db_data.T_local_ip, Port: set.Restya_db_data.T_local_port}, &rest_db_conn.Endpoint{Host: set.Restya_db_data.T_server_ip, Port: set.Restya_db_data.T_server_port}, &rest_db_conn.Endpoint{Host: set.Restya_db_data.T_remote_ip, Port: set.Restya_db_data.T_remote_port}, set.Restya_db_data.Db_host, set.Restya_db_data.Db_port, set.Restya_db_data.Db_user, set.Restya_db_data.Db_pwd, set.Restya_db_data.Db_name)
	defer conn.Close_rc()
	query := fmt.Sprintf(set.Slack_data.Sql_ptr_select_user_names, strings.Join(users_list, ","))
	name_rows, err := conn.Query(query)
	if err != nil {
		fmt.Println(err.Error())
	}
	for name_rows.Next() {
		err = name_rows.Scan(&temp_name)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			names_and_actions[temp_name] = 0
		}

	}
	query = fmt.Sprintf(set.Slack_data.Sql_ptr_select_activeties, fmt.Sprintf(PTR_START_DAY, start_date_str), fmt.Sprintf(PTR_END_DAY, end_date_str))
	name_rows, err = conn.Query(query)
	if err != nil {
		fmt.Println(err.Error())
	}
	for name_rows.Next() {
		err = name_rows.Scan(&temp_name, &temp_act_count)
		if err != nil {
			fmt.Println(err.Error())
		}
		names_and_actions[temp_name] = temp_act_count
	}
	result_table = set.Slack_data.Table_title + "\r\n"
	for name, act_count := range names_and_actions {
		result_table = result_table + fmt.Sprintf(format_row, name, act_count)
	}
	return strings.Trim(result_table, "\r\n")
}

func copy_restya_list() {
	dec = charmap.Windows1251.NewDecoder()
	toml.DecodeFile(CONFIG_PATH, &set)
	ra := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
	tfs_client = tc.TFS_api_client{Login: set.Tfs_data.User,
		Pwd:         set.Tfs_data.Pwd,
		Domain:      set.Tfs_data.Domain,
		Protocol:    set.Tfs_data.Protocol,
		Api_version: set.Tfs_data.Api_version}
	cards := ra.Get_no_arch_cards_from_list(420)
	for _, card := range cards {
		tfs_add_ticket(card)
	}
}

func tfs_add_ticket(card mym.R_card) {
	var card_params []tc.TFS_req_struct
	var temp_state string
	var temp_substate string
	temp_tfs_user_name := tc.TFS_STANDART_TFS_USER
	card_params = append(card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_TITLE,
		From:   nil,
		Value:  card.Name})
	card_params = append(card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_AREA,
		From:   nil,
		Value:  tc.TFS_DIGITAL_SUPPORT_AREA})
	card_params = append(card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_DESCRIPTION,
		From:   nil,
		Value:  card.Body})
	card_params = append(card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_TAGS,
		From:   nil,
		Value:  "BPM"})
	card_params = append(card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_DUE_DATE,
		From:   nil,
		Value:  "05/15/2018"})
	if len(card.Cards_users) > 0 {
		if set.Board.Members[`@`+card.Cards_users[0].Username].Tfs_name != "" {
			temp_tfs_user_name = set.Board.Members[`@`+card.Cards_users[0].Username].Tfs_name
		}
		card_params = append(card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_ASSIGNED_TO,
			From:   nil,
			Value:  temp_tfs_user_name})
	}
	if card.List_id != 417 {
		switch card.List_id {
		case 418:
			temp_state = tc.TFS_STATE_TO_DO
			temp_substate = tc.TFS_STANDART_SUBSTATUS
		case 419:
			temp_state = tc.TFS_STATE_IN_PROGRESS
			temp_substate = tc.TFS_IN_PROGRESS_SUBSTATUS
		case 420:
			temp_state = tc.TFS_STATE_TO_VENDOR
			temp_substate = tc.TFS_IN_TO_VENDOR_SUBSTATUS
		case 421:
			fmt.Println("OS X.")
		case 422:
			fmt.Println("OS X.")
		}
		card_params = append(card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_STATE,
			From:   nil,
			Value:  temp_state})
		card_params = append(card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_ACTIVATED_BY,
			From:   nil,
			Value:  temp_tfs_user_name})
		card_params = append(card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_SUBSTATUS,
			From:   nil,
			Value:  temp_substate})
	}
	r, _ := tfs_client.Add_card(card_params)
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	s := buf.String()

	fmt.Println(s)
}

func un_arch_card() {
	dec = charmap.Windows1251.NewDecoder()
	toml.DecodeFile(CONFIG_PATH, &set)
	m := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
	m.Un_arch_card(421, 4854)
}

func check_is_new_cron() (string, bool) {
	b, _ := ioutil.ReadFile(CRON_PATH)
	temp_md5 := fmt.Sprintf("%x", md5.Sum(b))

	if temp_md5 != cron_body_md5 {
		fmt.Println(temp_md5)
		cron_body_md5 = temp_md5
		return string(b), true
	} else {
		return strs.EMPTY_STRING, false
	}

}

func cron_create_card(tpl_file string) {
	if tpl_file != strs.EMPTY_STRING {
		var user_id int = 0

		type tpl struct {
			Desc        string
			Body        string
			Members     []int
			Check_lists map[string]mym.R_check_list
		}

		tpl_data := tpl{}

		tpl_path := PATH_TO_TPLS + tpl_file
		toml.DecodeFile(tpl_path, &tpl_data)
		if len(tpl_data.Members) > 0 {
			user_id = tpl_data.Members[0]
		}
		token = get_token()
		ra := mym.R_api{Debug: DEBUG, U_login: set.Api_data.Api_user.Login, U_pwd: set.Api_data.Api_user.Password, Api_domain: set.Api_data.Restya_api_domain, Board_id: set.Board.Id, Client: &http.Client{}}
		card := mym.R_card{Name: tpl_data.Desc, Board_id: set.Board.Id, List_id: set.Board.Sd_baclog_list_id, Cards_users: []mym.R_cards_user{mym.R_cards_user{User_id: user_id}}, Body: tpl_data.Body, Check_lists: tpl_data.Check_lists}
		ra.Create_card(card)
	}

}

func print_ln() {
	fmt.Println("+")
}

func test_c(cron_str string) *cron.Cron {
	const COMMENT_MARKER string = "#"
	const MIN_CRON_STR_ELEMENT_COUNT int = 6

	var last_entry_id cron.EntryID = 0
	var cron_param_name map[cron.EntryID]map[string]string
	cron_param_name = make(map[cron.EntryID]map[string]string)
	c := cron.New()

	cron_rows := strings.Split(cron_str, strs.NEW_LINE)
	for _, cron_row := range cron_rows {
		if !strings.HasPrefix(cron_row, COMMENT_MARKER) {
			cron_row = strings.Trim(cron_row, strs.SPACE)
			if cron_row != strs.EMPTY_STRING {
				temp_cron_line_data := strings.Split(cron_row, strs.SPACE)
				if len(temp_cron_line_data) > MIN_CRON_STR_ELEMENT_COUNT {

					cron_param_name[last_entry_id+1] = make(map[string]string)
					set_cron_func(c, temp_cron_line_data[6:], strings.Join(temp_cron_line_data[:6], strs.SPACE))
				}

			}
		}

	}

	return c
}

func cron_send_report_to_support() {
	const ROW_TPL string = "Инженер: %s, действия - %d\r\n"
	users_id := []int{22, 19, 17, 10, 9, 8, 4}

	send_report_to_slack(list_act_per_usr(users_id, ROW_TPL))
}

func set_cron_func(c *cron.Cron, param []string, cron_time_set string) {

	var command_name string
	var testMsg string
	var lists_for_arch string
	var esc_mark int
	var tpl_name string

	f := flag.NewFlagSet("f", flag.ContinueOnError)
	f.StringVar(&command_name, "c", strs.EMPTY_STRING, strs.EMPTY_STRING)
	f.StringVar(&testMsg, "msg", strs.EMPTY_STRING, strs.EMPTY_STRING)
	f.StringVar(&lists_for_arch, "list", strs.EMPTY_STRING, strs.EMPTY_STRING)
	f.StringVar(&tpl_name, "tpl", strs.EMPTY_STRING, strs.EMPTY_STRING)
	f.IntVar(&esc_mark, "esc", 0, strs.EMPTY_STRING)

	f.Parse(param)
	if _, err := c.AddFunc(cron_time_set, func() {
		//		fmt.Println(param["c"])
		switch command_name {
		case "wrt":
			cron_chat_write(testMsg)
		case "check_ticket":
			check_ticket()
		case "check_ref":
			check_ref()
		case "arch_list":
			cron_arch_list(strings.Split(lists_for_arch, ","), esc_mark)
		case "create_card":
			cron_create_card(tpl_name)
		case "send_report_to_support":
			cron_send_report_to_support()
		}

	}); err != nil {
		fmt.Println(err.Error())
	}
}

func (r refinements) is_exist_ref(bpm_id string) bool {
	res := false
	for _, ref := range r.refs {
		if ref.bpm_id == bpm_id {
			res = true
		}
	}
	return res
}

func (r refinements) str_write_to_file() string {
	var temp_row []string
	for _, ref := range r.refs {
		temp_row = append(temp_row, fmt.Sprintf("%s:%d:%s", ref.bpm_id, ref.add.Unix(), ref.member_name))
	}
	return strings.Join(temp_row, "\r\n|")
}

func (r *refinements) add_ref(bpm_id string, timestamp string, m_name string) {

	if !r.is_exist_ref(bpm_id) {
		if temp_int, err := strconv.ParseInt(timestamp, 10, 64); err == nil {

			r.refs = append(r.refs, refinement{bpm_id: bpm_id, add: time.Unix(temp_int, 0), member_name: m_name})
		}
	}

}

func get_ref() refinements {
	const BPM_ID int = 0
	const TIME int = 1
	const NAME int = 2
	var res refinements = refinements{}
	var row_data []string

	dat, _ = ioutil.ReadFile(set.Path_to_ref)
	dat, _ = dec.Bytes(dat)
	rows := strings.Split(strings.Trim(string(dat), strs.NEW_LINE), "\r\n|")
	for _, row := range rows {
		row_data = strings.Split(row, ":")
		res.add_ref(row_data[BPM_ID], row_data[TIME], row_data[NAME])

	}
	return res
}

func (c *restya_card) copy(card_tittle string) {
	const BODY_PTR_FOR_COPY_CARD string = `{
											  "copied_card_id": %d,
											  "keep_activities": "1",
											  "keep_attachments": "1",
											  "keep_checklists": "1",
											  "keep_labels": "1",
											  "keep_users": "1",
											  "position":"1",
											  "is_archived": false,
											  "list_id":"%d",
											  "name": "%s"
											}`

	var buf *bytes.Buffer
	token = get_token()
	url := fmt.Sprintf(RESTYA_API_URL_POST_СOPY_CARD, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, c.Id, token)

	body := fmt.Sprintf(BODY_PTR_FOR_COPY_CARD, c.Id, set.Board.Sd_baclog_list_id, card_tittle)
	fmt.Println(body)
	resp, _ := client.Post(url, "application/json", strings.NewReader(body))
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	fmt.Printf("%s\n", buf.String())
}

func check_ticket() {
	files, _ = filepath.Glob(set.Ticket_folder_path)
	is_new := true

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
					temp_title = strings.Replace(temp_title, val, strs.EMPTY_STRING, -1)
				}
				if strings.Contains(val, "@") {
					temp_title = strings.Replace(temp_title, val, strs.EMPTY_STRING, -1)
					temp_user = set.Board.Members[val].Id
					is_new = false
				}

			}
			os.Remove(file_path)
			if is_new {
				create_card(temp_title, strings.Replace(string(dat), "\t", strs.SPACE, -1), "BPM", temp_user)
			} else {
				//ToDo to conf
				const SERVICE_DESCK_IDS_PTR string = "(SR[0-9]{8}|TT[0-9]{7})"
				//ToDo to conf
				var newest_restya_card restya_card
				re := regexp.MustCompile(SERVICE_DESCK_IDS_PTR)
				find_cards := restya_cards_search(re.FindAllString(temp_title, -1)[0])

				for _, card := range find_cards {
					if card.Id > newest_restya_card.Id {
						newest_restya_card = card
					}
				}
				newest_restya_card.copy(temp_title)
			}

		}

	}
}

func restya_cards_search(search_str string) []restya_card {
	var result, find_res []restya_card
	token = get_token()
	url := fmt.Sprintf(RESTYA_API_URL_GET_SEARCH_CARD, set.Api_data.Restya_api_domain, set.Board.Id, token, search_str)
	fmt.Println(url)
	resp, _ := client.Get(url)
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &find_res)
	for _, card := range find_res {
		if card.Board_id == set.Board.Id {
			result = append(result, card)
		}

		//		fmt.Printf("Board - %d Name - %s\n", card.Board_id, card.Name)
	}

	return result
}

func create_card(title string, description string, label string, user_id int) {
	const JSON_BODY_PTR_ADD_CARD string = `{"board_id": %d,"list_id": %d,"name": "%s","position": 0}`
	const JSON_BODY_PTR_ADD_LABEL string = `{"name": "%s"}`
	const JSON_BODY_PTR_ADD_DESCRIPTION string = `{"description": "%s"}`
	const JSON_BODY_PTR_ADD_MEMBER string = `{"card_id":%s,"user_id":%d}`
	const JSON_BODY_PTR_ADD_DUE_DATE string = `{"to_date":"%s","due_date":"%s","start":"%s"}`
	const HYPERLINK_PTR string = `\[Перейти к задаче\]\((.+)\)`

	re := regexp.MustCompile(HYPERLINK_PTR)

	var temp_body string
	var objmap map[string]*json.RawMessage
	var buf *bytes.Buffer
	var tfs_card_params []tc.TFS_req_struct

	temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_CARD, set.Board.Id, set.Board.Sd_baclog_list_id, title)
	url := fmt.Sprintf(RESTYA_API_URL_POST_CREATE_CARD, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, token)

	tfs_card_params = append(tfs_card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_AREA,
		From:   nil,
		Value:  tc.TFS_DIGITAL_SUPPORT_AREA})
	tfs_card_params = append(tfs_card_params, tc.TFS_req_struct{
		Action: tc.TFS_ACTIONS_ADD,
		Field:  tc.TFS_FIELDS_TITLE,
		From:   nil,
		Value:  title})
	resp, _ := client.Post(url, "application/json", strings.NewReader(temp_body))
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	json.Unmarshal(buf.Bytes(), &objmap)
	tmp_json_val, _ := json.Marshal(objmap["id"])
	card_id := strings.Trim(string(tmp_json_val), strs.DOUBLE_QUOTES)
	if DEBUG {
		fmt.Println(buf.String())
	}

	buf.Reset()
	if label != "" {
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_LABEL, label)
		url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_LABEL_TO_CARD, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, card_id, token)
		client.Post(url, "application/json", strings.NewReader(temp_body))
	}
	if label == "BPM" {

		t := time.Now()
		t = t.Round(time.Hour)
		if t.Weekday() == time.Friday {
			t = t.AddDate(0, 0, 1)
		}
		for t.Weekday() != time.Friday {
			t = t.AddDate(0, 0, 1)
		}

		tfs_card_params = append(tfs_card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_TAGS,
			From:   nil,
			Value:  "BPM"})

		tfs_card_params = append(tfs_card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_DUE_DATE,
			From:   nil,
			Value:  t.Format("01/02/2006")})

		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_DUE_DATE, t.Format("2006-01-02"), t.Format("2006-01-02 15:04"), t.Format("2006-01-02T15:04"))
		url = fmt.Sprintf(RESTYA_API_URL_PUT_ADD_DUE_DATE, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, card_id, token)
		req, _ := http.NewRequest("PUT", url, strings.NewReader(temp_body))
		req.Header.Set("Content-Type", "application/json")
		client.Do(req)
	}

	if description != "" {

		description = strings.Replace(description, "\r\n", strs.SPACE, -1)
		description = strings.Replace(description, strs.DOUBLE_QUOTES, strs.QUOTATION, -1)
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_DESCRIPTION, description)
		url = fmt.Sprintf(RESTYA_API_URL_PUT_ADD_ACTIONS_TO_CARD, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, card_id, token)
		req, _ := http.NewRequest("PUT", url, strings.NewReader(temp_body))
		req.Header.Set("Content-Type", "application/json")
		fmt.Println(url)
		client.Do(req)

		if re.MatchString(description) {
			hyperlink := re.FindAllStringSubmatch(description, -1)[0][1]
			description = re.ReplaceAllString(description, fmt.Sprintf(tc.TFS_TEXT_PTR_HYPERLINK, hyperlink, hyperlink, "Перейти к задаче"))
		}

		tfs_card_params = append(tfs_card_params, tc.TFS_req_struct{
			Action: tc.TFS_ACTIONS_ADD,
			Field:  tc.TFS_FIELDS_DESCRIPTION,
			From:   nil,
			Value:  description})
	}
	if user_id != 0 {
		temp_body = fmt.Sprintf(JSON_BODY_PTR_ADD_MEMBER, card_id, user_id)
		url = fmt.Sprintf(RESTYA_API_URL_POST_ADD_MEMBER_TO_CARD, set.Api_data.Restya_api_domain, set.Board.Id, set.Board.Sd_baclog_list_id, card_id, user_id, token)
		resp, _ = client.Post(url, "application/json", strings.NewReader(temp_body))
		buf.ReadFrom(resp.Body)
		if DEBUG {
			fmt.Println(buf.String())
		}
	}
	tfs_client.Add_card(tfs_card_params)

}

func get_token() string {

	var JSON_BODY_REQ_TOKEN string = `{"email": "` + set.Api_data.Api_user.Login + `","password": "` + set.Api_data.Api_user.Password + `"}`

	var access_token string = ""
	var objmap map[string]*json.RawMessage
	var tmp_json_val []byte
	var resp *http.Response
	var temp_url string

	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_OAUTH_TOKEN, set.Api_data.Restya_api_domain)
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

	temp_url = fmt.Sprintf(RESTYA_API_URL_GET_LOGIN_TOKEN, set.Api_data.Restya_api_domain, access_token)
	resp, _ = client.Post(temp_url, "application/json", strings.NewReader(JSON_BODY_REQ_TOKEN))

	buf.ReadFrom(resp.Body)
	json.Unmarshal(buf.Bytes(), &objmap)
	buf.Reset()
	tmp_json_val, _ = json.Marshal(objmap["access_token"])
	access_token = string(tmp_json_val)
	if DEBUG {
		fmt.Println(access_token)
	}
	return strings.Trim(access_token, strs.DOUBLE_QUOTES)

}
