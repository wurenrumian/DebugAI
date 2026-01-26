<template>
    <div class="conversation-history-page">
        <h2>对话历史</h2>
        <div v-if="isLoading" class="loading-indicator">加载中...</div>
        <div v-else-if="errorMessage" class="error-message">{{ errorMessage }}</div>
        <div v-else class="history-list">
            <div v-if="history.evaluate_records && history.evaluate_records.length">
                <h3>代码评估历史</h3>
                <div v-for="record in history.evaluate_records" :key="record.ID" class="history-card evaluate-card">
                    <h4>评估记录 #{{ record.ID }} ({{ new Date(record.CreatedAt).toLocaleString() }})</h4>
                    <p><strong>学号:</strong> {{ record.StudentID }}</p>
                    <p><strong>会话ID:</strong> {{ record.ConversationID }}</p>
                    <p><strong>总分:</strong> {{ record.Score }}</p>
                    <p><strong>整体评价:</strong> {{ record.OverallEvaluation }}</p>
                    <details>
                        <summary><strong>详细评估</strong></summary>
                        <p><strong>可读性得分:</strong> {{ record.ReadabilityScore }} - {{ record.ReadabilityAnalysis }}</p>
                        <p><strong>逻辑严谨性得分:</strong> {{ record.LogicalRigorScore }} - {{ record.LogicalRigorAnalysis }}</p>
                        <p><strong>算法合理性得分:</strong> {{ record.AlgorithmQualityScore }} - {{ record.AlgorithmQualityAnalysis }}</p>
                        <p><strong>运行效率得分:</strong> {{ record.EfficiencyScore }} - {{ record.EfficiencyAnalysis }}</p>
                    </details>
                    <details>
                        <summary><strong>提交代码</strong></summary>
                        <pre>{{ record.Code }}</pre>
                    </details>
                    <details>
                        <summary><strong>题目描述</strong></summary>
                        <p>{{ record.ProblemDescription }}</p>
                    </details>
                </div>
            </div>

            <div v-if="history.debug_records && history.debug_records.length">
                <h3>代码调试历史</h3>
                <div v-for="record in history.debug_records" :key="record.ID" class="history-card debug-card">
                    <h4>调试记录 #{{ record.ID }} ({{ new Date(record.CreatedAt).toLocaleString() }})</h4>
                    <p><strong>学号:</strong> {{ record.StudentID }}</p>
                    <p><strong>会话ID:</strong> {{ record.ConversationID }}</p>
                    <p><strong>调试分析:</strong> {{ record.DebugAnalysis }}</p>
                    <details>
                        <summary><strong>薄弱点</strong></summary>
                        <ul v-if="parseJson(record.WeakPoints).length">
                            <li v-for="(point, index) in parseJson(record.WeakPoints)" :key="index">{{ point }}</li>
                        </ul>
                        <p v-else>无薄弱点记录</p>
                    </details>
                     <details>
                        <summary><strong>问题点</strong></summary>
                        <ul v-if="parseJson(record.Problems).length">
                            <li v-for="(problem, index) in parseJson(record.Problems)" :key="index">
                                <strong>位置:</strong> {{ problem.location || '未知' }}<br/>
                                <strong>描述:</strong> {{ problem.description || '无描述' }}<br/>
                                <strong>原因:</strong> {{ problem.root_cause || '未知原因' }}
                            </li>
                        </ul>
                        <p v-else>无问题记录</p>
                    </details>
                    <details>
                        <summary><strong>建议</strong></summary>
                        <ul v-if="parseJson(record.Suggestions).length">
                            <li v-for="(suggestion, index) in parseJson(record.Suggestions)" :key="index">{{ suggestion }}</li>
                        </ul>
                        <p v-else>无建议记录</p>
                    </details>
                     <details>
                        <summary><strong>提交代码</strong></summary>
                        <pre>{{ record.Code }}</pre>
                    </details>
                    <details>
                        <summary><strong>题目描述</strong></summary>
                        <p>{{ record.ProblemDescription }}</p>
                    </details>
                </div>
            </div>

            <div v-if="history.recommendation_records && history.recommendation_records.length">
                <h3>题目推荐历史</h3>
                <div v-for="record in history.recommendation_records" :key="record.ID" class="history-card recommend-card">
                    <h4>推荐记录 #{{ record.ID }} ({{ new Date(record.CreatedAt).toLocaleString() }})</h4>
                    <p><strong>学号:</strong> {{ record.StudentID }}</p>
                    <p><strong>会话ID:</strong> {{ record.ConversationID }}</p>
                    <p><strong>推荐分析:</strong> {{ record.Analysis }}</p>
                    <details>
                        <summary><strong>请求薄弱点</strong></summary>
                        <pre>{{ record.RequestedWeakPoints }}</pre>
                    </details>
                    <details>
                        <summary><strong>推荐列表</strong></summary>
                        <ul v-if="parseJson(record.Recommendations).length">
                            <li v-for="(rec, index) in parseJson(record.Recommendations)" :key="index">
                                <strong>标签:</strong> {{ rec.tag }}<br/>
                                <strong>相关度:</strong> {{ (rec.relevance * 100).toFixed(0) }}%<br/>
                                <strong>理由:</strong> {{ rec.reason }}
                            </li>
                        </ul>
                        <p v-else>无推荐题目</p>
                    </details>
                </div>
            </div>

            <p v-if="!history.evaluate_records.length && !history.debug_records.length && !history.recommendation_records.length">
                暂无AI交互历史记录。
            </p>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';

