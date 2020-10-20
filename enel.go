package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/knq/chromedp/kb"
)

var (
	configPath string
	user       userData
	timeout    int
)

type flow struct {
	c       context.Context
	user    userData
	invoice invoice
}

type invoice struct {
	dueDate string
	value   string
	barCode string
	status  string
}

type userData struct {
	email string
	pw    string
	name  string
}

func init() {
	flag.StringVar(&configPath, "user-data", "", "Sets the path for the user data JSON file")
	flag.IntVar(&timeout, "timeout", 30, "Sets the flow timeout in seconds")
}

func main() {
	flag.Parse()
	flow, cancel, err := newFlow(false)
	if err != nil {
		log.Printf("failed to create new flow: %v", err)
	}

	for i := range cancel {
		defer cancel[i]()
	}

	err = flow.login()
	if err != nil {
		log.Println(err)
		return
	}

	err = flow.invoiceList()
	if err != nil {
		log.Println(err)
		return
	}
}

func (flow *flow) login() error {
	log.Println("Starting login flow")
	name := ""
	err := chromedp.Run(flow.c,
		chromedp.Navigate(`https://portalhome.eneldistribuicaosp.com.br/#/login`),
		chromedp.WaitVisible(`h1.title`),
		chromedp.Sleep(2*time.Second),
		chromedp.Click(`#email`, chromedp.NodeVisible, chromedp.ByID),
		chromedp.SendKeys("#email", kb.End+flow.user.email, chromedp.ByID),
		chromedp.Click(`#senha`, chromedp.NodeVisible, chromedp.ByID),
		chromedp.SendKeys("#senha", kb.End+flow.user.pw, chromedp.ByID),
		chromedp.Click(`#btnLoginEmail`, chromedp.NodeVisible, chromedp.ByID),
		chromedp.WaitVisible(`i.aes-sair`),
		chromedp.Text(
			`document.querySelector("#troca-instalacao > div.user-data.layout-align-start-center.layout-column.flex-none > label.name > strong")`,
			&name,
			chromedp.ByJSPath,
		),
	)
	if strings.ToLower(name) != strings.ToLower(flow.user.name) {
		return fmt.Errorf("Login failure; user name did not match")
	}
	if err == nil {
		log.Println("Successfully logged in")
	}
	return err
}

func (flow *flow) invoiceList() error {
	log.Println("Starting invoiceList flow")
	table := ""
	err := chromedp.Run(flow.c,
		chromedp.Navigate("https://portalhome.eneldistribuicaosp.com.br/#/segunda-via"),
		chromedp.WaitVisible("i.aes-sair"),
		chromedp.WaitVisible("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list"),
		chromedp.Text(`document.querySelector("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list > md-list-item:nth-child(2)")`, &table, chromedp.ByJSPath),
	)
	if err != nil {
		return fmt.Errorf("invoiceFlow err: %v", err)
	}
	if strings.Contains(table, "Pendente") {
		detailHeader := ""
		err = chromedp.Run(flow.c,
			chromedp.Click(
				`document.querySelector("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list > md-list-item:nth-child(2) > div.action-group > div.item.act1.action.act-enable > span")`,
				chromedp.NodeVisible,
				chromedp.ByJSPath,
			),
			chromedp.WaitVisible(`#detalhamento > aes-content-header > div > div > h3`),
			chromedp.Text(
				`document.querySelector("#detalhamento > aes-content-header > div > div > h3")`,
				&detailHeader,
				chromedp.ByJSPath,
			),
		)
		log.Printf("detailheader %v", detailHeader)
		if err != nil {
			return fmt.Errorf("click details err: %v", err)
		}
		if detailHeader != "Detalhamento de conta" {
			return fmt.Errorf("missing header; loaded the wrong page")
		}
	}
	if err == nil {
		log.Println("Successfully selected the last listed invoice")
	}
	return nil
}

func (flow *flow) invoiceData() {

}

func newFlow(headless bool) (flow, []context.CancelFunc, error) {
	ctx, cancel := setContext(headless)
	user, err := setUser()
	return flow{c: ctx, user: user}, cancel, err
}

func setUser() (userData, error) {
	user, err := readFile()
	if err != nil {
		return user, fmt.Errorf("failed to read json file: %v", err)
	}
	if len(user.email) == 0 {
		return user, fmt.Errorf("invalid user email; cannot be empty")
	}
	if len(user.pw) == 0 {
		return user, fmt.Errorf("invalid user password; cannot be empty")
	}
	if len(user.name) == 0 {
		return user, fmt.Errorf("invalid user name; cannot be empty")
	}
	return user, nil
}

func readFile() (userData, error) {
	if len(configPath) == 0 {
		return userData{}, fmt.Errorf("invalid path; cannot be empty %s", configPath)
	}
	jsonFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return userData{}, err
	}
	user := map[string]string{}
	err = json.Unmarshal(jsonFile, &user)
	return userData{
		email: user["email"],
		pw:    user["pw"],
		name:  user["name"],
	}, err
}

func setContext(headless bool) (context.Context, []context.CancelFunc) {
	outputFunc := []context.CancelFunc{}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		// Set the headless flag to false to display the browser window
		chromedp.Flag("headless", headless),
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(int64(timeout))*time.Second)
	outputFunc = append(outputFunc, cancel)
	ctx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	outputFunc = append(outputFunc, cancel)
	ctx, cancel = chromedp.NewContext(ctx)
	outputFunc = append(outputFunc, cancel)
	return ctx, outputFunc
}

func (flow *flow) textByPath(path string) (string, error) {
	output := ""
	err := chromedp.Run(flow.c,
		chromedp.Text(
			path,
			&output,
			chromedp.ByJSPath,
		),
	)
	if err != nil {
		return "", fmt.Errorf("flow.textByPath err: %v", err)
	}
	return output, nil
}

func (flow *flow) textByID(id string) (string, error) {
	output := ""
	err := chromedp.Run(flow.c,
		chromedp.Text(
			id,
			&output,
			chromedp.ByID,
		),
	)
	if err != nil {
		return "", fmt.Errorf("flow.textByID err: %v", err)
	}
	return output, nil
}

func (flow *flow) waitVisible(something string) error {
	log.Printf("waiting for %v", something)
	return chromedp.Run(flow.c,
		chromedp.WaitVisible(something),
	)
}
