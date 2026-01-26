<template>
    <div class="ai-debug-core-page">
        <header class="ai-debug-header">
          <h1><i class="fas fa-robot"></i> AI 代码分析平台</h1>
          <p class="subtitle">输入代码和题目描述，获取AI的专业评价和调试建议</p>
        </header>

        <main class="ai-debug-main">
            <div class="input-section">
                <div class="form-group">
                    <label for="studentId">
                        <i class="fas fa-user"></i> 学生ID
                    </label>
                    <input type="text" id="studentId" v-model="studentId" placeholder="请输入学号">
                </div>

                <div class="form-group">
                    <label for="conversationId">
                        <i class="fas fa-comment"></i> 会话ID
                    </label>
                    <input type="text" id="conversationId" v-model="conversationId" placeholder="自动生成会话ID" readonly>
                </div>

                <div class="form-group">
                    <label>
                        <i class="fas fa-tasks"></i> 选择分析功能
                    </label>
                    <div class="task-checkboxes">
                        <div class="checkbox-group">
                            <input type="checkbox" id="enableEvaluate" v-model="enableEvaluate">
                            <label for="enableEvaluate">代码评价打分</label>
                        </div>
                        <div class="checkbox-group">
                            <input type="checkbox" id="enableDebug" v-model="enableDebug">
                            <label for="enableDebug">代码调试分析</label>
                        </div>
                        <div class="checkbox-group">
                            <input type="checkbox" id="enableRecommend" v-model="enableRecommend">
                            <label for="enableRecommend">题目推荐</label>
                        </div>
                    </div>
                </div>

                <div class="form-group full-width">
                    <label for="problemDescription">
                        <i class="fas fa-question-circle"></i> 题目描述
                    </label>
                    <textarea id="problemDescription" rows="3" v-model="problemDescription" placeholder="请输入题目要求..."></textarea>
                </div>

                <div class="form-group full-width">
                    <label for="codeEditor">
                        <i class="fas fa-code"></i> 学生代码 (C/C++)
                    </label>
                    <div class="code-editor-container">
                        <textarea id="codeEditor" rows="15" v-model="codeEditor" @input="updateLineNumbers" placeholder="请输入C/C++代码..."></textarea>
                        <div class="line-numbers" ref="lineNumbers"></div>
                    </div>
                </div>

                <div class="form-group full-width">
                    <label for="testPoints">
                        <i class="fas fa-vial"></i> 测试点 (JSON格式，可选)
                    </label>
                    <textarea id="testPoints" rows="4" v-model="testPoints" placeholder='[{"input": "测试输入", "status": "Accepted/Wrong Answer/Time Limit Exceeded"}]'></textarea>
                </div>

                <div class="recommend-input-section" id="recommendInputSection" v-show="enableRecommend">
                    <div class="form-group full-width">
                        <label for="weakPoints">
                            <i class="fas fa-exclamation-triangle"></i> 学生薄弱点信息 (JSON格式)
                        </label>
                        <textarea id="weakPoints" rows="4" v-model="weakPoints" placeholder='{"数组越界": 3, "时间复杂度高": 2, "边界条件错误": 4}'></textarea>
                    </div>
                
                    <div class="form-group full-width">
                        <label for="yojDatabase">
                            <i class="fas fa-database"></i> YOJ题库信息 (JSON格式，可选)
                        </label>
                        <textarea id="yojDatabase" rows="6" v-model="yojDatabase" placeholder='{"total_problems": 1000, "tags": ["数组", "字符串", "动态规划"]}'></textarea>
                    </div>
                </div>

                <div class="action-buttons">
                    <button id="submitBtn" class="btn primary" @click="submitAnalysis" :disabled="isLoading">
                        <i class="fas fa-paper-plane"></i> {{ isLoading ? '分析中...' : '提交分析' }}
                    </button>
                    <button id="clearBtn" class="btn secondary" @click="clearForm">
                        <i class="fas fa-broom"></i> 清空
                    </button>
                    <button id="exampleBtn" class="btn tertiary" @click="loadExample">
                        <i class="fas fa-magic"></i> 加载示例
                    </button>
                </div>
            </div>

            <div class="result-section">
                <h2><i class="fas fa-chart-line"></i> 分析结果</h2>
            
                <div class="result-tabs">
                    <button class="tab-btn" :class="{ active: activeTab === 'overview' }" @click="switchTab('overview')">概览</button>
                    <button class="tab-btn" :class="{ active: activeTab === 'detailed' }" @click="switchTab('detailed')">详细分析</button>
                    <button class="tab-btn" :class="{ active: activeTab === 'debug' }" @click="switchTab('debug')">调试分析</button>
                    <button class="tab-btn" :class="{ active: activeTab === 'recommend' }" @click="switchTab('recommend')">题目推荐</button>
                    <button class="tab-btn" :class="{ active: activeTab === 'raw' }" @click="switchTab('raw')">原始数据</button>
                </div>

                <div class="tab-content">
                    <div id="overviewTab" class="tab-pane" :class="{ active: activeTab === 'overview' }">
                        <div class="score-display">
                            <div class="score-circle">
                                <span id="scoreValue">{{ scoreValue }}</span>
                                <span class="score-label">总分</span>
                            </div>
                            <div class="score-breakdown">
                                <div class="breakdown-item">
                                    <span class="breakdown-label">可读性</span>
                                    <div class="breakdown-bar">
                                        <div class="breakdown-fill" :style="{ width: readabilityPercentage + '%' }"></div>
                                    </div>
                                    <span class="breakdown-value">{{ readabilityScore }}</span>
                                </div>
                                <div class="breakdown-item">
                                    <span class="breakdown-label">逻辑严谨性</span>
                                    <div class="breakdown-bar">
                                        <div class="breakdown-fill" :style="{ width: logicPercentage + '%' }"></div>
                                    </div>
                                    <span class="breakdown-value">{{ logicScore }}</span>
                                </div>
                                <div class="breakdown-item">
                                    <span class="breakdown-label">算法合理性</span>
                                    <div class="breakdown-bar">
                                        <div class="breakdown-fill" :style="{ width: algorithmPercentage + '%' }"></div>
                                    </div>
                                    <span class="breakdown-value">{{ algorithmScore }}</span>
                                </div>
                                <div class="breakdown-item">
                                    <span class="breakdown-label">运行效率</span>
                                    <div class="breakdown-bar">
                                        <div class="breakdown-fill" :style="{ width: efficiencyPercentage + '%' }"></div>
                                    </div>
                                    <span class="breakdown-value">{{ efficiencyScore }}</span>
                                </div>
                            </div>
                        </div>

                        <div class="overall-evaluation">
                            <h3><i class="fas fa-comment-dots"></i> 整体评价</h3>
                            <p>{{ overallEvaluationText }}</p>
                        </div>
                    </div>

                    <div id="detailedTab" class="tab-pane" :class="{ active: activeTab === 'detailed' }">
                        <div class="detail-category">
                            <h3><i class="fas fa-eye"></i> 可读性分析</h3>
                            <p>{{ readabilityDetail }}</p>
                        </div>
                        <div class="detail-category">
                            <h3><i class="fas fa-brain"></i> 逻辑严谨性分析</h3>
                            <p>{{ logicDetail }}</p>
                        </div>
                        <div class="detail-category">
                            <h3><i class="fas fa-cogs"></i> 算法合理性分析</h3>
                            <p>{{ algorithmDetail }}</p>
                        </div>
                        <div class="detail-category">
                            <h3><i class="fas fa-tachometer-alt"></i> 运行效率分析</h3>
                            <p>{{ efficiencyDetail }}</p>
                        </div>
                    </div>

                    <div id="debugTab" class="tab-pane" :class="{ active: activeTab === 'debug' }">
                        <div class="debug-section">
                            <h3><i class="fas fa-bug"></i> 调试分析</h3>
                            <p>{{ debugAnalysisText }}</p>
                            <div class="problems-list">
                                <div class="problem-item" v-for="(problem, index) in problemsList" :key="index">
                                    <div class="problem-location">位置: {{ problem.location || '未知' }}</div>
                                    <div class="problem-description">描述: {{ problem.description || '无描述' }}</div>
                                    <div class="problem-root-cause">原因: {{ problem.root_cause || '未知原因' }}</div>
                                </div>
                            </div>
                            <div class="suggestions">
                                <h4><i class="fas fa-lightbulb"></i> 修改建议</h4>
                                <ul>
                                    <li v-for="(suggestion, index) in suggestionsList" :key="index">{{ suggestion }}</li>
                                </ul>
                            </div>
                        </div>
                    </div>

                    <div id="recommendTab" class="tab-pane" :class="{ active: activeTab === 'recommend' }">
                        <div class="recommendation-results">
                            <h3><i class="fas fa-star"></i> 个性化题目推荐</h3>
                            <div class="recommendation-summary">
                                <p>{{ recommendationSummary }}</p>
                            </div>
                            <div class="recommendation-list">
                                <div class="recommendation-item" v-for="(rec, index) in recommendationList" :key="index">
                                    <h4>推荐 #{{ index + 1 }}</h4>
                                    <div>
                                        <span class="recommendation-tag">{{ rec.tag }}</span>
                                        <span class="recommendation-relevance">相关度: {{ (rec.relevance * 100).toFixed(0) }}%</span>
                                    </div>
                                    <div class="recommendation-reason">
                                        <strong>推荐理由:</strong> {{ rec.reason || '未提供理由' }}
                                    </div>
                                </div>
                            </div>
                            <div class="analysis-section">
                                <h4><i class="fas fa-chart-bar"></i> 推荐分析</h4>
                                <p>{{ recommendationAnalysis }}</p>
                            </div>
                        </div>
                    </div>
                    <div id="rawTab" class="tab-pane" :class="{ active: activeTab === 'raw' }">
                        <pre>{{ rawData }}</pre>
                    </div>
                </div>
            </div>
        </main>

        <footer class="ai-debug-footer">
            <p>AI 教学辅助平台演示版 | 后端服务: <span :class="{ connected: apiConnected }">{{ apiStatusText }}</span></p>
            <p class="api-url">API端点: <code>{{ apiEndpoint }}</code></p>
        </footer>
    </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import axios from 'axios';

