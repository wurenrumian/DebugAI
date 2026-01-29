<!-- views/DebugV2Test.vue -->
<template>
    <div class="simple-test">
        <h1>Debug V2 简单测试</h1>
        
        <div class="test-area">
            <div class="input-section">
                <h3>输入信息</h3>
                <div class="form-group">
                    <label>题目描述：</label>
                    <textarea v-model="problem" rows="5" placeholder="输入题目描述"></textarea>
                </div>
                
                <div class="form-group">
                    <label>学生代码（C/C++）：</label>
                    <textarea v-model="code" rows="10" placeholder="输入C/C++代码"></textarea>
                </div>
                
                <div class="form-group">
                    <label>测试点信息（JSON格式）：</label>
                    <div class="json-input">
                        <textarea v-model="testPointsJson" rows="8" placeholder='输入测试点JSON，例如：[{"input":"1 2", "status":"Accepted"}, {"input":"3 4", "status":"Wrong Answer"}]'></textarea>
                        <div class="json-info">
                            <small>input: 测试输入内容</small><br>
                            <small>status: 状态 (Accepted, Wrong Answer, Time Limit Exceeded, Memory Limit Exceeded, Runtime Error)</small>
                        </div>
                    </div>
                    <button class="btn-validate" @click="validateTestPoints" :disabled="isValidating">验证JSON格式</button>
                    <div v-if="testPoints.length > 0" class="test-points-summary">
                        已加载 {{ testPoints.length }} 个测试点
                        <span v-if="failedTestPoints > 0" class="failed">（失败: {{ failedTestPoints }}）</span>
                        <span v-if="passedTestPoints > 0" class="passed">（通过: {{ passedTestPoints }}）</span>
                    </div>
                </div>
                
                <button class="btn-start" @click="startTest" :disabled="isTesting || !code.trim() || !problem.trim()">
                    开始测试
                </button>
            </div>
            
            <div class="output-section">
                <div class="section-header">
                    <h3>对话过程</h3>
                    <div class="round-indicator">当前轮次: {{ currentRound }}</div>
                </div>
                
                <div v-if="conversation.length > 0" class="conversation">
                    <div v-for="(msg, idx) in conversation" :key="idx" class="message" :class="{'system': msg.role === 'system'}">
                        <div class="message-header">
                            <div class="role">{{ getRoleName(msg.role) }}</div>
                            <div class="round">第{{ msg.round }}轮</div>
                        </div>
                        <div class="content">
                            <pre v-if="isJson(msg.content)">{{ formatJson(msg.content) }}</pre>
                            <div v-else>{{ msg.content }}</div>
                        </div>
                    </div>
                </div>
                <div v-else class="empty-conversation">
                    <p>对话将在这里显示...</p>
                </div>
                
                <div v-if="showOptions && !conversationEnded" class="options">
                    <h4>请选择：</h4>
                    <div class="option-buttons">
                        <button v-for="(option, idx) in currentOptions" :key="idx" 
                                @click="selectOption(option.value)"
                                :class="{'btn-primary': option.type === 'positive', 'btn-secondary': option.type === 'negative'}">
                            {{ option.text }}
                        </button>
                    </div>
                </div>
                
                <div v-if="conversationEnded" class="end-message">
                    <div class="end-header">
                        <h3>🎉 对话已结束</h3>
                    </div>
                    <div class="end-content">
                        <p>对话已完成 {{ totalRounds }} 轮交互</p>
                        <p v-if="weakPoints.length > 0">识别到的薄弱点：</p>
                        <ul v-if="weakPoints.length > 0" class="weak-points">
                            <li v-for="(point, idx) in weakPoints" :key="idx">{{ point }}</li>
                        </ul>
                        <button class="btn-restart" @click="resetConversation">开始新的对话</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import axios from 'axios';

const code = ref('');
const problem = ref('');
const testPointsJson = ref('');
const testPoints = ref([]);
const conversation = ref([]);
const showOptions = ref(false);
const currentOptions = ref([]);
const conversationId = ref('');
const currentRound = ref(1);
const conversationEnded = ref(false);
const isTesting = ref(false);
const isValidating = ref(false);
const weakPoints = ref([]);

