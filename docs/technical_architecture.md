# Event 微服務技術架構設計文件

## 1. 架構概覽

### 1.1 系統定位
Event 微服務是 Partivo 平台的核心服務之一，負責活動資料的管理和查詢功能，採用微服務架構設計，提供高可用性和可擴展性。

### 1.2 技術棧
- **程式語言**: Go 1.21+
- **Web 框架**: gRPC + gRPC-Gateway
- **資料庫**: MongoDB 7.0+
- **訊息佇列**: 預留介面（未來整合）
- **容器化**: Docker
- **日誌**: 結構化日誌（JSON 格式）
- **監控**: 預留 OpenTelemetry 介面

### 1.3 架構圖

```
┌─────────────────┐    ┌─────────────────┐
│   Console Web   │    │   Public Web    │
│   (管理後台)     │    │   (前台使用者)   │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────────┬───────────┘
                     │ HTTP/gRPC
          ┌──────────▼───────────┐
          │   API Gateway        │
          │   (Authentication)   │
          └──────────┬───────────┘
                     │
          ┌──────────▼───────────┐
          │  Event Microservice  │
          │  ┌─────────────────┐ │
          │  │ gRPC Server     │ │
          │  └─────────────────┘ │
          │  ┌─────────────────┐ │
          │  │ gRPC-Gateway    │ │
          │  └─────────────────┘ │
          │  ┌─────────────────┐ │
          │  │ Business Logic  │ │
          │  └─────────────────┘ │
          │  ┌─────────────────┐ │
          │  │ Repository      │ │
          │  └─────────────────┘ │
          └──────────┬───────────┘
                     │
          ┌──────────▼───────────┐
          │     MongoDB          │
          │   ┌─────────────┐    │
          │   │   Events    │    │
          │   │ Collection  │    │
          │   └─────────────┘    │
          └──────────────────────┘

          ┌─────────────────┐
          │ Order Service   │ ◄─── 狀態轉換時調用
          └─────────────────┘

          ┌─────────────────┐
          │ Media Service   │ ◄─── 圖片上傳管理
          └─────────────────┘
```

## 2. 服務層架構

### 2.1 分層設計

```
┌─────────────────────────────────────┐
│           Transport Layer           │  ← gRPC + HTTP Gateway
├─────────────────────────────────────┤
│           Service Layer             │  ← 業務邏輯處理
├─────────────────────────────────────┤
│          Repository Layer           │  ← 資料存取抽象
├─────────────────────────────────────┤
│            Data Layer               │  ← MongoDB
└─────────────────────────────────────┘
```

### 2.2 目錄結構

```
event/
├── cmd/
│   └── server/
│       └── main.go                 # 服務入口點
├── internal/
│   ├── conf/
│   │   ├── config.go              # 配置管理
│   │   └── headers.go             # Header 定義
│   ├── service/
│   │   ├── event_service.go       # 業務邏輯層
│   │   ├── public_service.go      # 前台服務邏輯
│   │   └── validation.go          # 資料驗證
│   ├── dao/
│   │   └── repository/
│   │       ├── event_repository.go # 資料存取介面
│   │       └── mongodb_impl.go     # MongoDB 實作
│   ├── models/
│   │   ├── event.go               # Event 資料模型
│   │   ├── location.go            # Location 資料模型
│   │   └── session.go             # Session 資料模型
│   ├── dto/
│   │   ├── request.go             # 請求 DTO
│   │   ├── response.go            # 回應 DTO
│   │   └── error.go               # 錯誤定義
│   └── helper/
│       ├── pagination.go          # 分頁工具
│       ├── validator.go           # 驗證工具
│       └── converter.go           # 資料轉換
├── api/
│   ├── event/
│   │   └── event.proto            # gRPC 服務定義
│   └── common.proto               # 共用訊息定義
├── pkg/
│   └── vulpes/                    # 共用工具庫
└── deployments/
    ├── Dockerfile
    └── docker-compose.yml
```

## 3. 資料層設計

### 3.1 MongoDB 設計

**Collection: events**

