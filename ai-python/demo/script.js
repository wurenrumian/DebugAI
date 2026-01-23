// 全局配置
const API_BASE_URL = 'http://localhost:8000';
let currentConversationId = generateConversationId();

// DOM元素
const elements = {
    studentId: document.getElementById('studentId'),
    conversationId: document.getElementById('conversationId'),
    enableEvaluate: document.getElementById('enableEvaluate'),
    enableDebug: document.getElementById('enableDebug'),
    enableRecommend: document.getElementById('enableRecommend'),
    problemDescription: document.getElementById('problemDescription'),
    codeEditor: document.getElementById('codeEditor'),
    testPoints: document.getElementById('testPoints'),
    recommendInputSection: document.getElementById('recommendInputSection'),
    weakPoints: document.getElementById('weakPoints'),
    yojDatabase: document.getElementById('yojDatabase'),
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
    debugAnalysisText: document.getElementById('debugAnalysisText'),
    problemsList: document.getElementById('problemsList'),
    suggestionsList: document.getElementById('suggestionsList'),
    recommendationSummary: document.getElementById('recommendationSummary'),
    recommendationList: document.getElementById('recommendationList'),
    recommendationAnalysis: document.getElementById('recommendationAnalysis'),
    rawData: document.getElementById('rawData'),
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

    // 推荐功能复选框事件监听
    elements.enableRecommend.addEventListener('change', function () {
        elements.recommendInputSection.style.display = this.checked ? 'block' : 'none';
    });

    // 标签页切换
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', function () {
            const tabId = this.getAttribute('data-tab');
            switchTab(tabId);
        });
    });

    // 初始化显示推荐输入区域
    if (elements.enableRecommend.checked) {
        elements.recommendInputSection.style.display = 'block';
    }
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
    const lineNumbers = Array.from({ length: lines }, (_, i) => i + 1).join('\n');

    const lineNumbersElement = document.getElementById('lineNumbers');
    if (lineNumbersElement) {
        lineNumbersElement.textContent = lineNumbers;
    }
}

// 提交分析
async function submitAnalysis() {
    // 获取选中的功能
    const enableEvaluate = elements.enableEvaluate.checked;
    const enableDebug = elements.enableDebug.checked;
    const enableRecommend = elements.enableRecommend.checked;

    if (!enableEvaluate && !enableDebug && !enableRecommend) {
        alert('请至少选择一个分析功能！');
        return;
    }

    // 显示加载状态
    showLoading(true);

    try {
        const results = {};
        const baseData = {
            student_id: elements.studentId.value || 'anonymous',
            conversation_id: elements.conversationId.value,
            problem_description: elements.problemDescription.value,
            code: elements.codeEditor.value
        };

        // 解析测试点（用于评价和调试）
        let testPoints = [];
        try {
            if (elements.testPoints.value.trim()) {
                testPoints = JSON.parse(elements.testPoints.value);
            }
        } catch (error) {
            alert('测试点JSON格式错误！请检查格式。');
            showLoading(false);
            return;
        }

        // 执行评价功能
        if (enableEvaluate) {
            try {
                const requestData = {
                    ...baseData,
                    test_points: testPoints,
                    task_type: 'evaluate'
                };

                const response = await fetch(`${API_BASE_URL}/evaluate`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(requestData)
                });

                if (!response.ok) throw new Error(`评价功能HTTP错误: ${response.status}`);
                results.evaluate = await response.json();
            } catch (error) {
                results.evaluate = { error: `评价失败: ${error.message}` };
            }
        }

        // 执行调试功能
        if (enableDebug) {
            try {
                const requestData = {
                    ...baseData,
                    test_points: testPoints,
                    task_type: 'debug'
                };

                const response = await fetch(`${API_BASE_URL}/debug`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(requestData)
                });

                if (!response.ok) throw new Error(`调试功能HTTP错误: ${response.status}`);
                results.debug = await response.json();
            } catch (error) {
                results.debug = { error: `调试失败: ${error.message}` };
            }
        }

        // 执行推荐功能
        if (enableRecommend) {
            try {
                // 解析薄弱点信息
                let weakPoints = {};
                try {
                    if (elements.weakPoints.value.trim()) {
                        weakPoints = JSON.parse(elements.weakPoints.value);
                    }
                } catch (error) {
                    alert('薄弱点信息JSON格式错误！请检查格式。');
                    showLoading(false);
                    return;
                }

                // 解析YOJ题库信息（可选）
                let yojDatabase = null;
                try {
                    if (elements.yojDatabase.value.trim()) {
                        yojDatabase = JSON.parse(elements.yojDatabase.value);
                    }
                } catch (error) {
                    console.warn('YOJ题库信息JSON格式错误，将使用默认设置。');
                }

                // 构建推荐请求
                const recommendRequest = {
                    student_id: baseData.student_id,
                    conversation_id: baseData.conversation_id,
                    weak_points: weakPoints,
                    max_recommendations: 5
                };

                // 添加YOJ数据库信息到请求中（作为扩展字段）
                if (yojDatabase) {
                    recommendRequest.yoj_database = yojDatabase;
                }

                const response = await fetch(`${API_BASE_URL}/recommend`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(recommendRequest)
                });

                if (!response.ok) throw new Error(`推荐功能HTTP错误: ${response.status}`);
                results.recommend = await response.json();
            } catch (error) {
                results.recommend = { error: `推荐失败: ${error.message}` };
            }
        }

        // 显示结果
        displayResults(results);

        // 根据选中的功能切换到相应的标签页
        if (enableRecommend && results.recommend && !results.recommend.error) {
            switchTab('recommend');
        } else if (enableDebug && results.debug && !results.debug.error) {
            switchTab('debug');
        } else if (enableEvaluate && results.evaluate && !results.evaluate.error) {
            switchTab('overview');
        } else {
            switchTab('raw'); // 显示原始数据以查看错误
        }

    } catch (error) {
        console.error('请求失败:', error);
        alert(`分析请求失败: ${error.message}`);
        switchTab('raw');
    } finally {
        showLoading(false);
    }
}

