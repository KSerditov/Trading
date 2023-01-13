package brokerclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/KSerditov/Trading/pkg/broker/orders"
)

type BrokerClientHttp struct {
	BrokerBaseURL string
}

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (b *BrokerClientHttp) setAuth(userid string, req *http.Request) {
	creds := fmt.Sprintf("%v", userid)
	encreds := base64.StdEncoding.EncodeToString([]byte(creds))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %v", encreds))
	req.Header.Add("Content-Type", "application/json")
}

func (b *BrokerClientHttp) Register(userid string) error {
	url := fmt.Sprintf("%v/api/v1/register", b.BrokerBaseURL)

	lf := &LoginForm{
		Username: userid,
		Password: "", // do not use for oauth
	}
	post, _ := json.Marshal(lf)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(post))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	return nil
}

func (b *BrokerClientHttp) History(ticker string, userid string) (*orders.HistoryResponse, error) {
	if ticker == "" {
		return nil, errors.New("please provide ticker name")
	}

	url := fmt.Sprintf("%v/api/v1/history", b.BrokerBaseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	b.setAuth(userid, req)

	q := req.URL.Query()
	q.Add("ticker", ticker)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	history := &orders.HistoryResponse{}
	err1 := json.Unmarshal(body, history)
	if err1 != nil {
		return nil, err1
	}

	return history, nil
}

func (b *BrokerClientHttp) Positions(userid string) (*orders.StatusResponse, error) {
	url := fmt.Sprintf("%v/api/v1/status", b.BrokerBaseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	b.setAuth(userid, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	status := &orders.StatusResponse{}
	err1 := json.Unmarshal(body, status)
	if err1 != nil {
		return nil, err1
	}

	return status, nil
}

func (b *BrokerClientHttp) Deal(deal *orders.Deal, userid string) (*orders.DealIdResponse, error) {
	url := fmt.Sprintf("%v/api/v1/deal", b.BrokerBaseURL)
	post, _ := json.Marshal(deal)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(post))
	if err != nil {
		return nil, err
	}
	b.setAuth(userid, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	dealid := &orders.DealIdResponse{}
	err1 := json.Unmarshal(body, dealid)
	if err1 != nil {
		return nil, err1
	}

	return dealid, nil
}

func (b *BrokerClientHttp) Cancel(dealid int64, userid string) (bool, error) {
	url := fmt.Sprintf("%v/api/v1/cancel", b.BrokerBaseURL)
	d := &orders.DealId{
		Id: dealid,
	}
	post, _ := json.Marshal(d)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(post))
	if err != nil {
		return false, err
	}
	b.setAuth(userid, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	cancel := &orders.CancelResponse{}
	err1 := json.Unmarshal(body, cancel)
	if err1 != nil {
		return false, err1
	}
	if cancel.Body.Status != "Success" {
		return false, nil
	}

	return true, nil
}
