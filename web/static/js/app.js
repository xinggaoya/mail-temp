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
            },
            showEmailList: false,
            activeEmails: [],
            isLoading: false
        };
    },
    methods: {
        // 生成新的临时邮箱
        async generateEmail() {
            try {
                // 立即清空邮件列表，避免显示旧邮箱的邮件
                this.messages = [];
                this.isLoading = true;
                
                const response = await axios.get('/api/email/new');
                if (response.data.status === 'success') {
                    this.currentEmail = response.data.email;
                    this.startAutoRefresh();
                    this.showToast('已生成新的临时邮箱');
                }
            } catch (error) {
                console.error('生成邮箱失败', error);
                this.showToast('生成邮箱失败，请重试');
            } finally {
                this.isLoading = false;
            }
        },
        
        // 刷新邮件列表
        async refreshMessages(showLoading = true) {
            if (!this.currentEmail) return;
            
            try {
                // 只在手动刷新或者切换邮箱时显示加载状态
                if (showLoading) {
                    this.isLoading = true;
                }
                
                const response = await axios.get(`/api/email/${encodeURIComponent(this.currentEmail)}/messages`);
                if (response.data.status === 'success') {
                    // 排序邮件，最新的在前面
                    this.messages = response.data.messages.sort((a, b) => {
                        return new Date(b.timestamp) - new Date(a.timestamp);
                    });
                }
            } catch (error) {
                console.error('获取邮件失败', error);
                // 只在手动刷新时显示错误提示
                if (showLoading) {
                    this.showToast('获取邮件失败，请重试');
                }
            } finally {
                if (showLoading) {
                    this.isLoading = false;
                }
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
            if (!timestamp) return '';
            const date = new Date(timestamp);
            const now = new Date();
            const isToday = date.toDateString() === now.toDateString();
            
            if (isToday) {
                return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
            } else {
                return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
            }
        },
        
        // 解码邮件主题，处理特殊编码
        decodeEmailSubject(subject) {
            if (!subject) return '';
            
            // 尝试解码 =?UTF-8?B?...?= 格式（Base64编码的UTF-8）
            const b64Regex = /=\?utf-8\?B\?([a-zA-Z0-9+/=]+)\?=/i;
            if (b64Regex.test(subject)) {
                try {
                    const match = subject.match(b64Regex);
                    if (match && match[1]) {
                        const decoded = atob(match[1]);
                        // 将Base64解码后的二进制转为UTF-8字符串
                        const bytes = new Uint8Array(decoded.length);
                        for (let i = 0; i < decoded.length; i++) {
                            bytes[i] = decoded.charCodeAt(i);
                        }
                        return new TextDecoder('utf-8').decode(bytes);
                    }
                } catch (e) {
                    console.error('Base64解码失败:', e);
                }
            }
            
            // 尝试解码 =?UTF-8?Q?...?= 格式（Quoted-Printable编码的UTF-8）
            const qpRegex = /=\?utf-8\?Q\?([^\?]+)\?=/i;
            if (qpRegex.test(subject)) {
                try {
                    const match = subject.match(qpRegex);
                    if (match && match[1]) {
                        let qpText = match[1].replace(/_/g, ' ');
                        // 解码Quoted-Printable
                        qpText = qpText.replace(/=([0-9A-F]{2})/g, (_, hex) => {
                            return String.fromCharCode(parseInt(hex, 16));
                        });
                        return qpText;
                    }
                } catch (e) {
                    console.error('Quoted-Printable解码失败:', e);
                }
            }
            
            return subject;
        },
        
        // 开始自动刷新
        startAutoRefresh() {
            // 清除之前的定时器
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
            }
            
            // 设置新的定时器，每10秒刷新一次，但不显示加载状态
            this.refreshInterval = setInterval(() => {
                this.refreshMessages(false);
            }, 10000);
        },
        
        // 显示活跃邮箱列表
        async showActiveEmails() {
            try {
                await this.fetchActiveEmails();
                this.showEmailList = true;
            } catch (error) {
                console.error('获取活跃邮箱列表失败', error);
                this.showToast('获取活跃邮箱列表失败，请重试');
            }
        },
        
        // 关闭活跃邮箱列表
        closeEmailList() {
            this.showEmailList = false;
        },
        
        // 获取活跃邮箱列表
        async fetchActiveEmails() {
            try {
                const response = await axios.get('/api/email/list');
                if (response.data.status === 'success') {
                    this.activeEmails = response.data.emails;
                }
            } catch (error) {
                console.error('获取活跃邮箱列表失败', error);
                throw error;
            }
        },
        
        // 选择邮箱
        async selectEmail(email) {
            // 如果选择了不同的邮箱，则清空当前邮件列表
            if (this.currentEmail !== email) {
                this.currentEmail = email;
                // 立即清空邮件列表，避免显示旧邮箱的邮件
                this.messages = [];
                // 显示加载指示
                this.showToast(`正在加载邮箱: ${email}`);
            }
            
            this.closeEmailList();
            
            // 加载新邮箱的邮件，显示加载状态
            await this.refreshMessages(true);
            this.startAutoRefresh();
            this.showToast(`已切换到邮箱: ${email}`);
        },
        
        // 删除邮箱
        async deleteEmail(email) {
            try {
                const response = await axios.delete(`/api/email/${encodeURIComponent(email)}`);
                if (response.data.status === 'success') {
                    // 从列表中移除
                    this.activeEmails = this.activeEmails.filter(e => e !== email);
                    
                    // 如果删除的是当前邮箱，清空当前邮箱
                    if (this.currentEmail === email) {
                        this.currentEmail = '';
                        this.messages = [];
                        if (this.refreshInterval) {
                            clearInterval(this.refreshInterval);
                            this.refreshInterval = null;
                        }
                    }
                    
                    this.showToast('邮箱已删除');
                    
                    // 如果没有更多活跃邮箱，关闭列表
                    if (this.activeEmails.length === 0) {
                        this.closeEmailList();
                    }
                }
            } catch (error) {
                console.error('删除邮箱失败', error);
                this.showToast('删除邮箱失败，请重试');
            }
        },
        
        // 处理HTML内容，修复常见的编码问题
        processHtmlContent(html) {
            if (!html) return '';
            
            // 替换常见的编码问题
            html = html.replace(/=3D/g, '=');
            html = html.replace(/=\\r\\n/g, '');
            html = html.replace(/=\r\n/g, '');
            
            // 修复HTML中常见的引号和属性问题
            html = html.replace(/=22/g, '"');
            html = html.replace(/=27/g, "'");
            html = html.replace(/=20/g, ' ');
            
            // 移除邮件客户端特有的标记
            html = html.replace(/\(MISSING\)/g, '');
            
            // 处理邮件中的图片链接，确保使用https
            html = html.replace(/src="http:\/\//g, 'src="https://');
            
            // 为外部链接添加安全属性
            html = html.replace(/<a\s+(?!target)/g, '<a target="_blank" rel="noopener nofollow" ');
            
            // 对于没有指定样式的图片，设置最大宽度为100%
            html = html.replace(/<img(?!\s+style)/g, '<img style="max-width:100%;height:auto" ');
            
            return html;
        },
        
        // 判断是否应该显示滚动提示
        shouldShowScrollHint(body) {
            return body && (body.length > 300 || body.includes('DKIM-Signature') || body.includes('-------'));
        }
    },
    mounted() {
        // 检查是否有存储在localStorage中的邮箱
        const savedEmail = localStorage.getItem('tempEmail');
        if (savedEmail) {
            this.currentEmail = savedEmail;
            this.refreshMessages(true); // 初始加载显示加载状态
            this.startAutoRefresh();
        }
        
        // 监听页面关闭，保存当前邮箱
        window.addEventListener('beforeunload', () => {
            if (this.currentEmail) {
                localStorage.setItem('tempEmail', this.currentEmail);
            }
        });
        
        // 监听ESC键，关闭弹窗
        window.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.showEmailList) {
                this.closeEmailList();
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