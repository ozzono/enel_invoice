package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
	c    context.Context
	user userData
}

type userData struct {
	email string
	pw    string
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

	flow.login()

	// value := "teste"
	// dueDate := "string"
	// err = chromedp.Run(ctx,
	// 	chromedp.Navigate(`https://portalhome.eneldistribuicaosp.com.br/#/segunda-via`),
	// 	chromedp.WaitVisible(`div.page-info`),
	// 	chromedp.Text(`document.querySelector("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list > md-list-item > div.item.header1 > span")`, &value, chromedp.ByJSPath),
	// 	chromedp.Text(`document.querySelector("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list > md-list-item > div.item.header4.vencimento > span")`, &dueDate, chromedp.ByJSPath),
	// 	chromedp.Click(`document.querySelector("#segunda-via > div.aes-section.less-padding > div:nth-child(2) > div.faturas-list-container > md-list > md-list-item > div.action-group > div.item.act1.action.act-enable > span")`, chromedp.NodeVisible, chromedp.ByJSPath),
	// 	chromedp.Sleep(10*time.Second),
	// 	chromedp.Stop(),
	// )
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// log.Printf("value %q", value)
	// log.Printf("dueDate %q", dueDate)
}

func (flow *flow) login() error {
	log.Println("Starting login flow")
	output := ""
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
		chromedp.Text(`document.querySelector("#troca-instalacao > div.user-data.layout-align-start-center.layout-column.flex-none > label.text")`, &output, chromedp.ByJSPath),
	)
	log.Println(output)
	return err
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
	return userData{email: user["email"], pw: user["pw"]}, err
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