// Reactive state
const studentId = ref('2025001');
const conversationId = ref('');
const enableEvaluate = ref(true);
const enableDebug = ref(false);
const enableRecommend = ref(false);
const problemDescription = ref('');
const codeEditor = ref('');
const testPoints = ref('');
const weakPoints = ref('{\n    "数组越界": 3,\n    "时间复杂度高": 2,\n    "边界条件错误": 4\n}');
const yojDatabase = ref('{\n    "total_problems": 1000,\n    "tags": ["数组", "字符串", "动态规划", "贪心算法", "递归", "排序", "查找"],\n    "difficulty_distribution": {\n        "easy": 300,\n        "medium": 500,\n        "hard": 200\n    }\n}');

const activeTab = ref('overview');
const isLoading = ref(false);
const lineNumbers = ref(null); // Ref for line numbers div

const scoreValue = ref('--');
const readabilityScore = ref('0/10');
const logicScore = ref('0/40');
const algorithmScore = ref('0/25');
const efficiencyScore = ref('0/25');
const readabilityPercentage = ref(0);
const logicPercentage = ref(0);
const algorithmPercentage = ref(0);
const efficiencyPercentage = ref(0);

const overallEvaluationText = ref('等待分析结果...');
const readabilityDetail = ref('等待分析结果...');
const logicDetail = ref('等待分析结果...');
const algorithmDetail = ref('等待分析结果...');
const efficiencyDetail = ref('等待分析结果...');

