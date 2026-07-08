import { sortTasksByNewest } from './taskSort';

export type LayoutMode = 'solid' | 'minimal' | 'compact';
export type ThemeMode = 'solid' | 'light' | 'dark' | 'minimal';
export type Locale = 'es' | 'en';

export type User = {
  id: string;
  email: string;
  name: string;
  locale: Locale;
  layoutMode: LayoutMode;
};

export type AppSettings = {
  taskProjectRequired: boolean;
  defaultCurrency: string;
  timezone: string;
  themeMode: ThemeMode;
  timerStillRunningEnabled: boolean;
  timerStillRunningHours: number;
  backupEmailOnSuccess: boolean;
  backupEmailOnFailure: boolean;
  restoreEmailOnSuccess: boolean;
  restoreEmailOnFailure: boolean;
};

export type Profile = {
  id: string;
  email: string;
  name: string;
  locale: Locale;
  layoutMode: LayoutMode;
  settings: AppSettings;
  createdAt: string;
  updatedAt: string;
};

export type ProfileUpdateInput = {
  name: string;
  email: string;
  locale: Locale;
  layoutMode: LayoutMode;
  taskProjectRequired: boolean;
  defaultCurrency: string;
  timezone: string;
  themeMode: ThemeMode;
  timerStillRunningEnabled: boolean;
  timerStillRunningHours: number;
  backupEmailOnSuccess: boolean;
  backupEmailOnFailure: boolean;
  restoreEmailOnSuccess: boolean;
  restoreEmailOnFailure: boolean;
};

export type ChangePasswordInput = {
  currentPassword: string;
  newPassword: string;
};

export type BackupSettings = {
  enabled: boolean;
  endpoint: string;
  region: string;
  bucket: string;
  prefix: string;
  accessKeyId: string;
  secretAccessKeyConfigured: boolean;
  usePathStyle: boolean;
  scheduleHour: number;
  retentionDays: number;
  lastRunAt?: string | null;
  lastStatus: string;
  lastError: string;
  lastObjectKey: string;
  lastRestoreAt?: string | null;
  lastRestoreStatus: string;
  lastRestoreError: string;
  lastRestoreObjectKey: string;
};

export type BackupSettingsInput = {
  enabled: boolean;
  endpoint: string;
  region: string;
  bucket: string;
  prefix: string;
  accessKeyId: string;
  secretAccessKey?: string;
  usePathStyle: boolean;
  scheduleHour: number;
  retentionDays: number;
};

export type BackupObject = {
  key: string;
  sizeBytes: number;
  lastModified: string;
};

export type BackupRunResult = {
  status: string;
  objectKey?: string;
  sizeBytes?: number;
  startedAt: string;
  finishedAt: string;
  error?: string;
};

