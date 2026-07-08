import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, CloudUpload, DatabaseBackup, RefreshCcw, Save } from 'lucide-react';
import { FormEvent, useEffect, useState } from 'react';
import {
  fetchBackupObjects,
  fetchBackupSettings,
  isApiError,
  restoreBackup,
  runBackupNow,
  testBackupConnection,
  updateBackupSettings,
  type BackupObject,
  type BackupSettings,
  type BackupSettingsInput,
} from './api';
import type { MessageKey } from './i18n';
import { useToast } from './toast';

export type Translator = (key: MessageKey) => string;

type BackupFormState = BackupSettingsInput & {
  secretAccessKey: string;
};

function buildFormFromSettings(settings: BackupSettings): BackupFormState {
  return {
    enabled: settings.enabled,
    endpoint: settings.endpoint,
    region: settings.region,
    bucket: settings.bucket,
    prefix: settings.prefix,
    accessKeyId: settings.accessKeyId,
    secretAccessKey: '',
    usePathStyle: settings.usePathStyle,
    scheduleHour: settings.scheduleHour,
    retentionDays: settings.retentionDays,
  };
}

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

function formatBytes(sizeBytes: number): string {
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  if (sizeBytes < 1024 * 1024) {
    return `${(sizeBytes / 1024).toFixed(1)} KiB`;
  }
  return `${(sizeBytes / (1024 * 1024)).toFixed(1)} MiB`;
}