const debugAnalysisText = ref('等待调试结果...');
const problemsList = ref([]);
const suggestionsList = ref([]);

const recommendationSummary = ref('等待推荐结果...');
const recommendationList = ref([]);
const recommendationAnalysis = ref('等待分析结果...');

const rawData = ref('{"status": "等待请求数据..."}');

const apiStatusText = ref('未连接');
const apiConnected = ref(false);
const apiEndpoint = ref('http://localhost:8080/api/v1/ai'); // This will be the Go backend AI proxy endpoint

// Functions
const generateConversationId = () => {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 8);
    return `conv_${timestamp}_${random}`;
};

const checkApiConnection = async () => {
    try {
        const response = await axios.get(`${apiEndpoint.value}/health`); // Assuming a /health endpoint in Go
        if (response.ok) {
            apiStatusText.value = '已连接';
            apiConnected.value = true;
        } else {
            apiStatusText.value = '连接失败';
            apiConnected.value = false;
        }
    } catch (error) {
        apiStatusText.value = '未连接';
        apiConnected.value = false;
        console.warn('API连接检查失败:', error);
    }
};

const updateLineNumbers = () => {
    const code = codeEditor.value;
    const lines = code.split('\n').length;
    const numbers = Array.from({ length: lines }, (_, i) => i + 1).join('\n');
    if (lineNumbers.value) {
        lineNumbers.value.textContent = numbers;
    }
};

