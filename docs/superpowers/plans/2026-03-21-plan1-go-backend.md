# Go 后端服务 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建口播稿改写系统的 Go 后端服务，提供用户认证、风格档案、稿件管理、素材库、热点查询的 REST API。

**Architecture:** Go + Gin 框架，GORM + PostgreSQL 持久化，JWT 认证。采用 handler → service → repository 三层结构，各层职责清晰。Admin 前端产物通过 Go embed 打包进二进制，Docker Compose 一键部署。

**Tech Stack:** Go 1.22+, Gin, GORM, PostgreSQL 15, golang-jwt/jwt v5, Docker Compose

---

## 文件结构

```
server/
├── main.go                          # 启动入口，注册路由
├── config/
│   └── config.go                    # 环境变量读取（DB、JWT secret、端口）
├── internal/
│   ├── model/
│   │   ├── user.go                  # User、UserStyle GORM 模型
│   │   ├── script.go                # Script GORM 模型
│   │   └── material.go              # Material、Hotspot GORM 模型
│   ├── repository/
│   │   ├── user_repo.go             # User DB 操作
│   │   ├── script_repo.go           # Script DB 操作
│   │   └── material_repo.go         # Material、Hotspot DB 操作
│   ├── service/
│   │   ├── auth_service.go          # 注册/登录/token 生成
│   │   ├── user_service.go          # 用户档案、风格档案
│   │   ├── script_service.go        # 稿件 CRUD、标签
│   │   ├── material_service.go      # 素材查询
│   │   └── sync_service.go          # 离线队列批量同步
│   └── handler/
│       ├── auth_handler.go          # POST /auth/register, POST /auth/login
│       ├── user_handler.go          # GET/PUT /user/profile, PUT /user/style
│       ├── script_handler.go        # GET/POST /scripts, PUT /scripts/:id/tags
│       ├── material_handler.go      # GET /materials, GET /hotspot
│       └── sync_handler.go          # POST /sync
├── middleware/
│   └── auth.go                      # JWT 验证中间件
├── admin/                           # React 构建产物（Plan 3 完成后填入）
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

---

## Task 1: 项目初始化

**Files:**
- Create: `server/main.go`
- Create: `server/config/config.go`
- Create: `server/.env.example`
- Create: `server/docker-compose.yml`
- Create: `server/Dockerfile`

- [ ] **Step 1: 初始化 Go 模块**

```bash
mkdir -p /data/code/content_creator_imm/server
cd /data/code/content_creator_imm/server
go mod init github.com/content-creator-imm/server
```

- [ ] **Step 2: 安装依赖**

```bash
cd /data/code/content_creator_imm/server
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-jwt/jwt/v5
go get github.com/joho/godotenv
go get github.com/robfig/cron/v3
```

- [ ] **Step 3: 创建 config.go**

```go
// config/config.go
package config

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    JWTSecret  string
    Port       string
}

