import type { Client, Project } from './api';

export function hasBillableRate(project: Project | undefined | null, clients: Client[]): boolean {
  if ((project?.defaultHourlyRateMinor ?? 0) > 0) {
    return true;
  }
  const client = clients.find((item) => item.id === project?.clientId);
  return (client?.defaultHourlyRateMinor ?? 0) > 0;
}
