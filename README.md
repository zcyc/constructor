# Constructor

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful Go code generator that creates constructor code for structs with support for multiple design patterns.

## Features

- 🚀 **Multiple Constructor Patterns**: Generate all args, builder, or functional options patterns
- 🔧 **Flexible Configuration**: Customize output with various flags
- 🏷️ **Field Tagging**: Fine-grained control with `constructor:"-"`, `constructor:"getter:false"`, and
  `constructor:"setter:false"` tags
- 🎯 **Initialization Support**: Call init methods after construction
- 📦 **Value or Pointer**: Return values or pointers based on your needs
- 🔍 **Getter Generation**: Automatically generate getter methods for private fields
- 🛠️ **Import Management**: Automatic import handling via `goimports`

## Installation

```bash
go install github.com/zcyc/constructor@latest
```

Or download pre-built binaries from the [releases page](https://github.com/zcyc/constructor/releases).

### Dependencies

This tool requires [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) for automatic import management:

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

## Quick Start

### 1. Add go:generate comment to your struct

```go
package mypackage

import "time"

//go:generate constructor -type=User -constructorTypes=allArgs,builder,options
type User struct {
    id        int
    name      string
    email     string
    createdAt time.Time
    metadata  map[string]string `newc:"-"` // Skip this field
}
```

### 2. Run go generate

```bash
go generate ./...
```

### 3. Use the generated constructors

```go
// All args constructor
user1 := NewUser(1, "Alice", "alice@example.com", time.Now())

// Builder pattern
user2 := NewUserBuilder().
    Id(1).
    Name("Bob").
    Email("bob@example.com").
    CreatedAt(time.Now()).
    Build()

// Functional options pattern
user3 := NewUserWithOptions(
    WithId(1),
    WithName("Charlie"),
    WithEmail("charlie@example.com"),
    WithCreatedAt(time.Now()),
)
```

## Constructor Patterns

### All Args Constructor

Generates a simple constructor that accepts all fields as parameters.

```go
//go:generate constructor -type=Config -constructorTypes=allArgs
type Config struct {
    host string
    port int
}
```

**Generated code:**

```go
func NewConfig(host string, port int) *Config {
    return &Config{
        host: host,
        port: port,
    }
}
```

### Builder Pattern

Generates a builder struct with fluent setter methods.

```go
//go:generate constructor -type=Service -constructorTypes=builder -setterPrefix=With
type Service struct {
    db     *sql.DB
    cache  Cache
    logger Logger
}
```

**Generated code:**

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

// ... other setters ...

func (b *ServiceBuilder) Build() *Service {
    return &Service{
        db:     b.db,
        cache:  b.cache,
        logger: b.logger,
    }
}
```

### Functional Options Pattern

Generates option functions for flexible configuration.

```go
//go:generate constructor -type=Server -constructorTypes=options
type Server struct {
    host    string
    port    int
    timeout time.Duration
}
```

**Generated code:**

```go
type ServerOption func (*Server)

func WithHost(host string) ServerOption {
    return func (s *Server) {
        s.host = host
    }
}

func WithPort(port int) ServerOption {
    return func (s *Server) {
        s.port = port
    }
}

func WithTimeout(timeout time.Duration) ServerOption {
    return func (s *Server) {
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

## CLI Options

```bash
constructor [flags]
```

### Flags

| Flag                | Description                                 | Default         | Example                                     |
|---------------------|---------------------------------------------|-----------------|---------------------------------------------|
| `-type`             | **[Required]** Struct type name             | -               | `-type=User`                                |
| `-constructorTypes` | Comma-separated list of patterns            | `allArgs`       | `-constructorTypes=allArgs,builder,options` |
| `-output`           | Output file path                            | `<type>_gen.go` | `-output=constructors.go`                   |
| `-init`             | Init method name to call after construction | -               | `-init=initialize`                          |
| `-returnValue`      | Return value instead of pointer             | `false`         | `-returnValue`                              |
| `-setterPrefix`     | Prefix for builder setter methods           | -               | `-setterPrefix=With`                        |
| `-withGetter`       | Generate getter methods for private fields  | `false`         | `-withGetter`                               |
| `-version`          | Show version information                    | -               | `-version`                                  |

## Advanced Usage

### Initialization Function

Call an initialization method after construction:

```go
//go:generate constructor -type=Service -constructorTypes=allArgs -init=initialize
type Service struct {
    db     *sql.DB
    logger Logger
}

func (s *Service) initialize() {
    s.logger.Info("Service initialized")
}
```

**Generated:**

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

### Return Value Instead of Pointer

```go
//go:generate constructor -type=Config -constructorTypes=allArgs -returnValue
type Config struct {
    debug bool
    port  int
}
```

**Generated:**

```go
func NewConfig(debug bool, port int) Config {
    return Config{
        debug: debug,
        port:  port,
    }
}
```

### Generate Getter Methods

```go
//go:generate constructor -type=Repository -constructorTypes=allArgs -withGetter
type Repository struct {
    tableName string
    db        *sql.DB
}
```

**Generated:**

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

### Field Control with Tags

GoConstructor supports fine-grained control over field behavior using struct tags:

#### Complete Skip

Skip fields entirely (no constructor parameter, no getter):

```go
//go:generate constructor -type=User -constructorTypes=allArgs -withGetter
type User struct {
    name     string
    email    string
    internal string `constructor:"-"` // Completely skipped
}
```

**Generated:**

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
// No GetInternal() method
```

**Backward Compatibility:** Also supports `newc:"-"` and `gonstructor:"-"` tags.

#### Skip Getter Only

Include field in constructor but don't generate getter:

```go
//go:generate constructor -type=Product -constructorTypes=allArgs -withGetter
type Product struct {
    id          int
    name        string
    password    string `constructor:"getter:false"` // In constructor, but no getter
}
```

**Generated:**

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
// No GetPassword() method (security sensitive)
```

#### Skip Setter Only

Generate getter but don't include in constructor:

```go
//go:generate constructor -type=Service -constructorTypes=builder -withGetter
type Service struct {
    host        string
    port        int
    connCount   int `constructor:"setter:false"` // Has getter, but not in constructor
}
```

**Generated:**

```go
type ServiceBuilder struct {
    host string
    port int
    // No connCount field
}

func (b *ServiceBuilder) Host(host string) *ServiceBuilder { ... }
func (b *ServiceBuilder) Port(port int) *ServiceBuilder { ... }
// No ConnCount() setter method

func (s *Service) GetHost() string { return s.host }
func (s *Service) GetPort() int { return s.port }
func (s *Service) GetConnCount() int { return s.connCount } // Getter exists
```

This is useful for fields that are managed internally but need to be read externally.

### Builder with Setter Prefix

```go
//go:generate constructor -type=Client -constructorTypes=builder -setterPrefix=With
type Client struct {
    host string
    port int
}
```

**Generated setter methods:**

```go
func (b *ClientBuilder) WithHost(host string) *ClientBuilder { ... }
func (b *ClientBuilder) WithPort(port int) *ClientBuilder { ... }
```

### Multiple Patterns at Once

Generate multiple constructor patterns in a single file:

```go
//go:generate constructor -type=User -constructorTypes=allArgs,builder,options
type User struct {
    name  string
    email string
}
```

This generates all three patterns: `NewUser()`, `UserBuilder`, and `NewUserWithOptions()`.

## Usage Without Installation

For team collaboration, you can run the generator without manual installation:

```go
//go:generate go run github.com/zcyc/constructor@latest -type=User -constructorTypes=allArgs
type User struct {
    name string
}
```

This is especially useful in CI/CD pipelines and when working with teams where not everyone has the tool installed.

## Comparison with Similar Tools

### vs gonstructor

- ✅ **Additional Pattern**: Functional options pattern support
- ✅ **Cleaner Options**: More intuitive flag names
- ✅ **Better Defaults**: Sensible defaults for common use cases
- ✅ **Compatibility**: Supports `gonstructor:"-"` tags for migration

### vs newc

- ✅ **More Patterns**: Builder and functional options patterns
- ✅ **More Options**: Setter prefix, getter generation, etc.
- ✅ **Better Documentation**: Comprehensive examples and usage guide
- ✅ **Compatibility**: Supports `newc:"-"` tags for migration

## Examples

Check the [examples](./examples) directory for complete working examples organized by pattern:

### All Args Pattern

- `examples/allargs/user.go` - Basic all-args constructor with field skipping
- `examples/allargs/product.go` - All-args with getter control

### Builder Pattern

- `examples/builder/service.go` - Builder with init function and setter prefix
- `examples/builder/database.go` - Builder with fine-grained getter/setter control

### Options Pattern

- `examples/options/config.go` - Functional options with return value
- `examples/options/server.go` - Options with getter/setter control

### Mixed Patterns

- `examples/mixed/repository.go` - All three patterns in one struct

Run the demo:

```bash
cd examples/demo
go run main.go
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [gonstructor](https://github.com/moznion/gonstructor) - Inspiration for builder pattern
- [newc](https://github.com/Bin-Huang/newc) - Inspiration for clean API design
- Go team for excellent AST parsing tools

## Support

If you encounter any issues or have questions:

1. Check the [examples](./examples) directory
2. Search existing [issues](https://github.com/zcyc/constructor/issues)
3. Create a new issue with detailed information

---

**Made with ❤️ for the Go community**

