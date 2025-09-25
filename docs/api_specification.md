# Event 微服務 API 規格文件

## 1. 概述

Event 微服務提供活動的生命週期管理，支援 Console 管理端和 Public 用戶端。本文件詳細描述其 API 規格、業務規則和資料結構。

## 2. 權限管理

- **Console API**: 依賴 API Gateway 進行身份驗證和商戶成員資格校驗。服務透過 `X-Merchant-Id` Header 進行資料隔離。
- **Public API**: 匿名存取，無需身份驗證。
- **Internal API**: 無權限校驗，僅限內部 gRPC 呼叫。

*詳細權限模型請參考 `PERMISSION_REQUIREMENTS.md`*

## 3. 通用規範

### 3.1 回應格式

本服務的 gRPC 回應直接返回對應的 Protobuf Message，gRPC-Gateway 會將其轉換為 JSON。gRPC 的錯誤會被轉換為對應的 HTTP 狀態碼和錯誤訊息。

**成功回應 (以 GetEvent 為例):**
```json
{
  "id": "event_id",
  "title": "活動標題",
  "status": "published",
  // ... 其他 Event 欄位
}
```

**錯誤回應 (gRPC-Gateway 自動轉換):**
```json
{
  "code": 5,
  "message": "event not found",
  "details": []
}
```

### 3.2 結構化內容 (Detail)

`detail` 欄位為一個陣列，由多個內容區塊組成，取代了舊的單一 HTML 內容。每個區塊都是一個物件，定義了類型和對應的資料。

