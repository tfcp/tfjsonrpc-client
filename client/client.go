package client

import (
	"context"
	"encoding/json"
	"fmt"
	errs "github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type JsonRpcRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int64       `json:"id"`
}

type JsonRpcResponse struct {
	Version string        `json:"jsonrpc"`
	Error   *JsonRpcError `json:"error,omitempty"`
	Result  interface{}   `json:"result"`
	Id      int64         `json:"id"`
}

type JsonRpcError struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

const (
	// JsonRpc timeout setting
	jsonRpcTimeout = 5 * time.Second
	// JsonRpcDefaultVersion is the default request version for JsonRpc.
	JsonRpcDefaultVersion = "2.0"
)

var (
	requestRpc  *sync.Pool
	responseRpc *sync.Pool
)

func init() {
	// obj init
	requestRpc = &sync.Pool{
		New: func() interface{} {
			return JsonRpcRequest{
				Version: JsonRpcDefaultVersion,
				Method:  "",
				Params:  nil,
				Id:      1,
			}
		},
	}
	// obj init
	responseRpc = &sync.Pool{
		New: func() interface{} {
			return &JsonRpcResponse{
				Version: JsonRpcDefaultVersion,
				Error:   nil,
				Result:  nil,
				Id:      1,
			}
		},
	}
}

// JsonRpc do the request using JsonRpc protocol.
func JsonRpc(ctx context.Context, url, method string, params interface{}) (*JsonRpcResponse, error) {
	jsonRpcRes := responseRpc.Get().(*JsonRpcResponse)
	jsonRpcReq := requestRpc.Get().(JsonRpcRequest)
	jsonRpcReq.Method = method
	jsonRpcReq.Params = params
	defer func() {
		responseRpc.Put(jsonRpcRes)
		requestRpc.Put(jsonRpcReq)
	}()
	type resJsonRpc struct {
		jsonRpcRes *JsonRpcResponse
		err        error
	}
	ch := make(chan *resJsonRpc, 0)
	go func() {
		resJR := &resJsonRpc{
			jsonRpcRes: nil,
			err:        nil,
		}
		// break logic
		isBreak, isDryRun := JsonRpcBreakGet(url, method)
		if !isBreak {
			resJR.jsonRpcRes = new(JsonRpcResponse)
			resJR.jsonRpcRes.Result = ""
			ch <- resJR
			return
		}

		rq, err := json.Marshal(jsonRpcReq)
		if err != nil {
			resJR.err = err
			ch <- resJR
			return
		}
		rs, err := requestHttp(url, string(rq))
		if err != nil {
			resJR.err = err
			ch <- resJR
			return
		}
		defer rs.Body.Close()
		c, err := ioutil.ReadAll(rs.Body)
		if err != nil {
			resJR.err = err
			ch <- resJR
			return
		}
		err = json.Unmarshal(c, &resJR.jsonRpcRes)
		if err != nil {
			resJR.err = err
			ch <- resJR
			return
		}
		// break value set
		JsonRpcBreakSet(url, method, rs)
		// dry run breaker can close
		JsonRpcDryRun(url, method, isDryRun, rs)
		ch <- resJR
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		jsonRpcRes = res.jsonRpcRes
		return jsonRpcRes, nil
	case <-time.After(jsonRpcTimeout):
		// timeout watch
		errMsg := fmt.Sprintf("jsonrpc timeout: url:%v, method:%v,params:%v", url, method, params)
		return nil, errs.New(errMsg)
	}
}

func requestHttp(url, data string) (resp *http.Response, err error) {
	resp, err = http.Post(url,
		"application/x-www-form-urlencoded",
		strings.NewReader(data))
	if err != nil {
		return
	}
	return
}
