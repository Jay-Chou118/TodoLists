// 认证模块
const AuthModule = {
    // 存储认证信息的键名
    STORAGE_KEYS: {
        TOKEN: 'auth_token',
        USER: 'user_info',
        DEVICE_ID: 'device_id',
        LAST_SYNC: 'last_sync_time'
    },

    // 设备信息
    deviceInfo: {
        name: '',
        id: ''
    },

    // 初始化
    init() {
        // 生成或获取设备ID
        this.initDevice();
        
        // 检查是否已登录
        const token = this.getToken();
        if (token) {
            this.verifyToken();
        }
    },

    // 初始化设备信息
    initDevice() {
        let deviceId = localStorage.getItem(this.STORAGE_KEYS.DEVICE_ID);
        if (!deviceId) {
            deviceId = this.generateDeviceId();
            localStorage.setItem(this.STORAGE_KEYS.DEVICE_ID, deviceId);
        }
        
        this.deviceInfo = {
            name: this.getDeviceName(),
            id: deviceId
        };
    },

    // 生成设备ID
    generateDeviceId() {
        const random = Math.random().toString(36).substring(2, 15);
        const timestamp = Date.now().toString(36);
        return `${random}_${timestamp}`;
    },

    // 获取设备名称
    getDeviceName() {
        const platform = navigator.platform;
        const browser = this.getBrowserInfo();
        return `${platform} ${browser}`;
    },

    // 获取浏览器信息
    getBrowserInfo() {
        const userAgent = navigator.userAgent;
        let browserName = 'Unknown';
        
        if (userAgent.indexOf('Firefox') > -1) {
            browserName = 'Firefox';
        } else if (userAgent.indexOf('Chrome') > -1) {
            browserName = 'Chrome';
        } else if (userAgent.indexOf('Safari') > -1) {
            browserName = 'Safari';
        } else if (userAgent.indexOf('Edge') > -1) {
            browserName = 'Edge';
        } else if (userAgent.indexOf('MSIE') > -1 || userAgent.indexOf('Trident/') > -1) {
            browserName = 'IE';
        }
        
        return browserName;
    },

    // 注册
    async register(username, password, email) {
        try {
            const response = await fetch('/api/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    username,
                    password,
                    email,
                    device_name: this.deviceInfo.name,
                    device_id: this.deviceInfo.id
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || '注册失败');
            }

            const data = await response.json();
            this.saveAuthData(data.token, data.user);
            
            // 注册设备
            await this.registerDevice();
            
            return { success: true, user: data.user };
        } catch (error) {
            console.error('注册错误:', error);
            return { success: false, error: error.message };
        }
    },

    // 登录
    async login(username, password) {
        try {
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    username,
                    password,
                    device_name: this.deviceInfo.name,
                    device_id: this.deviceInfo.id
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || '登录失败');
            }

            const data = await response.json();
            this.saveAuthData(data.token, data.user);
            
            // 更新设备信息
            await this.registerDevice();
            
            return { success: true, user: data.user };
        } catch (error) {
            console.error('登录错误:', error);
            return { success: false, error: error.message };
        }
    },

    // 验证令牌
    async verifyToken() {
        try {
            const response = await fetch('/api/verify', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                this.logout();
                return false;
            }

            const data = await response.json();
            this.saveUserInfo(data.user);
            return true;
        } catch (error) {
            console.error('验证令牌错误:', error);
            this.logout();
            return false;
        }
    },

    // 注册设备
    async registerDevice() {
        try {
            const response = await fetch('/api/devices/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.getToken()}`
                },
                body: JSON.stringify({
                    name: this.deviceInfo.name,
                    device_id: this.deviceInfo.id
                })
            });

            if (!response.ok) {
                console.error('注册设备失败');
                return false;
            }

            return true;
        } catch (error) {
            console.error('注册设备错误:', error);
            return false;
        }
    },

    // 获取所有设备
    async getDevices() {
        try {
            const response = await fetch('/api/devices', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`
                }
            });

            if (!response.ok) {
                throw new Error('获取设备列表失败');
            }

            return await response.json();
        } catch (error) {
            console.error('获取设备错误:', error);
            return [];
        }
    },

    // 注销
    logout() {
        localStorage.removeItem(this.STORAGE_KEYS.TOKEN);
        localStorage.removeItem(this.STORAGE_KEYS.USER);
        // 保留设备ID以便下次登录使用
    },

    // 保存认证数据
    saveAuthData(token, user) {
        localStorage.setItem(this.STORAGE_KEYS.TOKEN, token);
        localStorage.setItem(this.STORAGE_KEYS.USER, JSON.stringify(user));
    },

    // 保存用户信息
    saveUserInfo(user) {
        localStorage.setItem(this.STORAGE_KEYS.USER, JSON.stringify(user));
    },

    // 获取令牌
    getToken() {
        return localStorage.getItem(this.STORAGE_KEYS.TOKEN);
    },

    // 获取用户信息
    getUserInfo() {
        const userStr = localStorage.getItem(this.STORAGE_KEYS.USER);
        return userStr ? JSON.parse(userStr) : null;
    },

    // 是否已登录
    isLoggedIn() {
        return !!this.getToken();
    },

    // 更新最后同步时间
    updateLastSyncTime() {
        localStorage.setItem(this.STORAGE_KEYS.LAST_SYNC, new Date().toISOString());
    },

    // 获取最后同步时间
    getLastSyncTime() {
        const timeStr = localStorage.getItem(this.STORAGE_KEYS.LAST_SYNC);
        return timeStr ? new Date(timeStr) : new Date(0);
    }
};

// 初始化认证模块
document.addEventListener('DOMContentLoaded', () => {
    AuthModule.init();
});

// 导出模块
window.AuthModule = AuthModule;
