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
LABEL maintainer="FormalLangLab Team"
LABEL version="1.0"
LABEL description="Email Service for FormalLangLab Project"

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