export type BackupRestoreResult = {
  status: string;
  objectKey?: string;
  safetySnapshotPath?: string;
  startedAt: string;
  finishedAt: string;
  error?: string;
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
  archivedAt: string;
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
  projectColor?: string;
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

export async function fetchProfile(): Promise<Profile> {
  return apiGet('/api/v1/profile');
}

export async function updateProfile(input: ProfileUpdateInput): Promise<Profile> {
  return apiJSON('/api/v1/profile', 'PATCH', input);
}

export async function changePassword(input: ChangePasswordInput): Promise<void> {
  const response = await fetch('/api/v1/profile/change-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function fetchBackupSettings(): Promise<BackupSettings> {
  return apiGet('/api/v1/backups/settings');
}

export async function updateBackupSettings(input: BackupSettingsInput): Promise<BackupSettings> {
  return apiJSON('/api/v1/backups/settings', 'PUT', input);
}

export async function testBackupConnection(input?: BackupSettingsInput): Promise<{ ok: boolean; message: string }> {
  return apiJSON('/api/v1/backups/test', 'POST', input ?? {});
}

export async function runBackupNow(): Promise<BackupRunResult> {
  return apiJSON('/api/v1/backups/run', 'POST', {});
}

export async function fetchBackupObjects(): Promise<{ objects: BackupObject[] }> {
  return apiGet('/api/v1/backups/objects');
}

export async function restoreBackup(input: { objectKey?: string; latest?: boolean; confirm: boolean }): Promise<BackupRestoreResult> {
  return apiJSON('/api/v1/backups/restore', 'POST', input);
}

export async function fetchOverview(): Promise<Overview> {
  return apiGet('/api/v1/overview');
}

export async function fetchDashboardStats(activityMonth?: string): Promise<DashboardStats> {
  const query = activityMonth ? `?activityMonth=${encodeURIComponent(activityMonth)}` : '';
  return apiGet(`/api/v1/dashboard/stats${query}`);
}

export async function fetchClients(options?: { includeArchived?: boolean }): Promise<ClientsResponse> {
  const query = options?.includeArchived ? '?includeArchived=true' : '';
  return apiGet(`/api/v1/clients${query}`);
}

export async function fetchProjects(options?: { includeArchived?: boolean }): Promise<ProjectsResponse> {
  const query = options?.includeArchived ? '?includeArchived=true' : '';
  return apiGet(`/api/v1/projects${query}`);
}

export async function fetchTasks(options?: { includeArchived?: boolean }): Promise<TasksResponse> {
  const query = options?.includeArchived ? '?includeArchived=true' : '';
  const response = await apiGet<TasksResponse>(`/api/v1/tasks${query}`);
  return {
    tasks: sortTasksByNewest(response.tasks),
  };
}

export async function fetchTags(options?: { includeArchived?: boolean }): Promise<TagsResponse> {
  const query = options?.includeArchived ? '?includeArchived=true' : '';
  return apiGet(`/api/v1/tags${query}`);
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

export async function restoreClient(clientId: string): Promise<Client> {
  return apiJSON(`/api/v1/clients/${clientId}/restore`, 'POST', {});
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

export async function restoreProject(projectId: string): Promise<Project> {
  return apiJSON(`/api/v1/projects/${projectId}/restore`, 'POST', {});
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

export async function restoreTask(taskId: string): Promise<Task> {
  return apiJSON(`/api/v1/tasks/${taskId}/restore`, 'POST', {});
}

export async function createTag(input: TagInput): Promise<Tag> {
  return apiJSON('/api/v1/tags', 'POST', input);
}

export async function updateTag(tagId: string, input: TagInput): Promise<Tag> {
  return apiJSON(`/api/v1/tags/${tagId}`, 'PATCH', input);
}

export async function archiveTag(tagId: string): Promise<void> {
  const response = await fetch(`/api/v1/tags/${tagId}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`request_failed:${response.status}`);
  }
}

export async function restoreTag(tagId: string): Promise<Tag> {
  return apiJSON(`/api/v1/tags/${tagId}/restore`, 'POST', {});
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

export async function requestPasswordReset(email: string): Promise<void> {
  const response = await fetch('/api/v1/auth/forgot-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ email }),
  });

  if (!response.ok) {
    throw new Error('password_reset_request_failed');
  }
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
  const response = await fetch('/api/v1/auth/reset-password', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ token, newPassword }),
  });

  if (!response.ok) {
    throw new Error('password_reset_failed');
  }
}

async function readApiError(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as { error?: string };
    if (payload.error?.trim()) {
      return payload.error;
    }
  } catch {
    // ignore non-json error bodies
  }
  return `request_failed:${response.status}`;
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

async function apiJSON<T>(path: string, method: 'POST' | 'PATCH' | 'PUT', body: unknown): Promise<T> {
  const response = await fetch(path, {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw new Error(await readApiError(response));
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

export type ImportEntityStats = {
  seen: number;
  created: number;
  updated: number;
  skipped: number;
};

export type SolidtimeImportSummary = {
  provider: string;
  exportId: string;
  version: string;
  dryRun: boolean;
  organization: ImportEntityStats;
  members: ImportEntityStats;
  clients: ImportEntityStats;
  projects: ImportEntityStats;
  tasks: ImportEntityStats;
  tags: ImportEntityStats;
  timeEntries: ImportEntityStats;
  warnings: string[];
  errors: string[];
};

type SolidtimeImportResponse = {
  summary: SolidtimeImportSummary;
};

const emptyImportStats = (): ImportEntityStats => ({
  seen: 0,
  created: 0,
  updated: 0,
  skipped: 0,
});

export function normalizeSolidtimeImportSummary(
  summary: Partial<SolidtimeImportSummary> | null | undefined,
): SolidtimeImportSummary {
  return {
    provider: summary?.provider ?? '',
    exportId: summary?.exportId ?? '',
    version: summary?.version ?? '',
    dryRun: summary?.dryRun ?? false,
    organization: summary?.organization ?? emptyImportStats(),
    members: summary?.members ?? emptyImportStats(),
    clients: summary?.clients ?? emptyImportStats(),
    projects: summary?.projects ?? emptyImportStats(),
    tasks: summary?.tasks ?? emptyImportStats(),
    tags: summary?.tags ?? emptyImportStats(),
    timeEntries: summary?.timeEntries ?? emptyImportStats(),
    warnings: summary?.warnings ?? [],
    errors: summary?.errors ?? [],
  };
}

export async function importSolidtimeExport(file: File, dryRun: boolean): Promise<SolidtimeImportSummary> {
  const form = new FormData();
  form.append('file', file);

  const response = await fetch(`/api/v1/imports/solidtime?dryRun=${dryRun ? 'true' : 'false'}`, {
    method: 'POST',
    body: form,
    credentials: 'include',
  });

  let payload: SolidtimeImportResponse & { error?: string } = { summary: {} as SolidtimeImportSummary };
  try {
    payload = (await response.json()) as SolidtimeImportResponse & { error?: string };
  } catch {
    throw new Error('import_failed');
  }

  if (!response.ok) {
    if (payload.summary) {
      return normalizeSolidtimeImportSummary(payload.summary);
    }
    throw new Error(payload.error ?? 'import_failed');
  }

  return normalizeSolidtimeImportSummary(payload.summary);
}
