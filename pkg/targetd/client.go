package targetd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Client struct {
	endpoint string
	username string
	password string
}

type Options struct {
	Insecure bool
	Address  string
	Port     int
	Username string
	Password string
}

func New(o *Options) *Client {
	scheme := "https"; if o.Insecure { scheme = "http" }
	endpoint := fmt.Sprintf("%v://%v:%v/targetrpc", scheme, o.Address, o.Port)

	return &Client{
		endpoint: endpoint,
		username: o.Username,
		password: o.Password,
	}
}

func (c *Client) do(method string, parameters, result interface{}) error {
	reqData, err := json.Marshal(&Request{
		Version:    "2.0",
		ID:         0,
		Method:     method,
		Parameters: parameters,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(reqData))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}

	if result != nil {
		var resmsg Response
		if err := json.NewDecoder(res.Body).Decode(&resmsg); err != nil {
			return err
		}

		if err := json.Unmarshal(resmsg.Result, result); err != nil {
			return err
		}
	}

	return nil
}
