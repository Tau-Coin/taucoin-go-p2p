package utils

import (
	"fmt"
	"testing"
)

func TestByteToPath(t *testing.T){
	strData := ""
	strByte := []byte(strData)
	path, err:= ByteToPath(strByte)
	if err != nil{
		t.Fatalf(err)
	}
	t.Logf(path)
}