// 轮次选项配置
const roundOptions = {
    1: [], // 第1轮不需要用户选项
    2: [
        { text: '确认理解正确', value: '确认理解正确', type: 'positive' },
        { text: '需要修正思路', value: '需要修正思路', type: 'negative' }
    ],
    3: [
        { text: '需要调试帮助', value: '需要调试帮助', type: 'positive' },
        { text: '不需要调试帮助', value: '不需要调试帮助', type: 'negative' }
    ],
    4: [
        { text: '需要详细指导', value: '需要详细指导', type: 'positive' },
        { text: '不需要详细指导', value: '不需要详细指导', type: 'negative' }
    ]
};

// 计算属性
const passedTestPoints = computed(() => {
    return testPoints.value.filter(tp => tp.status === 'Accepted').length;
});

const failedTestPoints = computed(() => {
    return testPoints.value.filter(tp => tp.status !== 'Accepted').length;
});

const totalRounds = computed(() => {
    return conversation.value.filter(msg => msg.role === 'assistant').length;
});

// 方法
const getRoleName = (role) => {
    const roleMap = {
        'assistant': 'AI助手',
        'user': '学生',
        'system': '系统'
    };
    return roleMap[role] || role;
};

const isJson = (str) => {
    try {
        JSON.parse(str);
        return true;
    } catch {
        return false;
    }
};

const formatJson = (jsonStr) => {
    try {
        const obj = JSON.parse(jsonStr);
        return JSON.stringify(obj, null, 2);
    } catch {
        return jsonStr;
    }
};

const validateTestPoints = () => {
    if (!testPointsJson.value.trim()) {
        testPoints.value = [];
        return;
    }
    
    isValidating.value = true;
    try {
        const parsed = JSON.parse(testPointsJson.value);
        if (Array.isArray(parsed)) {
            // 验证每个测试点都有必要的字段
            const validPoints = parsed.filter(point => 
                point && typeof point === 'object' && 
                'input' in point && 'status' in point
            );
            testPoints.value = validPoints;
            alert(`验证成功！加载了 ${validPoints.length} 个测试点`);
        } else {
            alert('测试点必须是数组格式');
            testPoints.value = [];
        }
    } catch (error) {
        alert(`JSON格式错误: ${error.message}`);
        testPoints.value = [];
    } finally {
        isValidating.value = false;
    }
};

const startTest = async () => {
    if (!code.value.trim() || !problem.value.trim()) {
        alert('请填写代码和题目描述');
        return;
    }
    
    resetConversation();
    isTesting.value = true;
    
    await sendRequest('');
    isTesting.value = false;
};

const sendRequest = async (studentResponse) => {
    try {
        // 生成对话ID（如果是新的对话）
        if (!conversationId.value) {
            conversationId.value = `test_${Date.now()}`;
        }
        
        const requestData = {
            student_id: 'test_user',
            conversation_id: conversationId.value,
            code: code.value,
            problem_description: problem.value,
            test_points: testPoints.value, // 使用解析后的测试点
            current_round: currentRound.value,
            student_response: studentResponse || null,
            dialogue_history: conversation.value
                .filter(msg => msg.role === 'assistant' || msg.role === 'user') // 过滤掉系统消息，保留AI和用户的对话历史
                .map(msg => ({
                    round_number: msg.round,
                    role: msg.role,
                    content: msg.content
                }))
        };
        
        console.log('发送请求:', requestData);
        
        const response = await axios.post('http://localhost:8000/debug_v2', requestData);
        
        // 添加AI响应到对话历史
        conversation.value.push({
            round: currentRound.value,
            role: 'assistant',
            content: JSON.stringify(response.data.ai_response, null, 2)
        });
        
        // 提取薄弱点（如果有）
        extractWeakPoints(response.data.ai_response);
        
        // 检查对话是否应该结束
        checkConversationEnd(studentResponse);
        
        // 如果不是结束状态，设置下一轮选项
        if (!conversationEnded.value && currentRound.value <= 4) {
            showOptions.value = true;
            currentOptions.value = roundOptions[currentRound.value];
        }
        
    } catch (error) {
        console.error('请求失败:', error);
        conversation.value.push({
            round: currentRound.value,
            role: 'assistant',
            content: `错误: ${error.response?.data?.detail?.message || error.message}`
        });
        conversationEnded.value = true;
        showOptions.value = false;
    }
};