```javascript
{
  "_id": ObjectId,
  "title": String,
  "brand_id": ObjectId,
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
  "sessions": [{
    "_id": ObjectId,
    "start_time": ISODate,
    "end_time": ISODate
  }],
  "detail": {
    "content": String,
    "content_type": String       // "html", "json", "markdown"
  },
  "faq": [{
    "question": String,
    "answer": String
  }],
  "created_at": ISODate,
  "created_by": ObjectId,
  "updated_at": ISODate,
  "updated_by": ObjectId
}
```

### 3.2 索引策略

**使用現有的 Migration 機制**

在 `internal/dao/mongodb/migration.go` 中新增 Event 相關索引：

```go
var migrations = []Migration{
  {
    Collection: "events",
    Indexes: []mongo.IndexModel{
      // 基本查詢索引
      {
        Keys: bson.D{
          {Key: "brand_id", Value: 1},
          {Key: "status", Value: 1},
          {Key: "visibility", Value: 1},
        },
      },
      // 時間範圍查詢索引
      {
        Keys: bson.D{
          {Key: "brand_id", Value: 1},
          {Key: "sessions.start_time", Value: 1},
        },
      },
      // 地理位置索引
      {
        Keys: bson.D{{Key: "location.coordinates", Value: "2dsphere"}},
      },
      // 全文搜尋索引
      {
        Keys: bson.D{{Key: "title", Value: "text"}},
      },
      // 排序索引
      {
        Keys: bson.D{
          {Key: "brand_id", Value: 1},
          {Key: "created_at", Value: -1},
        },
      },
      // 前台查詢索引
      {
        Keys: bson.D{
          {Key: "status", Value: 1},
          {Key: "visibility", Value: 1},
          {Key: "sessions.start_time", Value: 1},
        },
      },
    },
  },
}
```

### 3.3 資料庫連接管理

**使用現有的配置結構**

專案已有完整的 MongoDB 配置，位於 `internal/conf/config.go`：

```go
// 現有的 MongodbConfig 結構
type MongodbConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

// 連接建立在 internal/dao/mongodb/mongodb.go
func NewMongoDB(cfg *conf.MongodbConfig) (*mongo.Client, func(), error)
```

## 4. 業務邏輯層設計

### 4.1 Service Interface 設計

```go
type EventService interface {
    // Console API
    CreateEvent(ctx context.Context, req *CreateEventRequest) (*Event, error)
    GetEventList(ctx context.Context, req *GetEventListRequest) (*EventListResponse, error)
    GetEvent(ctx context.Context, id string) (*Event, error)
    UpdateEvent(ctx context.Context, id string, req *UpdateEventRequest) (*Event, error)
    PatchEvent(ctx context.Context, id string, req *PatchEventRequest) (*Event, error)
    DeleteEvent(ctx context.Context, id string) error
    UpdateEventStatus(ctx context.Context, id string, status string) (*Event, error)
}

type PublicService interface {
    // Public API
    SearchPublicEvents(ctx context.Context, req *SearchPublicEventsRequest) (*EventListResponse, error)
    GetPublicEvent(ctx context.Context, id string) (*Event, error)
}
```

### 4.2 Repository Interface 設計

```go
type EventRepository interface {
    Create(ctx context.Context, event *Event) (*Event, error)
    FindByID(ctx context.Context, id string) (*Event, error)
    FindByBrandID(ctx context.Context, brandID string, filter *EventFilter) ([]*Event, *Pagination, error)
    Update(ctx context.Context, id string, event *Event) (*Event, error)
    Delete(ctx context.Context, id string) error
    FindPublic(ctx context.Context, filter *PublicEventFilter) ([]*Event, *Pagination, error)
    
    // 地理位置查詢
    FindNearby(ctx context.Context, lat, lng float64, radius int, filter *PublicEventFilter) ([]*Event, error)
    
    // 全文搜尋
    SearchByTitle(ctx context.Context, query string, filter *EventFilter) ([]*Event, error)
}
```

### 4.3 狀態轉換邏輯

