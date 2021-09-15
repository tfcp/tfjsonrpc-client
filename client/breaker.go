package client

import (
	"fmt"
	"github.com/gogf/gf/os/gcache"
	"math/rand"
	"net/http"
	"time"
)

// breaker default config

var (
	// breaker cacheKey
	breakPrefix = "jsonrpcBreak#%s#%s"
	// limit dry-run config
	limitCanDryRunValue = 50
	// JsonRpc break times setting(default 500)
	jsonRpcBreakTimes = 500
	// JsonRpc break opened status (if breakStatus==999999)
	jsonRpcOpenedStatus = 999999
	// breaker key timeout
	//jsonRpcBreakerExpire = 5 * 60 * time.Second
	jsonRpcBreakerExpire = 5 * 60 * time.Second
	// dryRun percent (default 1% request can pass breaker)
	jsonRpcDryRunPercent = 99
)

func JsonRpcBreakGet(url, method string) (bool, bool) {
	breakTimes := getBreakValue(url, method)
	if breakTimes == jsonRpcBreakTimes {
		// set jsonrpc break status opened
		setBreakValue(url, method, jsonRpcOpenedStatus)
		return false, false
	}
	if breakTimes > jsonRpcBreakTimes {
		if true == limiting() {
			// break opened we try dry-run 1%
			// dry-run success
			return true, true
		}
		// dry-run fail
		return false, false
	}
	// normal breaker
	return true, false
}

func JsonRpcDryRun(url, method string, isDryRun bool, jsonRes *http.Response) {
	if isDryRun {
		// dry run
		if jsonRes.StatusCode == 200 {
			// dry run success can close break
			setBreakValue(url, method, 0)
		}
	}
	return
}

func JsonRpcBreakSet(url, method string, jsonRes *http.Response) bool {
	breakValue := getBreakValue(url, method)
	if jsonRes.StatusCode > 500 && breakValue < jsonRpcBreakTimes {
		// if jsonrpc response code 500 break
		setBreakValue(url, method, breakValue+1)
		return true
	}
	return true
}

func getBreakValue(url, method string) int {
	cacheKey := fmt.Sprintf(breakPrefix, url, method)
	breakTimes, _ := gcache.Get(cacheKey)
	if breakTimes == nil {
		gcache.Set(cacheKey, 0, jsonRpcBreakerExpire)
		return 0
	}
	return breakTimes.(int)
}

func setBreakValue(url, method string, breakValue int) {
	cacheKey := fmt.Sprintf(breakPrefix, url, method)
	gcache.Set(cacheKey, breakValue, jsonRpcBreakerExpire)
}

func limiting() bool {
	rand.Seed(time.Now().Unix())
	res := rand.Intn(jsonRpcDryRunPercent)
	if res == limitCanDryRunValue {
		// let us to do dry run
		return true
	}
	// unlucky do not to dry run
	return false
}
