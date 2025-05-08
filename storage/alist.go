package storage

import (
	"errors"
	"net/url"
	"qlist/config"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type AlistUploader struct {
}

var (
	Host     = ""
	Username = ""
	Password = ""
)

func Init() {
	Host = config.Instance.Alist.Host
	Username = config.Instance.Alist.Username
	Password = config.Instance.Alist.Password
}

func (a *AlistUploader) GetToken() (string, error) {
	resp, err := resty.New().R().SetBody(map[string]interface{}{
		"username": Username,
		"password": Password,
	}).Post(Host + "/api/auth/login")

	if err != nil {
		return "", err
	}
	return gjson.Get(resp.String(), "data.token").String(), nil
}

func (a *AlistUploader) PutObject(key string, data []byte, contentType string) (string, error) {
	token, err := a.GetToken()
	if err != nil {
		return "", err
	}
	resp, err := resty.New().R().
		SetHeader("File-Path", url.PathEscape(key)).
		SetHeader("Content-Type", contentType).
		SetHeader("Authorization", token).
		SetBody(data).Put(Host + "/api/fs/put")
	if err != nil {
		return "", err
	}

	if gjson.Get(resp.String(), "code").Int() != 200 {
		return "", errors.New(resp.String())
	}

	return key, nil
}
func (a *AlistUploader) CopyImage(originUrl string) (string, error) {
	return "", nil
}

func (a *AlistUploader) GetDownloadUrl(key string) (string, error) {
	token, err := a.GetToken()
	if err != nil {
		return "", err
	}
	resp, err := resty.New().R().
		SetHeader("Authorization", token).
		Get(Host + "/api/fs/get?path=" + url.PathEscape(key))
	if err != nil {
		return "", err
	}

	if gjson.Get(resp.String(), "code").Int() != 200 {
		return "", errors.New(resp.String())
	}

	return gjson.Get(resp.String(), "data.raw_url").String(), nil
}
