package onedrive

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type OneDrive struct {
	Api_url       string
	Auth_url      string
	Token_url     string
	Client_secret string
	Refresh_token string
	Client_id     string
	Access_token  string
	Upload_url    string
}

type TokenResponse struct {
	Token_type    string
	Expires_in    int
	Scope         string
	Access_token  string
	Refresh_token string
	User_id       string
}

type SessionResponse struct {
	UploadUrl string
}

// NewOneDrive comment
func NewOneDrive() *OneDrive {

	viper.SetConfigName("onedrive")
	viper.AddConfigPath("$GOPATH/conf")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("設定ファイルの読み込みに失敗しました")
	}

	return &OneDrive{
		viper.GetString("api.api_url"),
		viper.GetString("api.auth_url"),
		viper.GetString("api.token_url"),
		viper.GetString("api.client_secret"),
		viper.GetString("api.refresh_token"),
		viper.GetString("api.client_id"),
		"",
		"",
	}

}

func (d *OneDrive) CreateUploadSession(itemId string, fileName string) string {

	client := &http.Client{Timeout: time.Duration(10) * time.Second}
	path := "drive/items/" + itemId + ":/" + fileName + ":/upload.createSession"

	req, err := http.NewRequest("POST", d.Api_url+path, nil)
	req.Header.Add("Authorization", "bearer "+d.Access_token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Uploadセッション作成のAPI通信に失敗しました")
		return ""
	}

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Printf("%s", b)
		log.Fatal("Uploadセッションの作成が拒否されました")
		return ""
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var r SessionResponse
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		log.Fatal("Uploadセッション作成時のJsonデコードができませんでした")
		return ""
	}

	d.Upload_url = r.UploadUrl
	return r.UploadUrl
}

func (d *OneDrive) CreateAccessToken() {

	client := &http.Client{Timeout: time.Duration(10) * time.Second}

	values := url.Values{}
	values.Add("client_id", d.Client_id)
	values.Add("client_secret", d.Client_secret)
	values.Add("refresh_token", d.Refresh_token)
	values.Add("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", viper.GetString("api.token_url"), strings.NewReader(values.Encode()))
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("AccessTokenの作成に失敗しました")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var r TokenResponse
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		log.Fatal("AccessToken作成時のJsonデコードができませんでした")
	}
	d.Access_token = r.Access_token
}

func (d *OneDrive) ResumableUpload(start int64, length int64, data string) {

	client := &http.Client{}

	req, err := http.NewRequest("PUT", d.Upload_url, strings.NewReader(data))
	req.Header.Add("Authorization", "bearer "+d.Access_token)
	req.Header.Add("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+int64(len(data)-1), length))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("bytes %d-%d/%d\n", start, start+int64(len(data)-1), length)
		log.Fatal("ファイルの分割アップロード中にエラーが発生しました \n", err)
	}

	defer resp.Body.Close()

	fmt.Printf("sending bytes: %d-%d/%d\n", start, start+int64(len(data)-1), length)

}

func (d *OneDrive) UploadSession(filePath string) (string, error) {

	fmt.Println("Upload start:", time.Now())
	fp, _ := os.Open(filePath)
	st, _ := os.Stat(filePath)

	var bytes int64 = 1024 * 1024 * 30
	data := make([]byte, bytes)

	var start int64 = 0
	var err error
	var count int

	for err == nil {
		count, err = fp.ReadAt(data, start)
		d.ResumableUpload(start, st.Size(), string(data[:count]))
		start += bytes
	}

	fmt.Println("Upload end:", time.Now())
	return string(data), nil
}

func (d *OneDrive) Upload(itemId string, fileName string, filePath string) {
	d.CreateAccessToken()
	d.CreateUploadSession(itemId, fileName)
	d.UploadSession(filePath)
}
