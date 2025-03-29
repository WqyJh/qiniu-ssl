FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o qiniu-ssl ./cmd/qiniu-ssl

# 运行时镜像
FROM alpine:latest

# 添加必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制可执行文件
COPY --from=builder /app/qiniu-ssl /usr/local/bin/

# 设置工作目录
WORKDIR /app

# 设置入口点
ENTRYPOINT ["qiniu-ssl"]

# 默认命令行参数，可在运行容器时覆盖
CMD ["--help"]
