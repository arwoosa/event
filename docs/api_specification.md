# Event 微服務 API 規格文件

## 概述

Event 微服務提供活動管理功能，支援 Console 管理端和前台用戶端的不同需求。採用 gRPC + HTTP Gateway 雙協議設計。

## 權限管理

### API Gateway 權限檢查責任
API Gateway 負責統一的身份驗證和權限檢查，Event 微服務接收經過驗證的請求。

### Console API 權限要求
所有 Console API 都需要以下權限檢查：
- **用戶身份驗證**: 必須為已登入用戶
- **Merchant 成員驗證**: 用戶必須為請求中 Merchant 的成員
- **資源隔離**: 只能操作該 Merchant 下的 Event 資源

### Public API 權限要求
- **公開搜尋** (`GET /events`): 無需身份驗證，僅返回 `published` + `public` 的 Event
- **分享連結** (`GET /events/{id}`): 無需身份驗證，僅返回 `published` 狀態的 Event

## 通用規範

### 回應格式

**重要變更**：成功請求的 API 回應格式已調整：

**成功回應（直接返回資料）：**
```json
{
  "id": "event_id",
  "title": "活動標題",
  // 其他資料欄位...
}
```

**錯誤回應（保持原格式）：**
```json
{
  "status": "error",
  "code": 3,
  "message": "error details"
}
```

**說明：**
- 成功請求移除 `data` 包裝層，直接返回實際資料
- 前端使用 HTTP Status Code 判斷請求成功/失敗
- 錯誤處理格式由 EzGRPC 統一管理，保持不變

### 內容結構重要變更

**Detail 內容結構**已從單一內容欄位改為結構化內容區塊：

**舊格式（已棄用）：**
```json
{
  "detail": {
    "content": "HTML content",
    "content_type": "html"
  }
}
```

**新格式：**
```json
{
  "detail": [
    {
      "type": "text",
      "text_data": {
        "content": "文字內容"
      }
    },
    {
      "type": "image",
      "image_data": {
        "url": "https://example.com/image.jpg",
        "alt": "圖片替代文字",
        "caption": "圖片說明"
      }
    }
  ]
}
```

**變更說明：**
- 支援多種內容類型：文字 (text) 和圖片 (image)
- 使用 oneof 欄位設計，確保類型安全
- 最多支援 50 個內容區塊
- 文字內容最大 10,000 字
- 圖片支援 URL、alt 文字和說明文字
- 可為空陣列，但不可為 null

### 狀態碼

- `1000`: 成功
- 錯誤狀態碼遵循 gRPC status codes

### Header 管理

**Console API 必需 Headers：**
- `X-User-Id`: 用戶 ID（API Gateway 驗證後傳遞）
- `X-User-Email`: 用戶 Email
- `X-User-Name`: 用戶名稱
- `X-User-Avatar`: 用戶頭像 URL
- `X-Merchant-Id`: Merchant ID（需新增到 AllowedHeaders，用於權限檢查）

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
      "name": "場次名稱",  // 新增：可選欄位
      "capacity": 100,  // 新增：可選欄位，null 表示不限制
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T12:00:00Z"
    }
  ],
  "detail": [
    {
      "type": "text",
      "text_data": {
        "content": "這是活動的詳細描述內容"
      }
    },
    {
      "type": "image",
      "image_data": {
        "url": "https://example.com/detail-image.jpg",
        "alt": "活動詳細圖片",
        "caption": "活動現場圖片說明"
      }
    }
  ],
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
  "id": "event_id",
  "created_at": "2024-01-01T10:00:00Z"
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
- `page`: int (page-based pagination，從 1 開始)  
- `page_size`: int (預設 20，最大 100)
- `status`: string (draft|published|archived)
- `visibility`: string (public|private)
- `session_start_time_from`: string (RFC3339)
- `session_start_time_to`: string (RFC3339)
- `title_search`: string (title 全文搜尋)
- `sort_by`: string (created_at|updated_at|session_start_time)
- `sort_order`: string (asc|desc，預設 desc)

**分頁說明：**
- 使用 `page_token` 時採用 cursor-based 分頁（無限滾動）
- 使用 `page` 時採用 page-based 分頁（傳統分頁），會提供完整的頁數資訊
- 兩種分頁方式不可同時使用，`page` 參數會覆蓋 `page_token`

**回應：**
```json
{
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
          "name": "場次名稱",
          "capacity": 100,
          "start_time": "2024-01-01T10:00:00Z",
          "end_time": "2024-01-01T12:00:00Z"
        }
      ],
      "created_at": "2024-01-01T09:00:00Z",
      "updated_at": "2024-01-01T09:30:00Z"
    }
  ],
  "pagination": {
    // Cursor-based 分頁欄位（使用 page_token 時）
    "next_page_token": "next_cursor",
    "prev_page_token": "prev_cursor",
    
    // Page-based 分頁欄位（使用 page 時）
    "current_page": 2,
    "total_pages": 8,
    "total_count": 150,
    
    // 通用欄位（兩種分頁都有）
    "has_next": true,
    "has_prev": true
  }
}
```

