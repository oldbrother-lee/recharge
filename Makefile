.PHONY: build-worker run-worker

# 构建工作器
build-worker:
	go build -o bin/recharge cmd/worker/main.go

build-server:
	go build -o bin/server cmd/main.go

build-notify:
	go build -o bin/notify cmd/notification/main.go

build-server-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/main.go

build-server-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/server.exe cmd/main.go

build-server-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/server-mac-arm64 cmd/main.go

build-worker-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/worker-linux cmd/worker/main.go

build-worker-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/worker.exe cmd/worker/main.go

build-worker-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/worker-mac-arm64 cmd/worker/main.go

build-notify-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/notify-linux cmd/notification/main.go

build-notify-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/notify.exe cmd/notification/main.go

build-notify-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/notify-mac-arm64 cmd/notification/main.go

build-getorder-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/getorder-linux cmd/task/main.go

build-getorder-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/task.exe cmd/task/main.go

build-task-mac-arm64:
# 运行工作器
run-worker:
	go run cmd/worker/main.go

all:
	make build-worker-linux
	make build-server-linux
	make build-notify-linux
	make build-getorder-linux
