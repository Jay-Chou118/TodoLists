// TodoManager 模块，管理待办事项
const TodoManager = {
    todos: [],
    isInitialized: false,

    // 初始化
    async init() {
        // 等待AuthModule初始化
        while (!window.AuthModule) {
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        if (AuthModule.isLoggedIn()) {
            // 从服务器加载数据
            await this.loadTodosFromServer();
        } else {
            // 从本地存储加载数据
            this.loadTodosFromLocal();
        }
        
        this.renderTodos();
        this.setupEventListeners();
        this.isInitialized = true;

        // 监听同步完成事件
        window.addEventListener('syncComplete', () => {
            this.renderTodos();
            this.updateSyncStatus();
        });
    },

    // 从本地存储加载待办事项
    loadTodosFromLocal() {
        const storedTodos = localStorage.getItem('todos');
        if (storedTodos) {
            try {
                this.todos = JSON.parse(storedTodos);
            } catch (e) {
                console.error('Failed to parse todos from localStorage', e);
                this.todos = [];
            }
        } else {
            this.todos = [];
        }
    },

    // 从服务器加载待办事项
    async loadTodosFromServer() {
        if (!AuthModule.isLoggedIn()) return;

        try {
            const response = await fetch('/api/todos', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                }
            });

            if (response.ok) {
                this.todos = await response.json();
                // 同时保存到本地作为缓存
                this.saveTodosToLocal();
            } else {
                console.error('Failed to load todos from server');
                // 加载失败时使用本地缓存
                this.loadTodosFromLocal();
            }
        } catch (error) {
            console.error('Error loading todos from server:', error);
            // 出错时使用本地缓存
            this.loadTodosFromLocal();
        }
    },

    // 保存待办事项到本地存储
    saveTodosToLocal() {
        localStorage.setItem('todos', JSON.stringify(this.todos));
    },

    // 保存待办事项到服务器
    async saveTodoToServer(todo) {
        if (!AuthModule.isLoggedIn()) return false;

        try {
            const method = todo.id.startsWith('temp_') ? 'POST' : 'PUT';
            const url = method === 'POST' ? '/api/todos' : `/api/todos/${todo.id}`;

            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                },
                body: JSON.stringify({
                    name: todo.name,
                    description: todo.description,
                    completed: todo.completed,
                    deadline: todo.deadline,
                    category: todo.category,
                    priority: todo.priority
                })
            });

            if (response.ok) {
                const savedTodo = await response.json();
                // 更新本地ID为服务器生成的ID
                if (todo.id.startsWith('temp_')) {
                    const index = this.todos.findIndex(t => t.id === todo.id);
                    if (index !== -1) {
                        this.todos[index].id = savedTodo.id;
                    }
                }
                return true;
            }
            return false;
        } catch (error) {
            console.error('Error saving todo to server:', error);
            return false;
        }
    },

    // 从服务器删除待办事项
    async deleteTodoFromServer(id) {
        if (!AuthModule.isLoggedIn()) return false;

        try {
            const response = await fetch(`/api/todos/${id}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                }
            });
            return response.ok;
        } catch (error) {
            console.error('Error deleting todo from server:', error);
            return false;
        }
    },

    // 渲染待办事项列表
    renderTodos() {
        const todoList = document.getElementById('todo-list');
        const sortBy = document.getElementById('sort-by')?.value || 'created_at';
        
        if (!todoList) return;

        // 清空列表
        todoList.innerHTML = '';

        // 排序待办事项
        const sortedTodos = this.sortTodos([...this.todos], sortBy);

        if (sortedTodos.length === 0) {
            todoList.innerHTML = '<p class="no-todos">暂无待办事项，请添加新任务</p>';
            return;
        }

        // 创建并添加每个待办事项
        sortedTodos.forEach(todo => {
            const todoItem = this.createTodoElement(todo);
            todoList.appendChild(todoItem);
        });

        // 更新同步状态显示
        this.updateSyncStatus();
    },

    // 更新同步状态显示
    updateSyncStatus() {
        if (!AuthModule.isLoggedIn()) return;

        const syncStatusEl = document.getElementById('sync-status');
        if (!syncStatusEl) return;

        const status = SyncModule.getSyncStatus();
        
        if (status.isSyncing) {
            syncStatusEl.textContent = '正在同步...';
            syncStatusEl.className = 'syncing';
        } else if (status.syncError) {
            syncStatusEl.textContent = `同步失败: ${status.syncError}`;
            syncStatusEl.className = 'error';
        } else if (status.lastSync) {
            const time = status.lastSync.toLocaleTimeString();
            syncStatusEl.textContent = `上次同步: ${time}`;
            syncStatusEl.className = 'synced';
        } else {
            syncStatusEl.textContent = '未同步';
            syncStatusEl.className = '';
        }
    },

    // 排序待办事项
    sortTodos(todos, sortBy) {
        switch (sortBy) {
            case 'created_at':
                return todos.sort((a, b) => new Date(b.create_at) - new Date(a.create_at));
            case 'deadline':
                return todos.sort((a, b) => {
                    const dateA = a.deadline ? new Date(a.deadline) : new Date(0);
                    const dateB = b.deadline ? new Date(b.deadline) : new Date(0);
                    return dateA - dateB;
                });
            case 'priority':
                const priorityOrder = { 'high': 1, 'medium': 2, 'low': 3, '': 4 };
                return todos.sort((a, b) => {
                    return priorityOrder[a.priority || ''] - priorityOrder[b.priority || ''];
                });
            default:
                return todos;
        }
    },

    // 创建待办事项元素
    createTodoElement(todo) {
        const todoItem = document.createElement('div');
        todoItem.className = `todo-item ${todo.completed ? 'completed' : ''}`;
        todoItem.dataset.id = todo.id;

        const deadlineClass = this.getDeadlineClass(todo);

        todoItem.innerHTML = `
            <div class="todo-content">
                <div class="todo-header">
                    <h3>${todo.name}</h3>
                    <div class="todo-actions">
                        <button class="btn btn-sm btn-complete" data-id="${todo.id}">
                            ${todo.completed ? '✓' : '○'}
                        </button>
                        <button class="btn btn-sm btn-edit" data-id="${todo.id}">编辑</button>
                        <button class="btn btn-sm btn-delete" data-id="${todo.id}">删除</button>
                    </div>
                </div>
                ${todo.description ? `<p class="todo-description">${todo.description}</p>` : ''}
                <div class="todo-meta">
                    ${todo.deadline ? `<span class="todo-deadline ${deadlineClass}">截止: ${this.formatDateTime(todo.deadline)}</span>` : ''}
                    ${todo.category ? `<span class="todo-category">分类: ${todo.category}</span>` : ''}
                    ${todo.priority ? `<span class="todo-priority priority-${todo.priority}">优先级: ${this.capitalizeFirst(todo.priority)}</span>` : ''}
                    <span class="todo-created">创建: ${this.formatDateTime(todo.create_at)}</span>
                </div>
            </div>
        `;

        return todoItem;
    },

    // 获取截止日期样式类
    getDeadlineClass(todo) {
        if (!todo.deadline || todo.completed) return '';
        
        const now = new Date();
        const deadline = new Date(todo.deadline);
        const diffDays = Math.ceil((deadline - now) / (1000 * 60 * 60 * 24));

        if (diffDays < 0) return 'overdue';
        if (diffDays <= 1) return 'urgent';
        return '';
    },

    // 格式化日期时间
    formatDateTime(dateTime) {
        const date = new Date(dateTime);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    },

    // 首字母大写
    capitalizeFirst(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    },

    // 设置事件监听器
    setupEventListeners() {
        // 监听排序变化
        document.getElementById('sort-by')?.addEventListener('change', () => {
            this.renderTodos();
        });

        // 监听创建表单提交
        document.getElementById('todo-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.createOrUpdateTodo();
        });

        // 监听待办事项列表点击
        document.getElementById('todo-list')?.addEventListener('click', (e) => {
            if (e.target.closest('.btn-complete')) {
                const id = e.target.closest('.btn-complete').dataset.id;
                this.toggleComplete(id);
            } else if (e.target.closest('.btn-edit')) {
                const id = e.target.closest('.btn-edit').dataset.id;
                this.editTodo(id);
            } else if (e.target.closest('.btn-delete')) {
                const id = e.target.closest('.btn-delete').dataset.id;
                this.deleteTodo(id);
            }
        });

        // 监听手动同步按钮
        document.getElementById('sync-button')?.addEventListener('click', async () => {
            if (AuthModule.isLoggedIn()) {
                await SyncModule.forceSync();
            }
        });

        // 监听登录状态变化
        window.addEventListener('authChanged', () => {
            this.handleAuthChange();
        });
    },

    // 处理认证状态变化
    async handleAuthChange() {
        if (AuthModule.isLoggedIn()) {
            // 登录后从服务器加载数据
            await this.loadTodosFromServer();
        } else {
            // 登出后加载本地数据
            this.loadTodosFromLocal();
        }
        this.renderTodos();
    },

    // 创建或更新待办事项
    async createOrUpdateTodo() {
        const form = document.getElementById('todo-form');
        if (!form) return;

        const name = document.getElementById('todo-name').value;
        const description = document.getElementById('todo-description').value;
        const deadline = document.getElementById('todo-deadline').value;
        const category = document.getElementById('todo-category').value;
        const priority = document.getElementById('todo-priority').value;

        if (!name.trim()) {
            alert('请输入任务名称');
            return;
        }

        const isEditing = !!form.dataset.editId;
        const now = new Date().toISOString();

        if (isEditing) {
            // 更新现有任务
            const id = form.dataset.editId;
            const todoIndex = this.todos.findIndex(t => t.id === id);
            
            if (todoIndex !== -1) {
                this.todos[todoIndex] = {
                    ...this.todos[todoIndex],
                    name: name.trim(),
                    description: description.trim(),
                    deadline: deadline || null,
                    category: category.trim() || null,
                    priority: priority || null,
                    update_at: now
                };

                // 保存到本地
                this.saveTodosToLocal();
                
                // 保存到服务器
                if (AuthModule.isLoggedIn()) {
                    await this.saveTodoToServer(this.todos[todoIndex]);
                    // 触发同步
                    SyncModule.forceSync();
                }
            }
        } else {
            // 创建新任务
            const newTodo = {
                id: AuthModule.isLoggedIn() ? `temp_${Date.now()}` : Date.now().toString(),
                name: name.trim(),
                description: description.trim(),
                completed: false,
                create_at: now,
                update_at: now,
                deadline: deadline || null,
                category: category.trim() || null,
                priority: priority || null
            };

            this.todos.unshift(newTodo);
            
            // 保存到本地
            this.saveTodosToLocal();
            
            // 保存到服务器
            if (AuthModule.isLoggedIn()) {
                await this.saveTodoToServer(newTodo);
                // 触发同步
                SyncModule.forceSync();
            }
        }

        this.renderTodos();
        this.resetForm();
    },

    // 切换完成状态
    async toggleComplete(id) {
        const todo = this.todos.find(t => t.id === id);
        if (todo) {
            todo.completed = !todo.completed;
            todo.update_at = new Date().toISOString();
            
            // 保存到本地
            this.saveTodosToLocal();
            
            // 保存到服务器
            if (AuthModule.isLoggedIn()) {
                await this.saveTodoToServer(todo);
                // 触发同步
                SyncModule.forceSync();
            }
            
            this.renderTodos();
        }
    },

    // 编辑待办事项
    editTodo(id) {
        const todo = this.todos.find(t => t.id === id);
        if (!todo) return;

        const form = document.getElementById('todo-form');
        if (!form) return;

        // 填充表单
        document.getElementById('todo-name').value = todo.name;
        document.getElementById('todo-description').value = todo.description || '';
        document.getElementById('todo-deadline').value = todo.deadline ? this.formatDateTimeInput(todo.deadline) : '';
        document.getElementById('todo-category').value = todo.category || '';
        document.getElementById('todo-priority').value = todo.priority || '';

        // 保存当前编辑的ID
        form.dataset.editId = id;

        // 滚动到表单
        form.scrollIntoView({ behavior: 'smooth' });

        // 聚焦到名称输入框
        document.getElementById('todo-name').focus();
    },

    // 格式化日期时间输入
    formatDateTimeInput(dateTime) {
        const date = new Date(dateTime);
        return date.toISOString().slice(0, 16); // YYYY-MM-DDThh:mm
    },

    // 删除待办事项
    async deleteTodo(id) {
        if (confirm('确定要删除这个待办事项吗？')) {
            this.todos = this.todos.filter(t => t.id !== id);
            
            // 保存到本地
            this.saveTodosToLocal();
            
            // 从服务器删除
            if (AuthModule.isLoggedIn()) {
                await this.deleteTodoFromServer(id);
                // 触发同步
                SyncModule.forceSync();
            }
            
            this.renderTodos();
        }
    },

    // 重置表单
    resetForm() {
        const form = document.getElementById('todo-form');
        if (form) {
            form.reset();
            delete form.dataset.editId;
        }
    },

    // 获取所有待办事项
    getAllTodos() {
        return this.todos;
    },

    // 同步待办事项（从服务器到本地）
    syncTodos(serverTodos) {
        // 创建任务映射以便快速查找
        const serverTodoMap = new Map();
        serverTodos.forEach(todo => {
            serverTodoMap.set(todo.id, todo);
        });

        // 更新或添加任务
        const updatedTodos = [];
        
        // 处理服务器端的任务
        serverTodos.forEach(serverTodo => {
            const localIndex = this.todos.findIndex(t => t.id === serverTodo.id);
            if (localIndex !== -1) {
                // 更新本地任务
                this.todos[localIndex] = serverTodo;
            } else {
                // 添加新任务
                this.todos.push(serverTodo);
            }
            updatedTodos.push(serverTodo.id);
        });

        // 保存到本地
        this.saveTodosToLocal();
        
        return this.todos;
    },

    // 搜索待办事项
    searchTodos(query) {
        query = query.toLowerCase();
        return this.todos.filter(todo => 
            todo.name.toLowerCase().includes(query) ||
            (todo.description && todo.description.toLowerCase().includes(query)) ||
            (todo.category && todo.category.toLowerCase().includes(query))
        );
    },

    // 过滤待办事项
    filterTodos(filter) {
        switch (filter) {
            case 'completed':
                return this.todos.filter(todo => todo.completed);
            case 'pending':
                return this.todos.filter(todo => !todo.completed);
            case 'overdue':
                const now = new Date();
                return this.todos.filter(todo => 
                    !todo.completed && 
                    todo.deadline && 
                    new Date(todo.deadline) < now
                );
            default:
                return this.todos;
        }
    }
};

// 初始化TodoManager
document.addEventListener('DOMContentLoaded', () => {
    TodoManager.init();
});

// 导出模块
window.TodoManager = TodoManager;