package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// InitDatabase 初始化数据库连接
func InitDatabase() error {
	var err error

	// 确保数据目录存在
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return fmt.Errorf("创建数据目录失败: %v", err)
		}
	}

	// 打开SQLite数据库连接
	dbPath := "./data/todolist.db"
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %v", err)
	}

	// 测试连接
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}

	// 创建表
	err = createTables()
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	log.Println("数据库初始化成功")
	return nil
}

// createTables 创建数据库表
func createTables() error {
	// 创建用户表
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at TEXT NOT NULL
	);
	`

	// 创建设备表
	deviceTable := `
	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		device_id TEXT NOT NULL,
		last_seen TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(user_id, device_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	// 创建任务表
	todoTable := `
	CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		device_id TEXT,
		name TEXT NOT NULL,
		description TEXT,
		completed INTEGER DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		deadline TEXT,
		category TEXT,
		priority TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	// 执行建表语句
	_, err := db.Exec(userTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(deviceTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(todoTable)
	if err != nil {
		return err
	}

	// 创建索引以提高查询性能
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id)")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_todos_updated_at ON todos(updated_at)")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id)")
	if err != nil {
		return err
	}

	return nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// 将time.Time转换为字符串存储
func timeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

// 将字符串转换为time.Time
func stringToTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// 保存用户到数据库
func SaveUserToDB(user *User) error {
	query := `
	INSERT OR REPLACE INTO users (id, username, password, email, created_at)
	VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, user.ID, user.Username, user.Password, user.Email, timeToString(user.CreatedAt))
	return err
}

// 从数据库获取用户
func GetUserFromDB(userID string) (*User, error) {
	query := `
	SELECT id, username, password, email, created_at
	FROM users
	WHERE id = ?
	`

	var user User
	var createdAtStr string

	err := db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &createdAtStr,
	)

	if err != nil {
		return nil, err
	}

	user.CreatedAt, err = stringToTime(createdAtStr)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// 根据用户名获取用户
func GetUserByUsernameFromDB(username string) (*User, error) {
	query := `
	SELECT id, username, password, email, created_at
	FROM users
	WHERE username = ?
	`

	var user User
	var createdAtStr string

	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &createdAtStr,
	)

	if err != nil {
		return nil, err
	}

	user.CreatedAt, err = stringToTime(createdAtStr)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// 保存设备到数据库
func SaveDeviceToDB(device *Device) error {
	query := `
	INSERT OR REPLACE INTO devices (id, user_id, name, device_id, last_seen, created_at)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		device.ID, device.UserID, device.Name, device.DeviceID,
		timeToString(device.LastSeen), timeToString(device.CreatedAt),
	)
	return err
}

// 从数据库获取设备
func GetDeviceFromDB(deviceID string) (*Device, error) {
	query := `
	SELECT id, user_id, name, device_id, last_seen, created_at
	FROM devices
	WHERE device_id = ?
	`

	var device Device
	var lastSeenStr, createdAtStr string

	err := db.QueryRow(query, deviceID).Scan(
		&device.ID, &device.UserID, &device.Name, &device.DeviceID,
		&lastSeenStr, &createdAtStr,
	)

	if err != nil {
		return nil, err
	}

	device.LastSeen, err = stringToTime(lastSeenStr)
	if err != nil {
		return nil, err
	}

	device.CreatedAt, err = stringToTime(createdAtStr)
	if err != nil {
		return nil, err
	}

	return &device, nil
}

// 获取用户的所有设备
func GetUserDevicesFromDB(userID string) ([]Device, error) {
	query := `
	SELECT id, user_id, name, device_id, last_seen, created_at
	FROM devices
	WHERE user_id = ?
	ORDER BY last_seen DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		var lastSeenStr, createdAtStr string

		err := rows.Scan(
			&device.ID, &device.UserID, &device.Name, &device.DeviceID,
			&lastSeenStr, &createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		device.LastSeen, err = stringToTime(lastSeenStr)
		if err != nil {
			return nil, err
		}

		device.CreatedAt, err = stringToTime(createdAtStr)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// 保存任务到数据库
func SaveTodoToDB(todo *Todo) error {
	query := `
	INSERT OR REPLACE INTO todos (
		id, user_id, device_id, name, description, completed,
		created_at, updated_at, deadline, category, priority
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		todo.ID, todo.UserID, todo.DeviceID, todo.Name, todo.Description, boolToInt(todo.Completed),
		timeToString(todo.CreateAt), timeToString(todo.UpdateAt), todo.DeadLine, todo.Category, todo.Priority,
	)
	return err
}

// 从数据库获取用户的所有任务
func GetUserTodosFromDB(userID string) ([]Todo, error) {
	query := `
	SELECT id, user_id, device_id, name, description, completed,
	       created_at, updated_at, deadline, category, priority
	FROM todos
	WHERE user_id = ?
	ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		var completedInt int
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.DeviceID, &todo.Name, &todo.Description, &completedInt,
			&createdAtStr, &updatedAtStr, &todo.DeadLine, &todo.Category, &todo.Priority,
		)
		if err != nil {
			return nil, err
		}

		todo.Completed = intToBool(completedInt)
		todo.CreateAt, err = stringToTime(createdAtStr)
		if err != nil {
			return nil, err
		}

		todo.UpdateAt, err = stringToTime(updatedAtStr)
		if err != nil {
			return nil, err
		}

		todos = append(todos, todo)
	}

	return todos, nil
}