// 显示结果
function displayResults(results) {
    // 显示原始数据
    elements.rawData.textContent = JSON.stringify(results, null, 2);

    // 重置所有结果区域
    resetResultsDisplay();

    // 显示评价结果
    if (results.evaluate && !results.evaluate.error) {
        displayEvaluationResults(results.evaluate);
    } else if (results.evaluate?.error) {
        elements.overallEvaluationText.textContent = `评价失败: ${results.evaluate.error}`;
    }

    // 显示调试结果
    if (results.debug && !results.debug.error) {
        displayDebugResults(results.debug);
    } else if (results.debug?.error) {
        elements.debugAnalysisText.textContent = `调试失败: ${results.debug.error}`;
    }

    // 显示推荐结果
    if (results.recommend && !results.recommend.error) {
        displayRecommendResults(results.recommend);
    } else if (results.recommend?.error) {
        elements.recommendationAnalysis.textContent = `推荐失败: ${results.recommend.error}`;
    }
}

// 重置结果显示
function resetResultsDisplay() {
    // 重置评价结果
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

    // 重置调试结果
    elements.debugAnalysisText.textContent = '等待调试结果...';
    elements.problemsList.innerHTML = '';
    elements.suggestionsList.innerHTML = '';

    // 重置推荐结果
    elements.recommendationSummary.textContent = '等待推荐结果...';
    elements.recommendationList.innerHTML = '';
    elements.recommendationAnalysis.textContent = '等待分析结果...';

    // 重置分数条
    document.querySelectorAll('.breakdown-fill').forEach(bar => {
        bar.style.width = '0%';
    });
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

// 显示推荐结果
function displayRecommendResults(result) {
    // 更新推荐摘要
    elements.recommendationSummary.textContent = `为学生 ${result.student_id} 推荐了 ${result.recommendations?.length || 0} 个题目类型`;

    // 清空推荐列表
    elements.recommendationList.innerHTML = '';

    // 生成推荐列表
    if (result.recommendations && Array.isArray(result.recommendations)) {
        result.recommendations.forEach((rec, index) => {
            const recElement = document.createElement('div');
            recElement.className = 'recommendation-item';

            const relevancePercent = (rec.relevance * 100).toFixed(0);

            recElement.innerHTML = `
                <h4>推荐 #${index + 1}</h4>
                <div>
                    <span class="recommendation-tag">${rec.tag}</span>
                    <span class="recommendation-relevance">相关度: ${relevancePercent}%</span>
                </div>
                <div class="recommendation-reason">
                    <strong>推荐理由:</strong> ${rec.reason || '未提供理由'}
                </div>
            `;

            elements.recommendationList.appendChild(recElement);
        });
    } else {
        elements.recommendationList.innerHTML = '<p>暂无推荐</p>';
    }

    // 显示分析总结
    elements.recommendationAnalysis.textContent = result.analysis || '未提供分析总结';
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
    const bars = document.querySelectorAll(`.breakdown-fill`);
    if (bars.length > 0) {
        // 简单实现：更新第一个匹配的进度条
        bars[0].style.width = `${Math.min(percentage, 100)}%`;
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
        elements.weakPoints.value = '';
        elements.yojDatabase.value = '';
        updateLineNumbers();

        // 重置功能选择
        elements.enableEvaluate.checked = true;
        elements.enableDebug.checked = false;
        elements.enableRecommend.checked = false;
        elements.recommendInputSection.style.display = 'none';

        // 重置结果区域
        resetResultsDisplay();
    }
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

    // 设置推荐功能示例
    elements.enableEvaluate.checked = true;
    elements.enableDebug.checked = true;
    elements.enableRecommend.checked = true;
    elements.recommendInputSection.style.display = 'block';

    elements.weakPoints.value = `{
  "数组越界": 3,
  "时间复杂度高": 2,
  "边界条件错误": 4,
  "递归深度过大": 1,
  "内存泄漏": 2
}`;

    elements.yojDatabase.value = `{
  "total_problems": 1000,
  "tags": ["数组", "字符串", "链表", "栈", "队列", "树", "图", 
           "哈希表", "动态规划", "贪心算法", "回溯算法", "排序", "查找"],
  "difficulty_distribution": {
    "easy": 300,
    "medium": 500,
    "hard": 200
  }
}`;

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