const history = ref({
    evaluate_records: [],
    debug_records: [],
    recommendation_records: [],
});
const isLoading = ref(true);
const errorMessage = ref(null);

const fetchHistory = async () => {
    try {
        isLoading.value = true;
        errorMessage.value = null;
        // The /api/v1/ai prefix is added by the axios interceptor for Authorization
        const response = await axios.get('http://localhost:8080/api/v1/ai/history'); 
        history.value = response.data.history;
    } catch (error) {
        console.error('Error fetching history:', error);
        errorMessage.value = '未能加载历史记录。请确保已登录并重试。';
    } finally {
        isLoading.value = false;
    }
};

const parseJson = (jsonString) => {
    try {
        return JSON.parse(jsonString);
    } catch (e) {
        console.error("Error parsing JSON string:", jsonString, e);
        return [];
    }
};

onMounted(() => {
    fetchHistory();
});
</script>

<style scoped>
.conversation-history-page {
    padding: 20px;
    max-width: 1200px;
    margin: 20px auto;
    background-color: #f9f9f9;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
}

h2 {
    text-align: center;
    color: #333;
    margin-bottom: 30px;
    font-size: 2.2rem;
    border-bottom: 2px solid #eee;
    padding-bottom: 15px;
}

h3 {
    color: #0056b3;
    margin-top: 30px;
    margin-bottom: 20px;
    border-left: 5px solid #007bff;
    padding-left: 10px;
}

.loading-indicator, .error-message {
    text-align: center;
    padding: 20px;
    font-size: 1.1rem;
    color: #555;
}

.error-message {
    color: #d9534f;
    background-color: #fdd;
    border: 1px solid #d9534f;
    border-radius: 5px;
}

.history-card {
    background-color: #fff;
    border: 1px solid #e0e0e0;
    border-radius: 8px;
    padding: 20px 25px;
    margin-bottom: 20px;
    box-shadow: 0 1px 5px rgba(0, 0, 0, 0.03);
    transition: transform 0.2s ease-in-out;
}

.history-card:hover {
    transform: translateY(-3px);
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
}

.history-card h4 {
    color: #2c3e50;
    font-size: 1.3rem;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 1px dashed #eee;
}

.history-card p {
    margin-bottom: 8px;
    line-height: 1.6;
}

.history-card strong {
    color: #34495e;
}

details {
    margin-top: 15px;
    background-color: #f8f8f8;
    border: 1px solid #e9e9e9;
    border-radius: 5px;
}

details summary {
    padding: 10px 15px;
    cursor: pointer;
    font-weight: bold;
    color: #4a4a4a;
    outline: none;
}

details summary:hover {
    background-color: #f0f0f0;
}

details p, details ul {
    padding: 10px 15px 10px 30px;
    margin: 0;
    border-top: 1px solid #e9e9e9;
}

details ul {
    list-style-type: disc;
    padding-left: 40px;
}

details li {
    margin-bottom: 5px;
}

details pre {
    background-color: #eceff1;
    padding: 10px 15px;
    border-radius: 4px;
    overflow-x: auto;
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.9em;
    border-top: 1px solid #e9e9e9;
}

.evaluate-card {
    border-left: 5px solid #28a745;
}

.debug-card {
    border-left: 5px solid #dc3545;
}

.recommend-card {
    border-left: 5px solid #ffc107;
}

/* Add some spacing between record types */
.history-list > div:not(:last-child) {
    margin-bottom: 40px;
}
</style>