```go
type StateTransition struct {
    orderService OrderServiceClient
}

func (s *StateTransition) CanTransition(ctx context.Context, event *Event, newStatus string) error {
    switch event.Status {
    case "draft":
        if newStatus == "published" {
            return s.validatePublishRequirements(event)
        }
    case "published":
        if newStatus == "archived" {
            return nil // 無限制
        }
        return errors.New("published event can only be archived")
    case "archived":
        if newStatus == "published" {
            return s.validatePublishRequirements(event)
        }
        if newStatus == "draft" {
            hasOrders, err := s.orderService.HasOrders(ctx, event.ID)
            if err != nil {
                return err
            }
            if hasOrders {
                return errors.New("cannot modify event with existing orders")
            }
        }
    }
    return nil
}
```

### 4.4 事件建立邏輯與交易處理

**目前實作邏輯：**

Event 建立採用兩階段提交模式：
1. 先建立 Event 資料
2. 再建立相關的 Sessions 資料

```go
func (s *EventService) CreateEvent(ctx context.Context, req *CreateEventRequest) (*models.Event, error) {
    // 1. 驗證並轉換請求
    event, err := s.convertCreateRequestToModel(req)
    if err != nil {
        return nil, err
    }

    // 2. 建立 Event（草稿狀態）
    createdEvent, err := s.eventRepo.Create(ctx, event)
    if err != nil {
        return nil, err
    }

    // 3. 建立 Sessions（如果失敗則 Rollback Event）
    if len(req.Sessions) > 0 {
        _, err = s.sessionService.CreateSessionsForEvent(ctx, createdEvent.ID.Hex(), req.BrandID, req.Sessions)
        if err != nil {
            // Rollback: 刪除已建立的 Event
            s.eventRepo.Delete(ctx, createdEvent.ID.Hex())
            return nil, fmt.Errorf("failed to create sessions: %w", err)
        }
    }

    return createdEvent, nil
}
```

**設計考量：**

**優點：**
- ✅ **資料一致性**：避免產生孤兒 Event（無 Sessions 的 Event）
- ✅ **原子性操作**：CreateEvent 是完整的業務操作
- ✅ **錯誤處理清晰**：失敗時會回傳明確的錯誤原因

**潛在問題：**
- ⚠️ **使用者體驗**：Session 驗證錯誤會導致整個建立失敗
- ⚠️ **前端兼容性**：與自動儲存 + Patch 更新模式可能衝突
- ⚠️ **資料恢復**：使用者需要重新填寫所有資料

**【待決議】替代方案：**

**方案 1：條件式 Rollback**
```go
if err != nil {
    // 驗證錯誤：允許建立不完整草稿
    if isValidationError(err) {
        log.Warn("Session validation failed, event created without sessions")
        return createdEvent, nil
    }
    // 系統錯誤：執行 Rollback
    s.eventRepo.Delete(ctx, createdEvent.ID.Hex())
    return nil, err
}
```

**方案 2：取消 Rollback**
```go
if err != nil {
    log.Warn("Failed to create sessions, event created as draft")
    // 不 rollback，讓前端透過 Patch API 完善
}
return createdEvent, nil
```

**方案 3：真正的資料庫交易**
```go
// 使用 MongoDB Transaction
session.WithTransaction(ctx, func(ctx mongo.SessionContext) error {
    // 在交易中建立 Event 和 Sessions
})
```

**建議**：考慮前端自動儲存架構，建議採用**方案 1**，在 Session 驗證錯誤時允許草稿狀態，系統錯誤時才 Rollback。

## 5. API 層設計

### 5.1 gRPC 服務定義