// 删除任务
func DeleteTodoFromDB(todoID, userID string) error {
	query := `
	DELETE FROM todos
	WHERE id = ? AND user_id = ?
	`
	_, err := db.Exec(query, todoID, userID)
	return err
}

// 获取某个时间点之后更新的任务
func GetTodosUpdatedAfterFromDB(userID string, timestamp time.Time) ([]Todo, error) {
	query := `
	SELECT id, user_id, device_id, name, description, completed,
	       created_at, updated_at, deadline, category, priority
	FROM todos
	WHERE user_id = ? AND updated_at > ?
	ORDER BY updated_at ASC
	`

	rows, err := db.Query(query, userID, timeToString(timestamp))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		var completedInt int
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.DeviceID, &todo.Name, &todo.Description, &completedInt,
			&createdAtStr, &updatedAtStr, &todo.DeadLine, &todo.Category, &todo.Priority,
		)
		if err != nil {
			return nil, err
		}

		todo.Completed = intToBool(completedInt)
		todo.CreateAt, err = stringToTime(createdAtStr)
		if err != nil {
			return nil, err
		}

		todo.UpdateAt, err = stringToTime(updatedAtStr)
		if err != nil {
			return nil, err
		}

		todos = append(todos, todo)
	}

	return todos, nil
}

// 辅助函数：bool转int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// 辅助函数：int转bool
func intToBool(i int) bool {
	return i == 1
}

// 导出数据到JSON文件（用于备份）
func ExportDataToJSON(filePath string) error {
	// 获取所有用户
	users, err := getAllUsersFromDB()
	if err != nil {
		return err
	}

	// 获取所有设备
	devices, err := getAllDevicesFromDB()
	if err != nil {
		return err
	}

	// 获取所有任务
	todos, err := getAllTodosFromDB()
	if err != nil {
		return err
	}

	// 创建导出数据结构
	exportData := struct {
		Users   []User   `json:"users"`
		Devices []Device `json:"devices"`
		Todos   []Todo   `json:"todos"`
	}{users, devices, todos}

	// 转换为JSON
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(filePath, data, 0644)
}

// 辅助函数：获取所有用户（仅用于导出）
func getAllUsersFromDB() ([]User, error) {
	query := `SELECT id, username, password, email, created_at FROM users`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var createdAtStr string
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &createdAtStr)
		if err != nil {
			return nil, err
		}
		user.CreatedAt, _ = stringToTime(createdAtStr)
		users = append(users, user)
	}
	return users, nil
}

// 辅助函数：获取所有设备（仅用于导出）
func getAllDevicesFromDB() ([]Device, error) {
	query := `SELECT id, user_id, name, device_id, last_seen, created_at FROM devices`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		var lastSeenStr, createdAtStr string
		err := rows.Scan(&device.ID, &device.UserID, &device.Name, &device.DeviceID, &lastSeenStr, &createdAtStr)
		if err != nil {
			return nil, err
		}
		device.LastSeen, _ = stringToTime(lastSeenStr)
		device.CreatedAt, _ = stringToTime(createdAtStr)
		devices = append(devices, device)
	}
	return devices, nil
}

// 辅助函数：获取所有任务（仅用于导出）
func getAllTodosFromDB() ([]Todo, error) {
	query := `SELECT id, user_id, device_id, name, description, completed, created_at, updated_at, deadline, category, priority FROM todos`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		var completedInt int
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.DeviceID, &todo.Name, &todo.Description, &completedInt,
			&createdAtStr, &updatedAtStr, &todo.DeadLine, &todo.Category, &todo.Priority,
		)
		if err != nil {
			return nil, err
		}
		todo.Completed = intToBool(completedInt)
		todo.CreateAt, _ = stringToTime(createdAtStr)
		todo.UpdateAt, _ = stringToTime(updatedAtStr)
		todos = append(todos, todo)
	}
	return todos, nil
}
