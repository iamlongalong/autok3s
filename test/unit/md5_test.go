package unit

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestMd5(t *testing.T) {
	md5sum := md5.New()
	md5sum.Write([]byte("openit_longalong"))
	str := hex.EncodeToString(md5sum.Sum(nil))
	fmt.Println(str)
}
