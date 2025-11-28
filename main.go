package main

import (
	"TodoLists/db"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func main() {

	//create todo
	http.HandleFunc("/create", handleCreateTodo)
	//get all todos
	http.HandleFunc("/getAllTodos", handleGetAllTodos)
	//update
	http.HandleFunc("/update", handleUpdateTodo)
	//delete
	http.HandleFunc("/delete", handleDeleteTodo)

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

	//4.生成ID
	id := uuid.New().String()
	var newTodo db.Todo = db.Todo{
		ID:          id,
		Name:        name,
		Description: description,
		Completed:   false,
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

	for i, todo := range db.Todos {
		if todo.ID == id {
			db.Todos[i].Name = name
			db.Todos[i].Description = description
			db.Todos[i].Completed = completed == "true"
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
