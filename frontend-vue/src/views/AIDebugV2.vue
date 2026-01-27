<template>
    <div class="debug-v2-page">
        <header class="debug-header">
            <h1><i class="fas fa-robot"></i> AI 代码调试助手 V2 (4轮对话版)</h1>
            <p class="subtitle">通过4轮对话，逐步引导您理解代码问题并学习调试</p>
        </header>

        <div class="debug-container">
            <!-- 左侧：基本信息输入 -->
            <div class="input-panel">
                <div class="card">
                    <h2><i class="fas fa-info-circle"></i> 基本信息</h2>
                    
                    <div class="form-group">
                        <label for="studentId"><i class="fas fa-user"></i> 学生ID</label>
                        <input type="text" id="studentId" v-model="studentId" placeholder="请输入学号">
                    </div>

                    <div class="form-group">
                        <label for="conversationId"><i class="fas fa-comments"></i> 对话ID</label>
                        <div class="input-group">
                            <input type="text" id="conversationId" v-model="conversationId" readonly>
                            <button class="btn-small" @click="generateNewConversationId">重新生成</button>
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="problemDescription"><i class="fas fa-question-circle"></i> 题目描述</label>
                        <textarea id="problemDescription" rows="4" v-model="problemDescription" 
                                  placeholder="请输入题目要求和描述..."></textarea>
                    </div>

                    <div class="form-group">
                        <label for="codeEditor"><i class="fas fa-code"></i> 学生代码 (C/C++)</label>
                        <div class="code-editor">
                            <textarea id="codeEditor" rows="12" v-model="code" 
                                      placeholder="请输入C/C++代码..." @input="updateLineNumbers"></textarea>
                            <div class="line-numbers" ref="lineNumbers"></div>
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="testPoints"><i class="fas fa-vial"></i> 测试点 (JSON格式，可选)</label>
                        <textarea id="testPoints" rows="3" v-model="testPoints" 
                                  placeholder='[{"input": "测试输入", "status": "Accepted/Wrong Answer"}]'></textarea>
                    </div>

                    <button class="btn-primary" @click="startNewConversation" :disabled="isLoading">
                        <i class="fas fa-play"></i> {{ isLoading ? '正在初始化...' : '开始新对话' }}
                    </button>
                </div>

                <!-- 对话轮次指示器 -->
                <div class="card round-indicator">
                    <h2><i class="fas fa-list-ol"></i> 对话进度</h2>
                    <div class="round-steps">
                        <div class="step" :class="{ active: currentRound >= 1, completed: currentRound > 1 }">
                            <div class="step-number">1</div>
                            <div class="step-info">
                                <div class="step-title">理解思路</div>
                                <div class="step-desc">AI分析你的解题思路</div>
                            </div>
                        </div>
                        <div class="step" :class="{ active: currentRound >= 2, completed: currentRound > 2 }">
                            <div class="step-number">2</div>
                            <div class="step-info">
                                <div class="step-title">指出问题</div>
                                <div class="step-desc">分析问题点和薄弱点</div>
                            </div>
                        </div>
                        <div class="step" :class="{ active: currentRound >= 3, completed: currentRound > 3 }">
                            <div class="step-number">3</div>
                            <div class="step-info">
                                <div class="step-title">调试要点</div>
                                <div class="step-desc">提供调试思路指导</div>
                            </div>
                        </div>
                        <div class="step" :class="{ active: currentRound >= 4, completed: currentRound > 4 }">
                            <div class="step-number">4</div>
                            <div class="step-info">
                                <div class="step-title">详细指导</div>
                                <div class="step-desc">详细修改指导</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 右侧：对话界面 -->
            <div class="conversation-panel">
                <div class="card conversation-card">
                    <div class="conversation-header">
                        <h2><i class="fas fa-comments"></i> 对话窗口</h2>
                        <span class="round-badge">第 {{ currentRound }} 轮</span>
                    </div>

                    <!-- 对话历史 -->
                    <div class="conversation-history" ref="conversationHistory">
                        <div v-for="(message, index) in conversationHistory" :key="index" 
                             :class="['message', message.role]">
                            <div class="message-header">
                                <span class="message-role">
                                    <i :class="message.role === 'assistant' ? 'fas fa-robot' : 'fas fa-user'"></i>
                                    {{ message.role === 'assistant' ? 'AI助手' : '学生' }}
                                </span>
                                <span class="message-round">第{{ message.round_number }}轮</span>
                                <span class="message-time">{{ formatTime(message.timestamp) }}</span>
                            </div>
                            <div class="message-content">
                                <div v-if="message.metadata">
                                    <!-- 第1轮：显示AI理解的学生思路 -->
                                    <div v-if="message.round_number === 1 && message.role === 'assistant'">
                                        <div class="ai-understanding">
                                            <h4><i class="fas fa-lightbulb"></i> AI理解你的思路：</h4>
                                            <p>{{ message.metadata.student_thought || '未提供' }}</p>
                                            
                                            <div v-if="message.metadata.clarity_question" class="clarity-question">
                                                <h4><i class="fas fa-question-circle"></i> 需要确认：</h4>
                                                <p>{{ message.metadata.clarity_question }}</p>
                                            </div>
                                            
                                            <div v-if="message.metadata.suggested_correction" class="suggested-correction">
                                                <h4><i class="fas fa-exclamation-triangle"></i> 修正建议：</h4>
                                                <p>{{ message.metadata.suggested_correction }}</p>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- 第2轮：显示问题分析 -->
                                    <div v-if="message.round_number === 2 && message.role === 'assistant'">
                                        <div class="problem-analysis">
                                            <h4><i class="fas fa-bug"></i> 问题总述：</h4>
                                            <p>{{ message.metadata.problem_summary || '未提供' }}</p>
                                            
                                            <div v-if="message.metadata.key_issues" class="key-issues">
                                                <h4><i class="fas fa-exclamation-triangle"></i> 关键问题：</h4>
                                                <div v-for="(issue, i) in message.metadata.key_issues" :key="i" class="issue-item">
                                                    <div class="issue-location">{{ issue.location }}</div>
                                                    <div class="issue-description">{{ issue.description }}</div>
                                                    <div class="issue-severity">严重程度: {{ issue.severity }}</div>
                                                </div>
                                            </div>
                                            
                                            <div v-if="message.metadata.weak_points" class="weak-points">
                                                <h4><i class="fas fa-tachometer-alt"></i> 薄弱点：</h4>
                                                <div class="weak-points-tags">
                                                    <span v-for="(point, i) in message.metadata.weak_points" :key="i" 
                                                          class="weak-point-tag">{{ point }}</span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- 第3轮：显示调试指导 -->
                                    <div v-if="message.round_number === 3 && message.role === 'assistant'">
                                        <div class="debug-guidance">
                                            <h4><i class="fas fa-wrench"></i> 调试指导：</h4>
                                            <p>{{ message.metadata.debug_guidance || '未提供' }}</p>
                                        </div>
                                    </div>

                                    <!-- 第4轮：显示详细指导 -->
                                    <div v-if="message.round_number === 4 && message.role === 'assistant'">
                                        <div class="detailed-guidance">
                                            <h4><i class="fas fa-chalkboard-teacher"></i> 总体分析：</h4>
                                            <p>{{ message.metadata.debug_analysis || '未提供' }}</p>
                                            
                                            <div v-if="message.metadata.suggestions" class="suggestions-list">
                                                <h4><i class="fas fa-lightbulb"></i> 具体建议：</h4>
                                                <ul>
                                                    <li v-for="(suggestion, i) in message.metadata.suggestions" :key="i">
                                                        {{ suggestion }}
                                                    </li>
                                                </ul>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                
                                <!-- 纯文本内容 -->
                                <div v-else>
                                    {{ message.content }}
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- 当前轮次输入区域 -->
                    <div class="current-input" v-if="currentRound > 0 && currentRound <= 4 && !isCompleted">
                        <!-- 第1轮：学生确认或修正AI的理解 -->
                        <div v-if="currentRound === 1" class="round-input round1">
                            <h4><i class="fas fa-check-circle"></i> 请确认AI的理解是否正确：</h4>
                            <div class="confirmation-options">
                                <button class="btn-confirm" @click="confirmUnderstanding(true)">
                                    <i class="fas fa-check"></i> 是的，理解正确
                                </button>
                                <button class="btn-correct" @click="showCorrectionInput = true">
                                    <i class="fas fa-edit"></i> 需要修正
                                </button>
                            </div>
                            
                            <div v-if="showCorrectionInput" class="correction-input">
                                <textarea v-model="correctionText" rows="3" 
                                          placeholder="请描述你的实际思路，或指出AI理解不正确的地方..."></textarea>
                                <div class="correction-buttons">
                                    <button class="btn-primary" @click="confirmUnderstanding(false)">
                                        提交修正
                                    </button>
                                    <button class="btn-secondary" @click="showCorrectionInput = false">
                                        取消
                                    </button>
                                </div>
                            </div>
                        </div>

                        <!-- 第2轮：是否请求调试帮助 -->
                        <div v-if="currentRound === 2" class="round-input round2">
                            <h4><i class="fas fa-question-circle"></i> {{ getAIMessage('ask_for_help') || '是否需要我提供具体的调试建议？' }}</h4>
                            <div class="help-options">
                                <button class="btn-help" @click="requestHelp(true)">
                                    <i class="fas fa-hands-helping"></i> 是的，我需要帮助
                                </button>
                                <button class="btn-no-help" @click="requestHelp(false)">
                                    <i class="fas fa-user-check"></i> 不需要，我自己试试
                                </button>
                            </div>
                        </div>

                        <!-- 第3轮：是否需要详细指导 -->
                        <div v-if="currentRound === 3" class="round-input round3">
                            <h4><i class="fas fa-question-circle"></i> {{ getAIMessage('ask_for_detail') || '是否需要更详细的修改指导？' }}</h4>
                            <div class="detail-options">
                                <button class="btn-detail" @click="requestDetail(true)">
                                    <i class="fas fa-info-circle"></i> 是的，需要详细指导
                                </button>
                                <button class="btn-no-detail" @click="requestDetail(false)">
                                    <i class="fas fa-check"></i> 不需要，我已明白
                                </button>
                            </div>
                        </div>

                        <!-- 第4轮：对话完成 -->
                        <div v-if="currentRound === 4" class="round-input round4">
                            <div class="completion-message">
                                <h4><i class="fas fa-graduation-cap"></i> 调试指导完成！</h4>
                                <p>希望这次对话能帮助你理解代码问题并提高编程能力。</p>
                                <button class="btn-primary" @click="startNewConversation">
                                    <i class="fas fa-redo"></i> 开始新的调试对话
                                </button>
                                <button class="btn-secondary" @click="exportConversation">
                                    <i class="fas fa-download"></i> 导出对话记录
                                </button>
                            </div>
                        </div>
                    </div>

                    <!-- 加载状态 -->
                    <div v-if="isLoading" class="loading-state">
                        <div class="spinner"></div>
                        <p>AI助手正在思考...</p>
                    </div>
                </div>

                <!-- 对话信息汇总 -->
                <div class="card info-card">
                    <h3><i class="fas fa-chart-bar"></i> 对话信息</h3>
                    <div class="info-grid">
                        <div class="info-item">
                            <div class="info-label">当前轮次</div>
                            <div class="info-value">{{ currentRound }}/4</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">对话状态</div>
                            <div class="info-value">
                                <span :class="isCompleted ? 'status-completed' : 'status-active'">
                                    {{ isCompleted ? '已完成' : '进行中' }}
                                </span>
                            </div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">对话消息</div>
                            <div class="info-value">{{ conversationHistory.length }}</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">开始时间</div>
                            <div class="info-value">{{ formatTime(startTime) }}</div>
                        </div>
                    </div>
                    
                    <div v-if="weakPoints.length > 0" class="weak-points-summary">
                        <h4><i class="fas fa-exclamation-triangle"></i> 识别到的薄弱点</h4>
                        <div class="weak-points-tags">
                            <span v-for="(point, index) in weakPoints" :key="index" class="weak-point-tag">
                                {{ point }}
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue';
import axios from 'axios';

