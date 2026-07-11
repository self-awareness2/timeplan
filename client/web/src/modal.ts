import { api } from './api';
import type { ItemDraft, ScheduleItem } from './types';

export interface ModalCallbacks {
  onSave: () => void;
  onDelete?: (id: number) => void;
  defaultStartTime?: string;
  defaultEndTime?: string;
}

const quickTimes = ['08:00', '09:00', '10:00', '12:00', '14:00', '18:00', '20:00', '22:00'];
const quickDurations = [
  { label: '+30 分钟', minutes: 30 },
  { label: '+1 小时', minutes: 60 },
  { label: '+1.5 小时', minutes: 90 },
  { label: '+2 小时', minutes: 120 },
  { label: '+3 小时', minutes: 180 },
];

function emptyDraft(date: string): ItemDraft {
  return {
    title: '',
    description: '',
    date,
    startTime: '',
    endTime: '',
    repeat: 'none',
    priority: 'medium',
    status: 'pending',
    category: '',
  };
}

export function showItemModal(
  item: ScheduleItem | null,
  defaultDate: string,
  callbacks: ModalCallbacks
): void {
  const isEdit = item !== null;
  const draft = item
    ? {
        title: item.title,
        description: item.description,
        date: item.date,
        startTime: item.startTime === '00:00' && !item.hasTime ? '' : item.startTime,
        endTime: item.endTime === '00:00' ? '' : item.endTime,
        repeat: item.repeat,
        priority: item.priority,
        status: item.status,
        category: item.category,
      }
    : {
        ...emptyDraft(defaultDate),
        startTime: callbacks.defaultStartTime ?? '',
        endTime: callbacks.defaultEndTime ?? '',
      };

  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.innerHTML = `
    <div class="modal">
      <div class="modal-header">
        <div>
          <h3>${isEdit ? '编辑计划' : '添加计划'}</h3>
          <p>${draft.date}${draft.startTime ? ` · ${draft.startTime}` : ''}</p>
        </div>
        <button type="button" class="btn icon-btn modal-close" data-close aria-label="关闭">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>标题 *</label>
          <input name="title" value="${esc(draft.title)}" placeholder="例如：复习数学、健身、项目会议" required />
        </div>
        <div class="form-group">
          <label>描述</label>
          <textarea name="description" rows="3" placeholder="补充地点、材料或提醒">${esc(draft.description)}</textarea>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>日期 *</label>
            <input name="date" type="date" value="${draft.date}" required />
          </div>
          <div class="form-group">
            <label>分类</label>
            <input name="category" value="${esc(draft.category)}" placeholder="工作、学习、生活..." />
          </div>
        </div>
        <div class="time-editor">
          <div class="form-row">
            <div class="form-group">
              <label>开始时间</label>
              <input name="startTime" type="time" step="300" value="${draft.startTime}" />
            </div>
            <div class="form-group">
              <label>结束时间</label>
              <input name="endTime" type="time" step="300" value="${draft.endTime}" />
            </div>
          </div>
          <div class="quick-time-block">
            <span>开始</span>
            <div class="quick-time-list">
              ${quickTimes.map((time) => `<button type="button" class="chip-btn" data-start-time="${time}">${time}</button>`).join('')}
            </div>
          </div>
          <div class="quick-time-block">
            <span>时长</span>
            <div class="quick-time-list">
              ${quickDurations.map((d) => `<button type="button" class="chip-btn" data-duration="${d.minutes}">${d.label}</button>`).join('')}
              <button type="button" class="chip-btn subtle" data-clear-time>全天/无时间</button>
            </div>
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>重复</label>
            <select name="repeat">
              ${opt('none', '不重复', draft.repeat)}
              ${opt('daily', '每天', draft.repeat)}
              ${opt('weekly', '每周', draft.repeat)}
              ${opt('monthly', '每月', draft.repeat)}
              ${opt('yearly', '每年', draft.repeat)}
            </select>
          </div>
          <div class="form-group">
            <label>优先级</label>
            <select name="priority">
              ${opt('low', '低', draft.priority)}
              ${opt('medium', '中', draft.priority)}
              ${opt('high', '高', draft.priority)}
            </select>
          </div>
        </div>
        <div class="form-group">
          <label>状态</label>
          <select name="status">
            ${opt('pending', '待办', draft.status)}
            ${opt('in_progress', '进行中', draft.status)}
            ${opt('completed', '已完成', draft.status)}
            ${opt('cancelled', '已取消', draft.status)}
          </select>
        </div>
      </div>
      <div class="modal-footer">
        ${isEdit ? '<button type="button" class="btn btn-danger" data-delete>删除</button>' : '<span></span>'}
        <div class="modal-actions">
          <button type="button" class="btn" data-close>取消</button>
          <button type="button" class="btn btn-accent" data-save>保存</button>
        </div>
      </div>
    </div>
  `;

  bindTimeShortcuts(overlay);

  const close = () => overlay.remove();
  overlay.querySelectorAll('[data-close]').forEach((el) => el.addEventListener('click', close));
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) close();
  });

  overlay.querySelector('[data-save]')?.addEventListener('click', async () => {
    const form = readForm(overlay);
    if (!form.title.trim()) {
      alert('请填写标题');
      return;
    }
    try {
      if (isEdit && item) await api.updateItem(item.id, form);
      else await api.addItem(form);
      close();
      callbacks.onSave();
    } catch (err) {
      alert(String(err));
    }
  });

  overlay.querySelector('[data-delete]')?.addEventListener('click', async () => {
    if (!item || !callbacks.onDelete) return;
    if (!confirm('确定删除这个计划？')) return;
    try {
      await api.deleteItem(item.id);
      close();
      callbacks.onDelete(item.id);
    } catch (err) {
      alert(String(err));
    }
  });

  document.body.appendChild(overlay);
  (overlay.querySelector('input[name="title"]') as HTMLInputElement)?.focus();
}

