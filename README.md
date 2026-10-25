# 🚀 Will-Shark

> A lightweight and powerful Go development framework with code generation capabilities

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

### 🛠 Infrastructure
- MySQL connection pool
- Redis connection pool
- RocketMQ integration
- Cron job support
- Dependency injection (Wire)
- Distributed tracing (OpenTelemetry OTLP: Jaeger / SkyWalking / ARMS)

### 🔧 Code Generator
- Auto-generate CRUD code from SQL
- Support multiple table structures
- Generate standardized layered architecture
- Swagger documentation
- Parameter validation
- Unified error handling

### 🚀 Coming Soon
- gRPC support
- Etcd integration
- Prometheus metrics

### Local Development
> Go >= 1.26.10 is recommended (earlier 1.26.x patch releases have known stdlib vulnerabilities).

```bash
docker compose up -d        # MySQL 8 + Redis 7, seeded from sql/init
go run . dev                # loads envconfig/dev_config.yaml
go test -race ./...
```

### Configuration
Config file: `./envconfig/<env>_config.yaml`, env is the first CLI argument (default `dev`).
See `envconfig/dev_config.yaml` for all keys (`mysql`, `redis`, `rocketmq`, `tracing`, `cors`).

> Never commit real credentials. The dev config only contains local docker-compose credentials.

```yaml
dev:
  tracing:                 # OTLP/HTTP; disabled -> noop tracer
    enable: false
    endpoint: 127.0.0.1:4318
    insecure: true
  cors:                    # whitelist; "*" = any origin without credentials
    allowOrigins:
      - "*"
```

### Docker
```bash
docker build -t willshark --build-arg VERSION=$(git rev-parse --short HEAD) .
# base images can be redirected to a private registry:
#   --build-arg GO_IMAGE=harbor.example.com/library/golang:1.26-alpine
docker run -v $PWD/envconfig/pro_config.yaml:/app/envconfig/pro_config.yaml willshark pro
```

### Code Generation

#### Generate from SQL File
```bash
# Generate CRUD code from SQL file
./gen -name=willshark -sql-file=./sql/tables.sql
```

#### Example SQL
```sql
CREATE TABLE `user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL DEFAULT '',
  `age` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Project Structure
```
app/
├── controller/      # HTTP handlers
├── service/        # Business logic
├── router/         # Route definitions
├── modules/
│   ├── mysql/
│   │   ├── entity/ # Database models
│   │   └── dao/    # Data access
└── do/
    ├── request/    # Request DTOs
    └── response/   # Response DTOs
```

## 🚀 Run the Application

```bash
go run main.go

 __        ___ _ _     ____  _                _    
 \ \      / (_) | |   / ___|| |__   __ _ _ __| | __
  \ \ /\ / /| | | |   \___ \| '_ \ / _  | '__| |/ /
   \ V  V / | | | |    ___) | | | | (_| | |  |   <
    \_/\_/  |_|_|_|   |____/|_| |_|\__,_|_|  |_|\_\

Server Port: 8899
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
