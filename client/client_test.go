package client

import (
	"context"
	"testing"
)

func TestJsonRpc(t *testing.T) {
	var (
		url    = "http://test.com/rpc/user"
		method = "testRpc"
		params = map[string]interface{}{
			"param1": 1,
		}
	)
	result, err := JsonRpc(context.Background(), url, method, params)
	if err != nil {
		t.Fatal("jsonrpc request error:", err.Error())
		return
	}
	if result.Error != nil {
		t.Fatal("jsonrpc result error:", result.Error)
		return
	}
	if result.Result == nil {
		t.Fatal("jsonrpc result error: result is nil")
		return
	}
	t.Log("jsonrpc success:", result.Result)
}
