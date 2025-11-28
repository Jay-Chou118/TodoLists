package main

import (
	"TodoLists/db"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func main() {
	// 初始化数据库
	if err := db.InitDatabase(); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	defer db.CloseDatabase()

	// 添加静态文件服务，将static文件夹映射到根路径
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 认证相关路由（不需要验证）
	http.HandleFunc("/api/register", handleRegister)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/checkToken", handleCheckToken)

	// Todo相关路由（需要验证）
	http.HandleFunc("/api/create", authMiddleware(handleCreateTodo))
	http.HandleFunc("/api/getAllTodos", authMiddleware(handleGetAllTodos))
	http.HandleFunc("/api/update", authMiddleware(handleUpdateTodo))
	http.HandleFunc("/api/delete", authMiddleware(handleDeleteTodo))

	// 用户设备相关路由
	http.HandleFunc("/api/user/devices", authMiddleware(handleGetUserDevices))
	http.HandleFunc("/api/user/device/register", authMiddleware(handleRegisterDevice))
	http.HandleFunc("/api/user/device/update", authMiddleware(handleUpdateDevice))
	http.HandleFunc("/api/user/device/delete", authMiddleware(handleDeleteDevice))
	http.HandleFunc("/api/user/device/verify", authMiddleware(handleVerifyDevice))

	// 同步相关路由
	http.HandleFunc("/api/sync", authMiddleware(syncData))
	http.HandleFunc("/api/todos/batch", authMiddleware(batchUpdateTodos))
	http.HandleFunc("/api/conflicts/resolve", authMiddleware(resolveConflicts))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var jwtSecret = []byte("your-secret-key-change-in-production")

// 同步服务实例
var syncService = db.NewSyncService(db.StrategyTimeBased)

// 认证中间件
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理OPTIONS请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 获取Authorization头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "缺少认证token"})
			return
		}

		// 检查Bearer前缀
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "无效的认证格式"})
			return
		}

		// 验证token
		claims, err := db.ValidateToken(parts[1])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "无效的token: " + err.Error()})
			return
		}

		// 将用户信息存储在请求上下文中
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "device_id", claims.DeviceID)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// 用户注册处理
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var registerData struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&registerData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 验证输入
	if registerData.Username == "" || registerData.Password == "" || registerData.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "用户名、密码和邮箱不能为空"})
		return
	}

	// 注册用户
	user, err := db.RegisterUser(registerData.Username, registerData.Password, registerData.Email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 返回用户信息（不包含密码）
	user.Password = ""
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// 用户登录处理
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var loginData struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		DeviceName string `json:"device_name"`
		DeviceID   string `json:"device_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 验证输入
	if loginData.Username == "" || loginData.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "用户名和密码不能为空"})
		return
	}

	// 登录用户
	user, device, token, err := db.LoginUser(
		loginData.Username,
		loginData.Password,
		loginData.DeviceName,
		loginData.DeviceID,
	)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 返回登录成功信息
	user.Password = ""
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
		"device":  device,
		"token":   token,
	})
}

// 验证token处理
func handleCheckToken(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 获取Authorization头
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "缺少认证token"})
		return
	}

	// 检查Bearer前缀
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的认证格式"})
		return
	}

	// 验证token
	claims, err := db.ValidateToken(parts[1])
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的token"})
		return
	}

	// 获取用户信息
	user, err := db.GetUserByID(claims.UserID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "用户不存在"})
		return
	}

	// 返回成功信息
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"user":      user,
		"device_id": claims.DeviceID,
	})
}

// 获取用户设备列表
func handleGetUserDevices(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无法获取用户信息"})
		return
	}

	// 获取用户设备列表
	devices := db.GetUserDevices(userID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"devices": devices,
	})
}

// 注册新设备
func handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无法获取用户信息"})
		return
	}

	// 读取请求数据
	var deviceData struct {
		DeviceName string `json:"device_name"`
		DeviceID   string `json:"device_id"`
		UserAgent  string `json:"user_agent"`
	}

	err := json.NewDecoder(r.Body).Decode(&deviceData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 验证输入
	if deviceData.DeviceName == "" || deviceData.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "设备名称和设备ID不能为空"})
		return
	}

	// 解析用户代理信息
	deviceInfo := db.ParseUserAgent(deviceData.UserAgent)

	// 创建或更新设备
	device, err := db.CreateDevice(userID, deviceData.DeviceName, deviceData.DeviceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "创建设备失败: " + err.Error()})
		return
	}

	log.Printf("用户 %s 注册新设备: %s (%s)", userID, deviceData.DeviceName, deviceData.DeviceID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"device":      device,
		"device_info": deviceInfo,
	})
}

// 更新设备信息
func handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无法获取用户信息"})
		return
	}

	// 读取请求数据
	var updateData struct {
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
	}

	err := json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 更新设备名称
	err = db.UpdateDeviceName(userID, updateData.DeviceID, updateData.DeviceName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("用户 %s 更新设备名称: %s -> %s", userID, updateData.DeviceID, updateData.DeviceName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// 删除设备
func handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "无法获取用户信息"})
		return
	}

	// 读取请求数据
	var deleteData struct {
		DeviceID string `json:"device_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&deleteData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 删除设备
	err = db.DeleteDevice(userID, deleteData.DeviceID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("用户 %s 删除设备: %s", userID, deleteData.DeviceID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// 验证设备是否已授权
func handleVerifyDevice(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID和设备ID
	userID, _ := r.Context().Value("user_id").(string)
	deviceID, _ := r.Context().Value("device_id").(string)

	// 验证设备是否已授权
	authorized := db.IsDeviceAuthorized(userID, deviceID)

	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "设备未授权，请重新登录"})
		return
	}

	// 更新设备最后活跃时间
	db.UpdateDeviceLastSeen(deviceID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"authorized": true,
	})
}

func handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	// 从上下文获取用户ID和设备ID
	userID, _ := r.Context().Value("user_id").(string)
	deviceID, _ := r.Context().Value("device_id").(string)

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 读取前端数据
	var todoData struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		DeadLine    string `json:"deadline"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
	}

	// 解析数据
	err := json.NewDecoder(r.Body).Decode(&todoData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 验证输入
	if todoData.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "任务名称不能为空"})
		return
	}

	// 生成ID和创建时间
	id := uuid.New().String()
	now := time.Now()
	newTodo := db.Todo{
		ID:          id,
		UserID:      userID,
		DeviceID:    deviceID,
		Name:        todoData.Name,
		Description: todoData.Description,
		Completed:   false,
		CreateAt:    now,
		UpdateAt:    now,
		DeadLine:    todoData.DeadLine,
		Category:    todoData.Category,
		Priority:    todoData.Priority,
	}

	// 存储数据
	db.Todos = append(db.Todos, newTodo)
	log.Printf("创建任务: %s 由用户 %s 设备 %s", newTodo.Name, userID, deviceID)

	// 返回创建的任务
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"todo":    newTodo,
	})
}

// 同步数据
func syncData(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID和设备ID
	userID, _ := r.Context().Value("user_id").(string)
	deviceID, _ := r.Context().Value("device_id").(string)

	var syncReq db.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 验证请求
	syncReq.UserID = userID
	syncReq.DeviceID = deviceID

	// 验证同步数据
	if err := db.ValidateSyncData(userID, syncReq.Todos); err != nil {
		http.Error(w, fmt.Sprintf("数据验证失败: %v", err), http.StatusBadRequest)
		return
	}

	// 执行同步
	response, err := syncService.SyncData(&syncReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("同步失败: %v", err), http.StatusInternalServerError)
		log.Printf("同步失败: %v", err)
		return
	}

	json.NewEncoder(w).Encode(response)
}

// 批量更新任务
func batchUpdateTodos(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID和设备ID
	userID, _ := r.Context().Value("user_id").(string)
	deviceID, _ := r.Context().Value("device_id").(string)

	var todos []db.Todo
	if err := json.NewDecoder(r.Body).Decode(&todos); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 验证数据
	if err := db.ValidateSyncData(userID, todos); err != nil {
		http.Error(w, fmt.Sprintf("数据验证失败: %v", err), http.StatusBadRequest)
		return
	}

	// 批量更新
	if err := db.BatchUpdateTodos(userID, deviceID, todos); err != nil {
		http.Error(w, fmt.Sprintf("批量更新失败: %v", err), http.StatusInternalServerError)
		log.Printf("批量更新失败: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 解决冲突
func resolveConflicts(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 从上下文获取用户ID
	userID, _ := r.Context().Value("user_id").(string)

	var resolvedTodos []db.Todo
	if err := json.NewDecoder(r.Body).Decode(&resolvedTodos); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 解决冲突
	if err := db.ResolveConflicts(userID, resolvedTodos); err != nil {
		http.Error(w, fmt.Sprintf("解决冲突失败: %v", err), http.StatusInternalServerError)
		log.Printf("解决冲突失败: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleGetAllTodos(w http.ResponseWriter, r *http.Request) {
	// 从上下文获取用户ID
	userID, _ := r.Context().Value("user_id").(string)

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 过滤用户的任务
	var userTodos []db.Todo
	for _, todo := range db.Todos {
		if todo.UserID == userID {
			userTodos = append(userTodos, todo)
		}
	}

	log.Printf("获取用户 %s 的任务，共 %d 个", userID, len(userTodos))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userTodos)
}

func handleUpdateTodo(w http.ResponseWriter, r *http.Request) {
	// 从上下文获取用户ID
	userID, _ := r.Context().Value("user_id").(string)

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 读取前端数据
	var updateData struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Completed   bool   `json:"completed"`
		DeadLine    string `json:"deadline"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
	}

	err := json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 查找并更新任务
	found := false
	for i, todo := range db.Todos {
		if todo.ID == updateData.ID && todo.UserID == userID {
			// 只允许更新自己的任务
			db.Todos[i].Name = updateData.Name
			db.Todos[i].Description = updateData.Description
			db.Todos[i].Completed = updateData.Completed
			db.Todos[i].DeadLine = updateData.DeadLine
			db.Todos[i].Category = updateData.Category
			db.Todos[i].Priority = updateData.Priority
			db.Todos[i].UpdateAt = time.Now() // 更新时间戳
			found = true
			log.Printf("更新任务: %s 由用户 %s", updateData.ID, userID)
			break
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "任务不存在或无权修改"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	// 从上下文获取用户ID
	userID, _ := r.Context().Value("user_id").(string)

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 读取前端数据
	var deleteData struct {
		ID string `json:"id"`
	}

	err := json.NewDecoder(r.Body).Decode(&deleteData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "无效的请求数据"})
		return
	}

	// 查找并删除任务
	found := false
	for i, todo := range db.Todos {
		if todo.ID == deleteData.ID && todo.UserID == userID {
			// 只允许删除自己的任务
			db.Todos = append(db.Todos[:i], db.Todos[i+1:]...)
			found = true
			log.Printf("删除任务: %s 由用户 %s", deleteData.ID, userID)
			break
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "任务不存在或无权删除"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}
