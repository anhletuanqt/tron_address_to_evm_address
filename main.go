package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

type LogTransfer struct {
	Contract common.Address
	From     common.Address
	To       common.Address
	Tokens   *big.Int
}

type TronEventResult struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
}

type TronEventData struct {
	ContractAddress string          `json:"contract_address"`
	EventName       string          `json:"event_name"`
	Result          TronEventResult `json:"result"`
}

type TronEventResp struct {
	Data    []TronEventData `json:"data"`
	Success bool            `json:"success"`
}

func main() {
	url := "https://nile.trongrid.io/v1/transactions"
	txHash := "fbc0cd3350523de14d041521cc91dc348243632f96783f9d1d4481b3cfc0683a"
	amount, _ := new(big.Int).SetString("200000000000000000000000", 10) // 2000
	log, err := queryEventByTxHash(url, common.HexToHash(txHash), amount)
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(log)
}

func queryEventByTxHash(url string, txHash common.Hash, amount *big.Int) (*LogTransfer, error) {
	reqURL := fmt.Sprintf("%v/%v/events", url, txHash.Hex()[2:])
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var txResponse = TronEventResp{}
	if err := json.Unmarshal(body, &txResponse); err != nil {
		return nil, err
	}
	// rp := map[string]interface{}{}
	// if err := json.Unmarshal(body, &rp); err != nil {
	// 	return nil, err
	// }
	// spew.Dump(rp)
	// if !txResponse.Success {
	// 	return nil, errors.New("Tx_Fail")
	// }
	// get transfer event
	var transferEvent *TronEventData = nil
	for _, v := range txResponse.Data {
		if v.EventName == "Transfer" {
			transferEvent = &v
			break
		}
	}

	if transferEvent == nil {
		return nil, errors.New("Transfer_Not_Found")
	}
	tokens, ok := new(big.Int).SetString(transferEvent.Result.Value, 10)
	if !ok {
		return nil, errors.New("Parse_Amount_Fail")
	}
	if tokens.Cmp(amount) != 0 {
		return nil, errors.New("Invalid_Amount")
	}

	// TODO: Convert tron address to evm address
	ethAddress, err := convertETHAddress(transferEvent.ContractAddress)
	if err != nil {
		return nil, err
	}

	return &LogTransfer{
		Contract: ethAddress,
		From:     common.HexToAddress(transferEvent.Result.From),
		To:       common.HexToAddress(transferEvent.Result.To),
		Tokens:   tokens,
	}, nil
}

func convertETHAddress(tronAddress string) (common.Address, error) {
	validAddress, err := address.Base58ToAddress(tronAddress)
	if err != nil {
		return common.Address{}, err
	}
	a := &address.Address{}
	src := validAddress.Bytes()
	if err := a.Scan(src); err != nil {
		return common.Address{}, err
	}
	evmAddr := hexutil.Encode(src)
	addr := evmAddr[:2] + evmAddr[4:]
	return common.HexToAddress(addr), nil
}
