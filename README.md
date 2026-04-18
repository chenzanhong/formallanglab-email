# Email 服务

## 模块定位

Email 服务是 FormalLangLab 项目的邮件发送模块，负责系统内所有邮件的异步发送。作为独立的后端服务，Email 服务通过 Kafka 消息队列接收邮件发送任务，实现与业务系统的解耦，确保邮件发送的可靠性和高可用性。

## 功能特性

- **异步邮件发送**：基于 Kafka 消息队列的异步处理机制
- **模板邮件支持**：支持 HTML 模板邮件渲染
- **验证码邮件**：专用于注册验证码、密码重置验证码等场景
- **熔断保护**：内置熔断器机制，防止邮件服务故障导致系统雪崩
- **重试机制**：邮件发送失败自动重试
- **指标监控**：完整的 Prometheus 指标采集
- **速率控制**：防止邮件发送频率过高被邮件服务商封禁

## 技术栈

| 类别 | 技术 |
|------|------|
| 开发语言 | Go 1.23+ |
| 消息队列 | Kafka |
| 邮件协议 | SMTP |
| 日志系统 | 结构化日志（zlog） |
| 配置管理 | YAML 配置文件 |
| 监控指标 | Prometheus + Grafana |
| 熔断器 | 自研熔断器（Circuit Breaker） |

## 项目结构

```
backend/email/
├── main.go                     # 主程序入口
├── configs/                    # 配置管理
│   ├── config.go               # 配置结构定义与加载
│   └── config.yaml.example     # 配置文件示例
├── middleware/                 # 中间件
│   ├── breaker/                # 熔断器中间件
│   │   └── circuit_breaker.go  # 熔断器实现
│   └── metrics/                # 指标采集中间件
│       └── metrics.go          # Prometheus 指标定义
├── model/                      # 数据模型
│   └── model.go                # 邮件模型定义
├── logs/                       # 日志管理
│   └── logger.go               # 日志初始化配置
├── logs/                       # 日志文件目录
├── .gitignore                  # Git 忽略配置
├── .golangci.yml               # Go 代码检查配置
├── Dockerfile                  # Docker 镜像构建文件
├── Makefile                    # Make 命令配置
├── go.mod                      # Go 模块依赖
└── go.sum                      # Go 依赖校验文件
```

## 中间件

| 中间件 | 文件路径 | 功能描述 |
|--------|---------|---------|
| CircuitBreaker | `middleware/breaker/circuit_breaker.go` | 熔断器保护，防止邮件服务故障扩散 |
| Metrics | `middleware/metrics/metrics.go` | Prometheus 指标采集，监控邮件发送延迟、成功率和 QPS |

## 核心组件

### 熔断器管理器（Circuit Breaker Manager）

**文件**：`middleware/breaker/circuit_breaker.go`

**功能**：
- 实时监控邮件发送成功率
- 自动触发熔断保护（失败率超过阈值时）
- 熔断后自动进入半开状态探测恢复
- 支持手动复位和状态查询

**熔断状态**：
- **Closed（关闭）**：正常状态，允许请求通过
- **Open（打开）**：熔断状态，拒绝所有请求
- **Half-Open（半开）**：探测状态，允许少量请求测试恢复情况

### Kafka 消费者（Kafka Consumer）

**文件**：`main.go`

**功能**：
- 从 Kafka 订阅邮件发送主题
- 并发消费邮件发送任务
- 消息确认机制（ACK）
- 消费失败自动重试

### 邮件发送器（Email Sender）

**文件**：`main.go`

**功能**：
- SMTP 协议邮件发送
- HTML 邮件模板渲染
- 附件支持
- 发送日志记录

## 数据模型

### 邮件消息（Email Message）

**文件**：`model/model.go`

**主要字段**：
- `to`：收件人邮箱地址
- `subject`：邮件主题
- `content`：邮件内容（HTML 格式）
- `template_type`：模板类型（verification_code、notification 等）
- `variables`：模板变量
- `priority`：优先级
- `retry_count`：重试次数

### 验证码模板（Verification Code Template）

**功能**：
- 注册验证码邮件模板
- 密码重置验证码邮件模板
- 支持动态变量替换（验证码、有效期等）

## Kafka 主题配置

### 邮件发送主题

**主题名称**：`email-send`

**消息格式**：
```json
{
  "to": "user@example.com",
  "subject": "FormalLangLab 注册验证码",
  "content": "<html>...</html>",
  "template_type": "verification_code",
  "variables": {
    "code": "123456",
    "expire_minutes": 10
  },
  "priority": "high"
}
```

