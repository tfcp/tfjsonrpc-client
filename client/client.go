package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	errs "github.com/pkg/errors"
	"github.com/tfcp/tfgo-breaker/breaker"
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

type JsonClientReq struct {
	ctx         context.Context
	url, method string
	params      interface{}
}

type JsonRpcClient struct {
}

const (
	// JsonRpc timeout setting
	jsonRpcTimeout = 5 * time.Second
	// JsonRpcDefaultVersion is the default request version for JsonRpc.
	JsonRpcDefaultVersion = "2.0"
)

var (
	requestRpc    *sync.Pool
	responseRpc   *sync.Pool
	breakerErrMsg = "the breaker condition is reached"
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
func (this *JsonRpcClient) Call(ctx context.Context, url, method string, params interface{}) (*JsonRpcResponse, error) {
	cacheKey := fmt.Sprintf("%s#%s", url, method)
	// create breaker (default threshold:500 breakerCacheExpired:5min dryRunPercent: 1/100)
	breakerConf := breaker.NewBreakConf(cacheKey, 500,
		5*60*time.Second,
		10,
		this.normalJsonRpcFunc,
		this.breakJsonRpcFunc)
	breakerJsonRpc := breaker.NewBreaker(breakerConf)
	// jsonRpc request
	jsonReq := &JsonClientReq{
		ctx:    ctx,
		url:    url,
		method: method,
		params: params,
	}
	res, err := breakerJsonRpc.Run(jsonReq)
	return res.(*JsonRpcResponse), err
}

// breaker normal logic
func (this *JsonRpcClient) normalJsonRpcFunc(req interface{}) (interface{}, error, bool) {
	breakerReq := req.(*JsonClientReq)
	return this.jsonRpcBase(breakerReq.ctx, breakerReq.url, breakerReq.method, breakerReq.params)
}

// breaker opened logic
func (this *JsonRpcClient) breakJsonRpcFunc(req interface{}) (interface{}, error) {
	breakerRes := responseRpc.Get().(*JsonRpcResponse)
	defer func() {
		responseRpc.Put(breakerRes)
	}()
	breakerRes.Result = "breakerIsOpened"
	//err := errors.New("breaker is opened...")
	return breakerRes, nil
}

// JsonRpc base function
func (this *JsonRpcClient) jsonRpcBase(ctx context.Context, url, method string, params interface{}) (*JsonRpcResponse, error, bool) {
	jsonRpcRes := responseRpc.Get().(*JsonRpcResponse)
	jsonRpcReq := requestRpc.Get().(JsonRpcRequest)
	defer func() {
		requestRpc.Put(jsonRpcReq)
		responseRpc.Put(jsonRpcRes)
	}()
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
		// breaker condition reached, breaker value add 1
		// when httpStatus > 500 too many times, the breaker will open
		if rs.StatusCode > 500 {
			err := errors.New(breakerErrMsg)
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
		ch <- resJR
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			// reached breaker condition
			if res.err.Error() == breakerErrMsg {
				return nil, res.err, true
			}
			return nil, res.err, false
		}
		jsonRpcRes = res.jsonRpcRes
		return jsonRpcRes, nil, false
	case <-time.After(jsonRpcTimeout):
		// timeout watch
		errMsg := fmt.Sprintf("jsonrpc timeout: url:%v, method:%v,params:%v", url, method, params)
		return nil, errs.New(errMsg), false
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
