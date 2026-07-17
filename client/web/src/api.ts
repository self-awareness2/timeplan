import type {
  DayViewData,
  ItemDraft,
  MonthViewData,
  ScheduleItem,
  StatsData,
  TodayInfo,
  WeekViewData,
  YearViewData,
} from './types';
import type { RecurrenceScope } from './types';

interface ApiResponse<T> {
  ok: boolean;
  data?: T;
  error?: string;
}

export interface AuthSession {
  token: string;
  user: { username: string };
}

const tokenKey = 'chrona_token';
const authExpiredEvent = 'time-planner-auth-expired';

function apiBasePath(): string {
  return window.location.pathname === '/timeplan' || window.location.pathname.startsWith('/timeplan/')
    ? '/timeplan'
    : '';
}

function isDesktopBridge(): boolean {
  return false;
}

function authHeaders(): HeadersInit {
  const token = localStorage.getItem(tokenKey);
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function expireSession(): void {
  localStorage.removeItem(tokenKey);
  window.dispatchEvent(new CustomEvent(authExpiredEvent));
}

function handleAuthError(status: number, error?: string): void {
  if (status === 401 || error === '登录已失效' || error === '请先登录') {
    expireSession();
  }
}

async function call<T>(action: string, params?: object): Promise<T> {
  const body = JSON.stringify({ action, params: params ?? {} });
  const res = await fetch(`${apiBasePath()}/api`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body,
  });
  const parsed = JSON.parse(await res.text()) as ApiResponse<T>;
  handleAuthError(res.status, parsed.error);
  if (!parsed.ok) {
    throw new Error(parsed.error ?? 'unknown error');
  }
  return parsed.data as T;
}

async function authCall<T>(path: string, body?: object): Promise<T> {
  const res = await fetch(`${apiBasePath()}${path}`, {
    method: body ? 'POST' : 'GET',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: body ? JSON.stringify(body) : undefined,
  });
  const parsed = await res.json() as ApiResponse<T>;
  handleAuthError(res.status, parsed.error);
  if (!parsed.ok) {
    throw new Error(parsed.error ?? 'request failed');
  }
  return parsed.data as T;
}

async function downloadExport(): Promise<void> {
  const res = await fetch(`${apiBasePath()}/api/export`, {
    credentials: 'include',
    headers: authHeaders(),
  });
  if (!res.ok) {
    handleAuthError(res.status);
    throw new Error('export failed');
  }
  const blob = await res.blob();
  const link = document.createElement('a');
  const url = URL.createObjectURL(blob);
  link.href = url;
  link.download = `chrona-export-${todayString()}.json`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

export const api = {
  isDesktop: isDesktopBridge,
  authExpiredEvent,
  hasToken: () => Boolean(localStorage.getItem(tokenKey)),
  login: async (username: string, password: string) => {
    const session = await authCall<AuthSession>('/auth/login', { username, password });
    localStorage.setItem(tokenKey, session.token);
    return session.user;
  },
  register: async (username: string, password: string) => {
    const session = await authCall<AuthSession>('/auth/register', { username, password });
    localStorage.setItem(tokenKey, session.token);
    return session.user;
  },
  logout: () => localStorage.removeItem(tokenKey),
  me: () => authCall<{ user: { username: string } }>('/auth/me'),
  getToday: () => call<TodayInfo>('getToday'),
  listDay: (date: string) => call<DayViewData>('listDay', { date }),
  listWeek: (date: string) => call<WeekViewData>('listWeek', { date }),
  listMonth: (year: number, month: number) => call<MonthViewData>('listMonth', { year, month }),
  listYear: (year: number) => call<YearViewData>('listYear', { year }),
  getItem: (id: number) => call<ScheduleItem>('getItem', { id }),
  addItem: (item: ItemDraft) => call<{ id: number; item: ScheduleItem }>('addItem', { item }),
  updateItem: (id: number, item: ItemDraft, scope: RecurrenceScope = 'series', occurrenceDate = '') => call<ScheduleItem>('updateItem', { id, item, scope, occurrenceDate }),
  deleteItem: (id: number, scope: RecurrenceScope = 'series', occurrenceDate = '') => call<{ id: number }>('deleteItem', { id, scope, occurrenceDate }),
  setExecution: (id: number, executionStatus: string, failureReason = '', occurrenceDate = '') => call<ScheduleItem>('setExecution', { id, executionStatus, failureReason, occurrenceDate }),
  search: (keyword: string) => call<{ keyword: string; items: ScheduleItem[] }>('search', { keyword }),
  stats: () => call<StatsData>('stats'),
  categories: () => call<string[]>('categories'),
  downloadExport,
};

export function todayString(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function parseDateParts(date: string): { year: number; month: number; day: number } {
  const [y, m, d] = date.split('-').map(Number);
  return { year: y, month: m, day: d };
}

export function addDays(date: string, delta: number): string {
  const d = new Date(date + 'T12:00:00');
  d.setDate(d.getDate() + delta);
  return d.toISOString().slice(0, 10);
}

export function priorityLabel(p: string): string {
  return ({ low: '低', medium: '中', high: '高' } as Record<string, string>)[p] ?? p;
}

export function statusLabel(s: string): string {
  return ({
    pending: '待办',
    in_progress: '进行中',
    completed: '已完成',
    cancelled: '已取消',
  } as Record<string, string>)[s] ?? s;
}

export function repeatLabel(r: string): string {
  return ({
    none: '不重复',
    daily: '每天',
    weekly: '每周',
    monthly: '每月',
    yearly: '每年',
  } as Record<string, string>)[r] ?? r;
}
