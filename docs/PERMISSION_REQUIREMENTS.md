# Event 微服務權限需求文件

## 概述

本文件詳細列出 Event 微服務的權限設計和檢查項目。本文檔旨在闡明服務自身的職責以及對 API Gateway 的依賴。

## 權限檢查原則

- **信任邊界 (Trust Boundary)**: Event 微服務信任來自 API Gateway 的請求。所有對外 API 的身份驗證和權限檢查均由 API Gateway 統一處理。
- **集中式授權 (Centralized Authorization)**: API Gateway 整合 Ory Keto，負責資源級別的存取控制。例如，在請求 `GET /console/events/{id}` 時，Gateway 會驗證發起請求的用戶是否有權限存取 `{id}` 這個資源。
- **商戶隔離 (Merchant Isolation)**: 對於**列表查詢** (如 `GET /console/events`)，服務內部會使用 API Gateway 傳遞的 `X-Merchant-Id` Header 來實現嚴格的資料隔離。

## Console 管理 API (`/console/*`)

### 通用權限模型

所有 Console API 都依賴 API Gateway 執行以下檢查：

1.  **用戶身份驗證**: 檢查用戶 JWT Token 的有效性。
2.  **資源權限驗證 (使用 Ory Keto)**: 針對單一資源的操作 (如 `GET`, `PATCH`, `DELETE` 到 `/{id}` 的請求)，Gateway 會驗證用戶對該資源 ID 的存取權限。

驗證通過後，API Gateway 必須將以下 Headers 傳遞給 Event 微服務：

- `X-User-Id`, `X-User-Email`, `X-User-Name`, `X-User-Avatar`
- `X-Merchant-Id`

### 各 API 的資料存取邏輯

- **`POST /console/events` (建立 Event)**
  - **邏輯**: 這是唯一一個在服務層設定資源歸屬的寫入操作。由於資源尚不存在，Gateway 無法檢查權限。服務會使用 Header 中的 `X-Merchant-Id` 作為新 Event 的 `merchant_id`。

- **`GET /console/events` (查看 Event 列表)**
  - **邏輯**: 這是一個集合查詢，權限無法在 Gateway 層針對每一筆資料進行檢查。因此，服務**必須**使用 `X-Merchant-Id` 作為資料庫查詢條件，確保只返回該商戶下的 Events。

- **`GET /console/events/{id}` (查看單一 Event)**
  - **邏輯**: Gateway 已驗證用戶對 `{id}` 的權限。因此，服務層**不再需要**使用 `merchant_id` 進行二次過濾，而是直接透過 `_id: {id}` 查詢資源以提升效能。如果資源不存在，則返回 `Not Found`。

- **`PATCH /console/events/{id}` (更新 Event)**
  - **邏輯**: 同上。服務信任 Gateway 的授權，直接透過 ID 查找並更新資源。

- **`DELETE /console/events/{id}` (刪除 Event)**
  - **邏輯**: 同上。服務信任 Gateway 的授權，直接透過 ID 查找並刪除資源。

- **`PUT /console/events/{id}/status` (變更 Event 狀態)**
  - **邏輯**: 同上。服務信任 Gateway 的授權，直接透過 ID 查找並變更狀態。

- **`/console/events/{id}/form` (報名表單操作)**
  - **邏輯**: 同上。所有對表單的操作都信賴 Gateway 對其主資源 Event `{id}` 的權限校驗。

- **`/console/events/{id}/sessions/{sessionId}` (場次操作)**
  - **邏輯**: 同上。所有對場次的操作都信賴 Gateway 對其主資源 Event `{id}` 的權限校驗。

## Public API (`/events/*`)

Public API 無需身份驗證，權限控制由服務內部的業務邏輯實現。

- **`GET /events` (公開搜尋)**
  - **限制**: 服務邏輯只返回 `status: "published"` 且 `visibility: "public"` 的 Events。

- **`GET /events/{id}` (分享連結查詢)**
  - **限制**: 服務邏輯只返回 `status: "published"` 的 Events（不限 `visibility`）。如果活動為 `draft` 或 `archived` 狀態，將返回 `Not Found`。

## Internal API (gRPC only)

- **服務**: `InternalService`
- **權限**: **無任何權限檢查**。此服務僅供內部其他微服務（如 Order Service）透過 gRPC 呼叫，且必須部署在受信任的內部網路中。它能夠跨商戶查詢任何狀態的活動或場次資料。