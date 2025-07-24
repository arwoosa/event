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
   - 需要新增 x-brand-id 到 `headerTransMap`

3. **用戶資訊提取**：
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

4. **服務註冊**：
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

### 8.2 連接池管理
- MongoDB 連接池大小根據負載調整
- 設定合理的連接超時時間
- 監控連接池使用率

### 8.3 快取策略（未來擴展）
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

## 12. 未來擴展規劃

### 12.1 效能優化
- 引入 Redis 快取層
- 讀寫分離（MongoDB 副本集）
- 搜尋引擎整合（Elasticsearch）

### 12.2 功能擴展
- 事件驅動架構（消息佇列）
- 多語言支援
- 批次操作 API

### 12.3 可觀測性
- OpenTelemetry 整合
- 分散式追蹤
- 業務指標監控

這份技術架構文件為 Event 微服務的開發和維護提供了完整的技術指引，確保系統的可維護性、可擴展性和高效能。