**分頁欄位說明：**
- `next_page_token` / `prev_page_token`: cursor-based 分頁的游標（僅在使用 `page_token` 時出現）
- `current_page`: 當前頁數，從 1 開始（僅在使用 `page` 時出現）
- `total_pages`: 總頁數（僅在使用 `page` 時出現）
- `total_count`: 總筆數（僅在使用 `page` 時出現）
- `has_next` / `has_prev`: 是否有下一頁/上一頁（兩種分頁都有）

### 3. 取得單一 Event

**端點：** `GET /console/events/{id}`

**回應：**
```json
{
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
      "name": "場次名稱",
      "capacity": 100,
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T12:00:00Z"
    }
  ],
  "detail": [
    {
      "type": "text",
      "text_data": {
        "content": "活動詳細內容描述"
      }
    }
  ],
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
      "name": "更新後的場次名稱",
      "capacity": 150,
      "start_time": "2024-01-02T10:00:00Z",
      "end_time": "2024-01-02T12:00:00Z"
    },
    {
      "id": "",  // 空值表示新增場次
      "name": "新場次",
      "capacity": null,  // null 表示不限制容量
      "start_time": "2024-01-02T14:00:00Z",
      "end_time": "2024-01-02T16:00:00Z"
    }
  ]
}
```

**Session PATCH 更新機制（重要變更）：**
- **新增場次**：`id` 為空字串或不提供
- **修改場次**：`id` 為現有 session 的 ID
- **不再支援刪除**：PATCH API 不再處理場次刪除，改用專門的 DELETE API
- **批次操作**：使用 MongoDB BulkWrite 確保操作效率
- **權限檢查**：Published 狀態下的時間修改受限制

### 6. 刪除 Event

**端點：** `DELETE /console/events/{id}`

**回應：**
```json
{}
```

**HTTP Status**: 204 No Content

### 7. 刪除 Session

**端點：** `DELETE /console/events/{event_id}/sessions/{session_id}`

**新增的獨立 API**：從 Event PATCH 中分離出來的場次刪除功能

**權限檢查：**
- **Draft 狀態**：可自由刪除
- **Published 狀態**：需同時滿足以下條件：
  - 該 Session 沒有任何訂單（呼叫 OrderService 確認）
  - 不是最後一個 Session（Event 至少保留一個）
- **Archived 狀態**：完全禁止刪除

**回應：**
```json
{}
```

**HTTP Status**: 204 No Content

**錯誤情境：**
- `FailedPrecondition`：Session 有訂單或是最後一個 Session
- `NotFound`：Session 不存在

### 8. 變更 Event 狀態

**端點：** `PUT /console/events/{id}/status`

**請求參數：**
```json
{
  "status": "published"
}
```

**業務規則（重要更新）：**
- **單向轉換**：Draft → Published → Archived（不可逆）
- **草稿 → 發布**：檢查必填欄位完整性
- **發布 → 下架**：需要呼叫 OrderService 確認所有訂單都是 Cancelled 和 Refunded
- **編輯限制**：各狀態有不同的編輯權限（詳見編輯權限章節）

## Public API (前台用戶)

### 1. 公開搜尋 Event

**端點：** `GET /events`

**查詢參數：**
- `merchant_id`: string (選填，篩選特定 Merchant)
- `page_token`: string (cursor-based pagination)
- `page`: int (page-based pagination，從 1 開始)
- `page_size`: int (預設 20，最大 100)
- `title_search`: string
- `session_start_time_from`: string
- `session_start_time_to`: string
- `location_lat`: float (地理位置搜尋)
- `location_lng`: float
- `location_radius`: int (公尺，預設 1000)
- `sort_by`: string (session_start_time|created_at)
- `sort_order`: string (asc|desc)

**分頁說明：**
- 與 Console API 相同，支援 cursor-based 和 page-based 兩種分頁方式
- 使用 `page` 參數時會提供完整的分頁資訊（`current_page`, `total_pages`, `total_count`）

**限制：**
- 只返回 `status: "published"` 且 `visibility: "public"` 的 Event

**回應：** 與 Console API 的列表格式相同，包含完整的分頁資訊

### 2. 分享連結查詢

**端點：** `GET /events/{id}`

**限制：**
- 只能查看 `status: "published"` 的 Event
- 不限制 `visibility`

**回應：** 與 Console API 的單一 Event 格式相同，但簡化欄位

## Internal API (內部服務)

### 1. 取得 Event 詳細資料

**端點：** `InternalService.GetEventById` (僅 gRPC)

**用途：** 提供給其他內部微服務呼叫，取得 Event 完整資料

**權限：** 內部服務呼叫，**不需要 merchant 驗證**，可跨品牌查詢

**gRPC 方法：**
```protobuf
rpc GetEventById(api.ID) returns (Event);
```

**請求參數：**
```json
{
  "id": "event_id"
}
```

