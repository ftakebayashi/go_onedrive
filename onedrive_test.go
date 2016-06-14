package onedrive

import (
	"testing"
)

func TestCreateAccessToken(t *testing.T) {

	d := NewOneDrive()
	d.CreateAccessToken()

	if d.Access_token == "" {
		t.Error("アクセストークンが取得できません")
	}
}

func TestCreateUploadSession(t *testing.T) {

	d := NewOneDrive()
	d.CreateAccessToken()
	d.CreateUploadSession("786CFA0A1AA1452E!3036", "dummy")

	if d.Upload_url == "" {
		t.Error("アップロードセッションの作成に失敗しました")
	}

}
