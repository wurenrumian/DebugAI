<template>
  <div class="class-info-tab">
    <div v-if="classStore.loading" class="loading">
      加载中...
    </div>
    
    <div v-else-if="classStore.currentClass" class="class-info">
      <!-- 班级基本信息 -->
      <div class="info-section">
        <h4>班级信息</h4>
        <div class="info-item">
          <span class="label">班级名称：</span>
          <span class="value">{{ classStore.currentClass.name }}</span>
        </div>
        <div class="info-item">
          <span class="label">创建者：</span>
          <span class="value">{{ classStore.currentClass.creator_name || classStore.currentClass.creator_username || '未知' }}</span>
        </div>
        <div class="info-item">
          <span class="label">创建时间：</span>
          <span class="value">{{ formatDate(classStore.currentClass.created_at) }}</span>
        </div>
      </div>

      <!-- 成员列表 -->
      <div class="members-section">
        <h4>班级成员 ({{ members.length }})</h4>
        <div class="members-table">
          <table>
            <thead>
              <tr>
                <th>学号</th>
                <th>姓名</th>
                <th>角色</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members" :key="member.student_id">
                <td>{{ member.student_id }}</td>
                <td>{{ member.username }}</td>
                <td>
                  <span :class="['role-badge', getRoleClass(member.role)]">
                    {{ getRoleName(member.role) }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      请选择一个班级
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useClassStore } from '../../stores/class'

const classStore = useClassStore()

const members = computed(() => classStore.members)

const formatDate = (dateString) => {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const getRoleClass = (role) => {
  const roleMap = {
    'teacher': 'role-teacher',
    'ta': 'role-ta',
    'student': 'role-student'
  }
  return roleMap[role] || 'role-student'
}

const getRoleName = (role) => {
  const roleNameMap = {
    'teacher': '教师',
    'ta': '助教',
    'student': '学生'
  }
  return roleNameMap[role] || '学生'
}
</script>

<style scoped>
.class-info-tab {
  padding: 16px 0;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: #909399;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.info-section, .members-section {
  margin-bottom: 24px;
}

h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #303133;
  padding-bottom: 8px;
  border-bottom: 1px solid #ebeef5;
}

.info-item {
  display: flex;
  padding: 8px 0;
}

.info-item .label {
  width: 100px;
  color: #606266;
}

.info-item .value {
  color: #303133;
}

.members-table {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th, td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #ebeef5;
}

th {
  background-color: #f5f7fa;
  color: #606266;
  font-weight: 500;
}

td {
  color: #303133;
}

.role-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.role-teacher {
  background-color: #fdf6ec;
  color: #e6a23c;
}

.role-ta {
  background-color: #ecf5ff;
  color: #409eff;
}

.role-student {
  background-color: #f0f9ff;
  color: #909399;
}
</style>