```protobuf
syntax = "proto3";

package event;

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "api/common.proto";

service EventService {
  // Console API
  rpc CreateEvent(CreateEventRequest) returns (api.Response) {
    option (google.api.http) = {
      post: "/console/events"
      body: "*"
    };
  }
  
  rpc GetEventList(GetEventListRequest) returns (api.Response) {
    option (google.api.http) = {
      get: "/console/events"
    };
  }
  
  rpc GetEvent(api.ID) returns (api.Response) {
    option (google.api.http) = {
      get: "/console/events/{id}"
    };
  }
  
  rpc UpdateEvent(UpdateEventRequest) returns (api.Response) {
    option (google.api.http) = {
      put: "/console/events/{id}"
      body: "*"
    };
  }
  
  rpc PatchEvent(PatchEventRequest) returns (api.Response) {
    option (google.api.http) = {
      patch: "/console/events/{id}"
      body: "*"
    };
  }
  
  rpc DeleteEvent(api.ID) returns (api.Response) {
    option (google.api.http) = {
      delete: "/console/events/{id}"
    };
  }
  
  rpc UpdateEventStatus(UpdateEventStatusRequest) returns (api.Response) {
    option (google.api.http) = {
      put: "/console/events/{id}/status"
      body: "*"
    };
  }
}

service PublicEventService {
  // Public API
  rpc SearchEvents(SearchEventsRequest) returns (api.Response) {
    option (google.api.http) = {
      get: "/events"
    };
  }
  
  rpc GetEvent(api.ID) returns (api.Response) {
    option (google.api.http) = {
      get: "/events/{id}"
    };
  }
}
```

### 5.2 使用現有的 Vulpes Framework

**專案已整合 Vulpes EzGRPC 框架**

1. **自動化的 Interceptor 鏈**：
   - 專案使用 `interceptor.NewGrpcServerWithInterceptors()` 提供標準中間件
   - 包含日誌、指標、錯誤恢復等功能

2. **Header 處理**：
   - 現有 Header 映射：x-user-id, x-user-email, x-user-name 等
   - **需要新增 x-brand-id 到 `headerTransMap`** 以支援權限檢查

3. **權限驗證機制**：
   - API Gateway 負責統一的身份驗證和 Brand 成員驗證
   - 微服務接收經過驗證的 Headers，無需再次驗證權限
   - 所有 Console API 的請求都會包含驗證過的 user_id 和 brand_id

4. **用戶資訊提取**：
```go
// 使用現有的 GetUser 函數
user, err := ezgrpc.GetUser(ctx)
if err != nil {
    return nil, status.Error(codes.Unauthenticated, "user not authenticated")
}

// 需要擴展以支援 Brand ID
func GetBrandID(ctx context.Context) (string, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return "", fmt.Errorf("failed to get metadata from context")
    }
    if len(md.Get("brand-id")) == 0 {
        return "", fmt.Errorf("brand-id not found in metadata")
    }
    return md.Get("brand-id")[0], nil
}
```

5. **服務註冊**：
```go
// 在 service 包的 init() 函數中註冊
func init() {
    ezgrpc.InjectGrpcService(func(s grpc.ServiceRegistrar) {
        pb.RegisterEventServiceServer(s, &EventServiceServer{})
        pb.RegisterPublicEventServiceServer(s, &PublicEventServiceServer{})
    })
    
    ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterEventServiceHandlerFromEndpoint)
    ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterPublicEventServiceHandlerFromEndpoint)
}
```

## 6. 配置管理

### 6.1 使用現有配置結構

**專案已有完整的配置管理系統**，位於 `internal/conf/config.go`：

```go
// 現有的 AppConfig 結構
type AppConfig struct {
	Mode           string `mapstructure:"mode"`
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	Version        string `mapstructure:"version"`
	TimeZone       string `mapstructure:"time_zone"`
	*LogConfig     `mapstructure:"log"`
	*MongodbConfig `mapstructure:"mongodb"`
}

// 需要擴展的外部服務配置
type ExternalConfig struct {
    OrderService  ServiceConfig `mapstructure:"order_service"`
    MediaService  ServiceConfig `mapstructure:"media_service"`
}

type ServiceConfig struct {
    Endpoint string        `mapstructure:"endpoint"`
    Timeout  time.Duration `mapstructure:"timeout"`
}
```

### 6.2 現有配置檔案

**使用現有的 config.yaml**：

```yaml
# internal/conf/config.yaml
name: "partivo_event"
mode: "dev"
port: 8081
version: 1.0.0
time_zone: "Asia/Taipei"

log:
  level: "debug"
  filename: "logs/app.log"
  max_size: 200 #MB
  max_age: 30
  max_backups: 7

mongodb:
  host: "127.0.0.1"
  port: 27017
  db: "partivo_event"

# 需要新增的外部服務配置
external:
  order_service:
    endpoint: "localhost:9090"
    timeout: "10s"
  media_service:
    endpoint: "localhost:9091"
    timeout: "30s"
```

