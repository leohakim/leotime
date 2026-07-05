import { sortTasksByNewest } from './taskSort';

export type LayoutMode = 'solid' | 'minimal' | 'compact';
export type Locale = 'es' | 'en';

export type User = {
  id: string;
  email: string;
  name: string;
  locale: Locale;
  layoutMode: LayoutMode;
};

export type SessionResponse = {
  authenticated: boolean;
  user: User | null;
};

export type Overview = {
  clientsTotal: number;
  projectsTotal: number;
  tasksTotal: number;
  tagsTotal: number;
  timeEntriesTotal: number;
  invoicesTotal: number;
  openTimers: number;
};

export type Client = {
  id: string;
  name: string;
  email: string;
  taxId: string;
  billingAddress: string;
  defaultCurrency: string;
  defaultHourlyRateMinor: number;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
};

export type ClientInput = {
  name: string;
  email: string;
  taxId: string;
  billingAddress: string;
  defaultCurrency: string;
  defaultHourlyRateMinor: number;
};

export type ClientsResponse = {
  clients: Client[];
};

export type Project = {
  id: string;
  clientId: string;
  clientName: string;
  name: string;
  color: string;
  defaultHourlyRateMinor: number | null;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
};

export type ProjectInput = {
  clientId: string;
  name: string;
  color: string;
  defaultHourlyRateMinor: number | null;
};

export type ProjectsResponse = {
  projects: Project[];
};

export type Task = {
  id: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  name: string;
  billable: boolean;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
};

export type TaskInput = {
  projectId: string;
  name: string;
  billable: boolean;
};

export type TasksResponse = {
  tasks: Task[];
};

export type Tag = {
  id: string;
  name: string;
  color: string;
  createdAt: string;
  updatedAt: string;
};

export type TagInput = {
  name: string;
  color: string;
};

export type TagsResponse = {
  tags: Tag[];
};

export type TimeEntryTag = {
  id: string;
  name: string;
  color: string;
};

export type TimeEntry = {
  id: string;
  clientId: string;
  clientName: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  taskId: string;
  taskName: string;
  description: string;
  startedAt: string;
  endedAt: string;
  durationSeconds: number;
  billable: boolean;
  overlapWarning: boolean;
  source: string;
  tags: TimeEntryTag[];
  createdAt: string;
  updatedAt: string;
};

export type TimeEntryInput = {
  clientId: string;
  projectId: string;
  taskId: string;
  tagIds: string[];
  description: string;
  startedAt: string;
  endedAt: string;
  billable: boolean;
};

export type TimeEntriesResponse = {
  timeEntries: TimeEntry[];
};

export type TimerStartInput = {
  clientId: string;
  projectId: string;
  taskId: string;
  tagIds: string[];
  description: string;
  startedAt?: string;
  billable: boolean;
};

export type TimersResponse = {
  timers: TimeEntry[];
};

export type TimeReportGroupBy = 'day' | 'client' | 'project' | 'task';

export type TimeReportParams = {
  billableOnly?: boolean;
  from: string;
  groupBy?: TimeReportGroupBy;
  includeTimestamps?: boolean;
  to: string;
};

export type TimeReportGroup = {
  key: string;
  label: string;
  totalSeconds: number;
  entryCount: number;
};

export type TimeReport = {
  from: string;
  to: string;
  groupBy: TimeReportGroupBy;
  includeTimestamps: boolean;
  billableOnly: boolean;
  totalSeconds: number;
  entryCount: number;
  groups?: TimeReportGroup[];
  entries?: TimeEntry[];
};

export type DashboardRecentEntry = {
  id: string;
  clientId: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  taskId: string;
  description: string;
  startedAt: string;
  durationSeconds: number;
  billable: boolean;
};

export type DashboardDaySummary = {
  date: string;
  label: string;
  totalSeconds: number;
};

export type DashboardHeatmapDay = {
  date: string;
  totalSeconds: number;
  level: number;
  inMonth: boolean;
};

export type DashboardWeekDay = {
  date: string;
  weekday: string;
  totalSeconds: number;
};

export type DashboardProjectShare = {
  projectId: string;
  projectName: string;
  projectColor: string;
  totalSeconds: number;
};

export type DashboardStats = {
  activityMonth: string;
  recentEntries: DashboardRecentEntry[];
  lastSevenDays: DashboardDaySummary[];
  activityHeatmap: DashboardHeatmapDay[];
  weekDays: DashboardWeekDay[];
  weekSpentSeconds: number;
  weekBillableSeconds: number;
  weekBillableMinor: number;
  weekCurrency: string;
  projectBreakdown: DashboardProjectShare[];
};

