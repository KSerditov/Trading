# note: call scripts from /scripts
.PHONY: install gen test

install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

gen: check-env
	protoc --version # 3.19.3
	protoc-gen-go --version # v1.27.1
	protoc-gen-go-grpc --version # 1.2.0
	protoc --go_out=. --go-grpc_out=. ./api/exchange/*.proto
	# in combination with 'option go_package = ".";' in service.proto this will generate files in this folder with package main
	#sed -i 's/package exchange/package exchangeapi/g' ./api/exchange/*.pb.go 

#test:
#	go test -v -race

check-env:
ifndef GOBIN
	$(error GOBIN is undefined, set GOBIN so protoc can see installed plugins in PATH)
endif

start-docker:
	docker compose -f ./deployments/docker-compose.yml up

start-exchange:
	go run ./cmd/exchange/main.go

start-broker:
	go run ./cmd/broker/main.go

start-client:
	go run ./cmd/tgclient/main.go

stop-docker:
	docker compose -f ./deployments/docker-compose.yml down

generate-swagger:
	swagger generate spec -o ./api/broker/swagger.json