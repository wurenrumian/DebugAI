// 全局配置
const API_BASE_URL = 'http://localhost:8000';
let currentConversationId = generateConversationId();

// DOM元素
const elements = {
    studentId: document.getElementById('studentId'),
    conversationId: document.getElementById('conversationId'),
    problemType: document.getElementById('problemType'),
    problemDescription: document.getElementById('problemDescription'),
    codeEditor: document.getElementById('codeEditor'),
    testPoints: document.getElementById('testPoints'),
    submitBtn: document.getElementById('submitBtn'),
    clearBtn: document.getElementById('clearBtn'),
    exampleBtn: document.getElementById('exampleBtn'),
    scoreValue: document.getElementById('scoreValue'),
    readabilityScore: document.getElementById('readabilityScore'),
    logicScore: document.getElementById('logicScore'),
    algorithmScore: document.getElementById('algorithmScore'),
    efficiencyScore: document.getElementById('efficiencyScore'),
    overallEvaluationText: document.getElementById('overallEvaluationText'),
    readabilityDetail: document.getElementById('readabilityDetail'),
    logicDetail: document.getElementById('logicDetail'),
    algorithmDetail: document.getElementById('algorithmDetail'),
    efficiencyDetail: document.getElementById('efficiencyDetail'),
    rawData: document.getElementById('rawData'),
    debugAnalysisText: document.getElementById('debugAnalysisText'),
    problemsList: document.getElementById('problemsList'),
    suggestionsList: document.getElementById('suggestionsList'),
    debugSection: document.getElementById('debugSection'),
    apiStatus: document.getElementById('apiStatus'),
    apiEndpoint: document.getElementById('apiEndpoint')
};

// 初始化
function init() {
    // 设置API端点显示
    elements.apiEndpoint.textContent = API_BASE_URL;
    
    // 生成会话ID
    elements.conversationId.value = currentConversationId;
    
    // 检查API连接
    checkApiConnection();
    
    // 添加行号到代码编辑器
    updateLineNumbers();
    elements.codeEditor.addEventListener('input', updateLineNumbers);
    
    // 事件监听
    elements.submitBtn.addEventListener('click', submitAnalysis);
    elements.clearBtn.addEventListener('click', clearForm);
    elements.exampleBtn.addEventListener('click', loadExample);
    
    // 标签页切换
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            const tabId = this.getAttribute('data-tab');
            switchTab(tabId);
        });
    });
    
    // 任务类型切换时更新UI
    elements.problemType.addEventListener('change', function() {
        const isDebugMode = this.value === 'debug';
        elements.debugSection.style.display = isDebugMode ? 'block' : 'none';
    });
}

// 生成会话ID
function generateConversationId() {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 8);
    return `conv_${timestamp}_${random}`;
}

// 检查API连接
async function checkApiConnection() {
    try {
        const response = await fetch(`${API_BASE_URL}/docs`);
        if (response.ok) {
            elements.apiStatus.textContent = '已连接';
            elements.apiStatus.className = 'connected';
        } else {
            elements.apiStatus.textContent = '连接失败';
        }
    } catch (error) {
        elements.apiStatus.textContent = '未连接';
        console.warn('API连接检查失败:', error);
    }
}

// 更新代码行号
function updateLineNumbers() {
    const code = elements.codeEditor.value;
    const lines = code.split('\n').length;
    const lineNumbers = Array.from({length: lines}, (_, i) => i + 1).join('\n');
    
    const lineNumbersElement = document.getElementById('lineNumbers');
    if (lineNumbersElement) {
        lineNumbersElement.textContent = lineNumbers;
    }
}

