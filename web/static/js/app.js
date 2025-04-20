// 创建Vue应用
const app = Vue.createApp({
    data() {
        return {
            currentEmail: '',
            messages: [],
            refreshInterval: null,
            toast: {
                show: false,
                message: ''
            }
        };
    },
    methods: {
        // 生成新的临时邮箱
        async generateEmail() {
            try {
                const response = await axios.get('/api/email/new');
                if (response.data.status === 'success') {
                    this.currentEmail = response.data.email;
                    this.messages = [];
                    this.startAutoRefresh();
                    this.showToast('已生成新的临时邮箱');
                }
            } catch (error) {
                console.error('生成邮箱失败', error);
                this.showToast('生成邮箱失败，请重试');
            }
        },
        
        // 刷新邮件列表
        async refreshMessages() {
            if (!this.currentEmail) return;
            
            try {
                const response = await axios.get(`/api/email/${encodeURIComponent(this.currentEmail)}/messages`);
                if (response.data.status === 'success') {
                    // 排序邮件，最新的在前面
                    this.messages = response.data.messages.sort((a, b) => {
                        return new Date(b.timestamp) - new Date(a.timestamp);
                    });
                }
            } catch (error) {
                console.error('获取邮件失败', error);
                this.showToast('获取邮件失败，请重试');
            }
        },
        
        // 复制邮箱地址
        copyEmail() {
            navigator.clipboard.writeText(this.currentEmail)
                .then(() => {
                    this.showToast('邮箱地址已复制到剪贴板');
                })
                .catch(err => {
                    console.error('复制失败', err);
                    this.showToast('复制失败，请手动选择并复制');
                });
        },
        
        // 复制验证码
        copyCode(code) {
            navigator.clipboard.writeText(code)
                .then(() => {
                    this.showToast('验证码已复制到剪贴板');
                })
                .catch(err => {
                    console.error('复制失败', err);
                    this.showToast('复制失败，请手动选择并复制');
                });
        },
        
        // 显示提示信息
        showToast(message) {
            this.toast.message = message;
            this.toast.show = true;
            
            setTimeout(() => {
                this.toast.show = false;
            }, 3000);
        },
        
        // 格式化时间
        formatTime(timestamp) {
            const date = new Date(timestamp);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        },
        
        // 开始自动刷新
        startAutoRefresh() {
            // 清除之前的定时器
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
            }
            
            // 设置新的定时器，每10秒刷新一次
            this.refreshInterval = setInterval(() => {
                this.refreshMessages();
            }, 10000);
        }
    },
    mounted() {
        // 检查是否有存储在localStorage中的邮箱
        const savedEmail = localStorage.getItem('tempEmail');
        if (savedEmail) {
            this.currentEmail = savedEmail;
            this.refreshMessages();
            this.startAutoRefresh();
        }
        
        // 监听页面关闭，保存当前邮箱
        window.addEventListener('beforeunload', () => {
            if (this.currentEmail) {
                localStorage.setItem('tempEmail', this.currentEmail);
            }
        });
    },
    beforeUnmount() {
        // 清除定时器
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
});

// 挂载Vue应用
app.mount('#app'); 