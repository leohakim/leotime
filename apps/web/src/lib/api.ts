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

export async function fetchSession(): Promise<SessionResponse> {
  return apiGet('/api/v1/session');
}

export async function fetchOverview(): Promise<Overview> {
  return apiGet('/api/v1/overview');
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

