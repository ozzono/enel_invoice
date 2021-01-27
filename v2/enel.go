package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Flow struct {
	cookies []*http.Cookie
	User    User
	Invoice Invoice
}

// User has the required data to login
type User struct {
	Canal    string `json:"I_CANAL"`
	Email    string `json:"I_EMAIL"`
	Password string `json:"I_PASSWORD"`
}

// Invoice contains the needed payment data
type Invoice struct {
	Value   float64
	DueDate string
	BarCode string
}

// NewFlow creates a new Enel invoice flow
func NewFlow(email, password string) Flow {
	return Flow{User: User{Email: email, Password: password}}
}

func main() {
	flow := NewFlow("huvirgilio@gmail.com", "F52q7u4d2")
	bearer, err := flow.login()
	if err != nil {
		log.Printf("login err: %v", err)
		return
	}
	log.Printf("Bearer: %s", bearer)
}

func (f *Flow) login() (string, error) {
	f.User.Canal = "ZINT"
	url := "https://portalhome.eneldistribuicaosp.com.br/api/firebase/login"

	payload, err := json.Marshal(f.User)
	if err != nil {
		return "", fmt.Errorf("json.Marshal err: %v", err)
	}

	headers := map[string]string{
		"User-Agent":   "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0",
		"Accept":       "application/json, text/plain, */*",
		"Content-Type": "application/json;charset=utf-8",
	}

	res, err := request(url, "POST", headers, strings.NewReader(string(payload)), nil)
	if err != nil {
		return "", fmt.Errorf("request err: %v", err)
	}
	defer res.Body.Close()
	f.cookies = res.Cookies()

	data, err := parseBody(res)
	if len(fmt.Sprint(data["E_MSG"])) != 0 {
		return "", fmt.Errorf("E_MSG: %v", data["E_MSG"])
	}
	return fmt.Sprint(data["token"]), nil
}

func request(url, method string, headers map[string]string, payload *strings.Reader, cookies []*http.Cookie) (*http.Response, error) {
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}
	for key := range headers {
		req.Header.Add(key, headers[key])
	}
	for i := range cookies {
		req.AddCookie(cookies[i])
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func parseBody(res *http.Response) (map[string]interface{}, error) {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll err: %v", err)
	}
	data := make(map[string]interface{})
	json.Unmarshal(body, &data)
	return data, nil
}
