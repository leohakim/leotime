import type { IdMapping, QueuedMutation } from './types';

const DB_NAME = 'leotime-offline';
const DB_VERSION = 1;
const MUTATIONS_STORE = 'mutations';
const ID_MAP_STORE = 'idMap';

let memoryMutations: QueuedMutation[] = [];
let memoryIdMap = new Map<string, string>();

function storageAvailable(): boolean {
  return typeof indexedDB !== 'undefined';
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onerror = () => reject(request.error ?? new Error('open offline db failed'));
    request.onupgradeneeded = () => {
      const database = request.result;
      if (!database.objectStoreNames.contains(MUTATIONS_STORE)) {
        const store = database.createObjectStore(MUTATIONS_STORE, { keyPath: 'id' });
        store.createIndex('createdAt', 'createdAt', { unique: false });
      }
      if (!database.objectStoreNames.contains(ID_MAP_STORE)) {
        database.createObjectStore(ID_MAP_STORE, { keyPath: 'localId' });
      }
    };
    request.onsuccess = () => resolve(request.result);
  });
}

function runTransaction<T>(
  storeNames: string | string[],
  mode: IDBTransactionMode,
  run: (stores: IDBObjectStore[]) => IDBRequest<T> | void,
): Promise<T | void> {
  return openDatabase().then(
    (database) =>
      new Promise<T | void>((resolve, reject) => {
        const names = Array.isArray(storeNames) ? storeNames : [storeNames];
        const transaction = database.transaction(names, mode);
        const stores = names.map((name) => transaction.objectStore(name));
        let request: IDBRequest<T> | undefined;
        try {
          const result = run(stores);
          if (result) {
            request = result as IDBRequest<T>;
          }
        } catch (error) {
          reject(error);
          return;
        }

        transaction.oncomplete = () => resolve(request?.result);
        transaction.onerror = () => reject(transaction.error ?? new Error('offline db transaction failed'));
        transaction.onabort = () => reject(transaction.error ?? new Error('offline db transaction aborted'));
      }),
  );
}

export async function listQueuedMutations(): Promise<QueuedMutation[]> {
  if (!storageAvailable()) {
    return [...memoryMutations].sort((left, right) => left.createdAt.localeCompare(right.createdAt));
  }

  const items = await runTransaction<QueuedMutation[]>(MUTATIONS_STORE, 'readonly', ([store]) => store.getAll());
  return (items ?? []).sort((left, right) => left.createdAt.localeCompare(right.createdAt));
}

export async function putQueuedMutation(mutation: QueuedMutation): Promise<void> {
  if (!storageAvailable()) {
    memoryMutations = [...memoryMutations.filter((item) => item.id !== mutation.id), mutation];
    return;
  }

  await runTransaction(MUTATIONS_STORE, 'readwrite', ([store]) => store.put(mutation));
}

export async function deleteQueuedMutation(mutationId: string): Promise<void> {
  if (!storageAvailable()) {
    memoryMutations = memoryMutations.filter((item) => item.id !== mutationId);
    return;
  }

  await runTransaction(MUTATIONS_STORE, 'readwrite', ([store]) => store.delete(mutationId));
}

export async function clearQueuedMutations(): Promise<void> {
  if (!storageAvailable()) {
    memoryMutations = [];
    return;
  }

  await runTransaction(MUTATIONS_STORE, 'readwrite', ([store]) => store.clear());
}

export async function clearIdMappings(): Promise<void> {
  if (!storageAvailable()) {
    memoryIdMap = new Map();
    return;
  }

  await runTransaction(ID_MAP_STORE, 'readwrite', ([store]) => store.clear());
}

export async function resetOfflineStorage(): Promise<void> {
  await clearQueuedMutations();
  await clearIdMappings();
}

export async function getServerId(localId: string): Promise<string | null> {
  if (!storageAvailable()) {
    return memoryIdMap.get(localId) ?? null;
  }

  const mapping = await runTransaction<IdMapping>(ID_MAP_STORE, 'readonly', ([store]) => store.get(localId));
  return mapping?.serverId ?? null;
}

export async function setServerId(localId: string, serverId: string): Promise<void> {
  if (!storageAvailable()) {
    memoryIdMap.set(localId, serverId);
    return;
  }

  await runTransaction(ID_MAP_STORE, 'readwrite', ([store]) => store.put({ localId, serverId }));
}

export async function listIdMappings(): Promise<Map<string, string>> {
  if (!storageAvailable()) {
    return new Map(memoryIdMap);
  }

  const mappings = await runTransaction<IdMapping[]>(ID_MAP_STORE, 'readonly', ([store]) => store.getAll());
  return new Map((mappings ?? []).map((mapping) => [mapping.localId, mapping.serverId]));
}

export function resetOfflineStorageForTests() {
  memoryMutations = [];
  memoryIdMap = new Map();
}
