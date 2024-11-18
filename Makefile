# 定义变量
APP_NAME=minidocker
BUILD_DIR=build
GO=go
LDFLAGS="-s -w"  # 链接时去掉符号信息以减小可执行文件

# 默认目标，生成可执行文件
all: build

# 生成可执行文件
build: $(APP_NAME)

$(APP_NAME): main.go
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) main.go

# 运行 Go 测试
test:
	$(GO) test ./...

# 清理生成的文件
clean:
	rm -rf $(BUILD_DIR)

# 运行程序
run:
	$(BUILD_DIR)/$(APP_NAME)

# 格式化代码
fmt:
	$(GO) fmt ./...

# 打包为 Linux 目标平台
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-linux main.go

# 打包为 Windows 目标平台
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME).exe main.go