const submitAnalysis = async () => {
    if (!enableEvaluate.value && !enableDebug.value && !enableRecommend.value) {
        alert('请至少选择一个分析功能！');
        return;
    }

    isLoading.value = true;
    resetResultsDisplay();

    try {
        const results = {};
        const baseData = {
            student_id: studentId.value || 'anonymous',
            conversation_id: conversationId.value,
            problem_description: problemDescription.value,
            code: codeEditor.value
        };

        let parsedTestPoints = [];
        try {
            if (testPoints.value.trim()) {
                parsedTestPoints = JSON.parse(testPoints.value);
            }
        } catch (error) {
            alert('测试点JSON格式错误！请检查格式。');
            isLoading.value = false;
            return;
        }

        // Execute evaluation function
        if (enableEvaluate.value) {
            try {
                const requestData = {
                    ...baseData,
                    test_points: parsedTestPoints,
                    task_type: 'evaluate'
                };

                const response = await axios.post(`${apiEndpoint.value}/evaluate`, requestData);

                // axios automatically handles non-2xx status as errors, and response.data contains the JSON
                results.evaluate = response.data;
            } catch (error) {
                results.evaluate = { error: `评价失败: ${error.message || error.response?.data?.message || error.response?.data || '未知错误'}` };
            }
        }

        // Execute debug function
        if (enableDebug.value) {
            try {
                const requestData = {
                    ...baseData,
                    test_points: parsedTestPoints,
                    task_type: 'debug'
                };

                const response = await axios.post(`${apiEndpoint.value}/debug`, requestData);

                results.debug = response.data;
            } catch (error) {
                results.debug = { error: `调试失败: ${error.message || error.response?.data?.message || error.response?.data || '未知错误'}` };
            }
        }

        // Execute recommendation function
        if (enableRecommend.value) {
            try {
                let parsedWeakPoints = {};
                try {
                    if (weakPoints.value.trim()) {
                        parsedWeakPoints = JSON.parse(weakPoints.value);
                    }
                } catch (error) {
                    alert('薄弱点信息JSON格式错误！请检查格式。');
                    isLoading.value = false;
                    return;
                }

                let parsedYojDatabase = null;
                try {
                    if (yojDatabase.value.trim()) {
                        parsedYojDatabase = JSON.parse(yojDatabase.value);
                    }
                } catch (error) {
                    console.warn('YOJ题库信息JSON格式错误，将使用默认设置。');
                }

                const recommendRequest = {
                    student_id: baseData.student_id,
                    conversation_id: baseData.conversation_id,
                    weak_points: parsedWeakPoints,
                    max_recommendations: 5
                };

                if (parsedYojDatabase) {
                    recommendRequest.yoj_database = parsedYojDatabase;
                }

                const response = await axios.post(`${apiEndpoint.value}/recommend`, recommendRequest);

                results.recommend = response.data;
            } catch (error) {
                results.recommend = { error: `推荐失败: ${error.message || error.response?.data?.message || error.response?.data || '未知错误'}` };
            }
        }

        displayResults(results);

        if (enableRecommend.value && results.recommend && !results.recommend.error) {
            switchTab('recommend');
        } else if (enableDebug.value && results.debug && !results.debug.error) {
            switchTab('debug');
        } else if (enableEvaluate.value && results.evaluate && !results.evaluate.error) {
            switchTab('overview');
        } else {
            switchTab('raw');
        }

    } catch (error) {
        console.error('请求失败:', error);
        alert(`分析请求失败: ${error.message}`);
        switchTab('raw');
    } finally {
        isLoading.value = false;
    }
};

