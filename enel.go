package enel

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/knq/chromedp/kb"
)

var (
	configPath string
	user       UserData
)

// Flow contains and the data and methods needed to crawl through the enel webpage
type Flow struct {
	c       context.Context
	user    UserData
	invoice Invoice
	cancel  []context.CancelFunc
}

//Invoice has all the invoice data needed for payment
type Invoice struct {
	dueDate string
	value   string
	barCode string
	status  string
}

//UserData has all the needed data to login
type UserData struct {
	email string
	pw    string
	name  string
}

func init() {
	flag.StringVar(&configPath, "user-data", "", "Sets the path for the user data JSON file")
}

//InvoiceFlow crawls through the enel page
func (flow *Flow) InvoiceFlow() (Invoice, error) {
	for i := range flow.cancel {
		defer flow.cancel[i]()
	}

	err := flow.login()
	if err != nil {
		log.Println(err)
		return Invoice{}, err
	}

	err = flow.invoiceList()
	if err != nil {
		log.Println(err)
		return Invoice{}, err
	}

	err = flow.invoiceData()
	if err != nil {
		log.Println(err)
		return Invoice{}, err
	}
	return flow.invoice, nil
}

func (flow *Flow) login() error {
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

func (flow *Flow) invoiceList() error {
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
		flow.invoice.status = "pending"
	}
	if strings.Contains(table, "Vencido") {
		flow.invoice.status = "overdue"
	}

	if flow.invoice.status == "pending" || flow.invoice.status == "overdue" {
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

func (flow *Flow) invoiceData() error {
	err := chromedp.Run(flow.c,
		chromedp.Text(
			`document.querySelector("#detalhamento > div.aes-section.conta-header > div > div:nth-child(1) > div > div.layout-align-center-end.layout-column.flex > span.value")`,
			&flow.invoice.value,
			chromedp.ByJSPath,
		),
		chromedp.Text(
			`document.querySelector("#detalhamento > div.aes-section.conta-header > div > div:nth-child(1) > div > div.layout-align-center-start.layout-column.flex > span.value")`,
			&flow.invoice.dueDate,
			chromedp.ByJSPath,
		),
		chromedp.Text(
			`document.querySelector("#detalhamento > div.aes-section.conta-header > div > div.row-conta-detalhes.flex-100 > div > div.box-codigo-barras.layout-align-center-stretch.layout-column.flex-gt-sm-20.flex-100 > div:nth-child(2) > div.codigo-barras.layout-align-center-center.layout-row > span")`,
			&flow.invoice.barCode,
			chromedp.ByJSPath,
		),
	)
	if err != nil {
		return fmt.Errorf("chromedp.Run err: %v", err)
	}
	flow.formatInvoice()
	if err == nil {
		log.Println("Successfully fetched invoice data")
		log.Printf("invoice: %v", flow.invoice)
	}
	return nil
}

//NewFlow creates a flow with context besides user and invoice data
func NewFlow(headless bool) Flow {
	ctx, cancel := setContext(headless)
	return Flow{c: ctx, user: user, cancel: cancel}
}

func setContext(headless bool) (context.Context, []context.CancelFunc) {
	outputFunc := []context.CancelFunc{}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		// Set the headless flag to false to display the browser window
		chromedp.Flag("headless", headless),
	)
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	outputFunc = append(outputFunc, cancel)
	ctx, cancel = chromedp.NewContext(ctx)
	outputFunc = append(outputFunc, cancel)
	return ctx, outputFunc
}

func (flow *Flow) textByPath(path string) (string, error) {
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

func (flow *Flow) textByID(id string) (string, error) {
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

func (flow *Flow) waitVisible(something string) error {
	log.Printf("waiting for %v", something)
	return chromedp.Run(flow.c,
		chromedp.WaitVisible(something),
	)
}

func (flow *Flow) formatInvoice() {
	flow.invoice.barCode = strings.Replace(flow.invoice.barCode, " ", "", -1)
	flow.invoice.value = strings.Replace(strings.TrimPrefix(flow.invoice.value, "R$"), ",", ".", -1)
	//Discuss: should flow.invoice.dueDate be parsed to something else? unchanged: dd/mm/yyyy
}
