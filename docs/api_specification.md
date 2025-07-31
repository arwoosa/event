# Event 微服務 API 規格文件

## 概述

Event 微服務提供活動管理功能，支援 Console 管理端和前台用戶端的不同需求。採用 gRPC + HTTP Gateway 雙協議設計。

## 權限管理

### API Gateway 權限檢查責任
API Gateway 負責統一的身份驗證和權限檢查，Event 微服務接收經過驗證的請求。

### Console API 權限要求
所有 Console API 都需要以下權限檢查：
- **用戶身份驗證**: 必須為已登入用戶
- **Brand 成員驗證**: 用戶必須為請求中 Brand 的成員
- **資源隔離**: 只能操作該 Brand 下的 Event 資源

### Public API 權限要求
- **公開搜尋** (`GET /events`): 無需身份驗證，僅返回 `published` + `public` 的 Event
- **分享連結** (`GET /events/{id}`): 無需身份驗證，僅返回 `published` 狀態的 Event

## 通用規範

### 回應格式

所有 API 回應都使用統一的 `api.Response` 格式：

```json
{
  "status": "success|error",
  "code": 1000,
  "message": "optional error message",
  "data": {
    // 實際資料內容
  }
}
```

### 狀態碼

- `1000`: 成功
- 錯誤狀態碼遵循 gRPC status codes

### Header 管理

**Console API 必需 Headers：**
- `X-User-Id`: 用戶 ID（API Gateway 驗證後傳遞）
- `X-User-Email`: 用戶 Email
- `X-User-Name`: 用戶名稱
- `X-User-Avatar`: 用戶頭像 URL
- `X-Brand-Id`: Brand ID（需新增到 AllowedHeaders，用於權限檢查）

**Public API Headers：**
- 無必需 Headers，支援匿名存取


### 分頁機制

支援兩種分頁方式：

**1. Cursor-based Pagination（無限滾動）**
```json
{
  "page_token": "base64_encoded_cursor",
  "page_size": 20
}
```

**2. Page-based Pagination（傳統分頁）**
```json
{
  "page": 1,
  "page_size": 20
}
```

## Console 管理 API

### 1. 建立 Event

**端點：** `POST /console/events`

**請求參數：**
```json
{
  "title": "活動標題",
  "summary": "活動摘要",
  "status": "draft",
  "visibility": "private",
  "cover_image_url": "https://example.com/image.jpg",
  "location": {
    "name": "地點名稱",
    "address": "詳細地址",
    "place_id": "Google Places ID",
    "coordinates": {
      "type": "Point",
      "coordinates": [121.5654, 25.0330]
    }
  },
  "sessions": [
    {
      "id": "",  // 空值表示新增場次
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T12:00:00Z"
    }
  ],
  "detail": {
    "content": "Rich text content",
    "content_type": "html"
  },
  "faq": [
    {
      "question": "問題",
      "answer": "回答"
    }
  ]
}
```

