package db

type Todo struct {
	ID          string `json:"id"`
	Name        string `json:"name`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	CreateAt    string `json:"created_at"`
}

var Todos []Todo
