package async_request

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"testing"
)

type UserInfoRequest struct{}

func (u UserInfoRequest) Validate() error {
	return nil
}

func (u UserInfoRequest) GetUrl() string {
	return "https://localhost:8080"
}

func (u UserInfoRequest) GetUri() string {
	return "/api/login/getUserInfo"
}

func (u UserInfoRequest) GetBody() io.Reader {
	return nil
}

func (u UserInfoRequest) GetHeader() map[string]string {
	return map[string]string{
		"Authorization": "2dca1b52-0a30-40d9-abfa-a59058af66b8",
	}
}

func (u UserInfoRequest) GetMethod() string {
	return http.MethodGet
}

func (u UserInfoRequest) GetTimeout() int {
	return 60
}

type UserInfoResponse struct {
	Code   int             `json:"code"`
	Msg    string          `json:"msg"`
	Result json.RawMessage `json:"result"`
}

func (u *UserInfoResponse) IsOk() bool {
	return u.Code == 0
}

func (u *UserInfoResponse) GetCode() int {
	return u.Code
}

func (u *UserInfoResponse) GetMessage() string {
	return u.Msg
}

func (u *UserInfoResponse) Decode(v []byte) error {
	var res UserInfoResponse
	if err := json.Unmarshal(v, &res); err != nil {
		return errors.Wrapf(err, "decode response error: %s", string(v))
	}
	*u = res
	return nil
}

func TestRequest(t *testing.T) {
	type args struct {
		ctx       context.Context
		requests  []IRequest
		responses []IResponse
		options   []AsyncRequestWorker
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "验证并发请求",
			args: args{
				ctx:       context.Background(),
				requests:  []IRequest{UserInfoRequest{}},
				responses: []IResponse{&UserInfoResponse{}},
				options:   []AsyncRequestWorker{WorkerOption(1)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Request(tt.args.ctx, tt.args.requests, tt.args.responses, tt.args.options...); (err != nil) != tt.wantErr {
				t.Errorf("Request() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
