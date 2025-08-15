# Event 微服務需求規格

## 需求釐清紀錄

### 1. 權限管理和角色定義

**Event CRUD 權限：**
- 建立：登入使用者且為該 Merchant 成員（API Gateway 會驗證並傳遞 user_id, merchant_id 等 header 資訊）
- 編輯：該 Merchant 的成員即可編輯
- 刪除：該 Merchant 的成員即可刪除
- 狀態轉換權限：該 Merchant 的成員，但受狀態規則限制

**角色系統：**
- 目前為 Console 端（業主管理後台）
- 角色分類：一般用戶、Merchant 管理成員
- 暫無系統管理員角色
- MerchantID 用途：區分不同企業群組，只有該 Merchant 成員才能存取資源
- 注意：前台使用者與後台用戶分開管理

**跨服務權限：**
- API Gateway 負責身份驗證和權限檢查
- 微服務接收 header 中的 user_id, merchant_id 等資訊
- 不需調用其他微服務驗證權限
- CreatedBy/UpdatedBy 直接使用傳入的 user_id

### 2. Event 狀態轉換規則

**狀態定義：**
- `draft`（草稿）
- `published`（發布）  
- `archived`（下架）

**可見性定義：**（僅用於前台用戶查詢）
- `public`（公開）：可透過搜尋功能找到
- `private`（私人）：只能透過分享連結查看，不會出現在搜尋結果中
- 預設值：`private`
- 管理權限：Merchant 成員可修改（**備註：需考量用戶在 Merchant 中的細分權限**）

**狀態轉換規則（重要更新）：**
- **單向流程**：`Draft → Published → Archived`（不可逆轉）
- **草稿狀態（Draft）**：可自由修改、刪除、轉為發布狀態
- **發布狀態（Published）**：
  - 活動設定有限制編輯（詳見編輯權限章節）
  - 不可刪除整個 Event
  - 只能轉為下架狀態
- **下架狀態（Archived）**：
  - 完全不可修改任何內容
  - 不可刪除
  - 不可變更狀態（包含無法回到 Published）
  - 建議：使用者需複製成新的 Draft 來修改

**Published 狀態編輯權限詳細規範：**

*活動設定權限：*
- ❌ **不可編輯**：活動封面、活動標題、活動地點、活動簡介、活動內容
- ✅ **可編輯**：FAQ

*場次設定權限：*
- ❌ **不可編輯**：現有場次的時間資訊
- ✅ **可操作**：新增場次
- ✅ **有條件刪除場次**：
  - 條件1：該場次沒有任何訂單（需呼叫 OrderService 確認）
  - 條件2：不是最後一個場次（Event 至少需保留一個 Session）
  - 兩個條件必須同時滿足

**訂單檢查機制：**
- **狀態轉換檢查**：`Published → Archived` 需呼叫 OrderService 確認所有訂單都是 Cancelled 和 Refunded
- **場次刪除檢查**：呼叫 OrderService 檢查特定 Session 是否有訂單
- 訂單定義邏輯由訂單微服務負責處理
- 暫時無快取機制（**建議：未來可考慮快取以提升性能**）

### 3. Session 管理規則

**Session 與 Event 關係：**
- 一個 Event 可以有多個 Session
- 每個 Event 至少需要一個 Session
- Session 透過 Event 管理，不獨立存在
- **重要變更**：Session 不再儲存 merchant_id，透過 Event 繼承權限範圍

**Session 資料結構更新：**
- **新增欄位**：
  - `name`：場次名稱（可選，可空白）
  - `capacity`：容量限制（可選，可不限制）
- **移除欄位**：
  - `merchant_id`：改由 Event 統一管理

**Session 時間驗證規則：**
- 同一 Event 的 Session 時間不可重疊
- 重疊定義：start_time 和 end_time 的組合必須唯一
- StartTime 必須小於 EndTime
- 無時長限制（最小/最大時長）

**Session CRUD 邏輯重構：**
- **新增場次**：透過 Event PATCH API 或獨立新增 API
- **更新場次**：透過 Event PATCH API（時間、名稱、容量）
- **刪除場次**：獨立的 DELETE API，不再透過 PATCH 處理
- **權限檢查**：Published 狀態下的刪除需要 OrderService 確認

### 4. API 端點設計

**Console 管理 API：(路徑需要多一個console)**
- 建立 Event（包含多個 Session）
- 取得 Event 列表
  - 支援分頁（page-based pagination）
  - 支援無限滾動（cursor-based pagination）
  - 篩選功能：狀態（draft/published/archived）、可見性（public/private）、Session 時間範圍
- 取得單一 Event 詳細資料
- 更新 Event
  - PUT：全欄位更新
  - PATCH：部分欄位更新（支援前端自動儲存，減少資料傳輸）
- 刪除 Event
- 變更 Event 狀態（獨立端點）
- 必填欄位：title, merchant_id, sessions（至少一個）, cover_image_url, detail.content, location, visibility

**前台用戶 API：**
- GET /events：公開搜尋（只返回 published + public 的 Event）
  - 支援 merchant_id 參數篩選
  - 支援 title 全文搜尋
  - 支援地理位置範圍搜尋
  - 支援 Session 時間範圍篩選
- GET /events/{id}：分享連結查詢（只能查看 published 狀態，不限 visibility）

**Session 管理設計（重要更新）：**
- **新增/更新場次**：透過 Event PATCH API（PATCH /console/events/{id}）
- **刪除場次**：獨立 DELETE API（DELETE /console/events/{event_id}/sessions/{session_id}）
- **權限檢查**：Published 狀態需透過 OrderService 檢查訂單狀態
- Session 與 Event 保持資料一致性