export type InvoiceStatus = 'draft' | 'issued' | 'paid' | 'cancelled';

export type InvoiceLine = {
  id: string;
  timeEntryId: string;
  description: string;
  quantityMinutes: number;
  unitRateMinor: number;
  subtotalMinor: number;
  taxRateBasisPoints: number;
  createdAt: string;
};

export type Invoice = {
  id: string;
  clientId: string;
  invoiceNumber: string;
  status: InvoiceStatus;
  currency: string;
  issuedAt: string;
  dueAt: string;
  sellerName: string;
  sellerTaxId: string;
  sellerAddress: string;
  clientName: string;
  clientTaxId: string;
  clientAddress: string;
  subtotalMinor: number;
  taxMinor: number;
  withholdingMinor: number;
  totalMinor: number;
  notes: string;
  lines: InvoiceLine[];
  createdAt: string;
  updatedAt: string;
};

export type InvoicesResponse = {
  invoices: Invoice[];
};

export type InvoiceDraftFromTimeInput = {
  clientId: string;
  from: string;
  to: string;
  sellerName?: string;
  sellerTaxId?: string;
  sellerAddress?: string;
  taxRateBasisPoints?: number;
  withholdingMinor?: number;
  notes?: string;
  dueAt?: string;
};

export type InvoiceUpdateInput = {
  dueAt?: string;
  issuedAt?: string;
  sellerName?: string;
  sellerTaxId?: string;
  sellerAddress?: string;
  clientName?: string;
  clientTaxId?: string;
  clientAddress?: string;
  withholdingMinor?: number;
  notes?: string;
  taxRateBasisPoints?: number;
};

export async function fetchSession(): Promise<SessionResponse> {
  return apiGet('/api/v1/session');
}

export async function fetchOverview(): Promise<Overview> {
  return apiGet('/api/v1/overview');
}

export async function fetchDashboardStats(activityMonth?: string): Promise<DashboardStats> {
  const query = activityMonth ? `?activityMonth=${encodeURIComponent(activityMonth)}` : '';
  return apiGet(`/api/v1/dashboard/stats${query}`);
}

export async function fetchClients(): Promise<ClientsResponse> {
  return apiGet('/api/v1/clients');
}

export async function fetchProjects(): Promise<ProjectsResponse> {
  return apiGet('/api/v1/projects');
}

export async function fetchTasks(): Promise<TasksResponse> {
  const response = await apiGet<TasksResponse>('/api/v1/tasks');
  return {
    tasks: sortTasksByNewest(response.tasks),
  };
}

export async function fetchTags(): Promise<TagsResponse> {
  return apiGet('/api/v1/tags');
}

export type TimeEntryListParams = {
  clientId?: string;
  from?: string;
  projectId?: string;
  taskId?: string;
  to?: string;
};

export async function fetchTimeEntries(params?: TimeEntryListParams): Promise<TimeEntriesResponse> {
  const search = new URLSearchParams();
  if (params?.from) {
    search.set('from', params.from);
  }
  if (params?.to) {
    search.set('to', params.to);
  }
  if (params?.clientId) {
    search.set('clientId', params.clientId);
  }
  if (params?.projectId) {
    search.set('projectId', params.projectId);
  }
  if (params?.taskId) {
    search.set('taskId', params.taskId);
  }

  const query = search.toString();
  return apiGet(`/api/v1/time-entries${query ? `?${query}` : ''}`);
}

export async function fetchTimers(): Promise<TimersResponse> {
  return apiGet('/api/v1/timers');
}

export async function startTimer(input: TimerStartInput): Promise<TimeEntry> {
  return apiJSON('/api/v1/timers', 'POST', input);
}

