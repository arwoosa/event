# Event 微服務需求規格 (更新版)

## 1. 核心功能

Event 微服務負責管理 Partivo 平台上的所有活動，從建立、發布、查詢到封存的完整生命週期。服務區分為三個主要部分：

- **Console API**: 供商戶（Merchant）管理其名下活動的後台 API。
- **Public API**: 供終端使用者查詢公開活動的前台 API。
- **Internal API**: 供內部其他微服務（如訂單服務）調用的 gRPC 接口。

## 2. 權限管理

權限由 API Gateway 統一處理，Event 微服務信任 Gateway 傳遞的 Header 資訊。

- **Console API**: 所有請求必須包含有效的 `X-User-Id` 和 `X-Merchant-Id`。服務內部的所有資料操作都基於 `X-Merchant-Id` 進行嚴格的資料隔離。
- **Public API**: 匿名存取，無需身份驗證。
- **Internal API**: 無權限驗證，僅限內部受信任網路呼叫。

*詳細權限設計請參考 `PERMISSION_REQUIREMENTS.md`*

## 3. Event 狀態與生命週期管理

### 3.1 狀態定義

- `draft` (草稿): 活動建立後的初始狀態。可自由編輯所有內容。
- `published` (已發布): 活動對外可見的狀態。編輯權限受限。
- `archived` (已封存): 活動結束或下架後的最終狀態。內容完全鎖定，不可修改。

### 3.2 狀態轉換規則 (單向流程)

```
  draft ──(發布)──> published ──(封存)──> archived
```

- **`draft` → `published`**:
  - **條件**: 必須滿足所有發布條件，包括 `title`, `cover_image_url`, `detail`, `location` 欄位齊全，且至少存在一個 `session`。
  - **觸發**: 透過 `PUT /console/events/{id}/status` API。

- **`published` → `archived`**:
  - **條件**: 必須呼叫 `OrderService` 確認該活動所有相關訂單均已結束（如已取消或已退款）。
  - **觸發**: 透過 `PUT /console/events/{id}/status` API。

- **逆向轉換**: **不允許**。例如，`published` 無法回到 `draft`，`archived` 也無法回到任何狀態。

### 3.3 Published 狀態的編輯權限

當活動處於 `published` 狀態時，為確保活動內容的穩定性，僅以下欄位可透過 `PATCH /console/events/{id}` 修改：

- ✅ **可修改**: `visibility` (可見性), `faq` (常見問題)。
- ✅ **可新增**: `sessions` (可新增場次)。
- ❌ **不可修改/刪除**: `title`, `summary`, `cover_image_url`, `location`, `detail`，以及**已存在的場次**。

## 4. Session (場次) 管理

- **資料模型**: `Session` 作為一個獨立的資料集合 (collection)，透過 `event_id` 與 `Event` 關聯。
- **時間驗證**: 同一活動下的所有場次時間不可重疊。

### Session CRUD 邏輯

- **新增場次**:
  - 於建立活動時 (`POST /console/events`) 一併傳入。
  - 於更新活動時 (`PATCH /console/events/{id}`)，傳入不含 `id` 的 session 物件。

- **更新場次**:
  - 於更新活動時 (`PATCH /console/events/{id}`)，傳入包含 `id` 的 session 物件。
  - **限制**: 僅 `draft` 狀態的活動允許更新場次。

- **刪除場次**:
  - **端點**: `DELETE /console/events/{event_id}/sessions/{session_id}`。
  - **限制**: 僅 `draft` 狀態的活動允許刪除場次。

## 5. API 端點設計

### Console API (`/console/*`)

- `POST /console/events`: 建立新活動 (初始狀態為 `draft`)。
- `GET /console/events`: 取得商戶的活動列表 (支援分頁、篩選、排序)。
- `GET /console/events/{id}`: 取得單一活動的完整資料。
- `PATCH /console/events/{id}`: 部分更新活動資訊 (用於自動儲存等場景)。
- `DELETE /console/events/{id}`: 刪除活動 (**僅限 `draft` 狀態**)。
- `PUT /console/events/{id}/status`: 更新活動狀態 (`draft` -> `published`, `published` -> `archived`)。
- `DELETE /console/events/{id}/sessions/{session_id}`: 刪除活動的特定場次 (**僅限 `draft` 狀態**)。
- `PUT /console/events/{id}/form`: 新增或更新活動的報名表單。
- `GET /console/events/{id}/form`: 取得活動的報名表單。
- `DELETE /console/events/{id}/form`: 刪除活動的報名表單。

### Public API (`/events/*`)

- `GET /events`: 公開搜尋活動 (僅返回 `published` + `public` 的活動)。
- `GET /events/{id}`: 透過分享連結查詢單一活動 (僅返回 `published` 狀態的活動)。
- `GET /events/{id}/form`: 查詢公開活動的報名表單。

### Internal API (gRPC Only)

- `InternalService.GetEventById`: 供內部服務查詢任何狀態、任何商戶的活動。
- `InternalService.GetSessionById`: 供內部服務查詢任何場次。

## 6. 資料驗證規則

### 6.1 結構與必填欄位

- **Event (發布前必填)**: `title`, `cover_image_url`, `location`, `detail` (至少一個區塊), `sessions` (至少一個)。
- **Session**: `start_time`, `end_time`。
- **Location**: `name`, `address`, `place_id`, `coordinates`。
- **Detail (結構化內容)**:
  - 為一個陣列，最多 50 個區塊。
  - 每個區塊為一個物件，包含 `type` (`text` 或 `image`) 和 `data` (對應的 `text_data` 或 `image_data`)。
  - `text_data` 包含 `content` 欄位。
  - `image_data` 包含 `url` (必填), `alt`, `caption` 欄位。
- **FAQ**: 陣列，最多 20 個問答。若提供，`question` 和 `answer` 均必填。

### 6.2 欄位長度與格式限制

- `title`: 最大 60 字元。
- `summary`: 最大 160 字元。
- `detail.text_data.content`: 最大 10,000 字元。
- `faq.question`: 最大 100 字元。
- `faq.answer`: 最大 300 字元。
- **時間格式**: 所有時間欄位均使用 RFC 3339 格式。

*所有驗證規則均在 `.proto` 檔案中透過 `protoc-gen-validate` 標註，並在服務入口處由 gRPC 框架自動執行。*