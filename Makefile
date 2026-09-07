.PHONY: all build install clean test examples verify help generate fmt vet version release

# 默认目标
all: build

# 编译二进制文件
build:
	@echo "Building constructor..."
	@go build -o constructor -ldflags="-s -w" .
	@echo "✅ Build complete: ./constructor"

# 安装到 GOPATH/bin
install:
	@echo "Installing constructor..."
	@go install .
	@echo "✅ Installed to $(shell go env GOPATH)/bin/constructor"

# 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	@rm -f constructor
	@rm -rf dist
	@# Generated example sources are versioned and intentionally preserved.
	@echo "✅ Clean complete"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "✅ Tests passed"

# 生成示例代码
generate:
	@echo "Generating example constructors..."
	@cd examples && go generate ./...
	@echo "✅ Examples generated"

# 编译示例
examples: generate
	@echo "Building examples..."
	@go build ./examples/...
	@echo "✅ Examples built successfully"

# 验证项目
verify: build generate
	@echo "Running verification..."
	@go test -race ./...
	@go vet ./...
	@go build ./...
	@echo "✅ Verification passed"

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

# 代码检查
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✅ Vet checks passed"

# 显示版本
version:
	@./constructor -version

# 创建发布包
release: clean build
	@echo "Creating release package..."
	@mkdir -p dist
	@tar -czf dist/constructor-$(shell uname -s)-$(shell uname -m).tar.gz constructor README.md LICENSE
	@echo "✅ Release package created: dist/"

# 显示帮助信息
help:
	@echo "GoConstructor - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build      - Build the constructor binary"
	@echo "  install    - Install constructor to GOPATH/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  generate   - Generate example constructors"
	@echo "  examples   - Build examples"
	@echo "  verify     - Run race tests, vet, and build"
	@echo "  fmt        - Format code with gofmt"
	@echo "  vet        - Run go vet"
	@echo "  version    - Show version"
	@echo "  release    - Create release package"
	@echo "  help       - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build the project"
	@echo "  make install            # Install to GOPATH"
	@echo "  make verify             # Full verification"
