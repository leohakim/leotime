import type { Task } from './api';

export function sortTasksByNewest(tasks: Task[]): Task[] {
  return [...tasks].sort((left, right) => {
    const createdDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt);
    if (createdDiff !== 0) {
      return createdDiff;
    }
    return right.id.localeCompare(left.id);
  });
}
