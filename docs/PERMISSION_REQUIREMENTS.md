# Event 微服務權限需求文件

## 概述

本文件詳細列出 Event 微服務所需的權限檢查項目，供 API Gateway 開發者參考實作。

## 權限檢查原則

- **統一驗證**: 所有權限檢查由 API Gateway 統一處理
- **Merchant 隔離**: 確保用戶只能存取所屬 Merchant 的資源
- **請求傳遞**: 經過驗證的請求會包含用戶和 Merchant 資訊

## Console 管理 API 權限需求

### 通用權限檢查
所有 Console API (`/console/*`) 都需要進行以下檢查：

1. **用戶身份驗證**
   - 檢查用戶是否已登入
   - 驗證 JWT Token 有效性

2. **Merchant 成員驗證**  
   - 驗證用戶是否為指定 Merchant 的成員
   - 從請求路徑或參數中取得 Merchant ID
   - 確認用戶有存取該 Merchant 資源的權限

3. **Header 傳遞**
   - 驗證通過後，傳遞以下 Headers 給微服務：
     - `X-User-Id`: 用戶 ID
     - `X-User-Email`: 用戶 Email  
     - `X-User-Name`: 用戶名稱
     - `X-User-Avatar`: 用戶頭像 URL
     - `X-Merchant-Id`: Merchant ID

### 具體 API 權限

#### 1. 建立 Event
- **端點**: `POST /console/events`
- **權限**: 用戶為 Merchant 成員
- **額外檢查**: 從請求 body 中的 `merchant_id` 進行 Merchant 成員驗證

#### 2. 查看 Event 列表
- **端點**: `GET /console/events`
- **權限**: 用戶為 Merchant 成員
- **範圍**: 只返回該 Merchant 下的 Events

#### 3. 查看單一 Event
- **端點**: `GET /console/events/{id}`
- **權限**: 用戶為 Merchant 成員 + Event 屬於該 Merchant
- **檢查方式**: 需先查詢 Event 的 merchant_id，再驗證權限

#### 4. 更新 Event
- **端點**: `PUT /console/events/{id}`, `PATCH /console/events/{id}`
- **權限**: 用戶為 Merchant 成員 + Event 屬於該 Merchant
- **額外規則**: published 狀態的 Event 不可修改（由微服務處理）

#### 5. 刪除 Event
- **端點**: `DELETE /console/events/{id}`
- **權限**: 用戶為 Merchant 成員 + Event 屬於該 Merchant
- **額外規則**: published 狀態的 Event 不可刪除（由微服務處理）

#### 6. 變更 Event 狀態
- **端點**: `PUT /console/events/{id}/status`
- **權限**: 用戶為 Merchant 成員 + Event 屬於該 Merchant

## Public API 權限需求

### 1. 公開搜尋
- **端點**: `GET /events`
- **權限**: 無需身份驗證（匿名存取）
- **限制**: 僅返回 `status: "published"` 且 `visibility: "public"` 的 Events

### 2. 分享連結查詢
- **端點**: `GET /events/{id}`
- **權限**: 無需身份驗證（匿名存取）
- **限制**: 僅返回 `status: "published"` 的 Events（不限 visibility）

## 實作建議

### 1. Merchant 成員驗證邏輯
```
IF user_id NOT IN merchant_members(merchant_id) THEN
    RETURN 403 PermissionDenied
END IF
```

### 2. 資源隔離檢查
```
IF event.merchant_id != user.merchant_id THEN
    RETURN 403 PermissionDenied  
END IF
```

### 3. 錯誤回應
權限不足時返回：
- HTTP Status: 403 Forbidden
- gRPC Code: 7 (PermissionDenied)
- Message: "權限不足" 或 "Access denied"

## Headers 設定需求

### 新增 AllowedHeaders
需要在 API Gateway 的 CORS 設定中新增：
- `X-Merchant-Id`

### Headers 傳遞映射
```
x-user-id → X-User-Id
x-user-email → X-User-Email  
x-user-name → X-User-Name
x-user-avatar → X-User-Avatar
x-merchant-id → X-Merchant-Id
```

## 權限檢查流程圖

```
請求 → API Gateway
    ↓
身份驗證（JWT）
    ↓
Merchant 成員驗證
    ↓  
設定 Headers
    ↓
轉發到 Event 微服務
    ↓
微服務處理業務邏輯
```

## 注意事項

1. **不需要角色細分**: 目前所有 Merchant 成員都有相同權限
2. **業務規則檢查**: 狀態轉換等業務規則由微服務處理
3. **快取考量**: 未來可考慮快取 Merchant 成員資訊以提升效能
4. **錯誤處理**: 統一錯誤格式，避免洩露內部資訊