export async function stopTimer(timeEntryId: string): Promise<TimeEntry> {
  const response = await fetch(`/api/v1/timers/${timeEntryId}/stop`, {
    method: 'POST',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }

  return response.json();
}

export async function updateTimer(timeEntryId: string, input: TimerStartInput): Promise<TimeEntry> {
  return apiJSON(`/api/v1/timers/${timeEntryId}`, 'PATCH', input);
}

export async function createClient(input: ClientInput): Promise<Client> {
  return apiJSON('/api/v1/clients', 'POST', input);
}

export async function updateClient(clientId: string, input: ClientInput): Promise<Client> {
  return apiJSON(`/api/v1/clients/${clientId}`, 'PATCH', input);
}

export async function archiveClient(clientId: string): Promise<void> {
  const response = await fetch(`/api/v1/clients/${clientId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function createProject(input: ProjectInput): Promise<Project> {
  return apiJSON('/api/v1/projects', 'POST', input);
}

export async function updateProject(projectId: string, input: ProjectInput): Promise<Project> {
  return apiJSON(`/api/v1/projects/${projectId}`, 'PATCH', input);
}

export async function archiveProject(projectId: string): Promise<void> {
  const response = await fetch(`/api/v1/projects/${projectId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function createTask(input: TaskInput): Promise<Task> {
  return apiJSON('/api/v1/tasks', 'POST', input);
}

export async function updateTask(taskId: string, input: TaskInput): Promise<Task> {
  return apiJSON(`/api/v1/tasks/${taskId}`, 'PATCH', input);
}

export async function archiveTask(taskId: string): Promise<void> {
  const response = await fetch(`/api/v1/tasks/${taskId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function createTag(input: TagInput): Promise<Tag> {
  return apiJSON('/api/v1/tags', 'POST', input);
}

export async function updateTag(tagId: string, input: TagInput): Promise<Tag> {
  return apiJSON(`/api/v1/tags/${tagId}`, 'PATCH', input);
}

export async function deleteTag(tagId: string): Promise<void> {
  const response = await fetch(`/api/v1/tags/${tagId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function createTimeEntry(input: TimeEntryInput): Promise<TimeEntry> {
  return apiJSON('/api/v1/time-entries', 'POST', input);
}

export async function updateTimeEntry(timeEntryId: string, input: TimeEntryInput): Promise<TimeEntry> {
  return apiJSON(`/api/v1/time-entries/${timeEntryId}`, 'PATCH', input);
}

export async function deleteTimeEntry(timeEntryId: string): Promise<void> {
  const response = await fetch(`/api/v1/time-entries/${timeEntryId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function login(email: string, password: string): Promise<SessionResponse> {
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    throw new Error('login_failed');
  }

  return response.json();
}

export async function logout(): Promise<void> {
  const response = await fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error('logout_failed');
  }
}

async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }

  return response.json();
}

async function apiJSON<T>(path: string, method: 'POST' | 'PATCH', body: unknown): Promise<T> {
  const response = await fetch(path, {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }

  return response.json();
}

function buildTimeReportSearch(params: TimeReportParams & { format?: 'csv' | 'json' }) {
  const search = new URLSearchParams();
  search.set('from', params.from);
  search.set('to', params.to);
  if (params.groupBy) {
    search.set('groupBy', params.groupBy);
  }
  if (params.includeTimestamps) {
    search.set('includeTimestamps', 'true');
  }
  if (params.billableOnly) {
    search.set('billableOnly', 'true');
  }
  if (params.format) {
    search.set('format', params.format);
  }
  return search.toString();
}

export async function fetchTimeReport(params: TimeReportParams): Promise<TimeReport> {
  return apiGet(`/api/v1/reports/time?${buildTimeReportSearch(params)}`);
}

export async function downloadTimeReportExport(params: TimeReportParams, format: 'csv' | 'json'): Promise<Blob> {
  const response = await fetch(`/api/v1/reports/time/export?${buildTimeReportSearch({ ...params, format })}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }

  return response.blob();
}

export async function fetchInvoices(): Promise<InvoicesResponse> {
  return apiGet('/api/v1/invoices');
}

export async function fetchInvoice(invoiceId: string): Promise<Invoice> {
  return apiGet(`/api/v1/invoices/${invoiceId}`);
}

export async function createInvoiceDraftFromTime(input: InvoiceDraftFromTimeInput): Promise<Invoice> {
  return apiJSON('/api/v1/invoices/draft-from-time', 'POST', input);
}

export async function updateInvoice(invoiceId: string, input: InvoiceUpdateInput): Promise<Invoice> {
  return apiJSON(`/api/v1/invoices/${invoiceId}`, 'PATCH', input);
}

export async function updateInvoiceStatus(invoiceId: string, status: InvoiceStatus): Promise<Invoice> {
  return apiJSON(`/api/v1/invoices/${invoiceId}/status`, 'POST', { status });
}

export async function deleteInvoice(invoiceId: string): Promise<void> {
  const response = await fetch(`/api/v1/invoices/${invoiceId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function downloadInvoiceExport(invoiceId: string, format: 'html' | 'csv' | 'json'): Promise<Blob> {
  const response = await fetch(`/api/v1/invoices/${invoiceId}/export?format=${format}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }

  return response.blob();
}
