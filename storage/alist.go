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

// GetFileList 获取指定目录下的文件列表
func (a *AlistUploader) GetFileList(path string) ([]map[string]interface{}, error) {
	token, err := a.GetToken()
	if err != nil {
		return nil, err
	}

	resp, err := resty.New().R().
		SetHeader("Authorization", token).
		Get(Host + "/api/fs/list?path=" + url.PathEscape(path))
	if err != nil {
		return nil, err
	}

	if gjson.Get(resp.String(), "code").Int() != 200 {
		return nil, errors.New(resp.String())
	}

	var files []map[string]interface{}
	gjson.Get(resp.String(), "data.content").ForEach(func(key, value gjson.Result) bool {
		file := make(map[string]interface{})
		file["name"] = value.Get("name").String()
		file["path"] = path + "/" + value.Get("name").String()
		file["size"] = value.Get("size").Int()
		file["is_dir"] = value.Get("is_dir").Bool()
		file["modified"] = value.Get("modified").String()
		files = append(files, file)
		return true
	})

	return files, nil
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