**回應：**
```json
{
  "id": "event_id",
  "title": "活動標題",
  "merchant_id": "merchant_id",
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
      "name": "場次名稱",
      "capacity": 100,
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T12:00:00Z"
    }
  ],
  "detail": [
    {
      "type": "text",
      "text_data": {
        "content": "活動詳細內容"
      }
    }
  ],
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
```

**錯誤情境：**
- `NotFound`：Event 不存在

**與 Public API 的差異：**
- ✅ 不需要 Header 驗證
- ✅ 可查詢任何品牌的 Event（無品牌隔離）
- ✅ 可查詢任何狀態的 Event（包含 draft, archived）
- ✅ 僅限內部服務間 gRPC 呼叫

## 資料驗證規則

### 必填欄位
- **Event**: title, merchant_id, sessions, cover_image_url, detail (可為空陣列), location, visibility
- **Session**: start_time, end_time
- **Session 可選欄位**: name(場次名稱，可空白), capacity(容量限制，null表示不限制)
- **Location**: name, address, place_id, coordinates
- **DetailBlock**: type (text|image), 對應的 data 欄位 (text_data 或 image_data)
- **TextData**: content
- **ImageData**: url (必填), alt 和 caption (可選)
- **FAQ**: question, answer (當 FAQ 存在時，最多 20 個)

### 長度限制
- title: 最大 60 字
- summary: 最大 160 字
- detail 陣列: 最多 50 個內容區塊
- text_data.content: 最大 10,000 字
- image_data.alt: 最大 200 字
- image_data.caption: 最大 500 字
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
- `NotFound` (5): Event/Session 不存在
- `PermissionDenied` (7): 權限不足（用戶非 Merchant 成員或嘗試存取其他 Merchant 的資源）
- `FailedPrecondition` (9): 業務規則驗證失敗（狀態轉換、Session刪除限制等）
- `Internal` (13): 內部錯誤或外部服務呼叫失敗

### 錯誤回應格式
```json
{
  "status": "error",
  "code": 3,
  "message": "title is required"
}
```

## 事件狀態管理規範

### 狀態轉換流程

**單向狀態轉換（重要變更）：**
```
draft → published → archived
```

**狀態轉換規則：**
- ✅ `draft` → `published`：需滿足發佈前檢查(必要欄位等等)
- ✅ `published` → `archived`：需檢查是否有活躍訂單(與OrderService詢問狀態，需為canceled, refund)
- ❌ `published` → `draft`：不再允許（單向流程）
- ❌ `archived` → `published`：不再允許（單向流程）
- ❌ `archived` → `draft`：不再允許（單向流程）

### 事件/場次變更條件

- `draft`
  - 事件場次皆可修改、可刪除
- `published`
  - 不可刪除、僅能修改(FAQ, Visibility)、新增場次、不能修改場次
- `archived`
  - 不可修改、不可刪除

## 索引策略

### MongoDB 索引建議

```javascript
// 基本查詢索引
db.events.createIndex({"merchant_id": 1, "status": 1, "visibility": 1})

// 時間範圍查詢索引  
db.events.createIndex({"merchant_id": 1, "sessions.start_time": 1})

// 地理位置索引
db.events.createIndex({"location.coordinates": "2dsphere"})

// 全文搜尋索引
db.events.createIndex({"title": "text"})

// 排序索引
db.events.createIndex({"merchant_id": 1, "created_at": -1})
db.events.createIndex({"merchant_id": 1, "updated_at": -1})
```

## Published 狀態編輯權限詳細規範

### 活動設定權限

**可編輯欄位：**
- ✅ FAQ 內容 (可新增/修改問答)
- ✅ 可見性 (visibility) - public/private 切換

**不可編輯欄位：**
- ❌ 活動封面 (cover_image_url)
- ❌ 活動標題 (title)
- ❌ 活動地點 (location)
- ❌ 活動簡介 (summary)
- ❌ 活動內容 (detail 區塊內容)
- ❌ 活動狀態 (status) - 須使用專門 API

### 場次設定權限

**可操作（Draft）：**
- ✅ 新增場次 (所有欄位都可設定)
- ✅ 刪除場次
- ✅ 修改場次名稱 (name)  
- ✅ 修改場次容量 (capacity)

**不可操作（Publised）：**
- ❌ 修改現有場次 (任何欄位)
- ❌ 刪除現有場次
- ✅ 新增場次


## 外部服務整合

### 訂單微服務

**Event 狀態轉換檢查：**
- **端點：** `GET /orders/events/{event_id}/can-archive`
- **用途：** 檢查是否可以將 Event 轉為 Archived
- **回應：** `{"can_archive": true}`

**Session 訂單檢查：**
- **端點：** `GET /orders/sessions/{session_id}/has-orders`（假設接口）
- **用途：** 檢查 Session 是否有訂單
- **回應：** `{"has_orders": false}`

**注意：** 實際接口格式依 Robin 提供的 Proto 檔案為準