<!-- views/DebugV2Test.vue -->
<template>
    <div class="simple-test">
        <h1>Debug V2 简单测试</h1>
        
        <div class="test-area">
            <div class="input-section">
                <textarea v-model="code" rows="10" placeholder="输入C/C++代码"></textarea>
                <textarea v-model="problem" rows="5" placeholder="输入题目描述"></textarea>
                <button @click="startTest">开始测试</button>
            </div>
            
            <div class="output-section">
                <div v-if="conversation.length > 0" class="conversation">
                    <div v-for="(msg, idx) in conversation" :key="idx" class="message">
                        <div class="role">{{ msg.role === 'assistant' ? 'AI' : '用户' }}</div>
                        <div class="content">{{ msg.content }}</div>
                    </div>
                </div>
                
                <div v-if="showOptions" class="options">
                    <button v-for="(option, idx) in currentOptions" :key="idx" @click="selectOption(option)">
                        {{ option }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue';
import axios from 'axios';

const code = ref('');
const problem = ref('');
const conversation = ref([]);
const showOptions = ref(false);
const currentOptions = ref([]);
const conversationId = ref(`test_${Date.now()}`);
const currentRound = ref(1);

const startTest = async () => {
    if (!code.value.trim() || !problem.value.trim()) {
        alert('请填写代码和题目描述');
        return;
    }
    
    conversation.value = [];
    currentRound.value = 1;
    
    await sendRequest('');
};

const sendRequest = async (studentResponse) => {
    try {
        const requestData = {
            student_id: 'test_user',
            conversation_id: conversationId.value,
            code: code.value,
            problem_description: problem.value,
            test_points: [],
            current_round: currentRound.value,
            student_response: studentResponse,
            dialogue_history: conversation.value.map(msg => ({
                round_number: msg.round,
                role: msg.role,
                content: msg.content
            }))
        };
        
        const response = await axios.post('http://localhost:8000/debug_v2', requestData);
        
        // 添加AI响应
        conversation.value.push({
            round: currentRound.value,
            role: 'assistant',
            content: JSON.stringify(response.data.ai_response, null, 2)
        });
        
        // 根据轮次设置选项
        currentRound.value = response.data.next_round;
        
        if (currentRound.value <= 4) {
            showOptions.value = true;
            if (currentRound.value === 2) {
                currentOptions.value = ['确认理解正确', '需要修正'];
            } else if (currentRound.value === 3) {
                currentOptions.value = ['需要调试帮助', '不需要帮助'];
            } else if (currentRound.value === 4) {
                currentOptions.value = ['需要详细指导', '不需要详细指导'];
            }
        } else {
            showOptions.value = false;
        }
        
    } catch (error) {
        console.error('请求失败:', error);
        conversation.value.push({
            round: currentRound.value,
            role: 'assistant',
            content: `错误: ${error.message}`
        });
    }
};

const selectOption = async (option) => {
    // 添加用户选择
    conversation.value.push({
        round: currentRound.value - 1,
        role: 'user',
        content: option
    });
    
    showOptions.value = false;
    await sendRequest(option);
};
</script>

<style scoped>
.simple-test {
    padding: 20px;
    max-width: 800px;
    margin: 0 auto;
}

.test-area {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
}

.input-section, .output-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

textarea {
    width: 100%;
    padding: 10px;
    border: 1px solid #ccc;
    border-radius: 5px;
}

button {
    padding: 10px;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 5px;
    cursor: pointer;
}

.conversation {
    border: 1px solid #ddd;
    border-radius: 5px;
    padding: 10px;
    max-height: 400px;
    overflow-y: auto;
}

.message {
    margin-bottom: 10px;
    padding: 10px;
    background: #f5f5f5;
    border-radius: 5px;
}

.role {
    font-weight: bold;
    color: #007bff;
    margin-bottom: 5px;
}

.options {
    display: flex;
    flex-direction: column;
    gap: 10px;
}
</style>