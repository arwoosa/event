# Event Service

本專案是基於 Go 語言的事件管理微服務，使用 gRPC 和 gRPC-Gateway 構建，為活動操作提供 Console (管理) 和 Public (公開) 兩種 API。

## ✨ 功能特性

- **雙重 API**: 同時支援 gRPC 和 RESTful HTTP API。
- **雙重服務模式**:
  - **Console API**: 用於內部管理的完整 CRUD 操作。
  - **Public API**: 用於對外公開的唯讀操作。
- **清晰的架構**: 遵循乾淨架構 (Clean Architecture) 原則，分層清晰。
- **容器化**: 提供完整的 Docker 和 Docker Compose 設定，方便部署與開發。
- **開發自動化**: 使用 `Makefile` 封裝常用指令，簡化開發流程。

## 🛠️ 環境需求

- **Go**: 1.24.0+
- **Docker** & **Docker Compose**
- **Make**

## 🚀 快速入門 (使用 Docker Compose)

這是最推薦的開發方式，可以一鍵啟動所有相依服務 (包含資料庫)。

1.  **建置並啟動服務**:
    ```bash
    make docker-compose-build
    ```
    此指令會使用 `docker-compose.yml` 的設定來建置 Docker image 並在背景啟動所有服務。

2.  **確認服務狀態**:
    ```bash
    docker-compose ps
    ```
    您應該會看到 `partivo_event_console`, `partivo_event_public`, 和 `partivo_mongodb` 三個服務正在運行。

3.  **服務端點**:
    - Console API (管理): `http://localhost:8081`
    - Public API (公開): `http://localhost:8082`
    - MongoDB: `mongodb://localhost:27017`

4.  **停止服務**:
    ```bash
    make docker-compose-down
    ```

## 💻 開發指南

### 主要 `make` 指令

您可以使用 `make help` 查看所有可用的指令。

#### 執行服務 (本地)

如果您不想使用 Docker，也可以在本地執行服務，但您需要自行啟動 MongoDB。

```bash
# 啟動 Console (管理) API 服務
make run-console

# 啟動 Public (公開) API 服務
make run-public
```

#### 產生 gRPC 程式碼

本專案透過 Docker 容器來執行 `protoc`，您**不需要**在本地手動安裝 `protoc` 或相關的 Go gRPC 插件。

```bash
# 從 .proto 檔案產生所有 gRPC 相關程式碼
make grpc
```

#### 測試

```bash
# 執行所有測試
make test

# 僅執行單元測試 (推薦)
make test-unit

# 產生測試覆蓋率報告
make test-coverage
```
報告會生成在 `coverage.html`。

#### 程式碼品質

```bash
# 格式化並檢查您的程式碼
make gotool

# 執行 linter 檢查
make lint
```

### 組態設定

- **Docker 環境**: `conf/config_docker.yaml`
- **本地環境**: `conf/config.yaml`

`docker-compose` 預設會使用 `config_docker.yaml`。

### 專案結構

```
.
├── cmd/                # 服務啟動入口
├── conf/               # 組態設定檔
├── docs/               # 專案文件 (API, 架構圖)
├── gen/                # 自動產生的程式碼 (from proto)
├── internal/           # 主要的業務邏輯、服務、資料存取層
│   ├── dao/            # 資料存取物件 (Repository)
│   ├── models/         # 資料庫模型與業務邏輯
│   └── service/        # 服務層，處理核心業務(含grpc layer, service layer)
├── pkg/                # 共用的程式碼庫 (vulpes)
└── proto/              # Protobuf 定義檔
```