function formatTimestamp(value?: string | null): string {
  if (!value) {
    return '—';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

export function BackupSettingsPanel({ t }: { t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const settingsQuery = useQuery({
    queryKey: ['backup-settings'],
    queryFn: fetchBackupSettings,
    retry: false,
  });
  const objectsQuery = useQuery({
    queryKey: ['backup-objects'],
    queryFn: fetchBackupObjects,
    enabled: settingsQuery.data?.enabled === true,
    retry: false,
  });

  const [form, setForm] = useState<BackupFormState>(() => ({
    enabled: false,
    endpoint: '',
    region: '',
    bucket: '',
    prefix: 'leotime/backups/',
    accessKeyId: '',
    secretAccessKey: '',
    usePathStyle: false,
    scheduleHour: 1,
    retentionDays: 365,
  }));
  const [errors, setErrors] = useState<Partial<Record<keyof BackupFormState | 'form', string>>>({});
  const [selectedObjectKey, setSelectedObjectKey] = useState('');
  const [restoreConfirm, setRestoreConfirm] = useState(false);

  useEffect(() => {
    if (!settingsQuery.data) {
      return;
    }
    setForm(buildFormFromSettings(settingsQuery.data));
  }, [settingsQuery.data]);

  const saveMutation = useMutation({
    mutationFn: updateBackupSettings,
    onSuccess: (settings) => {
      queryClient.setQueryData(['backup-settings'], settings);
      setForm(buildFormFromSettings(settings));
      setErrors({});
      toast.success(t('backupSettingsSaved'));
    },
    onError: (error: Error) => {
      const secretsMissing =
        isApiError(error) && (error.status === 503 || error.code === 'backup_secrets_key_missing');
      const message = secretsMissing ? t('backupSecretsKeyMissing') : t('backupSettingsSaveFailed');
      setErrors({ form: message });
      toast.error(message);
    },
  });

  const testMutation = useMutation({
    mutationFn: testBackupConnection,
    onSuccess: () => toast.success(t('backupTestOk')),
    onError: (error: Error) => toast.error(error.message || t('backupTestFailed')),
  });

  const runMutation = useMutation({
    mutationFn: runBackupNow,
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['backup-settings'] });
      void queryClient.invalidateQueries({ queryKey: ['backup-objects'] });
      if (result.status === 'success') {
        toast.success(t('backupRunSuccess'));
      } else {
        toast.error(t('backupRunFailed'));
      }
    },
    onError: () => toast.error(t('backupRunFailed')),
  });

  const restoreMutation = useMutation({
    mutationFn: restoreBackup,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup-settings'] });
      setRestoreConfirm(false);
      toast.success(t('backupRestoreSuccess'));
    },
    onError: () => toast.error(t('backupRestoreFailed')),
  });

  function updateField<K extends keyof BackupFormState>(key: K, value: BackupFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined, form: undefined }));
  }

  function validate(next: BackupFormState): Partial<Record<keyof BackupFormState | 'form', string>> {
    const nextErrors: Partial<Record<keyof BackupFormState | 'form', string>> = {};
    if (next.enabled) {
      if (!next.bucket.trim()) {
        nextErrors.bucket = t('backupBucketRequired');
      }
      if (!next.accessKeyId.trim()) {
        nextErrors.accessKeyId = t('backupAccessKeyRequired');
      }
      if (!next.secretAccessKey.trim() && !settingsQuery.data?.secretAccessKeyConfigured) {
        nextErrors.secretAccessKey = t('backupSecretKeyRequired');
      }
    }
    if (next.scheduleHour < 0 || next.scheduleHour > 23) {
      nextErrors.scheduleHour = t('backupScheduleHourInvalid');
    }
    if (next.retentionDays < 1 || next.retentionDays > 3650) {
      nextErrors.retentionDays = t('backupRetentionInvalid');
    }
    return nextErrors;
  }

  function buildTestPayload(): BackupSettingsInput {
    const payload: BackupSettingsInput = {
      enabled: form.enabled,
      endpoint: form.endpoint.trim(),
      region: form.region.trim(),
      bucket: form.bucket.trim(),
      prefix: form.prefix.trim() || 'leotime/backups/',
      accessKeyId: form.accessKeyId.trim(),
      usePathStyle: form.usePathStyle,
      scheduleHour: form.scheduleHour,
      retentionDays: form.retentionDays,
    };
    if (form.secretAccessKey.trim()) {
      payload.secretAccessKey = form.secretAccessKey.trim();
    }
    return payload;
  }

  function validateForTest(next: BackupFormState): Partial<Record<keyof BackupFormState | 'form', string>> {
    const nextErrors: Partial<Record<keyof BackupFormState | 'form', string>> = {};
    if (!next.bucket.trim()) {
      nextErrors.bucket = t('backupBucketRequired');
    }
    if (!next.accessKeyId.trim()) {
      nextErrors.accessKeyId = t('backupAccessKeyRequired');
    }
    if (!next.secretAccessKey.trim() && !settingsQuery.data?.secretAccessKeyConfigured) {
      nextErrors.secretAccessKey = t('backupSecretKeyRequired');
    }
    return nextErrors;
  }

  function runConnectionTest() {
    const nextErrors = validateForTest(form);
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }
    testMutation.mutate(buildTestPayload());
  }

  function submitSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextErrors = validate(form);
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }

    saveMutation.mutate(buildTestPayload());
  }

  function submitRestore(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!restoreConfirm) {
      setErrors({ form: t('backupRestoreConfirmRequired') });
      return;
    }
    if (!selectedObjectKey) {
      setErrors({ form: t('backupRestoreSelectRequired') });
      return;
    }
    restoreMutation.mutate({ objectKey: selectedObjectKey, confirm: true });
  }

  const settings = settingsQuery.data;
  const objects = objectsQuery.data?.objects ?? [];

  return (
    <section className="panel-section" id="backups">
      <div className="panel-header">
        <div>
          <span>{t('backupSection')}</span>
          <h2>{t('backupHeading')}</h2>
          <p>{t('backupSubtitle')}</p>
        </div>
      </div>

      {settingsQuery.isError ? (
        <div className="form-alert" role="alert">
          <CircleAlert aria-hidden="true" />
          {t('backupLoadFailed')}
        </div>
      ) : null}

      <form className="client-editor profile-settings-form" noValidate onSubmit={submitSettings}>
        {errors.form ? (
          <div className="form-alert" role="alert">
            <CircleAlert aria-hidden="true" />
            {errors.form}
          </div>
        ) : null}

        <div className="settings-toggle-row backup-enabled-row">
          <input
            checked={form.enabled}
            id="backup-enabled"
            onChange={(event) => updateField('enabled', event.target.checked)}
            type="checkbox"
          />
          <label htmlFor="backup-enabled">{t('backupEnabled')}</label>
        </div>

        <div className="client-form-grid backup-settings-grid">
          <label className={fieldClass(errors.endpoint)} htmlFor="backup-endpoint">
            <span>{t('backupEndpoint')}</span>
            <input
              id="backup-endpoint"
              onChange={(event) => updateField('endpoint', event.target.value)}
              placeholder="https://s3.eu-central-1.amazonaws.com"
              value={form.endpoint}
            />
          </label>

          <label className={fieldClass(errors.region)} htmlFor="backup-region">
            <span>{t('backupRegion')}</span>
            <input id="backup-region" onChange={(event) => updateField('region', event.target.value)} value={form.region} />
          </label>

          <label className={fieldClass(errors.bucket)} htmlFor="backup-bucket">
            <span>{t('backupBucket')}</span>
            <input id="backup-bucket" onChange={(event) => updateField('bucket', event.target.value)} value={form.bucket} />
            {errors.bucket ? <span className="field-message">{errors.bucket}</span> : null}
          </label>

          <label className={fieldClass(errors.prefix)} htmlFor="backup-prefix">
            <span>{t('backupPrefix')}</span>
            <input id="backup-prefix" onChange={(event) => updateField('prefix', event.target.value)} value={form.prefix} />
          </label>

          <label className={fieldClass(errors.accessKeyId)} htmlFor="backup-access-key">
            <span>{t('backupAccessKeyId')}</span>
            <input id="backup-access-key" onChange={(event) => updateField('accessKeyId', event.target.value)} value={form.accessKeyId} />
            {errors.accessKeyId ? <span className="field-message">{errors.accessKeyId}</span> : null}
          </label>

          <label className={fieldClass(errors.secretAccessKey)} htmlFor="backup-secret-key">
            <span>{t('backupSecretAccessKey')}</span>
            <input
              autoComplete="off"
              id="backup-secret-key"
              onChange={(event) => updateField('secretAccessKey', event.target.value)}
              placeholder={settings?.secretAccessKeyConfigured ? t('backupSecretConfiguredPlaceholder') : ''}
              type="password"
              value={form.secretAccessKey}
            />
            {errors.secretAccessKey ? <span className="field-message">{errors.secretAccessKey}</span> : null}
          </label>
        </div>

        <div className="settings-toggle-row">
          <input
            checked={form.usePathStyle}
            id="backup-path-style"
            onChange={(event) => updateField('usePathStyle', event.target.checked)}
            type="checkbox"
          />
          <label htmlFor="backup-path-style">{t('backupUsePathStyle')}</label>
        </div>

        <div className="backup-compact-fields">
          <label className={fieldClass(errors.scheduleHour)} htmlFor="backup-schedule-hour">
            <span>{t('backupScheduleHour')}</span>
            <input
              className="settings-compact-input"
              id="backup-schedule-hour"
              max={23}
              min={0}
              onChange={(event) => updateField('scheduleHour', Number(event.target.value))}
              type="number"
              value={form.scheduleHour}
            />
            {errors.scheduleHour ? <span className="field-message">{errors.scheduleHour}</span> : null}
          </label>

          <label className={fieldClass(errors.retentionDays)} htmlFor="backup-retention-days">
            <span>{t('backupRetentionDays')}</span>
            <input
              className="settings-compact-input backup-retention-input"
              id="backup-retention-days"
              max={3650}
              min={1}
              onChange={(event) => updateField('retentionDays', Number(event.target.value))}
              type="number"
              value={form.retentionDays}
            />
            {errors.retentionDays ? <span className="field-message">{errors.retentionDays}</span> : null}
          </label>
        </div>

        {settings ? (
          <div className="backup-status-grid">
            <p>
              <strong>{t('backupLastRun')}:</strong> {formatTimestamp(settings.lastRunAt)} ({settings.lastStatus})
            </p>
            {settings.lastError ? <p>{settings.lastError}</p> : null}
            {settings.lastObjectKey ? <p>{settings.lastObjectKey}</p> : null}
            <p>
              <strong>{t('backupLastRestore')}:</strong> {formatTimestamp(settings.lastRestoreAt)} ({settings.lastRestoreStatus})
            </p>
          </div>
        ) : null}

        <div className="client-form-actions">
          <button disabled={saveMutation.isPending || settingsQuery.isLoading} type="submit">
            <Save aria-hidden="true" />
            {saveMutation.isPending ? t('loading') : t('backupSave')}
          </button>
          <button
            disabled={testMutation.isPending}
            onClick={runConnectionTest}
            type="button"
          >
            <CloudUpload aria-hidden="true" />
            {testMutation.isPending ? t('loading') : t('backupTest')}
          </button>
          <button disabled={runMutation.isPending || !form.enabled} onClick={() => runMutation.mutate()} type="button">
            <DatabaseBackup aria-hidden="true" />
            {runMutation.isPending ? t('loading') : t('backupRunNow')}
          </button>
        </div>
      </form>

      <form className="client-editor profile-password-form" noValidate onSubmit={submitRestore}>
        <div className="editor-header">
          <div>
            <span>{t('backupRestoreSection')}</span>
            <h3>{t('backupRestoreHeading')}</h3>
          </div>
        </div>

        {objectsQuery.isLoading ? <p>{t('loading')}</p> : null}
        {objects.length === 0 && !objectsQuery.isLoading ? <p>{t('backupNoObjects')}</p> : null}

        {objects.length > 0 ? (
          <div className="backup-object-list">
            {objects.map((object: BackupObject) => (
              <label className="backup-object-row" key={object.key}>
                <input
                  checked={selectedObjectKey === object.key}
                  name="backup-object"
                  onChange={() => setSelectedObjectKey(object.key)}
                  type="radio"
                />
                <span>
                  {object.key} · {formatBytes(object.sizeBytes)} · {formatTimestamp(object.lastModified)}
                </span>
              </label>
            ))}
          </div>
        ) : null}

        <div className="settings-toggle-row backup-restore-confirm">
          <input
            checked={restoreConfirm}
            id="backup-restore-confirm"
            onChange={(event) => setRestoreConfirm(event.target.checked)}
            type="checkbox"
          />
          <label htmlFor="backup-restore-confirm">{t('backupRestoreConfirmLabel')}</label>
        </div>

        <div className="client-form-actions">
          <button disabled={restoreMutation.isPending || !selectedObjectKey} type="submit">
            <RefreshCcw aria-hidden="true" />
            {restoreMutation.isPending ? t('loading') : t('backupRestoreAction')}
          </button>
        </div>
      </form>
    </section>
  );
}