## 7. 錯誤處理策略

### 7.1 錯誤分類

```go
// 業務錯誤
var (
    ErrEventNotFound       = errors.New("event not found")
    ErrInvalidStatus       = errors.New("invalid status transition")
    ErrHasOrders          = errors.New("event has existing orders")
    ErrInvalidTimeRange   = errors.New("invalid session time range")
    ErrSessionOverlap     = errors.New("session time overlap")
)

// 系統錯誤
var (
    ErrDatabaseConnection = errors.New("database connection failed")
    ErrExternalService   = errors.New("external service unavailable")
)
```

### 7.2 錯誤轉換

```go
func TranslateError(err error) (codes.Code, string) {
    switch {
    case errors.Is(err, ErrEventNotFound):
        return codes.NotFound, "Event not found"
    case errors.Is(err, ErrInvalidStatus):
        return codes.FailedPrecondition, "Invalid status transition"
    case errors.Is(err, ErrHasOrders):
        return codes.FailedPrecondition, "Cannot modify event with existing orders"
    case errors.Is(err, ErrInvalidTimeRange):
        return codes.InvalidArgument, "Invalid session time range"
    case errors.Is(err, ErrSessionOverlap):
        return codes.InvalidArgument, "Session times cannot overlap"
    default:
        return codes.Internal, "Internal server error"
    }
}
```

## 8. 效能優化策略

### 8.1 查詢優化
- 合理使用 MongoDB 索引
- 分頁查詢避免 skip() 大量資料
- 地理位置查詢使用 2dsphere 索引
- 全文搜尋使用 text 索引

### 8.2 Session 更新機制優化

**智慧型差異更新**：
- 採用陣列差異比對，支援新增、修改、刪除的混合操作
- 使用 MongoDB BulkWrite API 進行批次操作，提升性能

**BulkWrite 策略**：
```go
func (r *MongoSessionRepository) BulkUpdateSessions(ctx context.Context, 
    creates []*models.Session, updates []*models.Session, deleteIDs []string) error {
    
    writeModels := []mongo.WriteModel{}
    
    // Delete operations
    for _, id := range deleteIDs {
        deleteModel := mongo.NewDeleteOneModel().SetFilter(bson.M{"_id": objectID})
        writeModels = append(writeModels, deleteModel)
    }
    
    // Update operations  
    for _, session := range updates {
        updateModel := mongo.NewUpdateOneModel().
            SetFilter(bson.M{"_id": session.ID}).
            SetUpdate(bson.M{"$set": session})
        writeModels = append(writeModels, updateModel)
    }
    
    // Insert operations
    for _, session := range creates {
        insertModel := mongo.NewInsertOneModel().SetDocument(session)
        writeModels = append(writeModels, insertModel)
    }
    
    // Execute with unordered mode for better performance
    opts := options.BulkWrite().SetOrdered(false)
    _, err := r.collection.BulkWrite(ctx, writeModels, opts)
    return err
}
```

**效能優勢**：
- **網路往返減少**：從 N+M+1 次操作降為 1 次 BulkWrite
- **並行處理**：`SetOrdered(false)` 允許 MongoDB 並行執行操作
- **部分原子性**：失敗操作不影響成功操作，提升可用性

### 8.3 連接池管理
- MongoDB 連接池大小根據負載調整
- 設定合理的連接超時時間
- 監控連接池使用率

### 8.4 快取策略（未來擴展）
- Redis 快取熱門查詢結果
- Event 詳細資料快取
- 搜尋結果快取（短時間）

## 9. 監控與日誌

### 9.1 使用現有的 Vulpes 日誌系統

**專案已整合 Vulpes Log 套件**：

```go
import vulpeslog "github.com/arwoosa/vulpes/log"

// 結構化日誌
vulpeslog.Info("Event created successfully", 
    vulpeslog.String("event_id", eventID),
    vulpeslog.String("user_id", userID),
    vulpeslog.String("brand_id", brandID),
    vulpeslog.String("action", "create_event"),
    vulpeslog.Duration("duration", duration),
)

// 錯誤日誌
vulpeslog.Error("Failed to create event",
    vulpeslog.String("event_id", eventID),
    vulpeslog.Err(err),
)
```

