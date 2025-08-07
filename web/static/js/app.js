// 主应用类
class AnalysisApp {
    constructor() {
        this.eventSource = null;
        this.init();
    }
    
    init() {
        this.bindEvents();
        this.initSSE();
    }
    
    bindEvents() {
        const form = document.getElementById('analysisForm');
        if (form) {
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitAnalysis();
            });
        }
    }
    
    submitAnalysis() {
        const symbol = document.getElementById('symbol').value;
        const market = document.getElementById('market').value;
        
        if (!symbol || !market) {
            alert('请填写完整的股票代码和市场信息');
            return;
        }
        
        window.location.href = `/analysis/${symbol}.${market}`;
    }
    
    initSSE() {
        const taskId = this.getTaskIdFromUrl();
        if (!taskId) return;
        
        this.eventSource = new EventSource(`/api/stream/${taskId}`);
        
        this.eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleMessage(data);
            } catch (error) {
                console.error('Failed to parse SSE message:', error);
            }
        };
        
        this.eventSource.onerror = (error) => {
            console.error('SSE error:', error);
            this.showError('连接失败，请刷新页面重试');
            if (this.eventSource) {
                this.eventSource.close();
            }
        };
    }
    
    handleMessage(data) {
        switch (data.status) {
            case 'processing':
                this.updateProgress(data.message, data.percentage);
                break;
            case 'completed':
                this.showResult(data.result);
                if (this.eventSource) {
                    this.eventSource.close();
                }
                break;
            case 'failed':
                this.showError(data.error || '分析失败');
                if (this.eventSource) {
                    this.eventSource.close();
                }
                break;
        }
    }
    
    updateProgress(message, percentage) {
        const progressContainer = document.getElementById('progress');
        const progressText = document.getElementById('progressText');
        const progressFill = document.getElementById('progressFill');
        
        if (progressContainer) {
            progressContainer.style.display = 'block';
        }
        
        if (progressText) {
            progressText.textContent = message;
        }
        
        if (progressFill) {
            progressFill.style.width = percentage + '%';
        }
    }
    
    showResult(result) {
        const resultContainer = document.getElementById('result');
        const dailyChart = document.getElementById('dailyChart');
        const hourlyChart = document.getElementById('hourlyChart');
        const analysisContent = document.getElementById('analysisContent');
        
        if (resultContainer) {
            resultContainer.style.display = 'block';
        }
        
        if (result && result.daily_chart && dailyChart) {
            dailyChart.src = result.daily_chart;
        }
        
        if (result && result.hourly_chart && hourlyChart) {
            hourlyChart.src = result.hourly_chart;
        }
        
        if (result && result.analysis && analysisContent) {
            // 加载Markdown内容
            fetch(result.analysis)
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Failed to load analysis');
                    }
                    return response.text();
                })
                .then(markdown => {
                    const html = this.markdownToHtml(markdown);
                    analysisContent.innerHTML = html;
                })
                .catch(error => {
                    console.error('Failed to load analysis:', error);
                    analysisContent.innerHTML = '<p>分析内容加载失败</p>';
                });
        }
    }
    
    showError(error) {
        const errorContainer = document.getElementById('error');
        const errorMessage = document.getElementById('errorMessage');
        
        if (errorContainer) {
            errorContainer.style.display = 'block';
        }
        
        if (errorMessage) {
            errorMessage.textContent = error;
        }
    }
    
    markdownToHtml(markdown) {
        if (typeof marked !== 'undefined') {
            return marked.parse(markdown);
        }
        // 如果没有marked库，返回简单的HTML
        return `<pre>${markdown}</pre>`;
    }
    
    getTaskIdFromUrl() {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('taskId');
    }
}

// 全局函数，用于分析页面自动开始分析
function startAnalysis(symbolMarket) {
    if (!symbolMarket) return;
    
    // 创建分析任务
    fetch(`/api/analysis/${symbolMarket}`)
        .then(response => response.json())
        .then(data => {
            if (data.task_id) {
                // 开始SSE连接
                const app = new AnalysisApp();
                // 这里可以添加任务ID到URL或存储到localStorage
            }
        })
        .catch(error => {
            console.error('Failed to create analysis task:', error);
        });
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', function() {
    new AnalysisApp();
}); 