function bindTimeShortcuts(overlay: HTMLElement): void {
  const start = overlay.querySelector('input[name="startTime"]') as HTMLInputElement;
  const end = overlay.querySelector('input[name="endTime"]') as HTMLInputElement;

  overlay.querySelectorAll<HTMLElement>('[data-start-time]').forEach((button) => {
    button.addEventListener('click', () => {
      start.value = button.dataset.startTime ?? '';
      if (!end.value) end.value = addMinutes(start.value, 60);
      start.dispatchEvent(new Event('change'));
    });
  });

  overlay.querySelectorAll<HTMLElement>('[data-duration]').forEach((button) => {
    button.addEventListener('click', () => {
      if (!start.value) start.value = nearestTime();
      end.value = addMinutes(start.value, Number(button.dataset.duration ?? 60));
      end.dispatchEvent(new Event('change'));
    });
  });

  overlay.querySelector('[data-clear-time]')?.addEventListener('click', () => {
    start.value = '';
    end.value = '';
  });

  start.addEventListener('change', () => {
    if (start.value && end.value && minutes(end.value) <= minutes(start.value)) {
      end.value = addMinutes(start.value, 60);
    }
  });
}

function readForm(overlay: HTMLElement): ItemDraft {
  const q = (sel: string) => (overlay.querySelector(sel) as HTMLInputElement).value;
  return {
    title: q('input[name="title"]'),
    description: q('textarea[name="description"]'),
    date: q('input[name="date"]'),
    startTime: q('input[name="startTime"]') || '00:00',
    endTime: q('input[name="endTime"]') || '00:00',
    repeat: q('select[name="repeat"]') as ItemDraft['repeat'],
    priority: q('select[name="priority"]') as ItemDraft['priority'],
    status: q('select[name="status"]') as ItemDraft['status'],
    category: q('input[name="category"]'),
  };
}

function minutes(time: string): number {
  const [h, m] = time.split(':').map(Number);
  return h * 60 + m;
}

function addMinutes(time: string, delta: number): string {
  const total = Math.min(23 * 60 + 55, Math.max(0, minutes(time) + delta));
  const h = Math.floor(total / 60);
  const m = total % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function nearestTime(): string {
  const now = new Date();
  const rounded = Math.ceil((now.getHours() * 60 + now.getMinutes()) / 30) * 30;
  const h = Math.min(23, Math.floor(rounded / 60));
  const m = h === 23 ? Math.min(30, rounded % 60) : rounded % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function esc(s: string): string {
  return s.replace(/"/g, '&quot;').replace(/</g, '&lt;');
}

function opt(value: string, label: string, selected: string): string {
  return `<option value="${value}"${value === selected ? ' selected' : ''}>${label}</option>`;
}