**日誌配置**在 `main.go` 中已設定：
```go
vulpeslog.SetConfig(
    vulpeslog.WithDev(isDev),
    vulpeslog.WithLevel(appConfig.LogConfig.Level),
)
```

### 9.2 使用現有的 Prometheus 監控

**專案已整合 Prometheus 指標**：

1. **自動化的指標收集**：
   - `grpc_prometheus.Register(grpcService)` 已在 `ezgrpc.go` 中設定
   - 提供標準的 gRPC 指標（請求計數、延遲、錯誤率）

2. **指標端點**：
   - `/metrics` 端點已自動註冊
   - 可直接被 Prometheus 抓取

3. **自訂業務指標**（可選）：
```go
var (
    eventCreatedCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "events_created_total",
            Help: "Total number of events created",
        },
        []string{"brand_id", "status"},
    )
    
    eventQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "event_query_duration_seconds",
            Help: "Event query duration",
        },
        []string{"operation"},
    )
)

func init() {
    prometheus.MustRegister(eventCreatedCounter)
    prometheus.MustRegister(eventQueryDuration)
}
```

## 10. 部署策略

### 10.1 Docker 配置

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o event-service cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/event-service .
COPY --from=builder /app/internal/conf/config.yaml .

EXPOSE 8080 8081
CMD ["./event-service"]
```

### 10.2 健康檢查

```go
func (s *Server) HealthCheck(ctx context.Context, req *empty.Empty) (*api.Response, error) {
    // 檢查資料庫連接
    if err := s.db.Ping(ctx); err != nil {
        return service.ResponseError(codes.Internal, err)
    }
    
    // 檢查外部服務連接（可選）
    
    return service.ResponseSuccess(&HealthResponse{
        Status: "healthy",
        Timestamp: time.Now().Unix(),
    })
}
```

## 11. 安全考量

### 11.1 輸入驗證
- 所有使用者輸入都需要驗證
- 防止 NoSQL 注入攻擊
- 檔案上傳路徑驗證

### 11.2 權限控制
- Brand 隔離確保資料安全
- Header 驗證防止偽造請求
- 狀態轉換權限檢查

### 11.3 資料敏感性
- 不在日誌中記錄敏感資訊
- 錯誤訊息不洩露內部結構
- 適當的資料遮罩

## 12. 測試策略與覆蓋率

### 12.1 測試架構

```
┌─────────────────────────────────────┐
│          E2E Tests (未實作)          │  ← API完整流程測試
├─────────────────────────────────────┤
│        Integration Tests            │  ← 資料庫整合測試 (問題待修復)
├─────────────────────────────────────┤
│           Unit Tests                │  ← 業務邏輯單元測試 ✅
├─────────────────────────────────────┤
│         Model Tests                 │  ← 資料模型測試 ✅
└─────────────────────────────────────┘
```

### 12.2 當前測試覆蓋狀況

**✅ 完整測試覆蓋 (100%通過)**

**Models層** - `internal/models/`
- `errors_test.go` - 自定義錯誤類型測試
- `event_test.go` - Event模型業務邏輯測試
  - 狀態轉換邏輯 (`CanTransitionTo`)
  - 可見性檢查 (`IsPublic`, `IsShareable`)
  - 驗證函數 (`IsValidStatus`, `IsValidVisibility`)
  - 常數定義與轉換矩陣
- `session_test.go` - Session模型測試
  - 時間驗證 (`IsValid`, `ValidateTimeSequence`)
  - 重複檢查 (`IsDuplicateOf`)
  - 批次驗證 (`ValidateSessions`)
  - 工具函數 (`GetEarliestStartTime`, `GetLatestEndTime`)

**Service層** - `internal/service/`
- `event_service_basic_test.go` - Event服務核心功能
  - 創建活動 (`CreateEvent`)
  - 獲取活動 (`GetEvent`)
  - 輸入驗證測試
- `public_service_test.go` - 公開API服務
  - 搜尋功能 (`SearchEvents`)
  - 地理位置過濾
  - 分頁處理
  - 可見性權限檢查
- `session_service_test.go` - Session管理服務
  - Session CRUD操作
  - 事件關聯驗證
  - 品牌權限檢查
  - 時間衝突檢測

### 12.3 測試工具與框架

**測試框架**
```go
// 主要測試庫
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
)
```

**Mock系統** - `internal/service/mocks/`
- `MockEventRepository` - 資料庫操作Mock
- `MockSessionRepository` - Session資料Mock
- `MockOrderService` - 外部訂單服務Mock

**測試工具** - `internal/testutils/`
- `fixtures.go` - 測試資料生成器
- `matchers.go` - 自定義Mock匹配器
- `helpers.go` - 測試輔助函數

### 12.4 測試執行命令

```bash
# 所有測試
make test

