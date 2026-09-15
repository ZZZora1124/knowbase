package utils

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/imroc/req"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

var (
	DefaultRetry    = 2
	DefaultTimeout  = 3 * time.Second
	StatusCodeError = -1
)

const (
	ContentTypeJSON = "application/json; charset=utf-8"
	ContentTypeForm = "multipart/form-data"
)

type Request struct {
	RetryOnErrorCode bool
	URL              string
	Method           string
	Headers          req.Header
	Param            map[string]interface{}
	Body             []byte
	ContentType      string
	Timeout          time.Duration
	Retry            int
}

func (r *Request) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("URL is blank")
	}

	if r.Method == "" {
		return fmt.Errorf("method is blank")
	}

	if r.Headers == nil {
		r.Headers = make(map[string]string, 5)
	}

	if r.Param == nil {
		r.Param = make(map[string]interface{}, 5)
	}

	if r.ContentType != "" {
		r.Headers["Content-Type"] = r.ContentType
	}

	if r.Timeout < 0 {
		r.Timeout = DefaultTimeout
	}

	if r.Retry < 0 {
		r.Retry = DefaultRetry
	}

	return nil
}

func SendRequest(request *Request, res interface{}) ([]byte, int, error) {
	err := request.Validate()
	if err != nil {
		return nil, StatusCodeError, fmt.Errorf("request validate error: %w", err)
	}

	var resp *req.Resp

	queryParam := generateQueryParam(request.Param)

	if request.Timeout > 0 {
		req.SetTimeout(request.Timeout)
	}

	backOff := GetRetryBackOff()
	for i := 0; i < request.Retry+1; i++ {
		log.Logger.Infof("send request-[%s] start, params: %v, body: %s", request.URL, request.Param,
			string(request.Body))
		resp, err = req.Do(request.Method, request.URL, request.Headers, queryParam, request.Body)
		if err == nil {
			if request.RetryOnErrorCode {
				code := resp.Response().StatusCode
				if (code < 200 || code >= 300) && i < request.Retry {
					time.Sleep(backOff.NextBackOff())
					continue
				}
			}
			break
		} else {
			if i < request.Retry {
				time.Sleep(backOff.NextBackOff())
			}
		}
	}

	if err != nil {
		return nil, StatusCodeError, fmt.Errorf("send request-[%s] error: %w", request.URL, err)
	}

	if res != nil {
		err = resp.ToJSON(res)
		if err != nil {
			return resp.Bytes(), StatusCodeError, fmt.Errorf("parse response for URL-[%s] error: %w", request.URL, err)
		}
	}

	log.Logger.Infof("send request-[%s] success, statusCode: %v, response: %s", request.URL, resp.Response().StatusCode, resp.String())

	return resp.Bytes(), resp.Response().StatusCode, nil
}

func generateQueryParam(in map[string]interface{}) req.QueryParam {
	res := req.QueryParam{}
	if in == nil {
		return res
	}

	for k, v := range in {
		res[k] = v
	}

	return res
}

func GetRetryBackOff() backoff.BackOff {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.InitialInterval = time.Millisecond * 200
	exponentialBackOff.MaxInterval = time.Second * 3
	exponentialBackOff.MaxElapsedTime = time.Second * 16
	exponentialBackOff.Multiplier = 1.5
	exponentialBackOff.Reset()
	return exponentialBackOff
}
