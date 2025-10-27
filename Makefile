.PHONY: all build install clean test examples verify help generate

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

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "✅ Dependencies installed"

# 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	@rm -f constructor
	@rm -f examples/*_gen.go
	@rm -rf examples/demo/go.mod examples/demo/go.sum
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
	@go build ./examples
	@echo "✅ Examples built successfully"

# 运行演示
demo: build examples
	@echo "Running demo..."
	@cd examples/demo && go run main.go

# 验证项目
verify: build
	@echo "Running verification..."
	@./verify.sh

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .
	@echo "✅ Code formatted"

# 代码检查
lint:
	@echo "Running linters..."
	@go vet ./...
	@echo "✅ Lint checks passed"

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
	@echo "  deps       - Install dependencies (goimports)"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  generate   - Generate example constructors"
	@echo "  examples   - Build examples"
	@echo "  demo       - Run the demo application"
	@echo "  verify     - Run verification script"
	@echo "  fmt        - Format code with gofmt and goimports"
	@echo "  lint       - Run code linters"
	@echo "  version    - Show version"
	@echo "  release    - Create release package"
	@echo "  help       - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build the project"
	@echo "  make install            # Install to GOPATH"
	@echo "  make demo               # Run demonstration"
	@echo "  make verify             # Full verification"
