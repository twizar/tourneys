GOLANG_CI_LINT_IMAGE=golangci/golangci-lint:latest-alpine
GOLANG_IMAGE=golang:1.17.2

go-build:
	GOOS=linux CGO_ENABLED=0 go build -o tourneys ./cmd/main.go

docker-build:
	docker build --no-cache -t lambda-tourneys:0.0.1 .

run:
	docker run --env-file local.env --rm -p 9000:8080 lambda-tourneys:0.0.1

up: go-build docker-build run

test-call:
	curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{ "path":"/tourneys", "httpMethod":"GET" }'

go-lint:
	docker run -v ${PWD}:/app -v ~/.gitconfig:/root/.gitconfig -w /app $(GOLANG_CI_LINT_IMAGE) \
		go env -w GOPRIVATE=github.com/twizar/common
		golangci-lint run -v --timeout 600m --fix --sort-results

go-test:
	docker run \
		--env-file=./test.env \
		-v ${PWD}:/app \
		-v ~/.gitconfig:/root/.gitconfig \
		-w /app $(GOLANG_IMAGE) \
		go env -w GOPRIVATE=github.com/twizar/common
		go test -race -cover -v -coverpkg=./... -coverprofile=cover.out ./...
		go tool cover -html=cover.out