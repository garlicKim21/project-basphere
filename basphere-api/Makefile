.PHONY: build run clean test dev install

BINARY_NAME=basphere-api
BUILD_DIR=./build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# 빌드
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/basphere-api

# Linux용 빌드 (크로스 컴파일)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/basphere-api

# 개발 모드 실행 (mock provisioner 사용)
dev:
	@echo "Running in development mode..."
	go run ./cmd/basphere-api --dev --port 8080

# 프로덕션 실행
run:
	@echo "Running $(BINARY_NAME)..."
	go run ./cmd/basphere-api

# 테스트
test:
	go test -v ./...

# 의존성 정리
tidy:
	go mod tidy
	go mod download

# 클린
clean:
	rm -rf $(BUILD_DIR)

# 설치 (Bastion 서버용)
install: build-linux
	@echo "Installing $(BINARY_NAME)..."
	@if [ "$$(id -u)" -ne 0 ]; then echo "Must run as root"; exit 1; fi
	cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 /usr/local/bin/$(BINARY_NAME)
	chmod +x /usr/local/bin/$(BINARY_NAME)
	mkdir -p /var/lib/basphere/api/templates
	cp -r ./web/templates/* /var/lib/basphere/api/templates/
	@if [ ! -f /etc/basphere/api.yaml ]; then \
		cp ./config/api.yaml.example /etc/basphere/api.yaml; \
	fi
	@echo "Installation complete!"
	@echo "Start with: basphere-api"
