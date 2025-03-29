# Qiniu SSL Certificate Manager

一个用于申请Let's Encrypt SSL证书并上传到七牛云CDN的命令行工具。支持在非公网环境通过阿里云DNS Challenge方式验证域名所有权。

![Build and Release](https://github.com/WqyJh/qiniu-ssl/workflows/Build%20and%20Release/badge.svg)
![Docker Build and Push](https://github.com/WqyJh/qiniu-ssl/workflows/Docker%20Build%20and%20Push/badge.svg)

## 功能

- 通过阿里云DNS API自动申请Let's Encrypt免费SSL证书（DNS-01 challenge方式，适用于内网环境）
- 将证书上传到七牛云
- 自动检测七牛云域名是否支持HTTPS，并根据需要启用
- 为七牛云CDN域名绑定SSL证书
- 可选配置强制HTTPS和HTTP/2
- 自动检测证书过期时间并续期
- 通过七牛云API检查证书状态，确保准确判断证书是否需要更新
- 支持多域名批量管理和自动更新

## 安装

### 从源代码编译

```bash
# 克隆仓库
git clone https://github.com/WqyJh/qiniu-ssl.git
cd qiniu-ssl

# 编译
go build -o qiniu-ssl ./cmd/qiniu-ssl
```

### 直接下载二进制文件

请访问 [Releases](https://github.com/WqyJh/qiniu-ssl/releases) 页面下载适合您系统的二进制文件。

### 使用 Docker

您也可以通过 Docker 来运行本工具，无需本地安装 Go 环境。

#### 使用 Docker Compose（推荐）

1. 准备配置文件：

```bash
# 复制环境变量配置文件并修改
cp .env.example .env
# 编辑.env文件，填入您的七牛云和阿里云API凭证以及Let's Encrypt邮箱

# 创建配置目录
mkdir -p config
# 复制域名配置示例并修改
cp config/domains.txt.example config/domains.txt
# 编辑domains.txt，每行添加一个需要管理的域名
```

2. 启动容器：

```bash
docker-compose up -d
```

#### 直接使用 Docker 命令

```bash
# 构建镜像
docker build -t qiniu-ssl .

# 运行容器
docker run -d \
  --name qiniu-ssl \
  -v $(pwd)/certs:/app/certs \
  -v $(pwd)/config:/app/config \
  -e QINIU_ACCESS_KEY=您的七牛云AccessKey \
  -e QINIU_SECRET_KEY=您的七牛云SecretKey \
  -e ALIYUN_ACCESS_KEY=您的阿里云AccessKey \
  -e ALIYUN_SECRET_KEY=您的阿里云SecretKey \
  wqyjh/qiniu-ssl:latest --domains-file=/app/config/domains.txt --email=your@email.com --daemon
```

## 使用方法

您需要准备

- 七牛云 AccessKey 和 SecretKey: 用于七牛云 CDN 域名管理
- 阿里云 AccessKey 和 SecretKey: 用于阿里云 DNS API 管理 (AliyunDNSFullAccess 权限)
- 域名: 需要申请 SSL 证书的域名
- 邮箱: 用于 Let's Encrypt 身份认证

### 申请并配置证书

```bash
# 使用环境变量提供密钥
export QINIU_ACCESS_KEY=您的七牛云AccessKey
export QINIU_SECRET_KEY=您的七牛云SecretKey
export ALIYUN_ACCESS_KEY=您的阿里云AccessKey
export ALIYUN_SECRET_KEY=您的阿里云SecretKey
./qiniu-ssl --domain example.com --email your@email.com

# 完整参数
./qiniu-ssl \
    --qiniu-access-key 您的七牛云AccessKey \
    --qiniu-secret-key 您的七牛云SecretKey \
    --aliyun-access-key 您的阿里云AccessKey \
    --aliyun-secret-key 您的阿里云SecretKey \
    --aliyun-region cn-hangzhou \
    --domain example.com \
    --email your@email.com \
    --cert-dir ./certs \
    --force-https \
    --http2
```

### 自动检测并更新证书

工具支持自动检测证书有效期并更新，通过参数可以配置多域名和自动更新模式：

```bash
# 基本用法：检查单个域名证书
./qiniu-ssl --domain example.com --email your@email.com

# 多域名支持：从文件中读取域名列表
echo "example.com
example.org
sub.example.net" > domains.txt

./qiniu-ssl --domains-file domains.txt --email your@email.com

# 守护进程模式，并将日志输出到文件
./qiniu-ssl --domains-file domains.txt --email your@email.com --daemon --log-file /var/log/qiniu-ssl.log

# 自定义检查间隔和阈值
./qiniu-ssl --domains-file domains.txt --email your@email.com --daemon --check-interval 14 --threshold 30
```

注意：域名文件中的每行应包含一个域名，空行和以`#`开头的行将被忽略。

### 自动检测并更新证书（crontab）

您也可以通过设置系统定时任务（如crontab），实现证书的自动定期更新：

```bash
# 在crontab中添加，每周检查一次证书
0 0 * * 0 /path/to/qiniu-ssl --domains-file /etc/qiniu-ssl/domains.txt --email your@email.com --threshold 30 --log-file /var/log/qiniu-ssl.log 2>&1
```

### 可用选项

| 选项 | 短选项 | 描述 | 默认值 |
|------|------|------|------|
| `--qiniu-access-key` | `-qak` | 七牛云AccessKey (QINIU_ACCESS_KEY) | - |
| `--qiniu-secret-key` | `-qsk` | 七牛云SecretKey (QINIU_SECRET_KEY) | - |
| `--aliyun-access-key` | `-aak` | 阿里云AccessKey (ALIYUN_ACCESS_KEY) | - |
| `--aliyun-secret-key` | `-ask` | 阿里云SecretKey (ALIYUN_SECRET_KEY) | - |
| `--aliyun-region` | `-ar` | 阿里云区域 (ALIYUN_REGION) | `cn-hangzhou` |
| `--domain` | `-d` | 证书申请的域名 | - |
| `--domains-file` | `-df` | 包含域名列表的文件路径（每行一个域名） | - |
| `--email` | `-e` | 用于Let's Encrypt注册的邮箱地址 | - |
| `--cert-dir` | `-c` | 证书存储目录 | `certs` |
| `--force-https` | `-f` | 是否强制HTTPS | `false` |
| `--http2` | `-h2` | 是否启用HTTP/2 | `true` |
| `--check-interval` | `-i` | 证书检查间隔（单位：天） | 7 |
| `--threshold` | `-t` | 证书更新阈值（剩余有效期少于多少天触发更新，单位：天） | 30 |
| `--daemon` | - | 是否以守护进程模式运行，定期检查证书 | `false` |
| `--log-file` | - | 日志文件路径（不指定则输出到标准输出） | - |

## 工作原理

1. 申请Let's Encrypt免费SSL证书，通过阿里云DNS API创建必要的DNS TXT记录，以验证域名所有权
2. 将证书上传到七牛云
3. 检查域名是否已支持HTTPS：
   - 如果不支持，调用七牛云API启用HTTPS并同时绑定证书
   - 如果已支持，更新现有的HTTPS配置，绑定新证书
4. 根据参数配置强制HTTPS和HTTP/2选项

### 自动更新模式

1. 通过七牛云API检查域名对应的证书信息：
   - 获取域名HTTPS配置中的证书ID
   - 查询证书详细信息和有效期
   - 根据有效期计算是否需要更新
2. 如果证书不存在或有效期少于指定阈值（默认30天），则自动申请新证书并更新配置
3. 如启用daemon模式，将按指定间隔（默认7天）持续运行并检查证书状态

### 自动更新功能特点

1. 支持通过文件配置多个域名，便于批量管理
2. 直接从七牛云API获取证书信息，无需依赖本地证书文件
3. 提供专用的日志记录功能，方便排查问题
4. 友好的信号处理，可以安全地终止服务
5. 可以作为系统服务运行，提供长期稳定的证书更新服务

## 注意事项

- 本工具使用DNS Challenge方式验证域名所有权，**可以在内网环境中使用**，无需公网IP
- 您需要拥有 AliyunDNSFullAccess 权限的阿里云AccessKey和SecretKey
- 您需要拥有七牛云账号并获取 AccessKey 和 SecretKey
- **您需要先在七牛云控制台添加并配置好域名**，本工具不包含域名创建功能
- 本工具会自动检测域名是否已启用HTTPS，如未启用会自动为您启用
- 为避免 Let's Encrypt API 限制，建议不要过于频繁地执行证书申请操作

## 开发和贡献

### 自动化构建

本项目使用GitHub Actions来自动化构建过程：

1. **二进制构建**：当代码提交到main分支或创建新标签（tag）时，会自动构建适用于Linux、Windows和macOS的二进制文件。
   - 对于发布标签（如v0.1.0），会自动创建GitHub Release并附加构建的二进制文件。

2. **Docker镜像构建**：当代码提交到main分支或创建新标签（tag）时，会自动构建Docker镜像并推送到 Docker Hub: [wqyjh/qiniu-ssl](https://hub.docker.com/r/wqyjh/qiniu-ssl)

要使用自动构建的Docker镜像，可以执行：

```bash
# 使用最新版本
docker pull wqyjh/qiniu-ssl:latest

# 使用特定版本
docker pull wqyjh/qiniu-ssl:v0.1.0
```

### 开发

1. 克隆仓库
2. 安装依赖: `go mod download`
3. 编译: `go build -o qiniu-ssl ./cmd/qiniu-ssl`
4. 运行测试: `go test ./...`

## 许可证

MIT
