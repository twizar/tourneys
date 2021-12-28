package ports

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

type LambdaHandler struct {
	adapter                            *gorillamux.GorillaMuxAdapter
	httpHeaderAccessControlAllowOrigin string
}

func NewLambdaHandler(adapter *gorillamux.GorillaMuxAdapter, httpHeaderAccessControlAllowOrigin string) *LambdaHandler {
	return &LambdaHandler{
		adapter:                            adapter,
		httpHeaderAccessControlAllowOrigin: httpHeaderAccessControlAllowOrigin,
	}
}

func (lh LambdaHandler) Handle(req *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	resp, err := lh.adapter.Proxy(*req)
	if err != nil {
		return nil, fmt.Errorf("lambda proxy error occurred: %w", err)
	}

	if len(resp.Headers) == 0 {
		resp.Headers = make(map[string]string, 1)
	}

	resp.Headers["Access-Control-Allow-Origin"] = lh.httpHeaderAccessControlAllowOrigin

	return &resp, nil
}