// 提交分析
async function submitAnalysis() {
    // 获取表单数据
    const requestData = {
        student_id: elements.studentId.value || 'anonymous',
        conversation_id: elements.conversationId.value,
        problem_description: elements.problemDescription.value,
        code: elements.codeEditor.value,
        task_type: elements.problemType.value
    };
    
    // 解析测试点
    try {
        if (elements.testPoints.value.trim()) {
            requestData.test_points = JSON.parse(elements.testPoints.value);
        }
    } catch (error) {
        alert('测试点JSON格式错误！请检查格式。');
        return;
    }
    
    // 显示加载状态
    showLoading(true);
    
    // 确定API端点
    const endpoint = elements.problemType.value === 'debug' ? '/debug' : '/evaluate';
    
    try {
        // 发送请求
        const response = await fetch(`${API_BASE_URL}${endpoint}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestData)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP错误: ${response.status}`);
        }
        
        const result = await response.json();
        
        // 处理结果
        displayResults(result);
        
        // 切换到概览标签
        switchTab('overview');
        
    } catch (error) {
        console.error('请求失败:', error);
        alert(`分析请求失败: ${error.message}`);
    } finally {
        showLoading(false);
    }
}

// 显示结果
function displayResults(result) {
    // 显示原始数据
    elements.rawData.textContent = JSON.stringify(result, null, 2);
    
    if (elements.problemType.value === 'debug') {
        // 调试模式
        displayDebugResults(result);
    } else {
        // 评价模式
        displayEvaluationResults(result);
    }
}

// 显示评价结果
function displayEvaluationResults(result) {
    // 更新分数
    elements.scoreValue.textContent = result.score || '--';
    
    // 更新各项分数
    if (result.readability) {
        const readabilityScore = parseScore(result.readability.score);
        elements.readabilityScore.textContent = result.readability.score;
        updateBreakdownBar('readability', readabilityScore.percentage);
        elements.readabilityDetail.textContent = result.readability.analysis || '无分析';
    }
    
    if (result.logical_rigor) {
        const logicScore = parseScore(result.logical_rigor.score);
        elements.logicScore.textContent = result.logical_rigor.score;
        updateBreakdownBar('logic', logicScore.percentage);
        elements.logicDetail.textContent = result.logical_rigor.analysis || '无分析';
    }
    
    if (result.algorithm_quality) {
        const algorithmScore = parseScore(result.algorithm_quality.score);
        elements.algorithmScore.textContent = result.algorithm_quality.score;
        updateBreakdownBar('algorithm', algorithmScore.percentage);
        elements.algorithmDetail.textContent = result.algorithm_quality.analysis || '无分析';
    }
    
    if (result.efficiency) {
        const efficiencyScore = parseScore(result.efficiency.score);
        elements.efficiencyScore.textContent = result.efficiency.score;
        updateBreakdownBar('efficiency', efficiencyScore.percentage);
        elements.efficiencyDetail.textContent = result.efficiency.analysis || '无分析';
    }
    
    // 整体评价
    elements.overallEvaluationText.textContent = result.overall_evaluation || '无评价';
}

// 显示调试结果
function displayDebugResults(result) {
    // 调试分析
    elements.debugAnalysisText.textContent = result.debug_analysis || '无调试分析';
    
    // 具体问题
    elements.problemsList.innerHTML = '';
    if (result.problems && Array.isArray(result.problems)) {
        result.problems.forEach(problem => {
            const problemElement = document.createElement('div');
            problemElement.className = 'problem-item';
            problemElement.innerHTML = `
                <div class="problem-location">位置: ${problem.location || '未知'}</div>
                <div class="problem-description">描述: ${problem.description || '无描述'}</div>
                <div class="problem-root-cause">原因: ${problem.root_cause || '未知原因'}</div>
            `;
            elements.problemsList.appendChild(problemElement);
        });
    }
    
    // 修改建议
    elements.suggestionsList.innerHTML = '';
    if (result.suggestions && Array.isArray(result.suggestions)) {
        result.suggestions.forEach(suggestion => {
            const li = document.createElement('li');
            li.textContent = suggestion;
            elements.suggestionsList.appendChild(li);
        });
    }
}

