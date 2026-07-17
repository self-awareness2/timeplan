import { api } from './api';
import type { ItemDraft, RecurrenceScope, ScheduleItem } from './types';

export interface ModalCallbacks {
  onSave: () => void;
  onDelete?: (id: number) => void;
  defaultStartTime?: string;
  defaultEndTime?: string;
}

const draftKey = 'chrona_plan_draft';
const durations = [15, 30, 60, 90, 120];
const defaultCategories = ['工作', '学习', '生活', '健康', '家庭', '财务'];

export function showItemModal(item: ScheduleItem | null, defaultDate: string, callbacks: ModalCallbacks): void {
  const isEdit = item !== null;
  const savedDraft = !isEdit ? loadDraft() : null;
  const draft: ItemDraft = item ? {
    title: item.title, description: item.description, date: item.date,
    startTime: item.hasTime ? item.startTime : '', endTime: item.hasTime ? item.endTime : '',
    repeat: item.repeat, priority: item.priority, status: item.status, category: item.category,
    executionStatus: item.executionStatus, failureReason: item.failureReason,
  } : savedDraft ?? {
    title: '', description: '', date: defaultDate, startTime: callbacks.defaultStartTime ?? '', endTime: callbacks.defaultEndTime ?? '',
    repeat: 'none', priority: 'medium', status: 'pending', category: '', executionStatus: 'not_started', failureReason: '',
  };
  const categoryIsDefault = defaultCategories.includes(draft.category) || draft.category === '';
  const isRecurringEdit = Boolean(item && (item.repeat !== 'none' || item.seriesParentId > 0));
  const defaultScope: RecurrenceScope = item?.isVirtual || (item?.seriesParentId ?? 0) > 0 ? 'this' : 'series';

  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.innerHTML = `
    <div class="modal plan-editor" role="dialog" aria-modal="true" aria-label="${isEdit ? '编辑计划' : '添加计划'}">
      <div class="modal-header"><div><h3>${isEdit ? '编辑计划' : '添加计划'}</h3><p>常用内容先填，其他设置可按需展开</p></div><button class="btn icon-btn modal-close" type="button" data-close aria-label="关闭">×</button></div>
      <div class="modal-body">
        <div class="form-group"><label>标题 *</label><input name="title" value="${esc(draft.title)}" placeholder="例如：复习数学、项目会议" required /></div>
        <div class="quick-time-block"><span>日期</span><div class="quick-time-list"><button type="button" class="chip-btn" data-date="today">今天</button><button type="button" class="chip-btn" data-date="tomorrow">明天</button><button type="button" class="chip-btn" data-date="monday">下周一</button></div></div>
        <div class="form-row"><div class="form-group"><label>日期 *</label><input name="date" type="date" value="${draft.date}" required /></div><div class="form-group"><label>分类</label><select name="categorySelect"><option value="">未分类</option>${defaultCategories.map((category) => option(category, category, categoryIsDefault ? draft.category : '')).join('')}<option value="__custom__"${categoryIsDefault ? '' : ' selected'}>自定义分类...</option></select><input name="customCategory" value="${categoryIsDefault ? '' : esc(draft.category)}" placeholder="输入自定义分类"${categoryIsDefault ? ' hidden' : ''} /></div></div>
        <div class="form-row"><div class="form-group"><label>开始时间</label><input name="startTime" type="time" step="300" value="${draft.startTime}" /></div><div class="form-group"><label>结束时间</label><input name="endTime" type="time" step="300" value="${draft.endTime}" /></div></div>
        <div class="quick-time-block"><span>时长</span><div class="quick-time-list">${durations.map((minutes) => `<button type="button" class="chip-btn" data-duration="${minutes}">${minutes < 60 ? `${minutes} 分钟` : `${minutes / 60} 小时`}</button>`).join('')}<button type="button" class="chip-btn subtle" data-all-day>全天</button></div></div>
        ${isRecurringEdit ? `<div class="form-group recurrence-scope"><label>应用范围</label><select name="recurrenceScope"><option value="this"${defaultScope === 'this' ? ' selected' : ''}>仅此日期</option><option value="series"${defaultScope === 'series' ? ' selected' : ''}>整个重复计划</option></select></div>` : ''}
        <div class="form-row"><div class="form-group"><label>优先级</label><select name="priority">${option('low', '低', draft.priority)}${option('medium', '中', draft.priority)}${option('high', '高', draft.priority)}</select></div><div class="form-group"><label>计划状态</label><select name="status">${option('pending', '待办', draft.status)}${option('in_progress', '进行中', draft.status)}${option('completed', '已完成', draft.status)}${option('cancelled', '已取消', draft.status)}</select></div></div>
        <details class="plan-more"><summary>更多设置</summary><div class="form-group"><label>描述</label><textarea name="description" rows="3" placeholder="地点、材料或备注">${esc(draft.description)}</textarea></div><div class="form-row"><div class="form-group"><label>重复</label><select name="repeat">${option('none', '不重复', draft.repeat)}${option('daily', '每天', draft.repeat)}${option('weekly', '每周', draft.repeat)}${option('monthly', '每月', draft.repeat)}${option('yearly', '每年', draft.repeat)}</select></div><div class="form-group"><label>执行结果</label><select name="executionStatus">${option('not_started', '暂不确定', draft.executionStatus ?? 'not_started')}${option('executed', '已完成', draft.executionStatus ?? '')}${option('skipped', '未完成', draft.executionStatus ?? '')}</select></div></div><div class="form-group"><label>未完成原因</label><input name="failureReason" value="${esc(draft.failureReason ?? '')}" placeholder="时间不足、临时有事、优先级变化" /></div></details>
      </div>
      <div class="modal-footer">${isEdit ? '<button class="btn btn-danger" type="button" data-delete>删除</button>' : '<span></span>'}<div class="modal-actions"><button class="btn" type="button" data-close>取消</button><button class="btn btn-accent" type="button" data-save>保存</button></div></div>
    </div>`;

  const close = () => { overlay.remove(); };
  const save = async () => {
    const form = readForm(overlay);
    const scope = recurrenceScope(overlay, defaultScope);
    const occurrenceDate = item?.occurrenceDate || item?.date || '';
    if (!form.title.trim()) return showError(overlay, '请填写计划标题');
    if (form.startTime !== '00:00' && form.endTime !== '00:00' && minutes(form.endTime) <= minutes(form.startTime)) return showError(overlay, '结束时间需要晚于开始时间');
    const conflicts = (await api.listDay(form.date)).items.filter((other) => other.id !== item?.id && overlaps(form, other));
    if (conflicts.length && !confirm(`与“${conflicts[0].title}”时间重叠，仍要保存吗？`)) return;
    try {
      if (isEdit && item) await api.updateItem(item.id, form, scope, occurrenceDate); else await api.addItem(form);
      localStorage.removeItem(draftKey); close(); callbacks.onSave();
    } catch (error) { showError(overlay, error instanceof Error ? error.message : '保存失败，请稍后重试'); }
  };

  overlay.querySelectorAll('[data-close]').forEach((node) => node.addEventListener('click', close));
  overlay.querySelector('[data-save]')?.addEventListener('click', () => void save());
  overlay.querySelector('[data-delete]')?.addEventListener('click', async () => { if (item && callbacks.onDelete && confirm('确定删除这条计划吗？')) { await api.deleteItem(item.id, recurrenceScope(overlay, defaultScope), item.occurrenceDate || item.date); close(); callbacks.onDelete(item.id); } });
  overlay.querySelectorAll<HTMLElement>('[data-date]').forEach((button) => button.addEventListener('click', () => { const input = overlay.querySelector<HTMLInputElement>('[name="date"]')!; input.value = quickDate(button.dataset.date ?? 'today'); }));
  overlay.querySelector<HTMLSelectElement>('[name="categorySelect"]')?.addEventListener('change', (event) => {
    const custom = overlay.querySelector<HTMLInputElement>('[name="customCategory"]')!;
    custom.hidden = (event.target as HTMLSelectElement).value !== '__custom__';
    if (!custom.hidden) custom.focus();
  });
  overlay.querySelectorAll<HTMLElement>('[data-duration]').forEach((button) => button.addEventListener('click', () => { const start = overlay.querySelector<HTMLInputElement>('[name="startTime"]')!; const end = overlay.querySelector<HTMLInputElement>('[name="endTime"]')!; if (!start.value) start.value = nearestTime(); end.value = addMinutes(start.value, Number(button.dataset.duration)); }));
  overlay.querySelector('[data-all-day]')?.addEventListener('click', () => { overlay.querySelector<HTMLInputElement>('[name="startTime"]')!.value = ''; overlay.querySelector<HTMLInputElement>('[name="endTime"]')!.value = ''; });
  overlay.addEventListener('input', () => { if (!isEdit) localStorage.setItem(draftKey, JSON.stringify(readForm(overlay))); });
  overlay.addEventListener('keydown', (event) => { if (event.key === 'Escape') close(); if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') void save(); });
  document.body.appendChild(overlay);
  void populateCategorySuggestions(overlay);
  overlay.querySelector<HTMLInputElement>('[name="title"]')?.focus();
}

async function populateCategorySuggestions(root: HTMLElement): Promise<void> {
  try {
    const select = root.querySelector<HTMLSelectElement>('[name="categorySelect"]');
    if (!select) return;
    const known = new Set(Array.from(select.options, (option) => option.value));
    const customOption = select.querySelector<HTMLOptionElement>('option[value="__custom__"]');
    for (const category of await api.categories()) {
      if (!known.has(category)) select.insertBefore(new Option(category, category), customOption);
    }
  } catch {
    // Default categories remain available when the suggestion request fails.
  }
}

function readForm(root: HTMLElement): ItemDraft { const q = (selector: string) => (root.querySelector(selector) as HTMLInputElement).value; const selectedCategory = q('[name="categorySelect"]'); return { title: q('[name="title"]'), description: q('[name="description"]'), date: q('[name="date"]'), startTime: q('[name="startTime"]') || '00:00', endTime: q('[name="endTime"]') || '00:00', category: selectedCategory === '__custom__' ? q('[name="customCategory"]').trim() : selectedCategory, priority: q('[name="priority"]') as ItemDraft['priority'], status: q('[name="status"]') as ItemDraft['status'], repeat: q('[name="repeat"]') as ItemDraft['repeat'], executionStatus: q('[name="executionStatus"]') as ItemDraft['executionStatus'], failureReason: q('[name="failureReason"]') }; }
function recurrenceScope(root: HTMLElement, fallbackScope: RecurrenceScope): RecurrenceScope { return (root.querySelector<HTMLSelectElement>('[name="recurrenceScope"]')?.value as RecurrenceScope | undefined) ?? fallbackScope; }
function loadDraft(): ItemDraft | null { try { return JSON.parse(localStorage.getItem(draftKey) ?? 'null') as ItemDraft | null; } catch { return null; } }
function quickDate(kind: string): string { const date = new Date(); if (kind === 'tomorrow') date.setDate(date.getDate() + 1); if (kind === 'monday') { const days = (8 - date.getDay()) % 7 || 7; date.setDate(date.getDate() + days); } return date.toISOString().slice(0, 10); }
function minutes(time: string): number { const [hour, minute] = time.split(':').map(Number); return hour * 60 + minute; }
function addMinutes(time: string, delta: number): string { const total = Math.min(1439, minutes(time) + delta); return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`; }
function nearestTime(): string { const now = new Date(); const total = Math.ceil((now.getHours() * 60 + now.getMinutes()) / 15) * 15; return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`; }
function overlaps(draft: ItemDraft, item: ScheduleItem): boolean { if (draft.startTime === '00:00' || item.startTime === '00:00') return false; return minutes(draft.startTime) < minutes(item.endTime) && minutes(draft.endTime) > minutes(item.startTime); }
function option(value: string, label: string, selected: string): string { return `<option value="${value}"${value === selected ? ' selected' : ''}>${label}</option>`; }
function esc(value: string): string { return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;'); }
function showError(root: HTMLElement, message: string): void { const existing = root.querySelector('.form-error'); if (existing) existing.remove(); const error = document.createElement('div'); error.className = 'form-error'; error.textContent = message; root.querySelector('.modal-body')?.prepend(error); }
