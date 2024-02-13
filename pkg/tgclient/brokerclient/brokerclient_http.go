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

func (b *BrokerClientHttp) MakeRequest(
	userid string,
	url string,
	params map[string]string,
	bodyObj interface{},
	respObj interface{},
) error {
	url = fmt.Sprintf("%v%v", b.BrokerBaseURL, url)

	var req *http.Request
	var err error

	if bodyObj == nil {
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		q := req.URL.Query()
		if params != nil {
			for k, v := range params {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
		}
	} else {
		post, _ := json.Marshal(bodyObj)
		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(post))
		if err != nil {
			return err
		}
	}

	creds := fmt.Sprintf("%v", userid)
	encreds := base64.StdEncoding.EncodeToString([]byte(creds))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %v", encreds))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	msg := make(map[string]string, 1)
	err = json.Unmarshal(body, &msg)
	if err == nil {
		message, ok := msg["message"]
		if ok {
			return errors.New(message)
		}
	}
	fmt.Printf("RESPONSE: %v", string(body))
	err = json.Unmarshal(body, respObj)
	if err != nil {
		return err
	}
	return nil

}

func (b *BrokerClientHttp) Register(userid string) error {
	lf := &LoginForm{
		Username: userid,
		Password: "", // do not use for oauth
	}
	resp := make(map[string]interface{}, 1)

	err := b.MakeRequest(
		userid,
		"/api/v1/register",
		nil,
		lf,
		resp,
	)
	if err != nil {
		return err
	}

	return nil
}

func (b *BrokerClientHttp) History(ticker string, userid string) (*orders.HistoryResponse, error) {
	if ticker == "" {
		return nil, errors.New("please provide ticker name")
	}

	history := &orders.HistoryResponse{}
	err := b.MakeRequest(
		userid,
		"/api/v1/history",
		map[string]string{"ticker": ticker},
		nil,
		history,
	)
	if err != nil {
		return nil, err
	}

	return history, nil
}

func (b *BrokerClientHttp) Positions(userid string) (*orders.StatusResponse, error) {
	status := &orders.StatusResponse{}

	err := b.MakeRequest(
		userid,
		"/api/v1/status",
		nil,
		nil,
		status,
	)
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (b *BrokerClientHttp) Deal(deal *orders.Deal, userid string) (*orders.DealIdResponse, error) {
	dealid := &orders.DealIdResponse{}
	err := b.MakeRequest(
		userid,
		"/api/v1/deal",
		nil,
		deal,
		dealid,
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("DEALRESPONSE: %v", dealid)
	return dealid, nil
}

func (b *BrokerClientHttp) Cancel(dealid int64, userid string) (bool, error) {
	d := &orders.DealId{
		Id: dealid,
	}
	cancel := &orders.CancelResponse{}
	err := b.MakeRequest(
		userid,
		"/api/v1/deal",
		nil,
		d,
		cancel,
	)
	if err != nil {
		return false, err
	}
	if cancel.Body.Status != "Success" {
		return false, nil
	}

	return true, nil
}
