# Event 微服務技術架構設計文件

## 1. 架構概覽

### 1.1 系統定位
Event 微服務是 Partivo 平台的核心服務之一，負責活動資料的管理和查詢功能，採用微服務架構設計，提供高可用性和可擴展性。

### 1.2 Tech Stack
- **程式語言**: Go 1.24+
- **Web 框架**: gRPC + gRPC-Gateway
- **資料庫**: MongoDB 7.0+
- **容器化**: Docker
- **日誌**: 結構化日誌 (JSON 格式)
- **監控**: Prometheus

### 1.3 架構圖

```
          ┌──────────────────────┐
          │   API Gateway        │
          │   (Authentication)   │
          └──────────┬───────────┘
                     │ HTTP/gRPC
          ┌──────────▼───────────┐
          │                      │
┌─────────────────┐    ┌─────────────────┐
│   Console Web   │    │   Public Web    │
│   (管理後台)     │    │   (前台使用者)    │
└─────────┬───────┘    └─────────┬───────┘
          └──────────┬───────────┘
                     │ HTTP/gRPC
          ┌──────────▼───────────┐
          │  Event Microservice  │
          │  ┌─────────────────┐ │
          │  │ gRPC Server     │ │
          │  └─────────────────┘ │
          │  ┌─────────────────┐ │
          │  │ Business Logic  │ │
          │  │ (Service Layer) │ │
          │  └─────────────────┘ │
          │  ┌─────────────────┐ │
          │  │ Repository Layer│ │
          │  └─────────────────┘ │
          └──────────┬───────────┘
                     │
          ┌──────────▼───────────┐
          │       MongoDB        │
          │   ┌─────────────┐    │
          │   │   Events    │    │
          │   └─────────────┘    │
          │   ┌─────────────┐    │
          │   │   Sessions  │    │
          │   └─────────────┘    │
          │   ┌─────────────┐    │
          │   │    Forms    │    │
          │   └─────────────┘    │
          └──────────────────────┘

          ┌─────────────────┐
          │ Order Service   │ ◄─── 狀態轉換/刪除時調用
          └─────────────────┘
```

## 2. 服務層架構

### 2.1 分層設計

```
┌─────────────────────────────────────┐
│           Transport Layer           │  ← gRPC Handlers (grpc_*.go)
├─────────────────────────────────────┤
│            Service Layer            │  ← 業務邏輯 (event_service.go, etc.)
├─────────────────────────────────────┤
│          Repository Layer           │  ← 資料存取抽象 (repository/*.go)
├─────────────────────────────────────┤
│            Data Layer               │  ← MongoDB Driver
└─────────────────────────────────────┘
```

### 2.2 目錄結構 (更新後)

```
event/
├── cmd/
│   └── event-server/
│       └── main.go                 # 服務入口點
├── internal/
│   ├── conf/
│   │   └── config.go              # 配置管理
│   ├── constants/
│   │   └── constants.go           # 專案常數
│   ├── dao/
│   │   ├── mongodb/
│   │   │   └── mongodb.go         # DB 連線
│   │   │   └── migration.go       # DB Migration (初始化index)
│   │   └── repository/
│   │       ├── event.go           # Event Repository 介面
│   │       ├── session.go         # Session Repository 介面
│   │       └── form.go            # Form Repository 介面
│   ├── errors/
│   │   └── types.go               # 自定義錯誤類型
│   ├── models/
│   │   ├── event.go               # Event 資料模型
│   │   ├── session.go             # Session 資料模型
│   │   └── form.go                # Form 資料模型
│   ├── service/
│   │   ├── event_service.go       # Event 核心業務邏輯
│   │   ├── session_service.go     # Session 業務邏輯
│   │   ├── public_service.go      # Public API 業務邏輯
│   │   ├── grpc_event_server.go   # Console API gRPC Handler
│   │   ├── grpc_public_server.go  # Public API gRPC Handler
│   │   └── grpc_internal_server.go# Internal API gRPC Handler
│   └── testutils/                 # 測試工具
├── gen/
│   └── pb/                        # gRPC 產生的程式碼
├── proto/
│   ├── console_event.proto        # Console/Internal gRPC 服務定義
│   ├── public_event.proto         # Public gRPC 服務定義
│   └── common.proto               # 共用訊息定義
└── pkg/
    └── vulpes/                    # 共用工具庫
```

## 3. 資料層設計

### 3.1 MongoDB 設計 (更新後)

**Collection: `events`**
- **說明**: 儲存核心活動資訊，不再嵌入 Sessions。
```javascript
{
  "_id": ObjectId,
  "title": String,
  "merchant_id": String,
  "summary": String,
  "status": String,              // "draft", "published", "archived"
  "visibility": String,          // "public", "private"
  "cover_image_url": String,
  "location": {
    "name": String,
    "address": String,
    "place_id": String,
    "coordinates": {
      "type": "Point",
      "coordinates": [Number, Number]  // [lng, lat]
    }
  },
  "detail": [{                  // 改為陣列結構
    "type": String,              // "text", "image"
    "data": Object               // TextData or ImageData
  }],
  "faq": [{
    "question": String,
    "answer": String
  }],
  "created_at": ISODate,
  "created_by": String,
  "updated_at": ISODate,
  "updated_by": String
}
```