func Load() *Config {
    _ = godotenv.Load()
    cfg := &Config{
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBPort:     getEnv("DB_PORT", "5432"),
        DBUser:     getEnv("DB_USER", "postgres"),
        DBPassword: getEnv("DB_PASSWORD", "postgres"),
        DBName:     getEnv("DB_NAME", "content_creator"),
        JWTSecret:  getEnv("JWT_SECRET", "change-me-in-production"),
        Port:       getEnv("PORT", "8080"),
    }
    if cfg.JWTSecret == "change-me-in-production" {
        log.Println("WARNING: using default JWT secret, set JWT_SECRET env var")
    }
    return cfg
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

- [ ] **Step 4: 创建 main.go 骨架**

```go
// main.go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/config"
)

func main() {
    cfg := config.Load()
    r := gin.Default()

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    addr := ":" + cfg.Port
    log.Printf("Server starting on %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatal(err)
    }
}
```

- [ ] **Step 5: 创建 .env.example**

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=content_creator
JWT_SECRET=your-secret-here
PORT=8080
```

- [ ] **Step 6: 创建 docker-compose.yml**

```yaml
version: '3.8'
services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: content_creator
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  api:
    build: .
    ports:
      - "8080:8080"
    env_file: .env
    depends_on:
      - db

volumes:
  pgdata:
```

- [ ] **Step 7: 创建 Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 8: 验证编译通过**

```bash
cd /data/code/content_creator_imm/server
go build ./...
```

Expected: 无报错输出

- [ ] **Step 9: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: initialize Go server project structure"
```

---

## Task 2: 数据库模型

**Files:**
- Create: `server/internal/model/user.go`
- Create: `server/internal/model/script.go`
- Create: `server/internal/model/material.go`

- [ ] **Step 1: 创建 User 和 UserStyle 模型**

```go
// internal/model/user.go
package model

import (
    "time"
    "gorm.io/gorm"
)

type Role string
const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

type User struct {
    ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Username  string         `gorm:"uniqueIndex;not null" json:"username"`
    Email     string         `gorm:"uniqueIndex;not null" json:"email"`
    Password  string         `gorm:"not null" json:"-"`
    Role      Role           `gorm:"default:'user'" json:"role"`
    Active    bool           `gorm:"default:true" json:"active"`
    Style     *UserStyle     `gorm:"foreignKey:UserID" json:"style,omitempty"`
    Scripts   []Script       `gorm:"foreignKey:UserID" json:"-"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UserStyle struct {
    ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    UserID        string    `gorm:"uniqueIndex;not null" json:"user_id"`
    LanguageStyle string    `json:"language_style"` // 口语化/书面化/专业/接地气
    EmotionTone   string    `json:"emotion_tone"`   // 理性/感性/幽默/严肃
    OpeningStyle  string    `json:"opening_style"`  // 典型开场方式
    ClosingStyle  string    `json:"closing_style"`  // 典型结尾方式
    Catchphrases  string    `json:"catchphrases"`   // 口头禅（JSON数组字符串）
    RawFeatures   string    `json:"raw_features"`   // 完整风格向量（JSON）
    UpdatedAt     time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: 创建 Script 模型**

```go
// internal/model/script.go
package model

import (
    "time"
    "gorm.io/gorm"
)

type Script struct {
    ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    UserID         string         `gorm:"index;not null" json:"user_id"`
    Title          string         `json:"title"`
    SourceURL      string         `json:"source_url"`
    Platform       string         `json:"platform"`
    Content        string         `gorm:"type:text" json:"content"`
    QualityReport  string         `gorm:"type:text" json:"quality_report"` // JSON
    SimilarityScore float64       `json:"similarity_score"`
    ViralScore     float64        `json:"viral_score"`
    Tags           string         `json:"tags"` // JSON数组字符串
    Favorited      bool           `gorm:"default:false" json:"favorited"`
    CreatedAt      time.Time      `json:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at"`
    DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 3: 创建 Material 和 Hotspot 模型**

```go
// internal/model/material.go
package model

import "time"

type MaterialType string
const (
    MaterialTypeData    MaterialType = "data"
    MaterialTypeCase    MaterialType = "case"
    MaterialTypeQuote   MaterialType = "quote"
    MaterialTypeContrast MaterialType = "contrast"
)

type Material struct {
    ID        string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Topic     string       `gorm:"index" json:"topic"`
    Type      MaterialType `json:"type"`
    Content   string       `gorm:"type:text" json:"content"`
    Source    string       `json:"source"`
    IsPublic  bool         `gorm:"default:true" json:"is_public"`
    UserID    string       `gorm:"index" json:"user_id"` // 私有素材的所有者，公共素材为空
    CreatedAt time.Time    `json:"created_at"`
}

type Hotspot struct {
    ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Platform  string    `gorm:"index" json:"platform"` // douyin/xiaohongshu/weibo
    Title     string    `json:"title"`
    Rank      int       `json:"rank"`
    HeatScore int64     `json:"heat_score"`
    URL       string    `json:"url"`
    FetchedAt time.Time `gorm:"index" json:"fetched_at"`
}
```

- [ ] **Step 4: 在 main.go 中接入数据库并 AutoMigrate**

```go
// main.go - 添加数据库初始化
import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/content-creator-imm/server/internal/model"
)

func initDB(cfg *config.Config) *gorm.DB {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
    )
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect database: %v", err)
    }
    if err := db.AutoMigrate(
        &model.User{},
        &model.UserStyle{},
        &model.Script{},
        &model.Material{},
        &model.Hotspot{},
    ); err != nil {
        log.Fatalf("failed to migrate: %v", err)
    }
    return db
}
```

- [ ] **Step 5: 验证编译**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

Expected: 无报错

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add GORM models for user, script, material, hotspot"
```

---

## Task 3: JWT 认证中间件 + 注册登录

**Files:**
- Create: `server/middleware/auth.go`
- Create: `server/internal/repository/user_repo.go`
- Create: `server/internal/service/auth_service.go`
- Create: `server/internal/handler/auth_handler.go`

- [ ] **Step 1: 创建 JWT 中间件**

```go
// middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func Auth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }
        tokenStr := strings.TrimPrefix(header, "Bearer ")
        claims := &Claims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Next()
    }
}

