package docs

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Event struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title         string             `bson:"title" json:"title"`
	MerchantID    string             `bson:"merchant_id" json:"merchant_id"`
	Summary       string             `bson:"summary" json:"summary"`
	Status        string             `bson:"status" json:"status"`                   // 狀態: "draft", "published", "archived"
	Visibility    string             `bson:"visibility" json:"visibility"`           // 可見性: "public", "private"
	CoverImageURL string             `bson:"cover_image_url" json:"cover_image_url"` // 封面圖片 URL
	Location      Location           `bson:"location" json:"location"`
	Sessions      []Session          `bson:"sessions" json:"sessions"`
	Detail        []DetailBlock      `bson:"detail" json:"detail"`
	FAQ           []FAQ              `bson:"faq" json:"faq"`
	CreatedAt     primitive.DateTime `bson:"created_at" json:"created_at"`
	CreatedBy     primitive.ObjectID `bson:"created_by" json:"created_by"`
	UpdatedAt     primitive.DateTime `bson:"updated_at" json:"updated_at"`
	UpdatedBy     primitive.ObjectID `bson:"updated_by" json:"updated_by"`
}

// Location 地點資訊，支援 Google Maps 整合
type Location struct {
	Name        string       `bson:"name" json:"name"`               // 地點名稱
	Address     string       `bson:"address" json:"address"`         // 詳細地址
	PlaceID     string       `bson:"place_id" json:"place_id"`       // Google Places API Place ID
	Coordinates GeoJSONPoint `bson:"coordinates" json:"coordinates"` // 經緯度資訊
}

type GeoJSONPoint struct {
	Type        string    `bson:"type" json:"type"`               // 固定為 "Point"
	Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [lng, lat]
}

// DetailBlock 活動詳細內容區塊，支援多種類型內容
type DetailBlock struct {
	Type string      `bson:"type" json:"type"` // 區塊類型: "text", "image"
	Data interface{} `bson:"data" json:"data"` // 區塊資料，根據 Type 可為 TextData 或 ImageData
}

// TextData 文字區塊資料
type TextData struct {
	Content string `bson:"content" json:"content"` // 文字內容，最大 10,000 字
}

// ImageData 圖片區塊資料
type ImageData struct {
	URL     string `bson:"url" json:"url"`         // 圖片 URL (必填)
	Alt     string `bson:"alt" json:"alt"`         // 替代文字，最大 200 字
	Caption string `bson:"caption" json:"caption"` // 圖片說明，最大 500 字
}

// FAQ 常見問題
type FAQ struct {
	Question string `bson:"question" json:"question"` // 問題
	Answer   string `bson:"answer" json:"answer"`     // 回答
}

// Session 活動場次
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`             // 場次名稱 (可選)
	Capacity  *int32             `bson:"capacity" json:"capacity"`     // 容量限制 (可選，null 表示不限制)
	StartTime primitive.DateTime `bson:"start_time" json:"start_time"` // 開始時間
	EndTime   primitive.DateTime `bson:"end_time" json:"end_time"`     // 結束時間
}
