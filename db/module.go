package db

type Todo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	CreateAt    string `json:"created_at"`
	DeadLine    string `json:"deadline"` // 任务截止时间
	Category    string `json:"category"` // 任务分类
	Priority    string `json:"priority"` // 任务优先级
}

var Todos []Todo