// 基础数据
const studentId = ref('2025001');
const conversationId = ref('');
const problemDescription = ref('');
const code = ref('');
const testPoints = ref('');
const lineNumbers = ref(null);

// 对话状态
const currentRound = ref(0);
const isCompleted = ref(false);
const isLoading = ref(false);
const startTime = ref(new Date());

// 对话历史
const conversationHistory = ref([]);
const weakPoints = ref([]);
const conversationHistoryRef = ref(null);

// 第1轮相关
const showCorrectionInput = ref(false);
const correctionText = ref('');

// 生成对话ID
const generateConversationId = () => {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 8);
    return `conv_v2_${timestamp}_${random}`;
};

// 生成新对话ID
const generateNewConversationId = () => {
    conversationId.value = generateConversationId();
};

// 获取指定字段的AI消息
const getAIMessage = (field) => {
    const lastAIResponse = conversationHistory.value
        .filter(msg => msg.role === 'assistant')
        .slice(-1)[0];
    
    if (lastAIResponse && lastAIResponse.metadata) {
        return lastAIResponse.metadata[field];
    }
    return null;
};

// 开始新对话
const startNewConversation = async () => {
    if (!code.value.trim()) {
        alert('请输入代码');
        return;
    }

    if (!problemDescription.value.trim()) {
        alert('请输入题目描述');
        return;
    }

    // 生成对话ID（如果还没有）
    if (!conversationId.value) {
        conversationId.value = generateConversationId();
    }

    // 重置状态
    resetConversation();
    startTime.value = new Date();
    
    // 开始第1轮
    await sendRoundRequest(1, '');
};

