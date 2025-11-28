// 同步模块
const SyncModule = {
    // 同步状态
    syncStatus: {
        isSyncing: false,
        lastSync: null,
        syncError: null
    },

    // 初始化
    init() {
        // 设置定期同步
        this.setupPeriodicSync();
        
        // 监听网络状态变化
        this.setupNetworkListener();
    },

    // 设置定期同步
    setupPeriodicSync() {
        // 每30秒同步一次
        setInterval(() => {
            if (AuthModule.isLoggedIn() && navigator.onLine) {
                this.syncData();
            }
        }, 30000);
    },

    // 设置网络状态监听
    setupNetworkListener() {
        window.addEventListener('online', () => {
            if (AuthModule.isLoggedIn()) {
                // 网络恢复时立即同步
                this.syncData();
            }
        });
    },

    // 同步数据
    async syncData() {
        if (!AuthModule.isLoggedIn() || this.syncStatus.isSyncing) {
            return;
        }

        this.syncStatus.isSyncing = true;
        this.syncStatus.syncError = null;

        try {
            // 获取本地待同步的数据
            const localTodos = await this.getLocalTodos();
            
            // 获取最后同步时间
            const lastSyncAt = AuthModule.getLastSyncTime();
            
            // 发送同步请求
            const response = await fetch('/api/sync', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                },
                body: JSON.stringify({
                    last_sync_at: lastSyncAt.toISOString(),
                    todos: localTodos
                })
            });

            if (!response.ok) {
                throw new Error('同步请求失败');
            }

            const syncResponse = await response.json();
            
            // 处理服务器返回的数据
            await this.processSyncResponse(syncResponse);
            
            // 更新同步状态
            this.syncStatus.lastSync = new Date();
            AuthModule.updateLastSyncTime();
            
            return { success: true, data: syncResponse };
        } catch (error) {
            console.error('同步错误:', error);
            this.syncStatus.syncError = error.message;
            return { success: false, error: error.message };
        } finally {
            this.syncStatus.isSyncing = false;
            // 触发同步完成事件
            this.triggerSyncCompleteEvent();
        }
    },

    // 获取本地待同步的数据
    async getLocalTodos() {
        // 从TodoManager获取所有任务
        return TodoManager.getAllTodos();
    },

    // 处理同步响应
    async processSyncResponse(response) {
        const { todos, conflicts } = response;
        
        // 更新本地任务列表
        if (todos && todos.length > 0) {
            await TodoManager.syncTodos(todos);
        }
        
        // 处理冲突
        if (conflicts && conflicts.length > 0) {
            this.handleConflicts(conflicts);
        }
    },

    // 处理冲突
    handleConflicts(conflicts) {
        // 存储冲突到本地
        this.storeConflicts(conflicts);
        
        // 显示冲突解决界面
        this.showConflictResolutionUI(conflicts);
    },

    // 存储冲突
    storeConflicts(conflicts) {
        localStorage.setItem('sync_conflicts', JSON.stringify(conflicts));
    },

    // 获取存储的冲突
    getStoredConflicts() {
        const conflictsStr = localStorage.getItem('sync_conflicts');
        return conflictsStr ? JSON.parse(conflictsStr) : [];
    },

    // 显示冲突解决界面
    showConflictResolutionUI(conflicts) {
        // 创建冲突解决模态框
        const modal = this.createConflictModal(conflicts);
        document.body.appendChild(modal);
        
        // 显示模态框
        modal.style.display = 'block';
    },

    // 创建冲突解决模态框
    createConflictModal(conflicts) {
        const modal = document.createElement('div');
        modal.className = 'conflict-modal';
        modal.style.cssText = `
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.5);
            z-index: 1000;
            display: flex;
            align-items: center;
            justify-content: center;
        `;

        const modalContent = document.createElement('div');
        modalContent.className = 'conflict-modal-content';
        modalContent.style.cssText = `
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            max-width: 600px;
            width: 90%;
            max-height: 80vh;
            overflow-y: auto;
        `;

        modalContent.innerHTML = `
            <h3>检测到数据冲突</h3>
            <p>有 ${conflicts.length} 个任务在多个设备上被修改，请选择要保留的版本：</p>
            <div class="conflicts-list"></div>
            <div class="modal-actions" style="margin-top: 20px; text-align: right;">
                <button id="resolve-all-server" class="btn btn-secondary" style="margin-right: 10px;">全部保留服务器版本</button>
                <button id="resolve-all-local" class="btn btn-secondary" style="margin-right: 10px;">全部保留本地版本</button>
                <button id="close-conflict-modal" class="btn btn-primary">完成</button>
            </div>
        `;

        // 添加每个冲突项
        const conflictsList = modalContent.querySelector('.conflicts-list');
        conflicts.forEach((conflict, index) => {
            const conflictItem = this.createConflictItem(conflict, index);
            conflictsList.appendChild(conflictItem);
        });

        modal.appendChild(modalContent);

        // 添加事件监听器
        this.addConflictModalEvents(modal, conflicts);

        return modal;
    },

    // 创建冲突项
    createConflictItem(conflict, index) {
        const item = document.createElement('div');
        item.className = 'conflict-item';
        item.style.cssText = `
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            margin-bottom: 15px;
        `;

        const localTime = new Date(conflict.localTodo.update_at).toLocaleString();
        const serverTime = new Date(conflict.serverTodo.update_at).toLocaleString();

        item.innerHTML = `
            <div class="conflict-header" style="margin-bottom: 10px;">
                <h4>任务: ${conflict.localTodo.name}</h4>
            </div>
            <div class="conflict-versions">
                <div class="version local-version" style="margin-bottom: 10px; padding: 10px; background-color: #f8f9fa; border-left: 3px solid #2196F3; border-radius: 4px;">
                    <div class="version-header" style="font-weight: bold; margin-bottom: 5px;">
                        <label>
                            <input type="radio" name="conflict-${index}" value="local" checked> 本地版本 (${localTime})
                        </label>
                    </div>
                    <div class="version-content">
                        <p><strong>描述:</strong> ${conflict.localTodo.description || '无'}</p>
                        <p><strong>状态:</strong> ${conflict.localTodo.completed ? '已完成' : '未完成'}</p>
                        <p><strong>截止日期:</strong> ${conflict.localTodo.deadline ? new Date(conflict.localTodo.deadline).toLocaleString() : '无'}</p>
                        <p><strong>分类:</strong> ${conflict.localTodo.category || '无'}</p>
                        <p><strong>优先级:</strong> ${conflict.localTodo.priority || '无'}</p>
                    </div>
                </div>
                <div class="version server-version" style="padding: 10px; background-color: #f8f9fa; border-left: 3px solid #4CAF50; border-radius: 4px;">
                    <div class="version-header" style="font-weight: bold; margin-bottom: 5px;">
                        <label>
                            <input type="radio" name="conflict-${index}" value="server"> 服务器版本 (${serverTime})
                        </label>
                    </div>
                    <div class="version-content">
                        <p><strong>描述:</strong> ${conflict.serverTodo.description || '无'}</p>
                        <p><strong>状态:</strong> ${conflict.serverTodo.completed ? '已完成' : '未完成'}</p>
                        <p><strong>截止日期:</strong> ${conflict.serverTodo.deadline ? new Date(conflict.serverTodo.deadline).toLocaleString() : '无'}</p>
                        <p><strong>分类:</strong> ${conflict.serverTodo.category || '无'}</p>
                        <p><strong>优先级:</strong> ${conflict.serverTodo.priority || '无'}</p>
                    </div>
                </div>
            </div>
        `;

        return item;
    },

    // 添加冲突模态框事件
    addConflictModalEvents(modal, conflicts) {
        // 关闭按钮
        modal.querySelector('#close-conflict-modal').addEventListener('click', async () => {
            // 收集用户选择的解决方案
            const resolvedTodos = [];
            
            conflicts.forEach((conflict, index) => {
                const selectedVersion = modal.querySelector(`input[name="conflict-${index}"]:checked`).value;
                if (selectedVersion === 'local') {
                    resolvedTodos.push(conflict.localTodo);
                } else {
                    resolvedTodos.push(conflict.serverTodo);
                }
            });

            // 提交解决方案
            await this.submitConflictResolution(resolvedTodos);
            
            // 关闭模态框
            document.body.removeChild(modal);
            
            // 重新同步
            this.syncData();
        });

        // 全部保留服务器版本
        modal.querySelector('#resolve-all-server').addEventListener('click', () => {
            conflicts.forEach((_, index) => {
                modal.querySelector(`input[name="conflict-${index}"][value="server"]`).checked = true;
            });
        });

        // 全部保留本地版本
        modal.querySelector('#resolve-all-local').addEventListener('click', () => {
            conflicts.forEach((_, index) => {
                modal.querySelector(`input[name="conflict-${index}"][value="local"]`).checked = true;
            });
        });
    },

    // 提交冲突解决方案
    async submitConflictResolution(resolvedTodos) {
        try {
            const response = await fetch('/api/conflicts/resolve', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                },
                body: JSON.stringify(resolvedTodos)
            });

            if (!response.ok) {
                throw new Error('提交冲突解决方案失败');
            }

            // 清除存储的冲突
            localStorage.removeItem('sync_conflicts');
            
            return true;
        } catch (error) {
            console.error('提交冲突解决方案错误:', error);
            return false;
        }
    },

    // 批量更新任务
    async batchUpdateTodos(todos) {
        if (!AuthModule.isLoggedIn()) {
            return { success: false, error: '未登录' };
        }

        try {
            const response = await fetch('/api/todos/batch', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${AuthModule.getToken()}`
                },
                body: JSON.stringify(todos)
            });

            if (!response.ok) {
                throw new Error('批量更新失败');
            }

            // 更新同步时间
            AuthModule.updateLastSyncTime();
            
            return { success: true };
        } catch (error) {
            console.error('批量更新错误:', error);
            return { success: false, error: error.message };
        }
    },

    // 触发同步完成事件
    triggerSyncCompleteEvent() {
        const event = new CustomEvent('syncComplete', {
            detail: {
                lastSync: this.syncStatus.lastSync,
                error: this.syncStatus.syncError
            }
        });
        window.dispatchEvent(event);
    },

    // 获取同步状态
    getSyncStatus() {
        return this.syncStatus;
    },

    // 立即同步（手动触发）
    forceSync() {
        return this.syncData();
    }
};

// 初始化同步模块
document.addEventListener('DOMContentLoaded', () => {
    SyncModule.init();
});

// 导出模块
window.SyncModule = SyncModule;
