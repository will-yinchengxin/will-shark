# 🚀 Will-Shark

> A lightweight and powerful Go development framework with code generation capabilities

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

### 🛠 Infrastructure
- MySQL connection pool
- Redis connection pool
- RocketMQ integration
- Cron job support
- Dependency injection (Wire)
- Distributed tracing (Jaeger)

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

### Configuration
Create your configuration file in `./envconfig`, for example `dev_config.yaml`:

```yaml
dev:
  mysql:
    will:
      Host: 172.16.161.54
      Port: 3306
      User: root
      Password: '123456'
      DataBase: will
      ParseTime: True
      MaxIdleConns: 10
      MaxOpenConns: 30
      ConnMaxLifetime: 28800
      ConnMaxIdletime: 7200
  redis:
    will:
      host: 172.16.161.54:6379
      password: ""
      database: 0
      maxIdleNum: 50
      maxActive: 5000
      maxIdleTimeout: 600
      connectTimeout: 1
      readTimeout: 2
  rocketmq:
    GroupName: test-rocket
    Topic: test-rocket
    Host:
      - 127.0.0.1:9876
    Retry: 3
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