const displayResults = (results) => {
    rawData.value = JSON.stringify(results, null, 2);

    resetResultsDisplay();

    if (results.evaluate && !results.evaluate.error) {
        displayEvaluationResults(results.evaluate);
    } else if (results.evaluate?.error) {
        overallEvaluationText.value = `评价失败: ${results.evaluate.error}`;
    }

    if (results.debug && !results.debug.error) {
        displayDebugResults(results.debug);
    } else if (results.debug?.error) {
        debugAnalysisText.value = `调试失败: ${results.debug.error}`;
    }

    if (results.recommend && !results.recommend.error) {
        displayRecommendResults(results.recommend);
    } else if (results.recommend?.error) {
        recommendationAnalysis.value = `推荐失败: ${results.recommend.error}`;
    }
};

const resetResultsDisplay = () => {
    scoreValue.value = '--';
    readabilityScore.value = '0/10';
    logicScore.value = '0/40';
    algorithmScore.value = '0/25';
    efficiencyScore.value = '0/25';
    readabilityPercentage.value = 0;
    logicPercentage.value = 0;
    algorithmPercentage.value = 0;
    efficiencyPercentage.value = 0;

    overallEvaluationText.value = '等待分析结果...';
    readabilityDetail.value = '等待分析结果...';
    logicDetail.value = '等待分析结果...';
    algorithmDetail.value = '等待分析结果...';
    efficiencyDetail.value = '等待分析结果...';

    debugAnalysisText.value = '等待调试结果...';
    problemsList.value = [];
    suggestionsList.value = [];

    recommendationSummary.value = '等待推荐结果...';
    recommendationList.value = [];
    recommendationAnalysis.value = '等待分析结果...';

    rawData.value = '{"status": "等待请求数据..."}';
};

const displayEvaluationResults = (result) => {
    scoreValue.value = result.score || '--';

    if (result.readability) {
        const { obtained, total, percentage } = parseScore(result.readability.score);
        readabilityScore.value = result.readability.score;
        readabilityPercentage.value = percentage;
        readabilityDetail.value = result.readability.analysis || '无分析';
    }

    if (result.logical_rigor) {
        const { obtained, total, percentage } = parseScore(result.logical_rigor.score);
        logicScore.value = result.logical_rigor.score;
        logicPercentage.value = percentage;
        logicDetail.value = result.logical_rigor.analysis || '无分析';
    }

    if (result.algorithm_quality) {
        const { obtained, total, percentage } = parseScore(result.algorithm_quality.score);
        algorithmScore.value = result.algorithm_quality.score;
        algorithmPercentage.value = percentage;
        algorithmDetail.value = result.algorithm_quality.analysis || '无分析';
    }

    if (result.efficiency) {
        const { obtained, total, percentage } = parseScore(result.efficiency.score);
        efficiencyScore.value = result.efficiency.score;
        efficiencyPercentage.value = percentage;
        efficiencyDetail.value = result.efficiency.analysis || '无分析';
    }

    overallEvaluationText.value = result.overall_evaluation || '无评价';
};

const displayDebugResults = (result) => {
    debugAnalysisText.value = result.debug_analysis || '无调试分析';

    problemsList.value = [];
    if (result.problems && Array.isArray(result.problems)) {
        problemsList.value = result.problems;
    }

    suggestionsList.value = [];
    if (result.suggestions && Array.isArray(result.suggestions)) {
        suggestionsList.value = result.suggestions;
    }
};

const displayRecommendResults = (result) => {
    recommendationSummary.value = `为学生 ${result.student_id} 推荐了 ${result.recommendations?.length || 0} 个题目类型`;

    recommendationList.value = [];
    if (result.recommendations && Array.isArray(result.recommendations)) {
        recommendationList.value = result.recommendations;
    } else {
        recommendationList.value = [{ tag: '暂无推荐', relevance: 0, reason: '没有可用的推荐题目。' }];
    }

    recommendationAnalysis.value = result.analysis || '未提供分析总结';
};

const parseScore = (scoreStr) => {
    if (!scoreStr) return { obtained: 0, total: 1, percentage: 0 };

    const parts = scoreStr.split('/');
    if (parts.length !== 2) return { obtained: 0, total: 1, percentage: 0 };

    const obtained = parseFloat(parts[0]) || 0;
    const total = parseFloat(parts[1]) || 1;
    const percentage = total > 0 ? (obtained / total) * 100 : 0;

    return { obtained, total, percentage };
};

const switchTab = (tabId) => {
    activeTab.value = tabId;
};

