// 班级成员角色的显示名称与徽标色调
const ROLE_NAMES = { teacher: '教师', ta: '助教', student: '学生' }
const ROLE_TONES = { teacher: 'primary', ta: 'success', student: 'neutral' }

export const getRoleName = (role) => ROLE_NAMES[role] || '学生'
export const getRoleTone = (role) => ROLE_TONES[role] || 'neutral'
