package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xssnick/tonutils-go/tlb"
)

const URI = "https://anton.tools/api/v0/messages?operation_id=0&src_workchain=0"

type Transfer struct {
	Hash         string    `json:"hash"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Ton          tlb.Coins `json:"ton"`
	Comment      string    `json:"comment"`
	CreatedLt    string    `json:"created_lt"`
	ContractType string    `json:"dst_contract"`
}

func (t *Transfer) Dump() {
	fmt.Printf("Hash: %s\n", t.Hash)
	fmt.Printf("From: %s\n", t.From)
	fmt.Printf("To: %s\n", t.To)
	fmt.Printf("Ton: %s\n", t.Ton.String())
	fmt.Printf("CreatedLt: %s\n", t.CreatedLt)
	fmt.Printf("ContractType: %s\n", t.ContractType)
	fmt.Printf("Comment: %s\n", t.Comment)

	fmt.Println("")
	fmt.Println("--------------------------------------------------")
	decoded, err := base64.StdEncoding.DecodeString(t.Hash)
	if err != nil {
		fmt.Println("error decoding hash:", err)
		return
	}
	fmt.Println("https://tonviewer.com/transaction/" + hex.EncodeToString(decoded))
}

func (t *Transfer) DumpIfAmountGt(ton tlb.Coins) {
	if t.Ton.Nano().Cmp(ton.Nano()) > 0 {
		t.Dump()
	}
}

var (
	min = flag.String("min", "10", "min ton")
)

var lastHash = ""

func fetchLatestTransfer(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func main() {
	flag.Parse()
	for {
		var query url.Values = url.Values{
			"limit": []string{"300"},
			"order": []string{"DESC"},
		}

		uri := URI + "&" + query.Encode()
		body, err := fetchLatestTransfer(uri)
		if err != nil {
			// log error
			continue
		}

		tx := []*Transfer{}
		gjson.Get(body, "results").ForEach(func(key, value gjson.Result) bool {
			t := &Transfer{}

			t.Hash = value.Get("hash").String()
			t.From = value.Get("src_address.base64").String()
			t.To = value.Get("dst_address.base64").String()
			coin, ok := new(big.Int).SetString(value.Get("amount").String(), 10)
			if !ok {
				return false
			}
			t.Ton = tlb.MustFromNano(coin, 9)
			t.CreatedLt = value.Get("created_lt").String()
			t.Comment = value.Get("transfer_comment").String()
			t.ContractType = value.Get("dst_contract").String()

			tx = append(tx, t)

			return true
		})

		for _, t := range tx {
			t.DumpIfAmountGt(tlb.MustFromTON(*min))

			if t.Hash == lastHash {
				break
			}
		}

		lastHash = tx[0].Hash
		time.Sleep(5 * time.Second)
	}
}
