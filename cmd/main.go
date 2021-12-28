package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	serviceLambda "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/twizar/common/pkg/client"
	"github.com/twizar/tourneys/internal/application/service"
	"github.com/twizar/tourneys/internal/ports"
	"github.com/twizar/tourneys/internal/ports/converter"
)

const (
	envVarLambdaName                         = "TEAMS_LAMBDA_NAME"
	envVarLambdaEndpoint                     = "TEAMS_LAMBDA_ENDPOINT"
	envVarLambdaRegion                       = "TEAMS_LAMBDA_REGION"
	envVarHTTPHeaderAccessControlAllowOrigin = "HTTP_HEADER_ACCESS_CONTROL_ALLOW_ORIGIN"
)

func main() {
	lambdaName, lambdaEndpoint, lambdaRegion, accessControlAllowOrigin := requiredParams()

	lambdaTeamsClient := configureTeamsClient(lambdaName, lambdaEndpoint, lambdaRegion)
	tourneyGenerator := service.NewTourneyGenerator(lambdaTeamsClient)
	dtoConverter := converter.NewConverter(lambdaTeamsClient)

	server := ports.NewHTTPServer(tourneyGenerator, dtoConverter)
	r := ports.ConfigureRouter(server)
	adapter := gorillamux.New(r)
	handler := ports.NewLambdaHandler(adapter, accessControlAllowOrigin)
	lambda.Start(handler.Handle)
}

func requiredParams() (lambdaName, lambdaEndpoint, lambdaRegion, accessControlAllowOrigin string) {
	var varExists bool

	if lambdaName, varExists = os.LookupEnv(envVarLambdaName); !varExists {
		log.Panicf("reqiured env var `%s` doesn't exist", envVarLambdaName)
	}

	if lambdaEndpoint, varExists = os.LookupEnv(envVarLambdaEndpoint); !varExists {
		log.Panicf("reqiured env var `%s` doesn't exist", envVarLambdaEndpoint)
	}

	if lambdaRegion, varExists = os.LookupEnv(envVarLambdaRegion); !varExists {
		log.Panicf("reqiured env var `%s` doesn't exist", envVarLambdaRegion)
	}

	if accessControlAllowOrigin, varExists = os.LookupEnv(envVarHTTPHeaderAccessControlAllowOrigin); !varExists {
		log.Panicf("reqiured env var `%s` doesn't exist", envVarHTTPHeaderAccessControlAllowOrigin)
	}

	return
}

func configureTeamsClient(lambdaName, lambdaEndpoint, lambdaRegion string) *client.AWSLambdaTeams {
	conf := aws.NewConfig()
	conf.Region = aws.String(lambdaRegion)
	conf.Endpoint = aws.String(lambdaEndpoint)
	lambdaClient := serviceLambda.New(session.Must(session.NewSession()), conf)

	return client.NewAWSLambdaTeams(lambdaClient, lambdaName)
}
