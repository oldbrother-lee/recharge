.PHONY: build-server build-notification build-recharge build-worker build-task build-migrate run-server run-notification run-recharge run-worker run-task run-migrate

# 本地构建目标
build-server:
	go build -o bin/server cmd/server/main.go

build-notification:
	go build -o bin/notification cmd/notification/main.go

build-recharge:
	go build -o bin/recharge cmd/recharge/main.go

build-worker:
	go build -o bin/worker cmd/recharge/worker/main.go

build-task:
	go build -o bin/task cmd/task/main.go

build-migrate:
	go build -o bin/migrate cmd/migrate/main.go

# Linux 构建目标
build-server-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/server cmd/server/main.go

build-notification-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/notification cmd/notification/main.go

build-recharge-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/recharge cmd/recharge/main.go

build-worker-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/worker cmd/recharge/worker/main.go

build-task-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/task cmd/task/main.go

build-migrate-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/migrate cmd/migrate/main.go

# Windows 构建目标
build-server-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/server.exe cmd/server/main.go

build-notification-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/notification.exe cmd/notification/main.go

build-recharge-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/recharge.exe cmd/recharge/main.go

build-worker-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/worker.exe cmd/recharge/worker/main.go

build-task-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/task.exe cmd/task/main.go

build-migrate-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/migrate.exe cmd/migrate/main.go

# macOS ARM64 构建目标
build-server-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/server-mac-arm64 cmd/server/main.go

build-notification-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/notification-mac-arm64 cmd/notification/main.go

build-recharge-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/recharge-mac-arm64 cmd/recharge/main.go

build-worker-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/worker-mac-arm64 cmd/recharge/worker/main.go

build-task-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/task-mac-arm64 cmd/task/main.go

build-migrate-mac-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/migrate-mac-arm64 cmd/migrate/main.go

# 运行目标
run-server:
	go run cmd/server/main.go

run-notification:
	go run cmd/notification/main.go

run-recharge:
	go run cmd/recharge/main.go

run-worker:
	go run cmd/recharge/worker/main.go

run-task:
	go run cmd/task/main.go

run-migrate:
	go run cmd/migrate/main.go

# 批量构建目标
build-all:
	make build-server
	make build-notification
	make build-recharge
	make build-worker
	make build-task
	make build-migrate

build-all-linux:
	make build-server-linux
	make build-notification-linux
	make build-recharge-linux
	make build-worker-linux
	make build-task-linux
	make build-migrate-linux

build-all-windows:
	make build-server-windows
	make build-notification-windows
	make build-recharge-windows
	make build-worker-windows
	make build-task-windows
	make build-migrate-windows

build-all-mac-arm64:
	make build-server-mac-arm64
	make build-notification-mac-arm64
	make build-recharge-mac-arm64
	make build-worker-mac-arm64
	make build-task-mac-arm64
	make build-migrate-mac-arm64

# 清理构建文件
clean:
	rm -rf bin/

# 默认目标
all: build-all-linux
