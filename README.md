# Qiniu SSL Certificate Manager

一个用于申请Let's Encrypt SSL证书并上传到七牛云CDN的命令行工具。支持在非公网环境通过阿里云DNS Challenge方式验证域名所有权。

## 功能

- 通过阿里云DNS API自动申请Let's Encrypt免费SSL证书（DNS-01 challenge方式，适用于内网环境）
- 将证书上传到七牛云
- 自动检测七牛云域名是否支持HTTPS，并根据需要启用
- 为七牛云CDN域名绑定SSL证书
- 可选配置强制HTTPS和HTTP/2

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

## 使用方法

您需要准备

- 七牛云 AccessKey 和 SecretKey: 用于七牛云 CDN 域名管理
- 阿里云 AccessKey 和 SecretKey: 用于阿里云 DNS API 管理（AliyunDNSFullAccess 权限）
- 域名: 需要申请 SSL 证书的域名
- 邮箱: 用于 Let's Encrypt 身份认证

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

### 可用选项

| 选项 | 短选项 | 描述 | 默认值 |
|------|------|------|------|
| `--qiniu-access-key` | `-qak` | 七牛云AccessKey (也可通过QINIU_ACCESS_KEY环境变量设置) | - |
| `--qiniu-secret-key` | `-qsk` | 七牛云SecretKey (也可通过QINIU_SECRET_KEY环境变量设置) | - |
| `--aliyun-access-key` | `-aak` | 阿里云AccessKey (也可通过ALIYUN_ACCESS_KEY环境变量设置) | - |
| `--aliyun-secret-key` | `-ask` | 阿里云SecretKey (也可通过ALIYUN_SECRET_KEY环境变量设置) | - |
| `--aliyun-region` | `-ar` | 阿里云区域 | `cn-hangzhou` |
| `--domain` | `-d` | 证书申请的域名 | - |
| `--email` | `-e` | 用于Let's Encrypt注册的邮箱地址 | - |
| `--cert-dir` | `-c` | 证书存储目录 | `certs` |
| `--force-https` | `-f` | 是否强制HTTPS | `false` |
| `--http2` | `-h2` | 是否启用HTTP/2 | `true` |

## 工作原理

1. 通过阿里云DNS API创建必要的DNS TXT记录，以验证域名所有权
2. 申请Let's Encrypt免费SSL证书
3. 将证书上传到七牛云
4. 检查域名是否已支持HTTPS：
   - 如果不支持，调用七牛云API启用HTTPS并同时绑定证书
   - 如果已支持，更新现有的HTTPS配置，绑定新证书
5. 根据参数配置强制HTTPS和HTTP/2选项

## 注意事项

- 本工具使用DNS Challenge方式验证域名所有权，**可以在内网环境中使用**，无需公网IP
- 您需要拥有域名的阿里云DNS管理权限
- 您需要创建有权限管理DNS记录的阿里云AccessKey和SecretKey
- 您需要拥有七牛云账号并获取AccessKey和SecretKey
- **您需要先在七牛云控制台添加并配置好域名**，本工具不包含域名创建功能
- 本工具会自动检测域名是否已启用HTTPS，如未启用会自动为您启用

## 许可证

MIT 