// 重置对话
const resetConversation = () => {
    currentRound.value = 0;
    isCompleted.value = false;
    conversationHistory.value = [];
    weakPoints.value = [];
    showCorrectionInput.value = false;
    correctionText.value = '';
};

// 发送轮次请求
const sendRoundRequest = async (roundNumber, studentResponse) => {
    isLoading.value = true;
    
    try {
        // 解析测试点
        let parsedTestPoints = [];
        try {
            if (testPoints.value.trim()) {
                parsedTestPoints = JSON.parse(testPoints.value);
            }
        } catch (error) {
            alert('测试点JSON格式错误！');
            isLoading.value = false;
            return;
        }

        // 准备请求数据
        const requestData = {
            student_id: studentId.value || 'anonymous',
            conversation_id: conversationId.value,
            code: code.value,
            problem_description: problemDescription.value,
            test_points: parsedTestPoints,
            current_round: roundNumber,
            student_response: studentResponse,
            dialogue_history: conversationHistory.value.map(msg => ({
                round_number: msg.round_number,
                role: msg.role,
                content: msg.content,
                metadata: msg.metadata
            }))
        };

        // 发送请求到Go后端（需要后端实现/debug_v2端点）
        const response = await axios.post('http://localhost:8080/api/v1/ai/debug_v2', requestData);

        if (response.data.error) {
            throw new Error(response.data.message || 'AI服务错误');
        }

        // 更新对话状态
        currentRound.value = response.data.current_round;
        isCompleted.value = response.data.is_completed || false;

        // 添加AI响应到历史
        const aiMessage = {
            round_number: currentRound.value,
            role: 'assistant',
            content: response.data.ai_response?.debug_analysis || 'AI回复',
            metadata: response.data.ai_response,
            timestamp: new Date()
        };
        conversationHistory.value.push(aiMessage);

        // 收集薄弱点（从第2轮开始）
        if (currentRound.value >= 2 && response.data.ai_response?.weak_points) {
            const newWeakPoints = response.data.ai_response.weak_points;
            newWeakPoints.forEach(point => {
                if (!weakPoints.value.includes(point)) {
                    weakPoints.value.push(point);
                }
            });
        }

        // 滚动到最新消息
        scrollToBottom();

    } catch (error) {
        console.error('请求失败:', error);
        alert(`对话失败: ${error.message}`);
        
        // 添加错误消息到历史
        const errorMessage = {
            round_number: currentRound.value,
            role: 'assistant',
            content: `抱歉，处理请求时出现错误: ${error.message}`,
            timestamp: new Date()
        };
        conversationHistory.value.push(errorMessage);
    } finally {
        isLoading.value = false;
    }
};

