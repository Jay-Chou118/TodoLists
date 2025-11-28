package db

import (
	"time"
)

// User 用户结构体
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password_hash,omitempty"` // 存储密码哈希值，不返回给前端
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Device 设备结构体
type Device struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`      // 设备名称
	DeviceID  string    `json:"device_id"` // 唯一设备标识
	LastSeen  time.Time `json:"last_seen"` // 最后活跃时间
	CreatedAt time.Time `json:"created_at"`
}

// Todo 任务结构体，增加用户ID关联
type Todo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`             // 关联用户ID
	DeviceID    string    `json:"device_id,omitempty"` // 创建任务的设备ID
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreateAt    time.Time `json:"created_at"`
	UpdateAt    time.Time `json:"updated_at"` // 增加更新时间字段用于冲突解决
	DeadLine    string    `json:"deadline"`   // 任务截止时间
	Category    string    `json:"category"`   // 任务分类
	Priority    string    `json:"priority"`   // 任务优先级
}

// 内存存储（临时）
var Users []User
var Devices []Device
var Todos []Todo

// 数据库操作接口（后续会替换为实际数据库实现）
// 这里保留接口定义，便于后续实现数据库持久化
