# jsonrpc

## 1. intro
this is a funny client for jsonrpc. it can support timeout,breaker...

## 2. jsonrpc client Demo
```
	var (
		url    = "http://test.com/rpc/business"
		method = "testRpc"
		params = map[string]interface{}{
			"param1": 1,
		}
	)
	result, _ := JsonRpc(context.Background(),url, method, params)
	fmt.Println(result)
```