// 解析分数字符串（如 "8/10"）
function parseScore(scoreStr) {
    if (!scoreStr) return { obtained: 0, total: 1, percentage: 0 };
    
    const parts = scoreStr.split('/');
    if (parts.length !== 2) return { obtained: 0, total: 1, percentage: 0 };
    
    const obtained = parseFloat(parts[0]) || 0;
    const total = parseFloat(parts[1]) || 1;
    const percentage = total > 0 ? (obtained / total) * 100 : 0;
    
    return { obtained, total, percentage };
}

// 更新分数条
function updateBreakdownBar(type, percentage) {
    const barElement = document.querySelector(`.breakdown-fill[style*="width: 0%"]`);
    if (barElement) {
        barElement.style.width = `${Math.min(percentage, 100)}%`;
    }
}

// 切换标签页
function switchTab(tabId) {
    // 更新按钮状态
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.getAttribute('data-tab') === tabId) {
            btn.classList.add('active');
        }
    });
    
    // 更新内容区域
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
        if (pane.id === `${tabId}Tab`) {
            pane.classList.add('active');
        }
    });
}

// 清空表单
function clearForm() {
    if (confirm('确定要清空所有输入吗？')) {
        elements.problemDescription.value = '';
        elements.codeEditor.value = '';
        elements.testPoints.value = '';
        updateLineNumbers();
        
        // 重置结果区域
        resetResults();
    }
}

// 重置结果区域
function resetResults() {
    elements.scoreValue.textContent = '--';
    elements.readabilityScore.textContent = '0/10';
    elements.logicScore.textContent = '0/40';
    elements.algorithmScore.textContent = '0/25';
    elements.efficiencyScore.textContent = '0/25';
    elements.overallEvaluationText.textContent = '等待分析结果...';
    elements.readabilityDetail.textContent = '等待分析结果...';
    elements.logicDetail.textContent = '等待分析结果...';
    elements.algorithmDetail.textContent = '等待分析结果...';
    elements.efficiencyDetail.textContent = '等待分析结果...';
    elements.rawData.textContent = '{"status": "等待请求数据..."}';
    elements.debugAnalysisText.textContent = '等待调试结果...';
    elements.problemsList.innerHTML = '';
    elements.suggestionsList.innerHTML = '';
    
    // 重置分数条
    document.querySelectorAll('.breakdown-fill').forEach(bar => {
        bar.style.width = '0%';
    });
}

// 加载示例
function loadExample() {
    elements.problemDescription.value = `编写一个C函数，判断一个整数是否为素数。
要求：
1. 函数原型：int is_prime(int n)
2. 如果n是素数返回1，否则返回0
3. 注意处理特殊情况（n<=1）
4. 优化算法效率`;

    elements.codeEditor.value = `#include <stdio.h>
#include <math.h>

int is_prime(int n) {
    if (n <= 1) return 0;
    if (n == 2) return 1;
    if (n % 2 == 0) return 0;
    
    for (int i = 3; i <= sqrt(n); i += 2) {
        if (n % i == 0) {
            return 0;
        }
    }
    return 1;
}

int main() {
    int num;
    printf("请输入一个整数: ");
    scanf("%d", &num);
    
    if (is_prime(num)) {
        printf("%d 是素数\\n", num);
    } else {
        printf("%d 不是素数\\n", num);
    }
    
    return 0;
}`;

    elements.testPoints.value = `[
  {"input": "2", "status": "Accepted"},
  {"input": "17", "status": "Accepted"},
  {"input": "1", "status": "Accepted"},
  {"input": "4", "status": "Accepted"},
  {"input": "-5", "status": "Accepted"}
]`;

    updateLineNumbers();
}

// 显示/隐藏加载状态
function showLoading(show) {
    if (show) {
        elements.submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 分析中...';
        elements.submitBtn.disabled = true;
    } else {
        elements.submitBtn.innerHTML = '<i class="fas fa-paper-plane"></i> 提交分析';
        elements.submitBtn.disabled = false;
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', init);