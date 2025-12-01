# 第一阶段：构建阶段
FROM crpi-tcnuencv1iecgx03.cn-hangzhou.personal.cr.aliyuncs.com/chenzh2004/golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置国内Go代理镜像源，加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制所有源代码
COPY . .

# 构建应用，设置CGO_ENABLED=0以创建静态二进制文件
# email服务的入口文件是main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o email-worker main.go

# 第二阶段：运行阶段
FROM crpi-tcnuencv1iecgx03.cn-hangzhou.personal.cr.aliyuncs.com/chenzh2004/alpine:3.20

# 添加安全标签
LABEL maintainer="GDesign Team"
LABEL version="1.0"
LABEL description="Email Service for GDesign Project"

# 设置工作目录
WORKDIR /app

# 合并RUN指令，减少镜像层，同时创建非root用户
RUN apk --no-cache add ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    addgroup -g 1000 emailuser && \
    adduser -u 1000 -G emailuser -h /app -D emailuser && \
    mkdir -p /app/logs /app/configs /app/metrics && \
    chown -R emailuser:emailuser /app

# 切换到非root用户运行应用
USER emailuser

# 从构建阶段复制二进制文件
COPY --from=builder /app/email-worker /app/

# 复制配置文件
COPY configs/config.yaml /app/configs/

# 运行应用
CMD ["/app/email-worker"]

# ======================= 使用说明 =======================
# 1. 构建镜像：
#    docker build -t gdesign-email .
#
# 2. 准备环境：
#    - 创建.env文件（从.env.example复制并填写实际密钥）
#    - 确保Kafka服务正在运行
#
# 3. 运行容器（方式1：使用环境变量传递敏感信息）：
#    docker run -d \
#      --name gdesign-email \
#      -e KAFKA_BROKERS=host.docker.internal:9092 \
#      -e EMAIL_NAME=your_email@example.com \
#      -e EMAIL_PASSWORD=your_email_password \
#      -e SMTP_SERVER_HOST=smtp.example.com \
#      -e SMTP_SERVER_PORT=465 \
#      gdesign-email
#
# 4. 运行容器（方式2：使用卷挂载配置文件）：
#    docker run -d \
#      --name gdesign-email \
#      -v $(pwd)/.env:/app/.env \
#      -v $(pwd)/configs:/app/configs \
#      -v $(pwd)/logs:/app/logs \
#      gdesign-email

# docker build -t crpi-tcnuencv1iecgx03.cn-hangzhou.personal.cr.aliyuncs.com/chenzh2004/formallanglab-email .
# docker push crpi-tcnuencv1iecgx03.cn-hangzhou.personal.cr.aliyuncs.com/chenzh2004/formallanglab-email
