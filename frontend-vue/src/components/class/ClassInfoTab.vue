<template>
  <div class="class-info-tab">
    <AppSpinner v-if="classStore.loading" block label="加载中..." />

    <div v-else-if="classStore.currentClass" class="class-info">
      <!-- 班级基本信息 -->
      <section class="info-section" aria-labelledby="class-info-title">
        <h3 id="class-info-title" class="section-title">班级信息</h3>
        <dl class="info-grid">
          <div class="info-item">
            <dt>班级名称</dt>
            <dd>{{ classStore.currentClass.name }}</dd>
          </div>
          <div class="info-item">
            <dt>创建者</dt>
            <dd>{{ classStore.currentClass.creator_name || classStore.currentClass.creator_username || '未知' }}</dd>
          </div>
          <div class="info-item">
            <dt>创建时间</dt>
            <dd>{{ formatDate(classStore.currentClass.created_at) }}</dd>
          </div>
          <div class="info-item">
            <dt>成员人数</dt>
            <dd>{{ members.length }}</dd>
          </div>
        </dl>
      </section>

      <!-- 成员列表 -->
      <section class="members-section" aria-labelledby="class-members-title">
        <h3 id="class-members-title" class="section-title">
          班级成员
          <span class="count">{{ members.length }}</span>
        </h3>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th scope="col">学号</th>
                <th scope="col">姓名</th>
                <th scope="col">角色</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members" :key="member.student_id">
                <td class="mono">{{ member.student_id }}</td>
                <td>{{ member.username }}</td>
                <td>
                  <AppBadge :tone="getRoleTone(member.role)" size="sm">
                    {{ getRoleName(member.role) }}
                  </AppBadge>
                </td>
              </tr>
              <tr v-if="members.length === 0">
                <td colspan="3" class="empty-cell">暂无成员</td>
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
import { computed } from 'vue'
import { useClassStore } from '../../stores/class'
import AppBadge from '../ui/AppBadge.vue'
import AppSpinner from '../ui/AppSpinner.vue'
import { getRoleName, getRoleTone } from './roles'

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

</script>

<style scoped>
.class-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  font-size: var(--fs-md);
}

.count {
  padding: 0 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  line-height: 18px;
  color: var(--color-text-muted);
  background: var(--color-surface-3);
  border-radius: var(--radius-full);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
}

.info-item {
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.info-item dt {
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

.info-item dd {
  margin-top: 2px;
  font-weight: var(--fw-medium);
  word-break: break-word;
}

.mono {
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
}

.empty-cell,
.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--color-text-muted);
}
</style>
