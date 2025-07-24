package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Event struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title         string             `bson:"title" json:"title"`
	BrandID       primitive.ObjectID `bson:"brand_id" json:"brand_id"`
	Summary       string             `bson:"summary" json:"summary"`
	Status        string             `bson:"status" json:"status"`                   // 狀態: "draft", "published", "archived"
	Visibility    string             `bson:"visibility" json:"visibility"`           // 可見性: "public", "private"
	CoverImageURL string             `bson:"cover_image_url" json:"cover_image_url"` // 封面圖片 URL
	Location      Location           `bson:"location" json:"location"`
	Sessions      []Session          `bson:"sessions" json:"sessions"`
	Detail        Detail             `bson:"detail" json:"detail"`
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

// Detail 活動詳細內容，支援富文本編輯器
type Detail struct {
	Content     string `bson:"content" json:"content"`           // 富文本內容 (HTML 或 JSON 格式)
	ContentType string `bson:"content_type" json:"content_type"` // 內容類型: "html", "json", "markdown"
}

// FAQ 常見問題
type FAQ struct {
	Question string `bson:"question" json:"question"` // 問題
	Answer   string `bson:"answer" json:"answer"`     // 回答
}

// Session 活動場次
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StartTime primitive.DateTime `bson:"start_time" json:"start_time"`
	EndTime   primitive.DateTime `bson:"end_time" json:"end_time"`
}