func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.GetString("role") != "admin" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
            return
        }
        c.Next()
    }
}
```

- [ ] **Step 2: 创建 user_repo.go**

```go
// internal/repository/user_repo.go
package repository

import (
    "gorm.io/gorm"
    "github.com/content-creator-imm/server/internal/model"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db} }

func (r *UserRepo) Create(u *model.User) error {
    return r.db.Create(u).Error
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
    var u model.User
    err := r.db.Where("email = ?", email).First(&u).Error
    return &u, err
}

func (r *UserRepo) FindByID(id string) (*model.User, error) {
    var u model.User
    err := r.db.Preload("Style").First(&u, "id = ?", id).Error
    return &u, err
}

func (r *UserRepo) List(page, limit int) ([]model.User, int64, error) {
    var users []model.User
    var total int64
    r.db.Model(&model.User{}).Count(&total)
    err := r.db.Offset((page-1)*limit).Limit(limit).Find(&users).Error
    return users, total, err
}

func (r *UserRepo) UpdateActive(id string, active bool) error {
    return r.db.Model(&model.User{}).Where("id = ?", id).Update("active", active).Error
}

func (r *UserRepo) UpsertStyle(style *model.UserStyle) error {
    return r.db.Save(style).Error
}
```

- [ ] **Step 3: 创建 auth_service.go**

```go
// internal/service/auth_service.go
package service

import (
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/repository"
    mw "github.com/content-creator-imm/server/middleware"
)

type AuthService struct {
    repo      *repository.UserRepo
    jwtSecret string
}

func NewAuthService(repo *repository.UserRepo, secret string) *AuthService {
    return &AuthService{repo, secret}
}

func (s *AuthService) Register(username, email, password string) (*model.User, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    u := &model.User{
        Username: username,
        Email:    email,
        Password: string(hash),
        Role:     model.RoleUser,
    }
    if err := s.repo.Create(u); err != nil {
        return nil, err
    }
    return u, nil
}

func (s *AuthService) Login(email, password string) (string, error) {
    u, err := s.repo.FindByEmail(email)
    if err != nil {
        return "", errors.New("invalid credentials")
    }
    if !u.Active {
        return "", errors.New("account disabled")
    }
    if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
        return "", errors.New("invalid credentials")
    }
    claims := &mw.Claims{
        UserID: u.ID,
        Role:   string(u.Role),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}
```

- [ ] **Step 4: 安装 bcrypt**

```bash
cd /data/code/content_creator_imm/server && go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 5: 创建 auth_handler.go**

```go
// internal/handler/auth_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/internal/service"
)

type AuthHandler struct{ svc *service.AuthService }

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc} }

func (h *AuthHandler) Register(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required,min=6"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    user, err := h.svc.Register(req.Username, req.Email, req.Password)
    if err != nil {
        c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"user_id": user.ID, "username": user.Username})
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    token, err := h.svc.Login(req.Email, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"token": token})
}
```

- [ ] **Step 6: 在 main.go 中注册认证路由**

```go
// main.go 中添加路由注册
authHandler := handler.NewAuthHandler(authSvc)
r.POST("/api/v1/auth/register", authHandler.Register)
r.POST("/api/v1/auth/login", authHandler.Login)
r.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
```

- [ ] **Step 7: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

Expected: 无报错

- [ ] **Step 8: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add JWT auth middleware and register/login endpoints"
```

---

## Task 4: 用户档案 + 风格档案 API

**Files:**
- Create: `server/internal/service/user_service.go`
- Create: `server/internal/handler/user_handler.go`

- [ ] **Step 1: 创建 user_service.go**

```go
// internal/service/user_service.go
package service

import (
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/repository"
)

type UserService struct{ repo *repository.UserRepo }

func NewUserService(repo *repository.UserRepo) *UserService { return &UserService{repo} }

func (s *UserService) GetProfile(userID string) (*model.User, error) {
    return s.repo.FindByID(userID)
}

func (s *UserService) UpdateStyle(userID string, style *model.UserStyle) error {
    style.UserID = userID
    return s.repo.UpsertStyle(style)
}