const extractWeakPoints = (aiResponse) => {
    if (aiResponse && aiResponse.weak_points) {
        weakPoints.value = aiResponse.weak_points;
    } else if (aiResponse && typeof aiResponse === 'object') {
        // 尝试从不同可能的字段中提取薄弱点
        for (const key in aiResponse) {
            if (Array.isArray(aiResponse[key]) && 
                aiResponse[key].some(item => typeof item === 'string' && 
                (item.includes('错误') || item.includes('不足') || item.includes('不当')))) {
                weakPoints.value = [...new Set([...weakPoints.value, ...aiResponse[key]])];
            }
        }
    }
};

const checkConversationEnd = (studentResponse) => {
    // 第3轮：如果学生选择"不需要调试帮助"，结束对话
    if (currentRound.value === 3 && studentResponse === '不需要调试帮助') {
        conversationEnded.value = true;
        showOptions.value = false;
        conversation.value.push({
            round: currentRound.value,
            role: 'system',
            content: '对话结束：学生选择不需要调试帮助'
        });
        return;
    }
    
    // 第4轮：AI响应后都结束对话
    if (currentRound.value === 4) {
        conversationEnded.value = true;
        showOptions.value = false;
        conversation.value.push({
            round: currentRound.value,
            role: 'system',
            content: '对话结束：第4轮已完成'
        });
        return;
    }
    
    // 否则，进入下一轮
    if (currentRound.value < 4) {
        currentRound.value++;
    }
};

const selectOption = async (option) => {
    // 特殊处理：第4轮选择"不需要详细指导"时，直接结束，不发送请求
    if (currentRound.value === 4 && option === '不需要详细指导') {
        conversation.value.push({
            round: currentRound.value,
            role: 'user',
            content: option
        });
        conversation.value.push({
            round: currentRound.value,
            role: 'system',
            content: '对话结束：学生选择不需要详细指导'
        });
        conversationEnded.value = true;
        showOptions.value = false;
        return;
    }
    
    // 添加用户选择到对话历史
    conversation.value.push({
        round: currentRound.value,
        role: 'user',
        content: option
    });
    
    showOptions.value = false;
    await sendRequest(option);
};

const resetConversation = () => {
    conversation.value = [];
    currentRound.value = 1;
    conversationEnded.value = false;
    showOptions.value = false;
    currentOptions.value = [];
    weakPoints.value = [];
    // 注意：不清除conversationId，以便同一测试可以继续
};

// 示例数据
const loadExample = () => {
    problem.value = `编写一个C++程序，计算两个整数的和。
输入：两个整数a和b
输出：a + b的和
示例：
输入：3 4
输出：7`;
    
    code.value = `#include <iostream>
using namespace std;

int main() {
    int a, b;
    cin >> a >> b;
    
    if (a > 100) {
        cout << "a is too large" << endl;
    }
    
    int sum = a + b;
    cout << "Sum: " << sum << endl;
    
    return 0;
}`;
    
    testPointsJson.value = `[
    {"input": "3 4", "status": "Accepted"},
    {"input": "0 0", "status": "Accepted"},
    {"input": "-5 10", "status": "Accepted"},
    {"input": "101 5", "status": "Wrong Answer"},
    {"input": "2147483647 1", "status": "Time Limit Exceeded"}
]`;
    
    validateTestPoints();
};

onMounted(() => {
    // 可以加载示例数据，方便测试
    // loadExample();
});
</script>

<style scoped>
.simple-test {
    padding: 20px;
    max-width: 1200px;
    margin: 0 auto;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

h1 {
    text-align: center;
    color: #2c3e50;
    margin-bottom: 30px;
    border-bottom: 2px solid #3498db;
    padding-bottom: 10px;
}

.test-area {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 30px;
}

.input-section, .output-section {
    display: flex;
    flex-direction: column;
    gap: 15px;
}

.section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
}

.round-indicator {
    background-color: #3498db;
    color: white;
    padding: 5px 15px;
    border-radius: 20px;
    font-weight: bold;
    font-size: 0.9em;
}

.form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.form-group label {
    font-weight: bold;
    color: #2c3e50;
    font-size: 0.95em;
}

textarea {
    width: 100%;
    padding: 12px;
    border: 1px solid #ddd;
    border-radius: 8px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
    resize: vertical;
    transition: border-color 0.3s;
}

