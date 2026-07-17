export type RepeatType = 'none' | 'daily' | 'weekly' | 'monthly' | 'yearly';
export type Priority = 'low' | 'medium' | 'high';
export type Status = 'pending' | 'in_progress' | 'completed' | 'cancelled';
export type ExecutionStatus = 'not_started' | 'running' | 'paused' | 'executed' | 'delayed' | 'skipped' | 'cancelled';
export type RecurrenceScope = 'this' | 'series';

export interface ScheduleItem {
  id: number;
  title: string;
  description: string;
  date: string;
  startTime: string;
  endTime: string;
  repeat: RepeatType;
  priority: Priority;
  status: Status;
  executionStatus: ExecutionStatus;
  actualStartAt: string;
  actualEndAt: string;
  executionMinutes: number;
  failureReason: string;
  category: string;
  seriesParentId: number;
  occurrenceDate: string;
  isVirtual: boolean;
  createdAt: string;
  updatedAt: string;
  hasTime: boolean;
}

export interface TodayInfo {
  date: string;
  weekday: string;
  pending: number;
  completed: number;
  inProgress: number;
}

export interface DayViewData {
  date: string;
  weekday: string;
  items: ScheduleItem[];
}

export interface WeekDayData {
  date: string;
  weekday: string;
  items: ScheduleItem[];
}

export interface WeekViewData {
  weekStart: string;
  weekEnd: string;
  weekNumber: number;
  days: WeekDayData[];
}

export interface MonthDayData {
  date: string;
  weekday: string;
  count: number;
  items: ScheduleItem[];
}

export interface MonthViewData {
  year: number;
  month: number;
  monthName: string;
  daysInMonth: number;
  firstWeekday: number;
  days: MonthDayData[];
  items: ScheduleItem[];
}

export interface MonthSummary {
  month: number;
  monthName: string;
  count: number;
  items: ScheduleItem[];
}

export interface YearViewData {
  year: number;
  months: MonthSummary[];
  totalCount: number;
  items: ScheduleItem[];
}

export interface StatsData {
  pending: number;
  inProgress: number;
  completed: number;
  cancelled: number;
  total: number;
  byCategory: Record<string, number>;
  execution?: Record<string, number>;
  byReason?: Record<string, number>;
  daily?: Array<{ date: string; plannedMinutes: number; actualMinutes: number; executed: number; skipped: number }>;
}

export type ViewName = 'day' | 'week' | 'month' | 'year' | 'search' | 'stats';

export interface ItemDraft {
  title: string;
  description: string;
  date: string;
  startTime: string;
  endTime: string;
  repeat: RepeatType;
  priority: Priority;
  status: Status;
  executionStatus?: ExecutionStatus;
  failureReason?: string;
  category: string;
}

declare global {
  interface Window {
    api?: (body: string) => Promise<string | unknown>;
  }
}

export {};