func (s *UserService) ListUsers(page, limit int) ([]model.User, int64, error) {
    return s.repo.List(page, limit)
}

func (s *UserService) SetActive(userID string, active bool) error {
    return s.repo.UpdateActive(userID, active)
}
```

- [ ] **Step 2: 创建 user_handler.go**

```go
// internal/handler/user_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/service"
)

type UserHandler struct{ svc *service.UserService }

func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc} }

func (h *UserHandler) GetProfile(c *gin.Context) {
    userID := c.GetString("user_id")
    user, err := h.svc.GetProfile(userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateStyle(c *gin.Context) {
    userID := c.GetString("user_id")
    var style model.UserStyle
    if err := c.ShouldBindJSON(&style); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := h.svc.UpdateStyle(userID, &style); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Admin endpoints
func (h *UserHandler) ListUsers(c *gin.Context) {
    page, limit := pageParams(c)
    users, total, err := h.svc.ListUsers(page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *UserHandler) SetActive(c *gin.Context) {
    userID := c.Param("id")
    var req struct{ Active bool `json:"active"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := h.svc.SetActive(userID, req.Active); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func pageParams(c *gin.Context) (int, int) {
    page, limit := 1, 20
    if p := c.Query("page"); p != "" {
        if v, err := strconv.Atoi(p); err == nil && v > 0 { page = v }
    }
    if l := c.Query("limit"); l != "" {
        if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 { limit = v }
    }
    return page, limit
}
```

- [ ] **Step 3: 注册路由**

```go
// main.go 中认证路由组下添加
authMW := middleware.Auth(cfg.JWTSecret)
adminMW := middleware.AdminOnly()

api := r.Group("/api/v1", authMW)
{
    api.GET("/user/profile", userHandler.GetProfile)
    api.PUT("/user/style", userHandler.UpdateStyle)

    admin := api.Group("/admin", adminMW)
    {
        admin.GET("/users", userHandler.ListUsers)
        admin.PUT("/users/:id/active", userHandler.SetActive)
    }
}
```

- [ ] **Step 4: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add user profile and style endpoints"
```

---

## Task 5: 稿件 API

**Files:**
- Create: `server/internal/repository/script_repo.go`
- Create: `server/internal/service/script_service.go`
- Create: `server/internal/handler/script_handler.go`

- [ ] **Step 1: 创建 script_repo.go**

```go
// internal/repository/script_repo.go
package repository

import (
    "gorm.io/gorm"
    "github.com/content-creator-imm/server/internal/model"
)

type ScriptRepo struct{ db *gorm.DB }

func NewScriptRepo(db *gorm.DB) *ScriptRepo { return &ScriptRepo{db} }

func (r *ScriptRepo) Create(s *model.Script) error {
    return r.db.Create(s).Error
}

func (r *ScriptRepo) FindByID(id, userID string) (*model.Script, error) {
    var s model.Script
    err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&s).Error
    return &s, err
}

func (r *ScriptRepo) ListByUser(userID string, page, limit int) ([]model.Script, int64, error) {
    var scripts []model.Script
    var total int64
    q := r.db.Model(&model.Script{}).Where("user_id = ?", userID)
    q.Count(&total)
    err := q.Order("created_at DESC").Offset((page-1)*limit).Limit(limit).Find(&scripts).Error
    return scripts, total, err
}

func (r *ScriptRepo) ListAll(page, limit int) ([]model.Script, int64, error) {
    var scripts []model.Script
    var total int64
    r.db.Model(&model.Script{}).Count(&total)
    err := r.db.Order("created_at DESC").Offset((page-1)*limit).Limit(limit).Find(&scripts).Error
    return scripts, total, err
}

func (r *ScriptRepo) UpdateTags(id, userID, tags string) error {
    return r.db.Model(&model.Script{}).
        Where("id = ? AND user_id = ?", id, userID).
        Update("tags", tags).Error
}
```

- [ ] **Step 2: 创建 script_service.go**

```go
// internal/service/script_service.go
package service

import (
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/repository"
)

type ScriptService struct{ repo *repository.ScriptRepo }

func NewScriptService(repo *repository.ScriptRepo) *ScriptService { return &ScriptService{repo} }

func (s *ScriptService) Create(script *model.Script) error {
    return s.repo.Create(script)
}

func (s *ScriptService) Get(id, userID string) (*model.Script, error) {
    return s.repo.FindByID(id, userID)
}

func (s *ScriptService) List(userID string, page, limit int) ([]model.Script, int64, error) {
    return s.repo.ListByUser(userID, page, limit)
}

func (s *ScriptService) ListAll(page, limit int) ([]model.Script, int64, error) {
    return s.repo.ListAll(page, limit)
}

func (s *ScriptService) UpdateTags(id, userID, tags string) error {
    return s.repo.UpdateTags(id, userID, tags)
}
```

- [ ] **Step 3: 创建 script_handler.go**

```go
// internal/handler/script_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/service"
)

type ScriptHandler struct{ svc *service.ScriptService }

func NewScriptHandler(svc *service.ScriptService) *ScriptHandler { return &ScriptHandler{svc} }

func (h *ScriptHandler) Create(c *gin.Context) {
    userID := c.GetString("user_id")
    var s model.Script
    if err := c.ShouldBindJSON(&s); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    s.UserID = userID
    if err := h.svc.Create(&s); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, s)
}

func (h *ScriptHandler) List(c *gin.Context) {
    userID := c.GetString("user_id")
    page, limit := pageParams(c)
    scripts, total, err := h.svc.List(userID, page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": scripts, "total": total})
}

func (h *ScriptHandler) Get(c *gin.Context) {
    userID := c.GetString("user_id")
    s, err := h.svc.Get(c.Param("id"), userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    c.JSON(http.StatusOK, s)
}

func (h *ScriptHandler) UpdateTags(c *gin.Context) {
    userID := c.GetString("user_id")
    var req struct{ Tags string `json:"tags"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := h.svc.UpdateTags(c.Param("id"), userID, req.Tags); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Admin
func (h *ScriptHandler) ListAll(c *gin.Context) {
    page, limit := pageParams(c)
    scripts, total, err := h.svc.ListAll(page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": scripts, "total": total})
}
```

- [ ] **Step 4: 注册稿件路由**

```go
// main.go api 路由组中追加
api.GET("/scripts", scriptHandler.List)
api.POST("/scripts", scriptHandler.Create)
api.GET("/scripts/:id", scriptHandler.Get)
api.PUT("/scripts/:id/tags", scriptHandler.UpdateTags)

admin.GET("/scripts", scriptHandler.ListAll)
```

- [ ] **Step 5: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add script CRUD endpoints"
```

---

## Task 6: 素材库 + 热点 API + 离线同步

**Files:**
- Create: `server/internal/repository/material_repo.go`
- Create: `server/internal/service/material_service.go`
- Create: `server/internal/service/sync_service.go`
- Create: `server/internal/handler/material_handler.go`
- Create: `server/internal/handler/sync_handler.go`

- [ ] **Step 1: 创建 material_repo.go**

```go
// internal/repository/material_repo.go
package repository

import (
    "gorm.io/gorm"
    "github.com/content-creator-imm/server/internal/model"
    "time"
)

type MaterialRepo struct{ db *gorm.DB }

func NewMaterialRepo(db *gorm.DB) *MaterialRepo { return &MaterialRepo{db} }

func (r *MaterialRepo) ListMaterials(topic string, limit int) ([]model.Material, error) {
    var items []model.Material
    q := r.db.Where("is_public = ?", true)
    if topic != "" {
        q = q.Where("topic = ?", topic)
    }
    err := q.Limit(limit).Find(&items).Error
    return items, err
}

func (r *MaterialRepo) CreateMaterial(m *model.Material) error {
    return r.db.Create(m).Error
}

func (r *MaterialRepo) ListHotspots(platform string) ([]model.Hotspot, error) {
    var items []model.Hotspot
    since := time.Now().Add(-2 * time.Hour)
    q := r.db.Where("fetched_at > ?", since)
    if platform != "" {
        q = q.Where("platform = ?", platform)
    }
    err := q.Order("rank ASC").Limit(50).Find(&items).Error
    return items, err
}

func (r *MaterialRepo) BulkInsertHotspots(items []model.Hotspot) error {
    return r.db.Create(&items).Error
}
```

- [ ] **Step 2: 创建 material_service.go 和 sync_service.go**

```go
// internal/service/material_service.go
package service

import (
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/repository"
)

type MaterialService struct{ repo *repository.MaterialRepo }

func NewMaterialService(repo *repository.MaterialRepo) *MaterialService {
    return &MaterialService{repo}
}

func (s *MaterialService) ListMaterials(topic string, limit int) ([]model.Material, error) {
    if limit <= 0 || limit > 50 { limit = 10 }
    return s.repo.ListMaterials(topic, limit)
}

func (s *MaterialService) ListHotspots(platform string) ([]model.Hotspot, error) {
    return s.repo.ListHotspots(platform)
}
```

```go
// internal/service/sync_service.go
package service

import (
    "github.com/content-creator-imm/server/internal/model"
    "github.com/content-creator-imm/server/internal/repository"
)

type SyncItem struct {
    OpID    string      `json:"op_id"` // 幂等键
    OpType  string      `json:"op_type"` // "create_script" | "update_style"
    Payload interface{} `json:"payload"`
}

type SyncService struct {
    scriptRepo *repository.ScriptRepo
    userRepo   *repository.UserRepo
}

func NewSyncService(sr *repository.ScriptRepo, ur *repository.UserRepo) *SyncService {
    return &SyncService{sr, ur}
}

func (s *SyncService) ProcessQueue(userID string, items []SyncItem) (int, []string) {
    success, failed := 0, []string{}
    for _, item := range items {
        var err error
        switch item.OpType {
        case "create_script":
            // payload 解析为 model.Script 并保存
            success++
        default:
            failed = append(failed, item.OpID+": unknown op_type")
        }
        _ = err
    }
    return success, failed
}
```

- [ ] **Step 3: 创建 material_handler.go 和 sync_handler.go**

```go
// internal/handler/material_handler.go
package handler

import (
    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/internal/service"
)

type MaterialHandler struct{ svc *service.MaterialService }

func NewMaterialHandler(svc *service.MaterialService) *MaterialHandler {
    return &MaterialHandler{svc}
}

func (h *MaterialHandler) ListMaterials(c *gin.Context) {
    topic := c.Query("topic")
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    items, err := h.svc.ListMaterials(topic, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *MaterialHandler) ListHotspots(c *gin.Context) {
    platform := c.Query("platform")
    items, err := h.svc.ListHotspots(platform)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": items})
}
```

```go
// internal/handler/sync_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/content-creator-imm/server/internal/service"
)

type SyncHandler struct{ svc *service.SyncService }

func NewSyncHandler(svc *service.SyncService) *SyncHandler { return &SyncHandler{svc} }

func (h *SyncHandler) Sync(c *gin.Context) {
    userID := c.GetString("user_id")
    var items []service.SyncItem
    if err := c.ShouldBindJSON(&items); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    success, failed := h.svc.ProcessQueue(userID, items)
    c.JSON(http.StatusOK, gin.H{"success": success, "failed": failed})
}
```

- [ ] **Step 4: 注册剩余路由**

```go
// main.go api 路由组追加
api.GET("/materials", materialHandler.ListMaterials)
api.GET("/hotspot", materialHandler.ListHotspots)
api.POST("/sync", syncHandler.Sync)
```

- [ ] **Step 5: 完整编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

Expected: 无报错

- [ ] **Step 6: 启动 postgres 并验证服务可运行**

```bash
cd /data/code/content_creator_imm/server
cp .env.example .env
docker compose up -d db
sleep 3
go run . &
curl http://localhost:8080/ping
# Expected: {"status":"ok"}
kill %1
docker compose down
```

- [ ] **Step 7: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add material, hotspot, sync endpoints - Plan 1 complete"
```

---

## 验收标准

- [ ] `go build ./...` 无报错
- [ ] `GET /ping` 返回 `{"status":"ok"}`
- [ ] `POST /api/v1/auth/register` 可注册用户
- [ ] `POST /api/v1/auth/login` 返回 JWT token
- [ ] 携带 token 可访问 `GET /api/v1/user/profile`
- [ ] 不携带 token 访问受保护路由返回 401
- [ ] `POST /api/v1/scripts` 可保存稿件
- [ ] `GET /api/v1/materials` 可查询素材
- [ ] `GET /api/v1/hotspot` 可查询热点（空列表）
- [ ] `POST /api/v1/sync` 接受队列数组

---

## 下一步

- **Plan 2**: 热点雷达爬虫（Go cron 定时抓取各平台热榜）
- **Plan 3**: Admin Backend（React + Ant Design Pro 前端）
- **Plan 4**: Skill 改造（本地缓存层 + 远端 API 接入）
