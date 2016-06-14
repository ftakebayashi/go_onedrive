package onedrive

import (
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type OneDrive struct {
	ApiUrl       string
	AuthUrl      string
	TokenUrl     string
	ClientSecret string
	RefreshToken string
	ClientId     string
	AccessToken  string
	UploadUrl    string
}

// NewOneDrive comment
func NewOneDrive() *OneDrive {

	viper.SetConfigName("onedrive")
	viper.AddConfigPath("$GOPATH/conf")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("設定ファイルの読み込みに失敗しました")
	}

	d := &OneDrive{
		viper.GetString("api.api_url"),
		viper.GetString("api.auth_url"),
		viper.GetString("api.token_url"),
		viper.GetString("api.client_secret"),
		viper.GetString("api.refresh_token"),
		viper.GetString("api.client_id"),
		"",
		"",
	}
	d.CreateAccessToken()

	return d

}

func (d *OneDrive) CreateUploadSession(itemId string, fileName string) {

	client := &http.Client{Timeout: time.Duration(10) * time.Second}
	path := "drive/items/" + itemId + ":/" + fileName + ":/upload.createSession"

	req, err := http.NewRequest("POST", d.ApiUrl+path, nil)
	req.Header.Add("Authorization", "bearer "+d.AccessToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Uploadセッション作成のAPI通信に失敗しました")
	}

	defer resp.Body.Close()

	body, _ := jason.NewObjectFromReader(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("%v\n", body)
		log.Fatal("Uploadセッションの作成が拒否されました")
	}

	url, _ := body.GetString("uploadUrl")
	d.UploadUrl = url

}

func (d *OneDrive) CreateAccessToken() {

	client := &http.Client{Timeout: time.Duration(10) * time.Second}

	values := url.Values{}
	values.Add("client_id", d.ClientId)
	values.Add("client_secret", d.ClientSecret)
	values.Add("refresh_token", d.RefreshToken)
	values.Add("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", viper.GetString("api.token_url"), strings.NewReader(values.Encode()))
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("AccessTokenの作成に失敗しました")
	}

	defer resp.Body.Close()

	body, _ := jason.NewObjectFromReader(resp.Body)

	token, _ := body.GetString("access_token")
	d.AccessToken = token
}

func (d *OneDrive) ResumableUpload(start int64, length int64, data string) {

	client := &http.Client{}

	req, err := http.NewRequest("PUT", d.UploadUrl, strings.NewReader(data))
	req.Header.Add("Authorization", "bearer "+d.AccessToken)
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
	d.CreateUploadSession(itemId, fileName)
	d.UploadSession(filePath)
}

func (d *OneDrive) get(url string) *jason.Object {

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "bearer "+d.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%#v\n", resp.Body)
		log.Fatal("API GET に失敗しました。 \n", err)
	}

	defer resp.Body.Close()

	v, _ := jason.NewObjectFromReader(resp.Body)
	return v
}

func (d *OneDrive) GetDrive() {

	url := viper.GetString("api.api_url") + "drive"
	v := d.get(url)
	fmt.Printf("%v\n", v)

}

func (d *OneDrive) GetSharedFiles() {

	url := viper.GetString("api.api_url") + "drive/items/root?expand=children"
	v := d.get(url)
	fmt.Printf("%v\n", v)

}