# 單元測試 (推薦)
make test-unit          # models + service 
make test-models        # 僅模型測試
make test-service       # 僅服務測試

# 整合測試 (目前有問題)
make test-integration   # MongoDB testcontainer測試

# 測試覆蓋率報告
make test-coverage      # 生成HTML覆蓋率報告
```

### 12.5 已知問題與待修復項目

**❌ Integration Tests問題**
- **問題**: MongoDB序列化錯誤 ("document is nil")
- **影響**: testcontainers整合測試無法執行
- **範圍**: `event_repository_integration_test.go`, `session_repository_integration_test.go`
- **狀態**: 待調查修復

**⚠️ 測試覆蓋缺口**
- **gRPC Server層**: 協議轉換與錯誤處理
- **配置管理**: `internal/conf/` 配置載入測試
- **轉換器**: `converters.go` 資料轉換邏輯
- **工具類**: `helper/` 輔助函數測試

### 12.6 測試最佳實踐

**測試組織**
```go
func TestServiceMethod_Scenario(t *testing.T) {
    // Setup - 準備測試環境
    mockRepo := &mocks.MockRepository{}
    service := NewService(mockRepo)
    
    // Mock設定
    mockRepo.On("Method", args...).Return(result, nil)
    
    // Execute - 執行測試目標
    result, err := service.Method(ctx, args...)
    
    // Assert - 驗證結果
    require.NoError(t, err)
    assert.Equal(t, expected, result)
    
    // Verify - 確認Mock調用
    mockRepo.AssertExpectations(t)
}
```

**表格驅動測試**
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        wantErr bool
    }{
        {"valid case", validInput, false},
        {"invalid case", invalidInput, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 12.7 CI/CD測試整合

**建議的CI流程**
```yaml
# .github/workflows/test.yml
- name: Run Unit Tests
  run: make test-unit
  
- name: Run Integration Tests (when fixed)
  run: make test-integration
  
- name: Generate Coverage Report  
  run: make test-coverage
  
- name: Upload Coverage
  uses: codecov/codecov-action@v3
```

### 12.8 測試品質目標

**短期目標 (當前已達成)**
- ✅ 核心業務邏輯100%覆蓋
- ✅ 關鍵服務功能測試完整
- ✅ Mock系統建立完善

**中期目標**
- 🔄 修復Integration測試問題
- 📈 增加gRPC層測試覆蓋
- 🧪 添加端到端測試

**長期目標**
- 📊 整體測試覆蓋率達80%以上
- 🚀 自動化性能測試
- 🔍 契約測試(Pact)整合

## 13. 未來擴展規劃

### 13.1 效能優化
- 引入 Redis 快取層
- 讀寫分離（MongoDB 副本集）
- 搜尋引擎整合（Elasticsearch）

### 13.2 功能擴展
- 事件驅動架構（消息佇列）
- 多語言支援
- 批次操作 API

### 13.3 可觀測性
- OpenTelemetry 整合
- 分散式追蹤
- 業務指標監控

這份技術架構文件為 Event 微服務的開發和維護提供了完整的技術指引，確保系統的可維護性、可擴展性和高效能。完善的測試策略保障了代碼品質與業務邏輯的正確性。