const clearForm = () => {
    if (confirm('确定要清空所有输入吗？')) {
        problemDescription.value = '';
        codeEditor.value = '';
        testPoints.value = '';
        weakPoints.value = '';
        yojDatabase.value = '';
        updateLineNumbers();

        enableEvaluate.value = true;
        enableDebug.value = false;
        enableRecommend.value = false;

        resetResultsDisplay();
    }
};

const loadExample = () => {
    problemDescription.value = `编写一个C函数，判断一个整数是否为素数。\n要求：\n1. 函数原型：int is_prime(int n)\n2. 如果n是素数返回1，否则返回0\n3. 注意处理特殊情况（n<=1）\n4. 优化算法效率`;

    codeEditor.value = `#include <stdio.h>\n#include <math.h>\n\nint is_prime(int n) {\n    if (n <= 1) return 0;\n    if (n == 2) return 1;\n    if (n % 2 == 0) return 0;\n    \n    for (int i = 3; i <= sqrt(n); i += 2) {\n        if (n % i == 0) {\n            return 0;\n        }\n    }\n    return 1;\n}\n\nint main() {\n    int num;\n    printf(\"请输入一个整数: \");\n    scanf(\"%d\", &num);\n    \n    if (is_prime(num)) {\n        printf(\"%d 是素数\\n\", num);\n    } else {\n        printf(\"%d 不是素数\\n\", num);\n    }\n    \n    return 0;\n}`;

    testPoints.value = `[\n  {"input": "2", "status": "Accepted"},\n  {"input": "17", "status": "Accepted"},\n  {"input": "1", "status": "Accepted"},\n  {"input": "4", "status": "Accepted"},\n  {"input": "-5", "status": "Accepted"}\n]`;

    enableEvaluate.value = true;
    enableDebug.value = true;
    enableRecommend.value = true;

    weakPoints.value = `{\n  "数组越界": 3,\n  "时间复杂度高": 2,\n  "边界条件错误": 4,\n  "递归深度过大": 1,\n  "内存泄漏": 2\n}`;

    yojDatabase.value = `{\n  "total_problems": 1000,\n  "tags": ["数组", "字符串", "链表", "栈", "队列", "树", "图", \n           "哈希表", "动态规划", "贪心算法", "回溯算法", "排序", "查找"],\n  "difficulty_distribution": {\n    "easy": 300,\n    "medium": 500,\n    "hard": 200\n  }\n}`;

    updateLineNumbers();
};

onMounted(() => {
    conversationId.value = generateConversationId();
    checkApiConnection();
    updateLineNumbers();
});
</script>

<style scoped>
/* Imported from ai-python/demo/style.css with modifications for Vue scoping */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

