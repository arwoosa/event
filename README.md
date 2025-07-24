* 更改Makefile BINARY變數成新專案名稱，並執行make mod
* 如果要使用 Wire 進行依賴注入並使用 Protobuf 定義 gRPC 服務，要重新生成相關檔案
  * cd cmd/server
  * go run github.com/google/wire/cmd/wire
  * 會生成 `wire_gen.go`


* dao/mongodb/migration 可以定義collection以及index，會在初始化的時候創建該collection