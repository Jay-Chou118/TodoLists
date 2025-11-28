package db

import (
	"fmt"
	"sort"
	"time"
)

// SyncRequest 同步请求结构
type SyncRequest struct {
	UserID     string    `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	LastSyncAt time.Time `json:"last_sync_at"`
	Todos      []Todo    `json:"todos"`
}

// SyncResponse 同步响应结构
type SyncResponse struct {
	LastSyncAt time.Time `json:"last_sync_at"`
	Todos      []Todo    `json:"todos"`
	Conflicts  []Conflict `json:"conflicts,omitempty"`
}

// Conflict 冲突信息结构
type Conflict struct {
	LocalTodo  Todo `json:"local_todo"`
	ServerTodo Todo `json:"server_todo"`
}

// SyncStrategy 同步策略枚举
type SyncStrategy string

const (
	// StrategyServerWins 服务器优先策略
	StrategyServerWins SyncStrategy = "server_wins"
	// StrategyClientWins 客户端优先策略
	StrategyClientWins SyncStrategy = "client_wins"
	// StrategyManualResolve 手动解决冲突
	StrategyManualResolve SyncStrategy = "manual_resolve"
	// StrategyTimeBased 基于时间戳的策略
	StrategyTimeBased SyncStrategy = "time_based"
)

// SyncService 同步服务
type SyncService struct {
	strategy SyncStrategy
}

// NewSyncService 创建新的同步服务
func NewSyncService(strategy SyncStrategy) *SyncService {
	if strategy == "" {
		strategy = StrategyTimeBased // 默认使用基于时间戳的策略
	}
	return &SyncService{strategy: strategy}
}

// SyncData 执行数据同步
func (s *SyncService) SyncData(req *SyncRequest) (*SyncResponse, error) {
	// 验证输入
	if req.UserID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}
	if req.DeviceID == "" {
		return nil, fmt.Errorf("设备ID不能为空")
	}

	// 更新设备最后活跃时间
	device, err := GetDeviceFromDB(req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %v", err)
	}
	device.LastSeen = time.Now()
	err = SaveDeviceToDB(device)
	if err != nil {
		return nil, fmt.Errorf("更新设备信息失败: %v", err)
	}

	// 获取服务器端自上次同步以来的更新
	serverTodos, err := GetTodosUpdatedAfterFromDB(req.UserID, req.LastSyncAt)
	if err != nil {
		return nil, fmt.Errorf("获取服务器更新失败: %v", err)
	}

	// 处理客户端发送的更新
	conflicts, err := s.processClientUpdates(req.UserID, req.DeviceID, req.Todos, serverTodos)
	if err != nil {
		return nil, fmt.Errorf("处理客户端更新失败: %v", err)
	}

	// 获取最新的服务器端数据
	latestTodos, err := GetTodosUpdatedAfterFromDB(req.UserID, req.LastSyncAt)
	if err != nil {
		return nil, fmt.Errorf("获取最新数据失败: %v", err)
	}

	// 构建响应
	response := &SyncResponse{
		LastSyncAt: time.Now(),
		Todos:      latestTodos,
	}

	// 如果有冲突，添加到响应中
	if len(conflicts) > 0 {
		response.Conflicts = conflicts
	}

	return response, nil
}

// processClientUpdates 处理客户端发送的更新
func (s *SyncService) processClientUpdates(userID, deviceID string, clientTodos, serverTodos []Todo) ([]Conflict, error) {
	// 创建服务器端任务的映射
	serverTodoMap := make(map[string]Todo)
	for _, todo := range serverTodos {
		serverTodoMap[todo.ID] = todo
	}

	var conflicts []Conflict

	// 处理每个客户端任务
	for _, clientTodo := range clientTodos {
		// 确保任务属于当前用户
		clientTodo.UserID = userID
		clientTodo.DeviceID = deviceID

		// 检查服务器端是否有相同ID的任务
		if serverTodo, exists := serverTodoMap[clientTodo.ID]; exists {
			// 检测冲突
			if s.hasConflict(clientTodo, serverTodo) {
				// 根据策略处理冲突
				if s.strategy == StrategyManualResolve {
					// 记录冲突，需要用户手动解决
					conflicts = append(conflicts, Conflict{
						LocalTodo:  clientTodo,
						ServerTodo: serverTodo,
					})
					continue
				} else {
					// 根据策略选择保留哪个版本
					resolvedTodo := s.resolveConflict(clientTodo, serverTodo)
					// 保存解决后的任务
					err := SaveTodoToDB(&resolvedTodo)
					if err != nil {
						return nil, err
					}
				}
			} else {
				// 没有冲突，直接更新服务器数据
				clientTodo.UpdateAt = time.Now()
				err := SaveTodoToDB(&clientTodo)
				if err != nil {
					return nil, err
				}
			}
		} else {
			// 新任务，直接保存
			if clientTodo.CreateAt.IsZero() {
				clientTodo.CreateAt = time.Now()
			}
			clientTodo.UpdateAt = time.Now()
			err := SaveTodoToDB(&clientTodo)
			if err != nil {
				return nil, err
			}
		}
	}

	return conflicts, nil
}

// hasConflict 检测是否存在冲突
func (s *SyncService) hasConflict(clientTodo, serverTodo Todo) bool {
	// 如果两个任务的更新时间不同，并且不是同一个设备更新的，则认为存在冲突
	return !clientTodo.UpdateAt.Equal(serverTodo.UpdateAt) && 
		clientTodo.DeviceID != serverTodo.DeviceID
}

// resolveConflict 解决冲突
func (s *SyncService) resolveConflict(clientTodo, serverTodo Todo) Todo {
	switch s.strategy {
	case StrategyServerWins:
		return serverTodo
	case StrategyClientWins:
		return clientTodo
	case StrategyTimeBased:
		// 基于时间戳，保留最新的版本
		if clientTodo.UpdateAt.After(serverTodo.UpdateAt) {
			return clientTodo
		}
		return serverTodo
	default:
		// 默认使用基于时间戳的策略
		if clientTodo.UpdateAt.After(serverTodo.UpdateAt) {
			return clientTodo
		}
		return serverTodo
	}
}

// GetUserTodosWithSync 获取用户任务并包含同步信息
func GetUserTodosWithSync(userID string) ([]Todo, error) {
	return GetUserTodosFromDB(userID)
}

// BatchUpdateTodos 批量更新任务
func BatchUpdateTodos(userID, deviceID string, todos []Todo) error {
	for _, todo := range todos {
		// 确保任务属于当前用户
		todo.UserID = userID
		todo.DeviceID = deviceID
		todo.UpdateAt = time.Now()
		err := SaveTodoToDB(&todo)
		if err != nil {
			return fmt.Errorf("更新任务 %s 失败: %v", todo.ID, err)
		}
	}
	return nil
}

// ResolveConflicts 解决冲突
func ResolveConflicts(userID string, resolvedTodos []Todo) error {
	for _, todo := range resolvedTodos {
		// 确保任务属于当前用户
		if todo.UserID != userID {
			return fmt.Errorf("无权更新任务 %s", todo.ID)
		}
		todo.UpdateAt = time.Now()
		err := SaveTodoToDB(&todo)
		if err != nil {
			return fmt.Errorf("更新冲突任务 %s 失败: %v", todo.ID, err)
		}
	}
	return nil
}

// MergeChanges 合并变更（高级同步功能）
func MergeChanges(localTodos, serverTodos []Todo) []Todo {
	// 创建任务映射
	localMap := make(map[string]Todo)
	serverMap := make(map[string]Todo)

	for _, todo := range localTodos {
		localMap[todo.ID] = todo
	}

	for _, todo := range serverTodos {
		serverMap[todo.ID] = todo
	}

	// 合并结果
	merged := make(map[string]Todo)

	// 处理所有ID
	allIDs := make(map[string]bool)
	for id := range localMap {
		allIDs[id] = true
	}
	for id := range serverMap {
		allIDs[id] = true
	}

	// 合并每个任务
	for id := range allIDs {
		localTodo, localExists := localMap[id]
		serverTodo, serverExists := serverMap[id]

		if !serverExists {
			// 只在本地存在，保留本地版本
			merged[id] = localTodo
		} else if !localExists {
			// 只在服务器存在，保留服务器版本
			merged[id] = serverTodo
		} else {
			// 两边都存在，根据更新时间决定
			if localTodo.UpdateAt.After(serverTodo.UpdateAt) {
				merged[id] = localTodo
			} else {
				merged[id] = serverTodo
			}
		}
	}

	// 转换为切片并排序
	result := make([]Todo, 0, len(merged))
	for _, todo := range merged {
		result = append(result, todo)
	}

	// 按更新时间排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdateAt.After(result[j].UpdateAt)
	})

	return result
}

// ValidateSyncData 验证同步数据
func ValidateSyncData(userID string, todos []Todo) error {
	for i, todo := range todos {
		if todo.UserID != "" && todo.UserID != userID {
			return fmt.Errorf("任务 %d 不属于当前用户", i)
		}
		if todo.Name == "" {
			return fmt.Errorf("任务 %d 的名称不能为空", i)
		}
	}
	return nil
}