.ai-debug-core-page {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

/* 响应式布局 */
@media (max-width: 1024px) {
    .ai-debug-main {
        grid-template-columns: 1fr !important;
    }
}

/* ===== 头部区域 ===== */
.ai-debug-header {
    text-align: center;
    margin-bottom: 30px;
    padding: 20px;
    background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
    color: white;
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.ai-debug-header h1 {
    font-size: 2.5rem;
    margin-bottom: 10px;
}

.ai-debug-header .subtitle {
    font-size: 1.1rem;
    opacity: 0.9;
}

/* ===== 主体区域 ===== */
.ai-debug-main {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 30px;
    margin-bottom: 40px;
}

/* 输入区域和结果区域的通用样式 */
.input-section,
.result-section,
.tab-pane,
.recommendation-results {
    background: white;
    padding: 25px;
    border-radius: 10px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
}

/* ===== 表单组件 ===== */
.form-group {
    margin-bottom: 20px;
}

.form-group.full-width {
    grid-column: 1 / -1;
}

label {
    display: block;
    margin-bottom: 8px;
    font-weight: 600;
    color: #2c3e50;
}

label i {
    margin-right: 8px;
    color: #3498db;
}

/* 表单控件通用样式 */
input,
select,
textarea {
    width: 100%;
    padding: 12px 15px;
    border: 2px solid #e0e0e0;
    border-radius: 6px;
    font-size: 1rem;
    transition: border-color 0.3s;
    font-family: inherit;
}

input:focus,
select:focus,
textarea:focus {
    border-color: #3498db;
    outline: none;
}

textarea {
    resize: vertical;
}

/* ===== 任务选择复选框 ===== */
.task-checkboxes {
    display: flex;
    gap: 20px;
    margin-top: 10px;
    flex-wrap: wrap;
}

.checkbox-group {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background-color: #f8f9fa;
    border-radius: 6px;
    border: 1px solid #e0e0e0;
    cursor: pointer;
    transition: all 0.2s;
}

.checkbox-group:hover {
    background-color: #e9ecef;
    border-color: #3498db;
}

.checkbox-group input[type="checkbox"] {
    width: auto;
    transform: scale(1.2);
    margin: 0;
    cursor: pointer;
}

.checkbox-group label {
    margin: 0;
    font-weight: normal;
    cursor: pointer;
    user-select: none;
}

/* ===== 代码编辑器 ===== */
.code-editor-container {
    position: relative;
    border: 2px solid #e0e0e0;
    border-radius: 6px;
    overflow: hidden;
}

.code-editor-container textarea {
    border: none;
    padding-left: 50px;
    line-height: 1.5;
    font-family: 'Courier New', Courier, monospace;
    font-size: 14px;
}

.line-numbers {
    position: absolute;
    left: 0;
    top: 0;
    width: 40px;
    height: 100%;
    background-color: #f8f9fa;
    border-right: 1px solid #e0e0e0;
    padding: 12px 5px !important; /* Override padding from textarea */
    font-family: 'Courier New', Courier, monospace;
    font-size: 14px;
    color: #666;
    text-align: right;
    overflow: hidden;
}

/* ===== 按钮样式 ===== */
.action-buttons {
    display: flex;
    gap: 15px;
    margin-top: 25px;
}

.btn {
    padding: 12px 25px;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-family: inherit;
}

.btn i {
    font-size: 1.1rem;
}

.btn.primary {
    background-color: #3498db;
    color: white;
    flex: 2;
}

.btn.primary:hover:not(:disabled) {
    background-color: #2980b9;
    transform: translateY(-2px);
}

.btn.secondary {
    background-color: #e0e0e0;
    color: #333;
    flex: 1;
}

.btn.secondary:hover {
    background-color: #d0d0d0;
}

.btn.tertiary {
    background-color: #9b59b6;
    color: white;
    flex: 1;
}

.btn.tertiary:hover {
    background-color: #8e44ad;
}

.btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
    transform: none;
}

/* ===== 结果区域标题 ===== */
.result-section h2 {
    margin-bottom: 20px;
    color: #2c3e50;
    border-bottom: 2px solid #eee;
    padding-bottom: 10px;
}

/* ===== 标签页样式 ===== */
.result-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
    border-bottom: 2px solid #eee;
    margin-bottom: 20px;
}

.tab-btn {
    padding: 10px 20px;
    background: none;
    border: none;
    font-size: 1rem;
    font-weight: 600;
    color: #666;
    cursor: pointer;
    position: relative;
    transition: color 0.3s;
    border-radius: 6px 6px 0 0;
    font-family: inherit;
}

.tab-btn:hover {
    background-color: #f8f9fa;
}

.tab-btn.active {
    color: #3498db;
    background-color: #f8f9fa;
}

.tab-btn.active::after {
    content: '';
    position: absolute;
    bottom: -2px;
    left: 0;
    right: 0;
    height: 2px;
    background-color: #3498db;
}

.tab-pane {
    display: none;
}

.tab-pane.active {
    display: block;
}

/* ===== 概览标签页 ===== */
.score-display {
    display: flex;
    align-items: center;
    gap: 40px;
    margin-bottom: 30px;
}

.score-circle {
    width: 120px;
    height: 120px;
    border-radius: 50%;
    background: linear-gradient(135deg, #3498db, #2ecc71);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: bold;
    box-shadow: 0 4px 15px rgba(52, 152, 219, 0.3);
}

#scoreValue {
    font-size: 2.5rem;
}

.score-label {
    font-size: 0.9rem;
    opacity: 0.9;
}

.score-breakdown {
    flex: 1;
}

.breakdown-item {
    margin-bottom: 15px;
}

.breakdown-label {
    display: block;
    margin-bottom: 5px;
    font-weight: 600;
    color: #2c3e50;
}

.breakdown-bar {
    height: 10px;
    background-color: #eee;
    border-radius: 5px;
    overflow: hidden;
    margin: 8px 0;
}

