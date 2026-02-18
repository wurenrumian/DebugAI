<template>
  <div class="class-manage-tab">
    <div v-if="classStore.loading" class="loading">
      加载中...
    </div>
    
    <div v-else-if="classStore.currentClass" class="manage-content">
      <!-- 添加成员区域 -->
      <div class="add-member-section">
        <h4>添加成员</h4>
        <div class="add-form">
          <div class="form-item">
            <label>学号（多个用逗号分隔）：</label>
            <textarea 
              v-model="newMemberIds" 
              placeholder="请输入学号，多个学号用逗号分隔"
              rows="2"
            ></textarea>
          </div>
          <div class="form-item">
            <label>角色：</label>
            <select v-model="newMemberRole">
              <option value="student">学生</option>
              <option value="ta">助教</option>
              <option value="teacher">教师</option>
            </select>
          </div>
          <button 
            class="add-btn" 
            @click="addMembers"
            :disabled="!newMemberIds.trim() || classStore.loading"
          >
            添加成员
          </button>
        </div>
      </div>

      <!-- 成员列表区域 -->
      <div class="member-list-section">
        <h4>
          成员列表 
          <span class="selected-count" v-if="selectedMembers.length > 0">
            (已选择 {{ selectedMembers.length }} 人)
          </span>
        </h4>
        <div class="actions-bar">
          <button 
            class="remove-btn" 
            @click="removeSelectedMembers"
            :disabled="selectedMembers.length === 0 || classStore.loading"
          >
            移除选中成员
          </button>
        </div>
        
        <div class="members-table">
          <table>
            <thead>
              <tr>
                <th class="checkbox-col">
                  <input 
                    type="checkbox" 
                    :checked="allSelected"
                    :indeterminate="isIndeterminate"
                    @change="toggleSelectAll"
                  />
                </th>
                <th>学号</th>
                <th>姓名</th>
                <th>角色</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members" :key="member.student_id">
                <td class="checkbox-col">
                  <input 
                    type="checkbox" 
                    :value="member.student_id"
                    v-model="selectedMembers"
                    :disabled="member.is_creator"
                  />
                </td>
                <td>{{ member.student_id }}</td>
                <td>{{ member.username }}</td>
                <td>
                  <span :class="['role-badge', getRoleClass(member.role)]">
                    {{ getRoleName(member.role) }}
                  </span>
                </td>
                <td>
                  <button 
                    class="remove-single-btn"
                    @click="removeMember(member)"
                    :disabled="member.is_creator || classStore.loading"
                    :title="member.is_creator ? '创建者不可移除' : '移除'"
                  >
                    {{ member.is_creator ? '创建者' : '移除' }}
                  </button>
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
import { ref, computed, watch } from 'vue'
import { useClassStore } from '../../stores/class'

const classStore = useClassStore()

const newMemberIds = ref('')
const newMemberRole = ref('student')
const selectedMembers = ref([])

const members = computed(() => classStore.members)

const allSelected = computed(() => {
  const removableMembers = members.value.filter(m => !m.is_creator)
  return removableMembers.length > 0 && 
         removableMembers.every(m => selectedMembers.value.includes(m.student_id))
})

const isIndeterminate = computed(() => {
  const removableMembers = members.value.filter(m => !m.is_creator)
  const selectedCount = removableMembers.filter(m => selectedMembers.value.includes(m.student_id)).length
  return selectedCount > 0 && selectedCount < removableMembers.length
})

// 切换班级时清空选择
watch(() => classStore.currentClass?.id, () => {
  selectedMembers.value = []
  newMemberIds.value = ''
})

const toggleSelectAll = (e) => {
  const removableMembers = members.value.filter(m => !m.is_creator)
  if (e.target.checked) {
    selectedMembers.value = removableMembers.map(m => m.student_id)
  } else {
    selectedMembers.value = []
  }
}

const addMembers = async () => {
  if (!newMemberIds.value.trim() || !classStore.currentClass) return
  
  // 解析学号，支持逗号、中文逗号、空格分隔
  const ids = newMemberIds.value
    .split(/[,，\s]+/)
    .map(id => id.trim())
    .filter(id => id)
  
  if (ids.length === 0) return
  
  const result = await classStore.addMembers(
    classStore.currentClass.id, 
    ids, 
    newMemberRole.value
  )
  
  if (result.success) {
    newMemberIds.value = ''
  } else {
    alert(result.error || '添加成员失败')
  }
}

const removeSelectedMembers = async () => {
  if (selectedMembers.value.length === 0 || !classStore.currentClass) return
  
  if (!confirm(`确定要移除选中的 ${selectedMembers.value.length} 名成员吗？`)) {
    return
  }
  
  const result = await classStore.removeMembers(
    classStore.currentClass.id,
    selectedMembers.value
  )
  
  if (result.success) {
    selectedMembers.value = []
  } else {
    alert(result.error || '移除成员失败')
  }
}

const removeMember = async (member) => {
  if (member.is_creator || !classStore.currentClass) return
  
  if (!confirm(`确定要移除成员 ${member.username} (${member.student_id}) 吗？`)) {
    return
  }
  
  const result = await classStore.removeMembers(
    classStore.currentClass.id,
    [member.student_id]
  )
  
  if (!result.success) {
    alert(result.error || '移除成员失败')
  }
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
.class-manage-tab {
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

.manage-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #303133;
  padding-bottom: 8px;
  border-bottom: 1px solid #ebeef5;
}

.selected-count {
  font-size: 14px;
  font-weight: normal;
  color: #409eff;
}

.add-member-section {
  background-color: #f5f7fa;
  padding: 16px;
  border-radius: 8px;
}

.add-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-item label {
  font-size: 14px;
  color: #606266;
}

.form-item textarea,
.form-item select {
  padding: 8px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  font-family: inherit;
}

.form-item textarea:focus,
.form-item select:focus {
  outline: none;
  border-color: #409eff;
}

.add-btn {
  align-self: flex-start;
  padding: 8px 20px;
  background-color: #409eff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.add-btn:hover:not(:disabled) {
  background-color: #66b1ff;
}

.add-btn:disabled {
  background-color: #a0cfff;
  cursor: not-allowed;
}

.actions-bar {
  margin-bottom: 12px;
}

.remove-btn {
  padding: 6px 16px;
  background-color: #f56c6c;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.remove-btn:hover:not(:disabled) {
  background-color: #f78989;
}

.remove-btn:disabled {
  background-color: #fab6b6;
  cursor: not-allowed;
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

.checkbox-col {
  width: 40px;
  text-align: center;
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

.remove-single-btn {
  padding: 4px 12px;
  background-color: #f56c6c;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}

.remove-single-btn:hover:not(:disabled) {
  background-color: #f78989;
}

.remove-single-btn:disabled {
  background-color: #c8e0f0;
  color: #909399;
  cursor: not-allowed;
}
</style>
