<template>
  <div class="class-manage-tab">
    <AppSpinner v-if="classStore.loading" block label="加载中..." />

    <div v-else-if="classStore.currentClass" class="manage-content">
      <!-- 添加成员区域 -->
      <section class="add-member-section" aria-labelledby="add-member-title">
        <h3 id="add-member-title" class="section-title">添加成员</h3>
        <form class="add-form" @submit.prevent="addMembers">
          <div class="field field-ids">
            <label class="field-label" for="new-member-ids">学号</label>
            <textarea
              id="new-member-ids"
              v-model="newMemberIds"
              placeholder="请输入学号，多个学号用逗号分隔"
              rows="2"
              aria-describedby="new-member-ids-hint"
            ></textarea>
            <p id="new-member-ids-hint" class="field-hint">多个学号可用逗号、中文逗号或空格分隔</p>
          </div>
          <div class="field field-role">
            <label class="field-label" for="new-member-role">角色</label>
            <select id="new-member-role" v-model="newMemberRole">
              <option value="student">学生</option>
              <option value="ta">助教</option>
              <option value="teacher">教师</option>
            </select>
          </div>
          <AppButton
            type="submit"
            variant="primary"
            icon="plus"
            class="add-btn"
            :disabled="!newMemberIds.trim() || classStore.loading"
          >
            添加成员
          </AppButton>
        </form>
      </section>

      <!-- 成员列表区域 -->
      <section aria-labelledby="member-list-title">
        <div class="list-header">
          <h3 id="member-list-title" class="section-title">
            成员列表
            <span v-if="selectedMembers.length > 0" class="selected-count">
              已选择 {{ selectedMembers.length }} 人
            </span>
          </h3>
          <AppButton
            variant="danger-ghost"
            size="sm"
            icon="trash"
            :disabled="selectedMembers.length === 0 || classStore.loading"
            @click="removeSelectedMembers"
          >
            移除选中成员
          </AppButton>
        </div>

        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th scope="col" class="checkbox-col">
                  <input
                    type="checkbox"
                    aria-label="全选可移除的成员"
                    :checked="allSelected"
                    :indeterminate="isIndeterminate"
                    @change="toggleSelectAll"
                  />
                </th>
                <th scope="col">学号</th>
                <th scope="col">姓名</th>
                <th scope="col">角色</th>
                <th scope="col" class="action-col">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members" :key="member.student_id">
                <td class="checkbox-col">
                  <input
                    type="checkbox"
                    :aria-label="`选择 ${member.username}（${member.student_id}）`"
                    :value="member.student_id"
                    v-model="selectedMembers"
                    :disabled="member.is_creator"
                  />
                </td>
                <td class="mono">{{ member.student_id }}</td>
                <td>{{ member.username }}</td>
                <td>
                  <AppBadge :tone="getRoleTone(member.role)" size="sm">
                    {{ getRoleName(member.role) }}
                  </AppBadge>
                </td>
                <td class="action-col">
                  <span v-if="member.is_creator" class="creator-label" title="创建者不可移除">创建者</span>
                  <AppButton
                    v-else
                    variant="ghost"
                    size="sm"
                    class="remove-single-btn"
                    :aria-label="`移除 ${member.username}（${member.student_id}）`"
                    :disabled="classStore.loading"
                    @click="removeMember(member)"
                  >
                    移除
                  </AppButton>
                </td>
              </tr>
              <tr v-if="members.length === 0">
                <td colspan="5" class="empty-cell">暂无成员</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <p v-else class="empty-state">请选择一个班级</p>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useClassStore } from '../../stores/class'
import { toast } from '../../utils/toast'
import AppBadge from '../ui/AppBadge.vue'
import AppButton from '../ui/AppButton.vue'
import AppSpinner from '../ui/AppSpinner.vue'
import { getRoleName, getRoleTone } from './roles'

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
    toast.error(result.error || '添加成员失败')
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
    toast.error(result.error || '移除成员失败')
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
    toast.error(result.error || '移除成员失败')
  }
}

</script>

<style scoped>
.manage-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--fs-md);
}

.add-member-section {
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.add-member-section .section-title {
  margin-bottom: var(--space-3);
}

.add-form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 140px auto;
  gap: var(--space-3);
  align-items: start;
}

.add-form .field + .field {
  margin-top: 0;
}

.add-btn {
  margin-top: 26px;
}

.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.selected-count {
  font-size: var(--fs-sm);
  font-weight: var(--fw-normal);
  color: var(--color-primary-text);
}

.checkbox-col {
  width: 44px;
  text-align: center !important;
}

.action-col {
  width: 96px;
}

.mono {
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
}

.remove-single-btn {
  color: var(--color-danger);
}

.remove-single-btn:hover:not(:disabled) {
  color: var(--color-danger);
  background: var(--color-danger-soft);
}

.creator-label {
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.empty-cell,
.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--color-text-muted);
}

@media (max-width: 640px) {
  .add-member-section {
    padding: var(--space-4);
  }

  .add-form {
    grid-template-columns: minmax(0, 1fr);
  }

  .add-btn {
    margin-top: 0;
  }
}
</style>