// 第1轮：确认理解
const confirmUnderstanding = async (isCorrect) => {
    let responseText = '';
    
    if (isCorrect) {
        responseText = '是的，AI的理解正确，这就是我的思路。';
    } else {
        responseText = correctionText.value || '我需要修正AI的理解：' + correctionText.value;
    }

    // 添加学生响应到历史
    const studentMessage = {
        round_number: 1,
        role: 'student',
        content: responseText,
        timestamp: new Date()
    };
    conversationHistory.value.push(studentMessage);

    // 发送第2轮请求
    await sendRoundRequest(2, responseText);
};

// 第2轮：请求帮助
const requestHelp = async (needHelp) => {
    const responseText = needHelp ? '是的，请提供调试建议。' : '不需要，我想自己先尝试解决。';
    
    // 添加学生响应到历史
    const studentMessage = {
        round_number: 2,
        role: 'student',
        content: responseText,
        timestamp: new Date()
    };
    conversationHistory.value.push(studentMessage);

    // 发送第3轮请求
    await sendRoundRequest(3, responseText);
};

// 第3轮：请求详细指导
const requestDetail = async (needDetail) => {
    const responseText = needDetail ? '是的，请提供详细的修改指导。' : '不需要，我已经明白了。';
    
    // 添加学生响应到历史
    const studentMessage = {
        round_number: 3,
        role: 'student',
        content: responseText,
        timestamp: new Date()
    };
    conversationHistory.value.push(studentMessage);

    // 发送第4轮请求
    await sendRoundRequest(4, responseText);
};