.breakdown-fill {
    height: 100%;
    background: linear-gradient(90deg, #3498db, #2ecc71);
    border-radius: 5px;
    transition: width 1s ease-in-out;
}

.breakdown-value {
    float: right;
    font-weight: 600;
    color: #2c3e50;
}

/* ===== 整体评价和详细分析 ===== */
.overall-evaluation,
.detail-category,
.recommend-input-section {
    background-color: #f8f9fa;
    padding: 20px;
    border-radius: 8px;
    margin-bottom: 20px;
}

.overall-evaluation h3,
.detail-category h3 {
    margin-bottom: 10px;
    color: #2c3e50;
}

/* ===== 调试分析标签页 ===== */
.debug-section h3 {
    color: #e74c3c;
    border-bottom: 2px solid #f0f0f0;
    padding-bottom: 10px;
    margin-bottom: 20px;
}

.problems-list {
    margin: 25px 0;
}

.problem-item {
    background-color: #fff3cd;
    border-left: 4px solid #ffc107;
    padding: 15px;
    margin-bottom: 15px;
    border-radius: 0 6px 6px 0;
    transition: all 0.2s;
}

.problem-item:hover {
    transform: translateX(5px);
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.problem-location {
    font-weight: bold;
    color: #856404;
    margin-bottom: 5px;
    font-size: 1.1rem;
}

.problem-description {
    color: #856404;
    margin-bottom: 8px;
    padding-left: 10px;
}

.problem-root-cause {
    color: #856404;
    font-style: italic;
    padding-left: 10px;
    border-left: 2px solid #ffc107;
}

.suggestions {
    background-color: #e8f5e8;
    padding: 20px;
    border-radius: 8px;
    margin-top: 20px;
}

.suggestions h4 {
    color: #2ecc71;
    margin-bottom: 15px;
}

.suggestions ul {
    list-style-type: none;
    padding-left: 0;
}

.suggestions li {
    padding: 8px 0;
    position: relative;
    padding-left: 30px;
}

.suggestions li::before {
    content: '💡';
    position: absolute;
    left: 0;
    font-size: 1.2rem;
}

/* ===== 题目推荐标签页 ===== */
.recommendation-summary {
    background-color: #e8f5e8;
    border-left: 4px solid #2ecc71;
    padding: 15px;
    margin-bottom: 20px;
    border-radius: 4px;
}

.recommendation-list {
    margin: 20px 0;
}

.recommendation-item {
    background-color: white;
    border: 1px solid #e0e0e0;
    border-radius: 8px;
    padding: 15px;
    margin-bottom: 15px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
    transition: all 0.2s;
}

.recommendation-item:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.recommendation-tag {
    display: inline-block;
    background-color: #3498db;
    color: white;
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 0.9rem;
    font-weight: bold;
    margin-right: 8px;
    margin-bottom: 8px;
}

.recommendation-relevance {
    display: inline-block;
    background-color: #2ecc71;
    color: white;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 0.9rem;
    margin-right: 10px;
}

.recommendation-reason {
    margin-top: 10px;
    padding: 12px;
    background-color: #f8f9fa;
    border-left: 3px solid #f39c12;
    border-radius: 4px;
}

.analysis-section {
    margin-top: 30px;
    padding: 20px;
    background-color: #f8f9fa;
    border-radius: 8px;
}

/* ===== 原始数据标签页 ===== */
.ai-debug-core-page pre {
    background-color: #2c3e50;
    color: #ecf0f1;
    padding: 20px;
    border-radius: 6px;
    overflow: auto;
    max-height: 400px;
    font-family: 'Courier New', Courier, monospace;
    font-size: 13px;
    white-space: pre-wrap;
}

/* ===== 页脚 ===== */
.ai-debug-footer {
    text-align: center;
    padding: 20px;
    color: #666;
    border-top: 1px solid #eee;
}

.ai-debug-footer .connected {
    background-color: #2ecc71;
}

.ai-debug-footer .api-url {
    margin-top: 5px;
    font-size: 0.9rem;
}

/* ===== 加载动画 ===== */
.loading-spinner {
    border: 4px solid #f3f3f3;
    border-top: 4px solid #3498db;
    border-radius: 50%;
    width: 40px;
    height: 40px;
    animation: spin 1s linear infinite;
    margin: 0 auto 15px;
}

@keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
}
</style>
