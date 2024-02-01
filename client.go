package async_request

import (
	"crypto/tls"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct{}

type IRequest interface {
	// Validate 验证参数
	Validate() error
	// GetUrl 获取url
	GetUrl() string
	// GetUri 设置请求的uri
	GetUri() string
	// GetBody 获取body
	GetBody() io.Reader
	// GetHeader 获取header
	GetHeader() map[string]string
	// GetMethod 获取method
	GetMethod() string
	// GetTimeout 获取超时时间
	GetTimeout() int
}

type IResponse interface {
	IsOk() bool
	GetCode() int
	GetMessage() string
	Decode(v []byte) error
}

type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (r *Response) IsOk() bool {
	return r.Code == http.StatusOK || r.Code == 0
}

func (r *Response) GetCode() int {
	return r.Code
}

func (r *Response) GetMessage() string {
	return r.Message
}

func (r *Response) Decode(v []byte) error {
	var res Response
	if err := json.Unmarshal(v, &res); err != nil {
		return errors.Wrapf(err, "decode response error: %s", string(v))
	}
	*r = res
	return nil
}

func (r *Client) Do(httpReq IRequest, httpResp IResponse) error {
	if err := httpReq.Validate(); err != nil {
		return errors.Wrap(err, "invalid request parameter")
	}
	url := r.buildApiUrl(httpReq)
	req, err := http.NewRequest(httpReq.GetMethod(), url, httpReq.GetBody())
	if err != nil {
		return errors.Wrap(err, "build request error")
	}
	for hk, hv := range httpReq.GetHeader() {
		req.Header.Set(hk, hv)
	}
	cli := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: time.Duration(httpReq.GetTimeout()) * time.Second,
	}
	response, err := cli.Do(req)
	if err != nil {
		return errors.Wrap(err, "request error")
	}
	defer func() { _ = response.Body.Close() }()

	// 如果数据较大的话，这么读取会导致内存突刺
	respBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return errors.Wrapf(err, "read response error")
	}

	if err = httpResp.Decode(respBytes); err != nil {
		return errors.Wrapf(err, "unmarshal response error")
	}
	if !httpResp.IsOk() {
		return errors.Errorf("invalid response: %d, message: %s", httpResp.GetCode(), httpResp.GetMessage())
	}
	return nil
}

func (r *Client) buildApiUrl(req IRequest) string {
	return strings.Trim(req.GetUrl(), "/") + "/" + strings.Trim(req.GetUri(), "/")
}

func Do(req IRequest, v IResponse) error {
	c := Client{}
	return c.Do(req, v)
}
