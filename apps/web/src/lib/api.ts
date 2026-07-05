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

export async function fetchSession(): Promise<SessionResponse> {
  return apiGet('/api/v1/session');
}

export async function fetchOverview(): Promise<Overview> {
  return apiGet('/api/v1/overview');
}

export async function fetchClients(): Promise<ClientsResponse> {
  return apiGet('/api/v1/clients');
}

export async function fetchProjects(): Promise<ProjectsResponse> {
  return apiGet('/api/v1/projects');
}

export async function fetchTasks(): Promise<TasksResponse> {
  return apiGet('/api/v1/tasks');
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
