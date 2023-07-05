package unit

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestMd5(t *testing.T) {
	md5sum := md5.New()
	md5sum.Write([]byte("openit_longalong"))
	str := hex.EncodeToString(md5sum.Sum(nil))
	fmt.Println(str)
}

func TestSplitProxy(t *testing.T) {
	proxy := "github.com:socks5://127.0.0.1:8080"

	slices := strings.SplitN(proxy, ":", 2)
	if len(slices) != 2 {
		t.Fatal("invalid proxy format")
	}
}
