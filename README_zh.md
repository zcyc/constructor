# Constructor

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

一个强大的 Go 代码生成器，可以为结构体创建构造函数代码，支持多种设计模式。

[English](./README.md) | 简体中文

## 特性

- 🚀 **多种构造函数模式**：生成全参数构造函数、建造者模式或函数式选项模式
- 🔧 **灵活配置**：使用各种标志自定义输出
- 🏷️ **字段标签**：使用 `constructor:"-"`、`constructor:"getter:false"`、`constructor:"setter:false"`、`constructor:"default=..."` 和 `constructor:"required"` 标签进行细粒度控制
- 🎯 **初始化支持**：在构造后调用初始化方法
- 📦 **值或指针**：根据需要返回值或指针
- 🔍 **Getter 生成**：自动为私有字段生成 getter 方法
- 🛠️ **导入管理**：使用 Go 标准库自动处理导入

## 安装

```bash
go install github.com/zcyc/constructor@latest
```

或从 [releases 页面](https://github.com/zcyc/constructor/releases) 下载预构建的二进制文件。

## 快速开始

### 1. 在结构体上添加 go:generate 注释

```go
package mypackage

import "time"

//go:generate constructor -type=User -constructor-types=all-args,builder,options
type User struct {
	id        int
	name      string
	email     string
	createdAt time.Time
	metadata  map[string]string `newc:"-"` // 跳过此字段
}
```

### 2. 运行 go generate

```bash
go generate ./...
```

### 3. 使用生成的构造函数

```go
// 全参数构造函数
user1 := NewUser(1, "Alice", "alice@example.com", time.Now())

// 建造者模式
user2 := NewUserBuilder().
    Id(1).
    Name("Bob").
    Email("bob@example.com").
    CreatedAt(time.Now()).
    Build()

// 函数式选项模式
user3 := NewUserWithOptions(
    WithId(1),
    WithName("Charlie"),
    WithEmail("charlie@example.com"),
    WithCreatedAt(time.Now()),
)
```

## 构造函数模式

### 全参数构造函数

生成一个接受所有字段作为参数的简单构造函数。

```go
//go:generate constructor -type=Config -constructor-types=all-args
type Config struct {
    host string
    port int
}
```

**生成的代码：**

```go
func NewConfig(host string, port int) *Config {
    return &Config{
        host: host,
        port: port,
    }
}
```

### 建造者模式

生成一个带有流畅 setter 方法的建造者结构体。

```go
//go:generate constructor -type=Service -constructor-types=builder -setter-prefix=With
type Service struct {
    db     *sql.DB
    cache  Cache
    logger Logger
}
```

**生成的代码：**

```go
type ServiceBuilder struct {
    db     *sql.DB
    cache  Cache
    logger Logger
}

func NewServiceBuilder() *ServiceBuilder {
    return &ServiceBuilder{}
}

func (b *ServiceBuilder) WithDb(db *sql.DB) *ServiceBuilder {
    b.db = db
    return b
}

// ... 其他 setter ...

func (b *ServiceBuilder) Build() *Service {
    return &Service{
        db:     b.db,
        cache:  b.cache,
        logger: b.logger,
    }
}
```

### 函数式选项模式

生成用于灵活配置的选项函数。

```go
//go:generate constructor -type=Server -constructor-types=options
type Server struct {
    host    string
    port    int
    timeout time.Duration
}
```

**生成的代码：**

```go
type ServerOption func(*Server)

func WithHost(host string) ServerOption {
    return func(s *Server) {
        s.host = host
    }
}

func WithPort(port int) ServerOption {
    return func(s *Server) {
        s.port = port
    }
}

func WithTimeout(timeout time.Duration) ServerOption {
    return func(s *Server) {
        s.timeout = timeout
    }
}

func NewServerWithOptions(opts ...ServerOption) *Server {
    v := &Server{}
    for _, opt := range opts {
        opt(v)
    }
    return v
}
```

### 泛型结构体

泛型结构体的类型参数会同步保留到所有生成模式：

```go
//go:generate constructor -type=Box -constructor-types=all-args,builder,options
type Box[T any] struct {
    value T
}
```

会生成 `NewBox[T any](value T) *Box[T]`、`BoxBuilder[T]` 和 `BoxOption[T]`。

对于泛型 options，Go 只能推断出现在 option 参数中的类型参数。如果某个类型参数只出现在其他字段中，需要显式传入类型参数，或改用 builder 模式。

## 命令行选项

```bash
constructor [flags]
```

### 标志

| 标志                  | 描述                | 默认值             | 示例                                          |
|---------------------|-------------------|-----------------|---------------------------------------------|
| `-type`             | **[必需]** 结构体类型名称  | -               | `-type=User`                                |
| `-input`            | 包含结构体的 Go 源文件     | 自动查找          | `-input=user.go`                            |
| `-constructor-types` | 逗号分隔的模式列表         | `all-args`      | `-constructor-types=all-args,builder,options` |
| `-output`           | 输出文件路径            | `<type>_gen.go` | `-output=constructors.go`                   |
| `-init`             | 构造后调用的初始化方法名称     | -               | `-init=initialize`                          |
| `-init-returns-error` | 将初始化方法错误返回给调用方   | `false`         | `-init-returns-error`                         |
| `-return-value`      | 返回值而不是指针          | `false`         | `-return-value`                              |
| `-setter-prefix`     | 建造者 setter 方法的前缀  | -               | `-setter-prefix=With`                        |
| `-getters`           | 为私有字段生成 getter 方法 | `false`         | `-getters`                                   |
| `-dry-run`          | 校验生成结果但不写入文件   | `false`         | `-dry-run`                                  |
| `-stdout`           | 将生成代码输出到标准输出   | `false`         | `-stdout`                                   |
| `-validate-only`    | 只校验已有生成文件         | `false`         | `-validate-only`                            |
| `-version`          | 显示版本信息            | -               | `-version`                                  |

## 高级用法

### 初始化函数

在构造后调用初始化方法：

```go
//go:generate constructor -type=Service -constructor-types=all-args -init=initialize
type Service struct {
    db     *sql.DB
    logger Logger
}

func (s *Service) initialize() {
    s.logger.Info("Service initialized")
}
```

如果初始化可能失败，让方法返回 `error`，并启用 `-init-returns-error`。此时所有生成的构造函数都会返回 `(*Service, error)`（配合 `-return-value` 时返回 `(Service, error)`）：

```go
//go:generate constructor -type=Service -constructor-types=all-args -init=initialize -init-returns-error

func (s *Service) initialize() error {
    return nil
}
```

**生成：**

```go
func NewService(db *sql.DB, logger Logger) *Service {
    v := &Service{
        db:     db,
        logger: logger,
    }
    v.initialize()
    return v
}
```

### 返回值而不是指针

```go
//go:generate constructor -type=Config -constructor-types=all-args -return-value
type Config struct {
    debug bool
    port  int
}
```

### 默认值和必填字段

默认值使用 Go 表达式，`required` 用于禁止字段保持零值：

```go
type Server struct {
    name    string        `constructor:"required"`
    port    int           `constructor:"default=8080"`
    timeout time.Duration `constructor:"default=time.Second"`
}
```

带默认值的字段会从 all-args 参数列表中移除，并由三种模式自动初始化。必填字段会让生成的构造函数返回 `(*Server, error)`（配合 `-return-value` 时返回 `(Server, error)`）。

`-dry-run` 用于只校验不写文件，`-stdout` 用于管道输出代码，`-validate-only` 用于校验已有生成文件。

**生成：**

```go
func NewConfig(debug bool, port int) Config {
    return Config{
        debug: debug,
        port:  port,
    }
}
```

### 生成 Getter 方法

```go
//go:generate constructor -type=Repository -constructor-types=all-args -getters
type Repository struct {
    tableName string
    db        *sql.DB
}
```

**生成：**

```go
func NewRepository(tableName string, db *sql.DB) *Repository {
    return &Repository{
        tableName: tableName,
        db:        db,
    }
}

func (r *Repository) GetTableName() string {
    return r.tableName
}

func (r *Repository) GetDb() *sql.DB {
    return r.db
}
```

### 使用标签控制字段

GoConstructor 支持使用结构体标签对字段行为进行细粒度控制：

#### 完全跳过

完全跳过字段（不作为构造函数参数，也不生成 getter）：

```go
//go:generate constructor -type=User -constructor-types=all-args -getters
type User struct {
    name     string
    email    string
    internal string `constructor:"-"` // 完全跳过
}
```

**生成：**

```go
func NewUser(name string, email string) *User {
    return &User{
        name:  name,
        email: email,
    }
}

func (u *User) GetName() string {
    return u.name
}

func (u *User) GetEmail() string {
    return u.email
}
// 没有 GetInternal() 方法
```

**向后兼容性：** 也支持 `newc:"-"` 和 `gonstructor:"-"` 标签。

#### 仅跳过 Getter

在构造函数中包含字段但不生成 getter：

```go
//go:generate constructor -type=Product -constructor-types=all-args -getters
type Product struct {
    id          int
    name        string
    password    string `constructor:"getter:false"` // 在构造函数中，但没有 getter
}
```

**生成：**

```go
func NewProduct(id int, name string, password string) *Product {
    return &Product{
        id:       id,
        name:     name,
        password: password,
    }
}

func (p *Product) GetId() int {
    return p.id
}

func (p *Product) GetName() string {
    return p.name
}
// 没有 GetPassword() 方法（安全敏感）
```

#### 仅跳过 Setter

生成 getter 但不包含在构造函数中：

```go
//go:generate constructor -type=Service -constructor-types=builder -getters
type Service struct {
    host        string
    port        int
    connCount   int `constructor:"setter:false"` // 有 getter，但不在构造函数中
}
```

**生成：**

```go
type ServiceBuilder struct {
    host string
    port int
    // 没有 connCount 字段
}

func (b *ServiceBuilder) Host(host string) *ServiceBuilder { ... }
func (b *ServiceBuilder) Port(port int) *ServiceBuilder { ... }
// 没有 ConnCount() setter 方法

func (s *Service) GetHost() string { return s.host }
func (s *Service) GetPort() int { return s.port }
func (s *Service) GetConnCount() int { return s.connCount } // Getter 存在
```

这对于内部管理但需要对外读取的字段很有用。

### 带前缀的建造者

```go
//go:generate constructor -type=Client -constructor-types=builder -setter-prefix=With
type Client struct {
    host string
    port int
}
```

**生成的 setter 方法：**

```go
func (b *ClientBuilder) WithHost(host string) *ClientBuilder { ... }
func (b *ClientBuilder) WithPort(port int) *ClientBuilder { ... }
```

### 一次生成多种模式

在单个文件中生成多种构造函数模式：

```go
//go:generate constructor -type=User -constructor-types=all-args,builder,options
type User struct {
    name  string
    email string
}
```

这将生成所有三种模式：`NewUser()`、`UserBuilder` 和 `NewUserWithOptions()`。

## 无需安装即可使用

对于团队协作，您可以在不手动安装的情况下运行生成器：

```go
//go:generate go run github.com/zcyc/constructor@latest -type=User -constructor-types=all-args
type User struct {
    name string
}
```

这在 CI/CD 管道中以及与未安装该工具的团队成员协作时特别有用。

## 与类似工具的比较

### vs gonstructor

- ✅ **额外模式**：支持函数式选项模式
- ✅ **更清晰的选项**：更直观的标志名称
- ✅ **更好的默认值**：常见用例的合理默认值
- ✅ **兼容性**：支持 `gonstructor:"-"` 标签以便迁移

### vs newc

- ✅ **更多模式**：建造者模式和函数式选项模式
- ✅ **更多选项**：Setter 前缀、getter 生成等
- ✅ **更好的文档**：全面的示例和使用指南
- ✅ **兼容性**：支持 `newc:"-"` 标签以便迁移

## 示例

查看 [examples](./examples) 目录获取按模式组织的完整工作示例：

### 全参数模式

- `examples/allargs/user.go` - 基本全参数构造函数与字段跳过
- `examples/allargs/product.go` - 全参数与 getter 控制

### 建造者模式

- `examples/builder/service.go` - 带初始化函数和 setter 前缀的建造者
- `examples/builder/database.go` - 带细粒度 getter/setter 控制的建造者

### 选项模式

- `examples/options/config.go` - 带返回值的函数式选项
- `examples/options/server.go` - 带 getter/setter 控制的选项

### 混合模式

- `examples/mixed/repository.go` - 一个结构体中的所有三种模式

## 贡献

欢迎贡献！请随时提交 Pull Request。

1. Fork 此仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -am 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开一个 Pull Request

## 许可证

此项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [gonstructor](https://github.com/moznion/gonstructor) - 建造者模式的灵感来源
- [newc](https://github.com/Bin-Huang/newc) - 清晰 API 设计的灵感来源
- Go 团队提供的优秀 AST 解析工具

## 支持

如果您遇到任何问题或有疑问：

1. 查看 [examples](./examples) 目录
2. 搜索现有的 [issues](https://github.com/zcyc/constructor/issues)
3. 创建一个包含详细信息的新 issue

---

**用 ❤️ 为 Go 社区制作**