// 导出对话记录
const exportConversation = () => {
    const conversationData = {
        conversation_id: conversationId.value,
        student_id: studentId.value,
        start_time: startTime.value.toISOString(),
        end_time: new Date().toISOString(),
        weak_points: weakPoints.value,
        dialogue_history: conversationHistory.value
    };

    const dataStr = JSON.stringify(conversationData, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    
    const link = document.createElement('a');
    link.href = url;
    link.download = `debug_v2_conversation_${conversationId.value}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
};

// 格式化时间
const formatTime = (time) => {
    if (!time) return '';
    const date = new Date(time);
    return date.toLocaleTimeString('zh-CN', { 
        hour: '2-digit', 
        minute: '2-digit',
        second: '2-digit'
    });
};

// 更新行号
const updateLineNumbers = () => {
    if (!lineNumbers.value) return;
    const lines = code.value.split('\n').length;
    const numbers = Array.from({ length: lines }, (_, i) => i + 1).join('\n');
    lineNumbers.value.textContent = numbers;
};

// 滚动到底部
const scrollToBottom = () => {
    nextTick(() => {
        if (conversationHistoryRef.value) {
            conversationHistoryRef.value.scrollTop = conversationHistoryRef.value.scrollHeight;
        }
    });
};

// 加载示例数据
const loadExample = () => {
    problemDescription.value = `编写一个C函数，计算两个整数的最大公约数（GCD）。
要求：
1. 函数原型：int gcd(int a, int b)
2. 使用欧几里得算法
3. 处理特殊情况（如负数）`;

    code.value = `#include <stdio.h>

int gcd(int a, int b) {
    if (b == 0) {
        return a;
    }
    return gcd(b, a % b);
}

int main() {
    int num1, num2;
    printf("请输入两个整数: ");
    scanf("%d %d", &num1, &num2);
    
    // 这里有一个问题：没有处理负数
    int result = gcd(num1, num2);
    
    printf("最大公约数是: %d\\n", result);
    return 0;
}`;

    testPoints.value = `[
  {"input": "12 18", "status": "Accepted"},
  {"input": "7 13", "status": "Accepted"},
  {"input": "0 5", "status": "Accepted"},
  {"input": "-12 18", "status": "Wrong Answer"},
  {"input": "12 0", "status": "Runtime Error"}
]`;

    updateLineNumbers();
};

// 监听对话历史变化，自动滚动
watch(conversationHistory, () => {
    scrollToBottom();
}, { deep: true });

// 初始化
onMounted(() => {
    conversationId.value = generateConversationId();
    loadExample();
    updateLineNumbers();
});
</script>

<style scoped>
.debug-v2-page {
    max-width: 1600px;
    margin: 0 auto;
    padding: 20px;
}

.debug-header {
    text-align: center;
    margin-bottom: 30px;
    padding: 20px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border-radius: 10px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.debug-header h1 {
    font-size: 2.2rem;
    margin-bottom: 10px;
}

.debug-header .subtitle {
    font-size: 1.1rem;
    opacity: 0.9;
}

.debug-container {
    display: grid;
    grid-template-columns: 400px 1fr;
    gap: 25px;
    height: calc(100vh - 200px);
}

.card {
    background: white;
    border-radius: 10px;
    padding: 25px;
    margin-bottom: 20px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
    border: 1px solid #eaeaea;
}

.card h2 {
    color: #2c3e50;
    margin-bottom: 20px;
    font-size: 1.4rem;
    border-bottom: 2px solid #f0f0f0;
    padding-bottom: 10px;
}

.card h2 i {
    margin-right: 10px;
    color: #667eea;
}

/* 左侧输入面板 */
.input-panel {
    display: flex;
    flex-direction: column;
}

.form-group {
    margin-bottom: 20px;
}

.form-group label {
    display: block;
    margin-bottom: 8px;
    font-weight: 600;
    color: #2c3e50;
}

.form-group label i {
    margin-right: 8px;
    color: #667eea;
}

.input-group {
    display: flex;
    gap: 10px;
}

.input-group input {
    flex: 1;
}

.btn-small {
    padding: 8px 15px;
    background: #f0f0f0;
    border: 1px solid #ddd;
    border-radius: 5px;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.3s;
}

.btn-small:hover {
    background: #e0e0e0;
}

textarea, input {
    width: 100%;
    padding: 12px 15px;
    border: 2px solid #e0e0e0;
    border-radius: 6px;
    font-size: 1rem;
    transition: border-color 0.3s;
    font-family: inherit;
}

textarea:focus, input:focus {
    border-color: #667eea;
    outline: none;
}

.code-editor {
    position: relative;
    border: 2px solid #e0e0e0;
    border-radius: 6px;
    overflow: hidden;
}

.code-editor textarea {
    border: none;
    padding-left: 50px;
    line-height: 1.5;
    font-family: 'Courier New', Courier, monospace;
    font-size: 14px;
    background: #f8f9fa;
}

.line-numbers {
    position: absolute;
    left: 0;
    top: 0;
    width: 40px;
    height: 100%;
    background-color: #f0f0f0;
    border-right: 1px solid #e0e0e0;
    padding: 12px 5px;
    font-family: 'Courier New', Courier, monospace;
    font-size: 14px;
    color: #666;
    text-align: right;
    overflow: hidden;
}

.btn-primary {
    width: 100%;
    padding: 14px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 1.1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
}

.btn-primary:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}

.btn-primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.btn-secondary {
    padding: 10px 20px;
    background: #f0f0f0;
    color: #333;
    border: 1px solid #ddd;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s;
}

.btn-secondary:hover {
    background: #e0e0e0;
}

/* 轮次指示器 */
.round-indicator {
    margin-top: 20px;
}

.round-steps {
    display: flex;
    flex-direction: column;
    gap: 15px;
}

.step {
    display: flex;
    align-items: center;
    padding: 15px;
    border-radius: 8px;
    border: 2px solid #f0f0f0;
    opacity: 0.6;
    transition: all 0.3s;
}

.step.active {
    opacity: 1;
    border-color: #667eea;
    background: #f8f9ff;
}

.step.completed {
    opacity: 1;
    border-color: #2ecc71;
    background: #f0fff4;
}

.step-number {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    font-size: 1.2rem;
    margin-right: 15px;
    color: #666;
}

.step.active .step-number {
    background: #667eea;
    color: white;
}

.step.completed .step-number {
    background: #2ecc71;
    color: white;
}

.step-info {
    flex: 1;
}

.step-title {
    font-weight: 600;
    color: #2c3e50;
    margin-bottom: 4px;
}

.step-desc {
    font-size: 0.9rem;
    color: #666;
}

/* 右侧对话面板 */
.conversation-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.conversation-card {
    flex: 1;
    display: flex;
    flex-direction: column;
    max-height: calc(100vh - 400px);
}

.conversation-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
}

.round-badge {
    background: #667eea;
    color: white;
    padding: 6px 15px;
    border-radius: 20px;
    font-weight: bold;
    font-size: 0.9rem;
}

.conversation-history {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    background: #f8f9fa;
    border-radius: 8px;
    margin-bottom: 20px;
    max-height: 500px;
}

.message {
    margin-bottom: 20px;
    animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
}

.message-header {
    display: flex;
    align-items: center;
    gap: 15px;
    margin-bottom: 8px;
    font-size: 0.9rem;
    color: #666;
}

.message-role {
    font-weight: bold;
    color: #2c3e50;
}

.message-role i {
    margin-right: 5px;
}

.message.student .message-role {
    color: #e74c3c;
}

.message.assistant .message-role {
    color: #3498db;
}

.message-content {
    background: white;
    padding: 15px;
    border-radius: 10px;
    box-shadow: 0 2px 5px rgba(0,0,0,0.05);
    border-left: 4px solid #667eea;
}

.message.student .message-content {
    border-left-color: #e74c3c;
    background: #fff5f5;
}

.message.assistant .message-content {
    border-left-color: #3498db;
    background: #f0f8ff;
}

/* AI响应样式 */
.ai-understanding,
.problem-analysis,
.debug-guidance,
.detailed-guidance {
    padding: 15px;
    background: white;
    border-radius: 8px;
    border: 1px solid #eaeaea;
}

.ai-understanding h4,
.problem-analysis h4,
.debug-guidance h4,
.detailed-guidance h4 {
    color: #2c3e50;
    margin-bottom: 10px;
    display: flex;
    align-items: center;
    gap: 8px;
}

.clarity-question {
    margin-top: 15px;
    padding: 15px;
    background: #fff9e6;
    border-left: 4px solid #f39c12;
    border-radius: 6px;
}

.suggested-correction {
    margin-top: 15px;
    padding: 15px;
    background: #ffe6e6;
    border-left: 4px solid #e74c3c;
    border-radius: 6px;
}

.key-issues {
    margin-top: 15px;
}

.issue-item {
    padding: 12px;
    margin-bottom: 10px;
    background: #fff9e6;
    border-left: 4px solid #f1c40f;
    border-radius: 6px;
}

.issue-location {
    font-weight: bold;
    color: #d35400;
    margin-bottom: 5px;
}

.issue-description {
    margin-bottom: 5px;
}

.issue-severity {
    font-size: 0.9rem;
    color: #7f8c8d;
}

.weak-points {
    margin-top: 15px;
}

.weak-points-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 10px;
}

.weak-point-tag {
    background: #ffeaa7;
    color: #d35400;
    padding: 5px 12px;
    border-radius: 20px;
    font-size: 0.9rem;
    font-weight: 500;
}

.suggestions-list ul {
    padding-left: 20px;
    margin-top: 10px;
}

.suggestions-list li {
    margin-bottom: 8px;
    line-height: 1.5;
}

/* 当前输入区域 */
.current-input {
    padding: 20px;
    background: #f8f9ff;
    border-radius: 10px;
    border: 2px dashed #667eea;
}

.round-input h4 {
    color: #2c3e50;
    margin-bottom: 20px;
    font-size: 1.2rem;
}

.confirmation-options,
.help-options,
.detail-options {
    display: flex;
    gap: 15px;
    flex-wrap: wrap;
}

.btn-confirm,
.btn-help,
.btn-detail {
    flex: 1;
    padding: 15px;
    background: linear-gradient(135deg, #2ecc71 0%, #27ae60 100%);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 1.1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
}

.btn-confirm:hover,
.btn-help:hover,
.btn-detail:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 15px rgba(46, 204, 113, 0.4);
}

.btn-correct,
.btn-no-help,
.btn-no-detail {
    flex: 1;
    padding: 15px;
    background: #f0f0f0;
    color: #333;
    border: 2px solid #ddd;
    border-radius: 8px;
    font-size: 1.1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
}

.btn-correct:hover,
.btn-no-help:hover,
.btn-no-detail:hover {
    background: #e0e0e0;
}

.correction-input {
    margin-top: 20px;
    padding: 20px;
    background: white;
    border-radius: 8px;
    border: 1px solid #eaeaea;
}

.correction-input textarea {
    width: 100%;
    margin-bottom: 15px;
}

.correction-buttons {
    display: flex;
    gap: 15px;
}

.completion-message {
    text-align: center;
    padding: 30px;
}

.completion-message h4 {
    color: #2ecc71;
    font-size: 1.5rem;
    margin-bottom: 15px;
}

.completion-message p {
    color: #666;
    margin-bottom: 25px;
}

.completion-message button {
    margin: 0 10px;
    min-width: 180px;
}

/* 加载状态 */
.loading-state {
    text-align: center;
    padding: 40px;
}

.spinner {
    border: 4px solid #f3f3f3;
    border-top: 4px solid #667eea;
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

/* 信息卡片 */
.info-card h3 {
    margin-bottom: 20px;
    font-size: 1.3rem;
}

.info-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 15px;
    margin-bottom: 25px;
}

.info-item {
    background: #f8f9fa;
    padding: 15px;
    border-radius: 8px;
    border-left: 4px solid #667eea;
}

.info-label {
    font-size: 0.9rem;
    color: #666;
    margin-bottom: 5px;
}

.info-value {
    font-weight: bold;
    color: #2c3e50;
    font-size: 1.2rem;
}

.status-active {
    color: #3498db;
    font-weight: bold;
}

.status-completed {
    color: #2ecc71;
    font-weight: bold;
}

.weak-points-summary {
    padding-top: 20px;
    border-top: 2px solid #f0f0f0;
}

.weak-points-summary h4 {
    margin-bottom: 15px;
    color: #2c3e50;
}

/* 响应式设计 */
@media (max-width: 1200px) {
    .debug-container {
        grid-template-columns: 1fr;
        height: auto;
    }
    
    .input-panel {
        order: 2;
    }
    
    .conversation-panel {
        order: 1;
    }
}

@media (max-width: 768px) {
    .confirmation-options,
    .help-options,
    .detail-options {
        flex-direction: column;
    }
    
    .info-grid {
        grid-template-columns: 1fr;
    }
    
    .completion-message button {
        width: 100%;
        margin: 10px 0;
    }
}
</style>