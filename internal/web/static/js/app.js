// API 请求封装
const api = {
    getToken() {
        return localStorage.getItem('jwt_token');
    },
    setToken(token) {
        localStorage.setItem('jwt_token', token);
    },
    removeToken() {
        localStorage.removeItem('jwt_token');
    },
    async request(method, url, data = null) {
        const headers = { 'Content-Type': 'application/json' };
        const token = this.getToken();
        if (token) headers['Authorization'] = 'Bearer ' + token;
        const options = { method, headers };
        if (data) options.body = JSON.stringify(data);
        const resp = await fetch(url, options);
        // 401 时跳转登录页
        if (resp.status === 401) {
            this.removeToken();
            window.location.href = '/login';
            return;
        }
        return await resp.json();
    },
    get(url) { return this.request('GET', url); },
    post(url, data) { return this.request('POST', url, data); },
    put(url, data) { return this.request('PUT', url, data); },
    del(url) { return this.request('DELETE', url); },
};

// Token 自动刷新（JWT 剩余不足 2 小时时刷新）
function setupTokenRefresh() {
    setInterval(async () => {
        const token = api.getToken();
        if (!token) return;
        try {
            // 解码 JWT payload 检查过期时间
            const payload = JSON.parse(atob(token.split('.')[1]));
            const exp = payload.exp * 1000;
            const remaining = exp - Date.now();
            if (remaining < 2 * 60 * 60 * 1000 && remaining > 0) {
                const result = await api.post('/api/v1/admin/refresh-token');
                if (result && result.code === 0) {
                    api.setToken(result.data.token);
                }
            }
        } catch (e) { console.error('Token refresh error:', e); }
    }, 5 * 60 * 1000); // 每 5 分钟检查一次
}

// 格式化日期
function formatDate(dateStr) {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleString('zh-CN');
}

// 格式化日期（仅日期部分）
function formatDateOnly(dateStr) {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-CN');
}

// 截断文本
function truncate(text, maxLength = 50) {
    if (!text) return '-';
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
}

// 格式化 JSON 显示
function formatJSON(obj) {
    try {
        if (typeof obj === 'string') {
            obj = JSON.parse(obj);
        }
        return JSON.stringify(obj, null, 2);
    } catch (e) {
        return String(obj);
    }
}

// 复制到剪贴板
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch (err) {
        console.error('复制失败:', err);
        return false;
    }
}

// 显示 Toast 通知
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    const bgColor = type === 'success' ? 'bg-green-500' : type === 'error' ? 'bg-red-500' : 'bg-blue-500';
    toast.className = `fixed top-4 right-4 ${bgColor} text-white px-6 py-3 rounded-lg shadow-lg z-50 transition-opacity duration-300`;
    toast.textContent = message;
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 登出
function logout() {
    api.removeToken();
    window.location.href = '/login';
}

// 检查登录状态
function checkAuth() {
    const token = api.getToken();
    if (!token) {
        window.location.href = '/login';
        return false;
    }
    return true;
}