**Collection: `sessions`**
- **說明**: 獨立儲存活動場次，透過 `event_id` 關聯。
```javascript
{
  "_id": ObjectId,
  "event_id": ObjectId,
  "name": String,
  "capacity": Number, // 可為 null
  "start_time": ISODate,
  "end_time": ISODate,
  "created_at": ISODate,
  "updated_at": ISODate
}
```

**Collection: `event_forms`**
- **說明**: 獨立儲存活動報名表單，透過 `event_id` 關聯。
```javascript
{
  "_id": ObjectId,
  "event_id": ObjectId,
  "schema": Object,   // JSON Schema
  "uischema": Object, // UI Schema
  "created_at": ISODate,
  "created_by": String,
  "updated_at": ISODate,
  "updated_by": String
}
```

### 3.2 索引策略
索引定義於 `internal/dao/mongodb/migration.go` 中，並在服務啟動時自動建立。主要索引包括：
- **events**:
  - `merchant_id`, `status`, `visibility` 複合索引
  - `location.coordinates` 2dsphere 地理空間索引
  - `title` text 全文檢索索引
- **sessions**:
  - `event_id` 索引，用於快速查找特定活動的所有場次
- **event_forms**:
  - `event_id` 唯一索引，確保一個活動只有一個表單

## 4. 業務邏輯層設計

### 4.1 Service Interface 設計 (更新後)
實際的 gRPC 服務定義在 `.proto` 檔案中。Service 層的 Go 程式碼圍繞以下幾個核心結構：
- `EventService`: 處理 Console 端的 Event 和 Form 核心業務邏輯。
- `SessionService`: 處理所有 Session 相關的 CRUD 和批次操作。
- `PublicService`: 處理 Public API 的查詢邏輯。

### 4.2 Repository Interface 設計 (更新後)
- `EventRepository`: 定義 Event 的 CRUD、多條件查詢 (`Find`) 和公開查詢 (`FindPublic`)。查詢結果會透過 Aggregation Pipeline 聚合 `sessions` 資料。
- `SessionRepository`: 定義 Session 的 CRUD、批次查詢 (`FindByEventIDs`) 和批次更新 (`BulkUpdateSessions`)。
- `FormRepository`: 定義 Form 的 CRUD 和基於 `event_id` 的查詢。

### 4.3 狀態轉換邏輯
- **單向流程**: `draft` → `published` → `archived`，此規則在 `models/event.go` 的 `CanTransitionTo` 方法中定義。
- **草稿轉發布**: `UpdateEventStatus` 會呼叫 `validatePublishRequirements` 檢查所有必填欄位（如 title, cover_image_url, detail, location 及至少一個 session）是否齊全。
- **發布轉封存**: `UpdateEventStatus` 會呼叫 `OrderService` 確認該活動沒有任何未完成的訂單。
- **編輯限制**: `PatchEvent` 中的 `validateEventChanges` 方法會根據活動狀態實施編輯限制。`published` 狀態下只允許修改 `faq` 和 `visibility`。
- **刪除限制**: `DeleteEvent` 中的 `IsValidStatusForDelete` 方法限制只有 `draft` 狀態的活動可被刪除。

### 4.4 事件建立邏輯
`EventService.CreateEvent` 採用兩階段操作：
1.  在 `events` collection 中建立一筆 `draft` 狀態的活動。
2.  呼叫 `SessionService.CreateSessionsForEvent` 將場次資料批次寫入 `sessions` collection。
3.  若第二步失敗，則會回滾（刪除）第一步已建立的活動，以確保資料一致性。

## 5. API 層設計

### 5.1 gRPC 服務定義 (更新後)
專案定義了三個 gRPC 服務:
- **`EventService` (`console_event.proto`)**:
  - `CreateEvent`, `GetEventList`, `GetEvent`, `PatchEvent`, `DeleteEvent`, `UpdateEventStatus`
  - `DeleteSession`: 獨立的刪除場次 API。
  - `SetEventForm`, `GetEventForm`, `DeleteEventForm`: 獨立的報名表單管理 API。
- **`PublicEventService` (`public_event.proto`)**:
  - `SearchEvents`: 公開搜尋活動。
  - `GetEvent`: 透過分享連結查詢單一活動。
  - `GetEventForm`: 查詢公開活動的報名表單。
- **`InternalService` (`console_event.proto`)**:
  - `GetEventById`, `GetSessionById`: 供內部服務（如 OrderService）呼叫，無商戶權限驗證。

### 5.2 Vulpes EzGRPC 框架
- **Interceptor**: 自動整合日誌、錯誤恢復、Prometheus 指標等中介軟體。
- **Header 傳遞**: 透過 `ezgrpc.GetUser(ctx)` 從 gRPC Context 中獲取 API Gateway 傳遞的 `X-User-Id`, `X-Merchant-Id` 等資訊。
- **服務註冊**: 在 `cmd/event-server/root.go` 中，透過 `ezgrpc.InjectGrpcService` 和 `ezgrpc.RegisterHandlerFromEndpoint` 自動註冊 gRPC 服務和 HTTP Gateway。

## 6. 測試策略
專案的測試策略涵蓋了模型層和服務層的單元測試。
- **測試工具**: 使用 `testify` 套件進行斷言和 Mock。
- **Mock**: `EventRepository`, `SessionRepository`, `OrderService` 等外部依賴均有 Mock 實作。
- **已知問題**: `testcontainers` 的整合測試目前存在問題，尚待修復。
