# 打包与构建指引

本文档介绍如何从源码构建 chatlog、为不同平台打包发布以及验证产物。

## 环境准备

- Go 1.21 或以上版本（与 `go.mod` 对齐）
- Git（用于获取版本信息）
- C/C++ 编译工具链（CGO 会在解密相关模块中启用）
- macOS 交叉编译到 Linux/Windows 时，建议安装 [xcode-select](https://developer.apple.com/support/xcode/) 对应的命令行工具

> 如果使用容器环境，可以直接参考仓库提供的 `Dockerfile`。

## 单平台本地构建

```bash
make build
```

命令会在 `bin/` 目录下生成当前平台的 `chatlog` 可执行文件。构建完成后可以通过下述方式验证：

```bash
./bin/chatlog --help
./bin/chatlog server
```

## 多平台交叉编译

仓库内置了常见桌面平台（macOS、Linux、Windows）的交叉编译配置：

```bash
make crossbuild
```

生成的可执行文件会命名为 `chatlog_<os>_<arch>` 并保存在 `bin/` 目录。例如：

- `bin/chatlog_darwin_arm64`
- `bin/chatlog_linux_amd64`
- `bin/chatlog_windows_amd64`

如需压缩特定产物，可在命令前加上 `ENABLE_UPX=1`（需要本地安装 `upx`）。

## 一键打包脚本

如果需要生成适合分发的压缩包和校验文件，可以使用脚本目录下的打包工具：

```bash
bash script/package.sh
```

脚本会执行以下步骤：

1. 调用 `make crossbuild` 生成全部平台二进制
2. 按平台打包为 `tar.gz` 或 `zip`，并放置在 `packages/` 目录
3. 输出 `packages/checksums.txt`，内含每个压缩包及常用裸二进制的 SHA256 校验值

执行成功后，发布所需的产物会集中在 `packages/` 目录中，可直接上传到 Release。

## 常见问题

### 如何指定版本号？

脚本会默认使用 `git describe --tags --always --dirty="-dev"`。若需自定义，可以在执行前设置 `VERSION` 环境变量：

```bash
VERSION=v1.2.3 make build
VERSION=v1.2.3 bash script/package.sh
```

### 仅构建特定平台

可以直接调用 Go 的交叉编译能力：

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -trimpath -o bin/chatlog_linux_arm64 main.go
```

或在运行 `make crossbuild` 前修改 `Makefile` 中的 `PLATFORMS` 变量。

### 构建后如何验证标签功能？

启动 HTTP 服务后，调用联系人接口即可验证新引入的标签字段：

```bash
./bin/chatlog server &
curl 'http://127.0.0.1:5030/api/v1/contact?label=客户' | jq .
```

若需要浏览器调试，也可以访问 `http://127.0.0.1:5030/` 并在「联系人」页的“按标签筛选”输入框中填写需要过滤的标签。

---

如有更多构建 / 发布需求，欢迎在 Issue 中反馈。
