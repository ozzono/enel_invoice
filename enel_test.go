package enel

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"

	enel "github.com/ozzono/enel_invoice"
)

func Test(t *testing.T) {
	f := enel.NewFlow(true)
	config, err := readFile("config_test.json")
	if err != nil {
		t.Fatal(err)
	}
	f.User = enel.UserData{
		Email: config["Email"],
		Pw:    config["Pw"],
		Name:  config["Name"],
	}
	invoiceData, err := f.InvoiceFlow()
	if err != nil {
		log.Printf("f.InvoiceFlow err: %v", err)
		t.Fatal(err)
	}
	if len(invoiceData.BarCode) == 0 {
		t.Fatal("invalid invoice BarCode; cannot be empty")
	}
	if len(invoiceData.DueDate) == 0 {
		t.Fatal("invalid invoice DueDate; cannot be empty")
	}
	if len(invoiceData.Value) == 0 {
		t.Fatal("invalid invoice Value; cannot be empty")
	}
	if len(invoiceData.Status) == 0 {
		t.Fatal("invalid invoice Status; cannot be empty")
	}
	t.Logf("%#v", invoiceData)
}

func readFile(filename string) (map[string]string, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return map[string]string{}, err
	}
	output := map[string]string{}
	err = json.Unmarshal(file, &output)
	if err != nil {
		return map[string]string{}, err
	}
	return output, nil
}
