export type OfflineOperationType =
  | 'createClient'
  | 'createProject'
  | 'createTask'
  | 'createTag'
  | 'createTimeEntry'
  | 'updateTimeEntry'
  | 'startTimer'
  | 'updateTimer'
  | 'stopTimer';

export type QueuedMutation = {
  id: string;
  operation: OfflineOperationType;
  localId?: string;
  entityId?: string;
  payload: Record<string, unknown>;
  createdAt: string;
  retryCount: number;
  lastError: string;
};

export type IdMapping = {
  localId: string;
  serverId: string;
};
