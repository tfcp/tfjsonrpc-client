package client

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestJsonRpc(t *testing.T) {
	var (
		url    = "http://test.com/rpc/user"
		method = "test-rpc"
		params = map[string]interface{}{
			"param1": 1,
		}
	)
	jsonClient := &JsonRpcClient{}
	result, err := jsonClient.Call(context.Background(), url, method, params)
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

// test jsonrpc breaker
func TestBreakerJsonRpc(t *testing.T) {
	var (
		url    = "http://test.com/rpc/user"
		method = "test-rpc"
		params = map[string]interface{}{
			"param1": 1,
		}
	)
	jsonClient := &JsonRpcClient{}
	for i := 0; i < 100; i++ {
		result, err := jsonClient.Call(context.Background(), url, method, params)
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
		fmt.Println("times:", i)
		fmt.Println("result:", result.Result)
		time.Sleep(1 * time.Second)
	}

	t.Log("jsonrpc success")
}
