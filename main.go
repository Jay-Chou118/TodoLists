package main

import (
	"TodoLists/db"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func main() {
	// 添加静态文件服务，将static文件夹映射到根路径
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API路由
	//create todo
	http.HandleFunc("/api/create", handleCreateTodo)
	//get all todos
	http.HandleFunc("/api/getAllTodos", handleGetAllTodos)
	//update
	http.HandleFunc("/api/update", handleUpdateTodo)
	//delete
	http.HandleFunc("/api/delete", handleDeleteTodo)

	log.Println("Server starting on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	//1.读取前端数据
	params := map[string]string{}

	//2.解析数据
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//3.处理参数
	name := params["name"]
	description := params["description"]
	deadline := params["deadline"]
	category := params["category"]
	priority := params["priority"]

	//4.生成ID和创建时间
	id := uuid.New().String()
	now := time.Now().Format("2006-01-02 15:04:05")
	var newTodo db.Todo = db.Todo{
		ID:          id,
		Name:        name,
		Description: description,
		Completed:   false,
		CreateAt:    now,
		DeadLine:    deadline,
		Category:    category,
		Priority:    priority,
	}

	//5.存储数据（todo，存入数据库）
	db.Todos = append(db.Todos, newTodo)
	log.Println("todos: ", db.Todos)

	//6.返回结果
	w.WriteHeader(http.StatusOK)
}

func handleGetAllTodos(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Println("handleGetAllTodos: ", db.Todos)
	json.NewEncoder(w).Encode(db.Todos)
}

func handleUpdateTodo(w http.ResponseWriter, r *http.Request) {

	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := params["id"]
	name := params["name"]
	description := params["description"]
	completed := params["completed"]
	deadline := params["deadline"]
	category := params["category"]
	priority := params["priority"]

	for i, todo := range db.Todos {
		if todo.ID == id {
			db.Todos[i].Name = name
			db.Todos[i].Description = description
			db.Todos[i].Completed = completed == "true"
			// 更新新字段
			if deadline != "" {
				db.Todos[i].DeadLine = deadline
			}
			if category != "" {
				db.Todos[i].Category = category
			}
			if priority != "" {
				db.Todos[i].Priority = priority
			}
			break
		}
	}

	w.WriteHeader(http.StatusOK)

}

func handleDeleteTodo(w http.ResponseWriter, r *http.Request) {

	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := params["id"]

	for i, todo := range db.Todos {
		if todo.ID == id {
			db.Todos = append(db.Todos[:i], db.Todos[i+1:]...)
			break
		}
	}

	w.WriteHeader(http.StatusOK)

}