**回應：**
```json
{
  "status": "success",
  "code": 1000,
  "data": {
    "id": "event_id",
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

**交易處理：**
- Event 建立採用兩階段提交：先建立 Event，再建立 Sessions
- 如果 Sessions 建立失敗，會自動刪除已建立的 Event（Rollback）
- 建立失敗時會回傳詳細的錯誤訊息，使用者需重新提交請求

### 2. 取得 Event 列表

**端點：** `GET /console/events`

**查詢參數：**
- `page_token`: string (cursor-based pagination)
- `page`: int (page-based pagination)  
- `page_size`: int (預設 20，最大 100)
- `status`: string (draft|published|archived)
- `visibility`: string (public|private)
- `session_start_time_from`: string (RFC3339)
- `session_start_time_to`: string (RFC3339)
- `title_search`: string (title 全文搜尋)
- `sort_by`: string (created_at|updated_at|session_start_time)
- `sort_order`: string (asc|desc，預設 desc)

**回應：**
```json
{
  "status": "success",
  "code": 1000,
  "data": {
    "events": [
      {
        "id": "event_id",
        "title": "活動標題",
        "summary": "活動摘要",
        "status": "published",
        "visibility": "public",
        "cover_image_url": "https://example.com/image.jpg",
        "location": {
          "name": "地點名稱",
          "address": "詳細地址"
        },
        "sessions": [
          {
            "id": "session_id",
            "start_time": "2024-01-01T10:00:00Z",
            "end_time": "2024-01-01T12:00:00Z"
          }
        ],
        "created_at": "2024-01-01T09:00:00Z",
        "updated_at": "2024-01-01T09:30:00Z"
      }
    ],
    "pagination": {
      "next_page_token": "next_cursor",
      "has_next": true,
      "total_count": 150
    }
  }
}
```

### 3. 取得單一 Event

**端點：** `GET /console/events/{id}`

**回應：**
```json
{
  "status": "success",
  "code": 1000,
  "data": {
    "id": "event_id",
    "title": "活動標題",
    "summary": "活動摘要",
    "status": "published",
    "visibility": "public",
    "cover_image_url": "https://example.com/image.jpg",
    "location": {
      "name": "地點名稱",
      "address": "詳細地址",
      "place_id": "Google Places ID",
      "coordinates": {
        "type": "Point",
        "coordinates": [121.5654, 25.0330]
      }
    },
    "sessions": [
      {
        "id": "session_id",
        "start_time": "2024-01-01T10:00:00Z",
        "end_time": "2024-01-01T12:00:00Z"
      }
    ],
    "detail": {
      "content": "Rich text content",
      "content_type": "html"
    },
    "faq": [
      {
        "question": "問題",
        "answer": "回答"
      }
    ],
    "created_at": "2024-01-01T09:00:00Z",
    "created_by": "user_id",
    "updated_at": "2024-01-01T09:30:00Z",
    "updated_by": "user_id"
  }
}
```

### 4. 更新 Event (全欄位)

**端點：** `PUT /console/events/{id}`

**請求參數：** 與建立 Event 相同的完整結構

### 5. 更新 Event (部分欄位)

**端點：** `PATCH /console/events/{id}`

**請求參數：**
```json
{
  "title": "新標題",
  "sessions": [
    {
      "id": "existing_session_id",  // 有值表示修改現有場次
      "start_time": "2024-01-02T10:00:00Z",
      "end_time": "2024-01-02T12:00:00Z"
    },
    {
      "id": "",  // 空值表示新增場次
      "start_time": "2024-01-02T14:00:00Z",
      "end_time": "2024-01-02T16:00:00Z"
    }
  ]
}
```

**Session 陣列更新機制：**
- **新增場次**：`id` 為空字串或不提供
- **修改場次**：`id` 為現有 session 的 ID
- **刪除場次**：不在陣列中的現有 session 會被刪除
- **批次操作**：使用 MongoDB BulkWrite 確保操作效率和部分原子性

### 6. 刪除 Event

**端點：** `DELETE /console/events/{id}`

**回應：**
```json
{
  "status": "success",
  "code": 1000,
  "data": null
}
```

### 7. 變更 Event 狀態

**端點：** `PUT /console/events/{id}/status`

**請求參數：**
```json
{
  "status": "published"
}
```

**業務規則：**
- 草稿 → 發布：檢查必填欄位
- 發布 → 下架：無限制
- 下架 → 發布：檢查必填欄位
- 下架狀態的修改/刪除：需檢查訂單微服務

## Public API (前台用戶)

### 1. 公開搜尋 Event

**端點：** `GET /events`

**查詢參數：**
- `brand_id`: string (選填，篩選特定 Brand)
- `page_token`: string
- `page_size`: int
- `title_search`: string
- `session_start_time_from`: string
- `session_start_time_to`: string
- `location_lat`: float (地理位置搜尋)
- `location_lng`: float
- `location_radius`: int (公尺，預設 1000)
- `sort_by`: string (session_start_time|created_at)
- `sort_order`: string (asc|desc)

**限制：**
- 只返回 `status: "published"` 且 `visibility: "public"` 的 Event

**回應：** 與 Console API 的列表格式相同，但簡化欄位

### 2. 分享連結查詢

**端點：** `GET /events/{id}`

**限制：**
- 只能查看 `status: "published"` 的 Event
- 不限制 `visibility`

**回應：** 與 Console API 的單一 Event 格式相同，但簡化欄位

## 資料驗證規則

### 必填欄位
- **Event**: title, brand_id, sessions, cover_image_url, detail.content, location, visibility
- **Session**: start_time, end_time  
- **Location**: name, address, place_id, coordinates
- **Detail**: content
- **FAQ**: question, answer (當 FAQ 存在時，最多 20 個)

### 長度限制
- title: 最大 60 字
- summary: 最大 160 字
- detail.content: 最大 64KB
- faq.question: 最大 100 字
- faq.answer: 最大 300 字
- faq 數量: 最多 20 個

### 業務規則驗證
- Sessions 至少一個
- Session start_time < end_time
- 同一 Event 的 Sessions 時間不可重疊 (start_time, end_time 組合唯一)
- 時間格式必須為 RFC 3339
- visibility 預設值為 "private"

## 錯誤處理

### 常見錯誤碼

- `InvalidArgument` (3): 參數驗證失敗
- `NotFound` (5): Event 不存在
- `PermissionDenied` (7): 權限不足（用戶非 Brand 成員或嘗試存取其他 Brand 的資源）
- `FailedPrecondition` (9): 業務規則驗證失敗
- `Internal` (13): 內部錯誤

### 錯誤回應格式
```json
{
  "status": "error",
  "code": 3,
  "message": "title is required"
}
```

## 索引策略

### MongoDB 索引建議

```javascript
// 基本查詢索引
db.events.createIndex({"brand_id": 1, "status": 1, "visibility": 1})

// 時間範圍查詢索引  
db.events.createIndex({"brand_id": 1, "sessions.start_time": 1})

// 地理位置索引
db.events.createIndex({"location.coordinates": "2dsphere"})

// 全文搜尋索引
db.events.createIndex({"title": "text"})

// 排序索引
db.events.createIndex({"brand_id": 1, "created_at": -1})
db.events.createIndex({"brand_id": 1, "updated_at": -1})
```

## 外部服務整合

### 訂單微服務

**端點：** `GET /orders/events/{event_id}/exists`

**用途：** 檢查 Event 是否有相關訂單

**回應：**
```json
{
  "has_orders": true
}
```