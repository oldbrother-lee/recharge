package signature

import "testing"

func TestKayixinSign_DocExample(t *testing.T) {
	appID := "10000"
	secret := "652fbdd2f637606a2afbb2fe0ca72419"
	version := KayixinAPIVersion
	timestamp := "1760767397"
	body := `{"orderNumber":"20251016165825069","outerNumber":""}`
	want := "6a414e658265ff3ca665cee3b161c4e9"

	got := KayixinSign(appID, secret, version, timestamp, body)
	if got != want {
		t.Fatalf("KayixinSign() = %q, want %q", got, want)
	}
	if !KayixinVerify(appID, secret, version, timestamp, body, want) {
		t.Fatal("KayixinVerify should accept doc example signature")
	}
}