textarea:focus {
    outline: none;
    border-color: #3498db;
    box-shadow: 0 0 0 2px rgba(52, 152, 219, 0.2);
}

.json-input {
    position: relative;
}

.json-info {
    background-color: #f8f9fa;
    border-left: 4px solid #3498db;
    padding: 8px 12px;
    margin-top: 5px;
    font-size: 0.85em;
    color: #666;
    border-radius: 0 4px 4px 0;
}

.btn-validate, .btn-start, .btn-restart {
    padding: 12px 20px;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: bold;
    font-size: 1em;
    transition: all 0.3s;
}

.btn-validate {
    background-color: #f39c12;
    color: white;
    align-self: flex-start;
}

.btn-validate:hover:not(:disabled) {
    background-color: #e67e22;
}

.btn-start {
    background-color: #2ecc71;
    color: white;
    margin-top: 10px;
}

.btn-start:hover:not(:disabled) {
    background-color: #27ae60;
}

.btn-start:disabled {
    background-color: #95a5a6;
    cursor: not-allowed;
}

.btn-restart {
    background-color: #3498db;
    color: white;
    padding: 12px 30px;
    font-size: 1.1em;
}

.btn-restart:hover {
    background-color: #2980b9;
    transform: translateY(-2px);
}

.test-points-summary {
    padding: 8px 12px;
    background-color: #ecf0f1;
    border-radius: 6px;
    font-size: 0.9em;
    margin-top: 10px;
}

.failed {
    color: #e74c3c;
    font-weight: bold;
}

.passed {
    color: #27ae60;
    font-weight: bold;
}

.conversation {
    border: 1px solid #ddd;
    border-radius: 10px;
    padding: 15px;
    max-height: 500px;
    overflow-y: auto;
    background-color: #f8f9fa;
}

.empty-conversation {
    border: 2px dashed #ddd;
    border-radius: 10px;
    padding: 40px;
    text-align: center;
    color: #95a5a6;
    background-color: #f8f9fa;
}

.message {
    margin-bottom: 15px;
    padding: 15px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 5px rgba(0,0,0,0.1);
    border-left: 4px solid #3498db;
}

.message.system {
    border-left-color: #95a5a6;
    background-color: #ecf0f1;
}

.message-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    padding-bottom: 5px;
    border-bottom: 1px solid #eee;
}

.role {
    font-weight: bold;
    color: #3498db;
    font-size: 0.9em;
}

.message.system .role {
    color: #7f8c8d;
}

.round {
    font-size: 0.8em;
    color: #95a5a6;
    background-color: #ecf0f1;
    padding: 2px 8px;
    border-radius: 10px;
}

.content {
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.85em;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.5;
}

pre {
    margin: 0;
    white-space: pre-wrap;
}

.options {
    margin-top: 20px;
    padding: 20px;
    border: 2px solid #3498db;
    border-radius: 10px;
    background-color: #f8f9fa;
}

.options h4 {
    margin-bottom: 15px;
    color: #2c3e50;
    text-align: center;
}

.option-buttons {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 15px;
}

button[class^="btn-"] {
    padding: 12px;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: bold;
    transition: all 0.3s;
}

.btn-primary {
    background-color: #2ecc71;
    color: white;
}

.btn-primary:hover {
    background-color: #27ae60;
    transform: translateY(-2px);
}

.btn-secondary {
    background-color: #e74c3c;
    color: white;
}

.btn-secondary:hover {
    background-color: #c0392b;
    transform: translateY(-2px);
}

.end-message {
    margin-top: 20px;
    padding: 25px;
    border: 2px solid #2ecc71;
    border-radius: 10px;
    background-color: #f8f9fa;
    text-align: center;
}

.end-header {
    margin-bottom: 15px;
}

.end-header h3 {
    color: #27ae60;
    margin-bottom: 10px;
}

.end-content {
    color: #2c3e50;
}

.weak-points {
    list-style-type: none;
    padding: 0;
    margin: 10px 0 20px 0;
}

.weak-points li {
    background-color: #f39c12;
    color: white;
    padding: 8px 15px;
    margin-bottom: 8px;
    border-radius: 20px;
    display: inline-block;
    margin-right: 10px;
    font-size: 0.9em;
}

button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none !important;
}
</style>