**分区策略**：
- 按收件人邮箱哈希分区，确保同一用户的邮件顺序处理
- 支持多消费者并发处理

## 配置说明

**文件**：`configs/config.yaml.example`

**主要配置项**：

### SMTP 配置
- `smtp_host`：SMTP 服务器地址
- `smtp_port`：SMTP 端口
- `smtp_username`：SMTP 用户名
- `smtp_password`：SMTP 密码
- `smtp_from`：发件人邮箱地址

### Kafka 配置
- `kafka_brokers`：Kafka 集群地址列表
- `kafka_topic`：邮件发送主题
- `kafka_consumer_group`：消费者组 ID
- `kafka_concurrency`：并发消费者数量

### 熔断器配置
- `failure_threshold`：失败次数阈值
- `success_threshold`：成功次数阈值（半开状态）
- `timeout`：熔断超时时间
- `window_size`：统计窗口大小

### 速率限制配置
- `send_rate_limit`：每分钟最大发送数量
- `rate_limit_window`：速率限制时间窗口

### 日志配置
- 日志级别
- 日志格式（JSON/Text）
- 日志输出路径

## 监控指标

### Prometheus 指标

**文件**：`middleware/metrics/metrics.go`

**主要指标**：
- `email_send_total`：邮件发送总数（Counter）
- `email_send_success_total`：成功发送数量（Counter）
- `email_send_failure_total`：发送失败数量（Counter）
- `email_send_duration_seconds`：发送延迟直方图（Histogram）
- `circuit_breaker_state`：熔断器状态（Gauge）
- `kafka_consumer_lag`：Kafka 消费延迟（Gauge）

## 错误处理

### 邮件发送错误

**常见错误类型**：
- SMTP 连接失败
- 认证失败
- 收件人地址无效
- 邮件内容过大
- 速率限制触发

### 重试策略

- 指数退避重试（Exponential Backoff）
- 最大重试次数限制（默认 3 次）
- 重试失败后记录错误日志
- 支持死信队列（可选）

### 熔断保护

- 连续失败达到阈值时触发熔断
- 熔断期间直接拒绝请求
- 熔断超时后进入半开状态
- 半开状态成功则恢复，失败则继续熔断

## 使用场景

### 用户注册验证码

- 触发时机：用户在 Auth 服务注册时
- 邮件内容：6 位数字验证码
- 有效期：10 分钟
- 发送限制：1 分钟内只能发送 1 次

### 密码重置验证码

- 触发时机：用户申请重置密码时
- 邮件内容：6 位数字验证码 + 重置链接
- 有效期：10 分钟
- 发送限制：1 分钟内只能发送 1 次

### 系统通知

- 触发时机：系统重要事件通知
- 邮件内容：HTML 格式通知邮件
- 优先级：低
- 支持批量发送

## 最佳实践

### 邮件内容优化

- 使用响应式 HTML 模板，适配移动端
- 控制邮件大小在 100KB 以内
- 避免使用敏感词汇触发垃圾邮件过滤
- 添加退订链接（针对营销邮件）

### 发送频率控制

- 单个用户每日发送上限
- 同一 IP 地址发送频率限制
- 新域名逐步提升发送量（预热）
- 避开邮件服务商的敏感时段

### 监控与告警

- 实时监控发送成功率
- 熔断器状态变化告警
- Kafka 消费延迟告警
- SMTP 服务可用性监控

## Docker 与 Makefile 使用说明

### Dockerfile 说明

Dockerfile 采用多阶段构建，分为构建阶段和运行阶段：

**构建阶段**：
- 使用 Go 1.24 Alpine 镜像作为构建环境
- 配置国内 Go 代理镜像源加速依赖下载
- 先下载依赖再复制源代码，优化 Docker 缓存层
- 构建静态二进制文件（CGO_ENABLED=0）

**运行阶段**：
- 使用 Alpine 3.20 精简镜像
- 创建非 root 用户（emailuser）运行应用，提升安全性
- 配置 Asia/Shanghai 时区
- 复制二进制文件和配置文件
- 暴露服务端口（默认 8083）

### Makefile 命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `make lint` | 运行代码质量检查 | `make lint` |
| `make lint-fix` | 运行代码质量检查并自动修复 | `make lint-fix` |
| `make build` | 构建并推送 Docker 镜像 | `make build` |
| `make build T=false` | 仅构建镜像，不推送 | `make build T=false` |
| `make deploy` | 构建镜像并部署服务 | `make deploy` |

**快速开始**：
```bash
# 1. 代码质量检查
make lint

# 2. 构建镜像（本地测试，不推送）
make build T=false

# 3. 构建并推送镜像
make build

# 4. 部署服务
make deploy
```
