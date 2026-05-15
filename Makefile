container_runtime := $(shell which podman || which docker)

golint:
	golangci-lint run -E gocritic -v ./...

swagger:
	swag init -g main.go --parseDependency --parseInternal -o ./docs --outputTypes go

up: down
	${container_runtime} compose up --build -d

down:
	${container_runtime} compose down