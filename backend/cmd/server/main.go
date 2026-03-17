package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// 占位确保依赖被引用
var _ = jwt.New
var _ *pgx.Conn
var _ = uuid.New
var _ = websocket.DefaultDialer
var _ = godotenv.Load
var _ = bcrypt.GenerateFromPassword

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","version":"0.0.1"}`)
}

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用系统环境变量")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 注册路由
	http.HandleFunc("/api/v1/health", healthHandler)

	log.Printf("TE-DNA 2.0 服务启动，端口: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
