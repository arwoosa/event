package pkg

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var ExpirePageToken = errors.New("token expired")

type PageToken string
type Page struct {
	NextID        string `json:"next_id"`
	NextTimeAtUTC int64  `json:"next_time_at_utc"`
}

func (p Page) Encode() (PageToken, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return PageToken(base64.StdEncoding.EncodeToString(b)), nil
}

func (t PageToken) Decode() (*Page, error) {
	var p Page
	if len(t) == 0 {
		return nil, nil
	}

	bytes, err := base64.StdEncoding.DecodeString(string(t))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &p)
	if err != nil {
		return nil, err
	}

	//check if the token is expired
	if p.NextTimeAtUTC < time.Now().Add(-24*time.Hour).Unix() {
		err = ExpirePageToken
		return nil, err
	}

	return &p, nil
}

func GeneratePageToken(id string) (p PageToken, err error) {
	p, err = Page{NextID: id, NextTimeAtUTC: time.Now().Unix()}.Encode()
	return
}