```json
{
  "detail": [
    {
      "type": "text",
      "text_data": {
        "content": "這是一個文字區塊。"
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

### 3.3 分頁機制 (Pagination)

列表查詢 API (如 `GET /console/events`, `GET /events`) 同時支援兩種分頁方式，客戶端擇一使用。

- **Cursor-based**: 透過傳遞 `page_token` 進行無限滾動式分頁。
- **Page-based**: 透過傳遞 `page` 和 `page_size` 進行傳統頁碼分頁。

**注意**: 若同時提供 `page` 和 `page_token`，`page` 的優先級更高。

**分頁回應結構 (`pagination` 物件):**
```json
{
  "pagination": {
    // --- 通用欄位 ---
    "has_next": true,
    "has_prev": false,

    // --- Page-based 專用 ---
    "total_count": 150,
    "current_page": 1,
    "total_pages": 8,

    // --- Cursor-based 專用 ---
    "next_page_token": "base64_encoded_cursor_string",
    "prev_page_token": null
  }
}
```

## 4. Console 管理 API (`/console/*`)

### 4.1 建立 Event

- **端點**: `POST /console/events`
- **說明**: 建立一個新的活動，其初始狀態永遠為 `draft`。
- **請求 Body**:
  ```json
  {
    "title": "我的新活動",
    "summary": "活動摘要...",
    "visibility": "private", // 可選，預設為 private
    "cover_image_url": "https://.../cover.jpg",
    "location": { /* Location 物件 */ },
    "sessions": [ { /* Session 物件，不含 id */ } ],
    "detail": [ { /* DetailBlock 物件 */ } ],
    "faq": [ { /* FAQ 物件 */ } ]
  }
  ```
- **回應**: `200 OK`，返回 `CreateEventResponse` 物件。
  ```json
  {
    "id": "new_event_id",
    "created_at": "2025-09-25T10:00:00Z"
  }
  ```

### 4.2 取得 Event 列表

- **端點**: `GET /console/events`
- **查詢參數**:
  - 分頁: `page`, `page_size`, `page_token`
  - 篩選: `status`, `visibility`, `session_start_time_from`, `session_start_time_to`, `title_search`
  - 排序: `sort_by`, `sort_order`
- **回應**: `200 OK`，返回 `EventListResponse` 物件，包含 `events` 陣列和 `pagination` 物件。

### 4.3 取得單一 Event

- **端點**: `GET /console/events/{id}`
- **回應**: `200 OK`，返回完整的 `Event` 物件，包含聚合的 `sessions` 資訊。

### 4.4 更新 Event (部分欄位)

- **端點**: `PATCH /console/events/{id}`
- **說明**: 用於部分更新活動欄位，是主要的編輯 API。**不支援 `PUT` (全量更新)**。
- **Session 更新機制**:
  - **新增場次**: 在 `sessions` 陣列中傳入不含 `id` 的新 Session 物件。
  - **更新場次**: 傳入包含 `id` 的 Session 物件。**僅在 `draft` 狀態下有效**。
  - **刪除場次**: **不支援**。請使用獨立的 `DELETE` 端點。
- **權限**: `published` 狀態下僅能修改 `visibility` 和 `faq`，但可新增場次。
- **請求 Body**: 包含要修改的欄位。
- **回應**: `200 OK`，返回更新後的 `Event` 物件。

### 4.5 刪除 Event

- **端點**: `DELETE /console/events/{id}`
- **限制**: 只能刪除 `draft` 狀態的活動。
- **回應**: `200 OK`，返回空物件 `{}` (對應 gRPC-Gateway 的 `204 No Content`)。

### 4.6 變更 Event 狀態

- **端點**: `PUT /console/events/{id}/status`
- **請求 Body**: `{ "status": "published" }` 或 `{ "status": "archived" }`
- **業務規則**:
  - `draft` → `published`: 檢查所有必填欄位是否齊全。
  - `published` → `archived`: 檢查 `OrderService` 確認無活躍訂單。
  - **不允許逆向轉換**。
- **回應**: `200 OK`，返回更新後的 `Event` 物件。

### 4.7 刪除 Session

- **端點**: `DELETE /console/events/{id}/sessions/{session_id}`
- **說明**: 獨立的場次刪除功能。
- **限制**: 只能刪除 `draft` 狀態活動的場次。
- **回應**: `200 OK`，返回空物件 `{}`。

### 4.8 Event Form 管理

- **`PUT /console/events/{id}/form`**: 新增或更新活動的報名表單 (JSON Schema)。
- **`GET /console/events/{id}/form`**: 取得活動的報名表單。
- **`DELETE /console/events/{id}/form`**: 刪除活動的報名表單。

## 5. Public API (`/events/*`)

### 5.1 公開搜尋 Event

- **端點**: `GET /events`
- **限制**: 只返回 `status: "published"` 且 `visibility: "public"` 的活動。
- **查詢參數**: `page`, `page_size`, `page_token`, `merchant_id`, `title_search`, `session_start_time_from`, `session_start_time_to`, `location_lat`, `location_lng`, `location_radius`, `sort_by`, `sort_order`。
- **回應**: `200 OK`，返回 `EventListResponse` 物件。

### 5.2 分享連結查詢

- **端點**: `GET /events/{id}`
- **限制**: 只返回 `status: "published"` 的活動 (不論 `visibility`)。
- **回應**: `200 OK`，返回 `Event` 物件。

### 5.3 查詢 Event Form

- **端點**: `GET /events/{id}/form`
- **限制**: 僅 `published` 狀態的活動可查詢。
- **回應**: `200 OK`，返回 `EventForm` 物件。

## 6. Internal API (gRPC Only)

- **服務**: `InternalService`
- **用途**: 供內部其他微服務呼叫，無權限驗證。
- **方法**:
  - `GetEventById(id)`: 取得任何狀態、任何商戶的 Event 完整資料。
  - `GetSessionById(id)`: 取得單一 Session 的資料。

## 7. 資料驗證規則

所有驗證規則均定義在 `.proto` 檔案中。

- **Event (發布前必填)**: `title`, `cover_image_url`, `location`, `detail` (至少一個區塊), `sessions` (至少一個)。
- **Session**: `start_time` 必須早于 `end_time`。
- **Detail**: 陣列，最多 50 個區塊。
- **TextData**: `content` 最大 10,000 字元。
- **ImageData**: `url` 為必填的 URI 格式。
- **FAQ**: 陣列，最多 20 組問答。

## 8. 事件狀態管理規範

### 8.1 狀態轉換流程 (單向)

```
  draft ──(發布)──> published ──(封存)──> archived
```

### 8.2 各狀態編輯權限

| 狀態 | 可執行操作 |
| :--- | :--- |
| `draft` | - ✅ 可修改所有欄位。<br>- ✅ 可新增、更新、刪除場次。<br>- ✅ 可刪除活動本身。 |
| `published` | - ✅ **僅可修改**: `visibility`, `faq`。<br>- ✅ **僅可新增**: `sessions`。<br>- ❌ **不可修改**: `title`, `summary`, `cover_image_url`, `location`, `detail` 及已存在的場次。<br>- ❌ **不可刪除**: 活動本身及已存在的場次。 |
| `archived` | - ❌ **完全鎖定**，不可進行任何修改或刪除操作。 |