**不實作功能：**
- 批次操作（建立/更新/刪除/狀態變更）
- 複製 Event 功能
- Event 統計資訊
- Event 預覽功能

**新增 API 端點：**
- **IsPublished API**：專門給 OrderService 呼叫，檢查 Event 是否為 Published 狀態
- **獨立 Session 刪除 API**：處理場次刪除邏輯與權限檢查

### 5. 資料驗證規則

**必填欄位驗證：**
- Event 必填欄位：title, merchant_id, sessions（至少一個）, cover_image_url, detail.content, location, visibility
- **Session 必填欄位**：start_time, end_time
- **Session 可選欄位**：name（場次名稱，可空白）, capacity（容量限制，可空值表示不限制）
- Location 必填欄位：name, address, place_id, coordinates（支援 Google Maps 整合和地理位置搜尋）
- Detail 必填欄位：content（不可空白）
- FAQ：為可選欄位，但若新增則 question, answer 都必填（最多 20 個）
- Visibility 預設值：private

**欄位長度限制：**（**備註：需與 PO 確認最終規格**）
- title：最大 60 字
- summary：最大 160 字
- Location.name, address：**待評估是否需要限制**
- Detail.content：最大 64KB（考慮 HTML/Markdown 語法）
- FAQ question：最大 100 字
- FAQ answer：最大 300 字
- FAQ 數量：最多 20 個

**特殊驗證規則：**
- 時間格式：一律使用 RFC 3339
- 狀態轉換：暫時使用必填欄位驗證（**草稿轉發布的完整性驗證待確認**）
- 經緯度驗證：暫時不實作

**Location 資料結構技術建議：**
您更新的 GeoJSONPoint 結構很好，符合 MongoDB 地理空間索引標準。建議必填欄位：
- name：必填（用於顯示）
- address：必填（用於搜尋和顯示）
- place_id：建議必填（Google Maps 整合的關鍵）
- coordinates：可選（可從 place_id 取得）

### 6. 檔案上傳和媒體處理

**Event 封面圖片：**
- 每個 Event 需要一張封面圖片
- Event 資料結構需新增 cover_image_url 欄位

**圖片處理流程：**
- Event 微服務僅儲存圖片 URL，不處理實際檔案
- 前端透過媒體微服務上傳圖片後取得 URL
- Detail 內容中的圖片也採用相同流程（**重要：不可使用 base64 格式**）

**媒體儲存架構：**
- 使用 Cloudflare 圖片儲存服務
- 由獨立媒體微服務處理檔案上傳
- **未來考慮：圖片壓縮和多尺寸生成**

### 7. 搜尋、分頁、篩選需求

**分頁功能：**
- Page-based pagination（傳統分頁）
- Cursor-based pagination（無限滾動）

**篩選功能：**
- 按狀態篩選（draft/published/archived）
- 按可見性篩選（public/private）
- 按 Session 時間範圍篩選（查詢特定時間範圍的活動）

**前台用戶查詢限制：**
- Event 必須同時滿足 `status: "published"` 和 `visibility: "public"` 才能被搜尋到
- `private` Event 只能透過分享連結（直接 ID 查詢）存取
- **分享連結實作：**GET /events/{event_id}（**備註：需與 PO 討論具體實作方式**）

**搜尋功能：**
- 按 Event title 全文搜尋（使用 MongoDB 文字索引）
- **未來考慮：**跨欄位全文搜尋引擎整合

**排序功能：**
- 按建立時間排序（created_at）
- 按更新時間排序（updated_at）
- 按 Session 開始時間排序（取最早的 session.start_time）

**進階功能：**
- **地理位置範圍搜尋**（附近活動，使用 MongoDB 地理空間查詢）
- 標籤系統：暫不實作

**資料結構設計考量：**
**Session 獨立 Collection vs 嵌入式設計：**

**目前嵌入式設計優點：**
- 資料一致性佳，原子性操作
- 減少 Join 查詢，讀取效能好
- API 設計簡單

**獨立 Collection 優點：**
- Session 時間範圍查詢更容易
- 更靈活的索引策略
- 未來擴展性佳

**技術建議：暫時保持嵌入式設計**，理由：
1. Session 與 Event 強相關，很少單獨查詢
2. 可透過 MongoDB 複合索引解決時間查詢問題
3. 避免過早優化，保持架構簡單

### 8. 最終確認事項

**已整合至前述章節的內容：**
- Location 必填欄位已更新至第 5 章
- Event 必填欄位已更新至第 4、5 章
- API 端點設計已更新至第 4 章
- 前台用戶 API 已整合至第 4 章

**待 PO 確認的項目：**
- 欄位長度限制規格
- 草稿轉發布的完整性驗證規則
- Merchant 成員的細分權限控制
- 分享連結的具體實作方式

**公司範本設計規範：**
- **回應格式**：統一使用 `api.Response` 結構（status, code, message, data）
- **錯誤處理**：使用 gRPC status codes，標準錯誤訊息
- **Header 管理**：支援 X-User-Id, X-User-Email, X-User-Name, X-User-Avatar
- **分頁機制**：Cursor-based pagination 使用 PageToken 編碼（24小時過期）
- **協議支援**：gRPC + HTTP Gateway 雙協議
- **成功回應碼**：使用 `1000` 作為成功狀態碼

**需要新增到 Header 管理的欄位：**
- X-Merchant-Id：Merchant 識別（需新增到 AllowedHeaders）

**技術實作注意事項：**
- MongoDB 索引策略：地理空間索引、複合索引支援時間範圍查詢
- 訂單微服務整合：狀態轉換時的訂單檢查
- 前後台 API 分離：不同的權限和資料過濾邏輯