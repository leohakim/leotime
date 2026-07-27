import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, Save, Sparkles } from 'lucide-react';
import { FormEvent, useEffect, useState } from 'react';
import {
  fetchAISettings,
  isApiError,
  updateAISettings,
  type AISettings,
  type AISettingsInput,
} from './api';
import type { MessageKey } from './i18n';
import { SettingsCard, SettingsPanel } from './settingsChrome';
import { useToast } from './toast';

export type Translator = (key: MessageKey) => string;

type AISettingsFormState = AISettingsInput & {
  cursorApiKey: string;
};

function buildFormFromSettings(settings: AISettings): AISettingsFormState {
  return {
    enabled: settings.enabled,
    gitAuthorEmail: settings.gitAuthorEmail,
    cursorApiKey: '',
    cursorCostPerMillionUsd: settings.cursorCostPerMillionUsd || 2,
  };
}

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

export function AISettingsPanel({ t }: { t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const settingsQuery = useQuery({
    queryKey: ['ai-settings'],
    queryFn: fetchAISettings,
    retry: false,
  });

  const [form, setForm] = useState<AISettingsFormState>(() => ({
    enabled: false,
    gitAuthorEmail: '',
    cursorApiKey: '',
    cursorCostPerMillionUsd: 2,
  }));
  const [cursorApiKeyConfigured, setCursorApiKeyConfigured] = useState(false);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (!settingsQuery.data) {
      return;
    }
    setForm(buildFormFromSettings(settingsQuery.data));
    setCursorApiKeyConfigured(settingsQuery.data.cursorApiKeyConfigured);
  }, [settingsQuery.data]);

  const saveMutation = useMutation({
    mutationFn: updateAISettings,
    onSuccess: (settings) => {
      setForm(buildFormFromSettings(settings));
      setCursorApiKeyConfigured(settings.cursorApiKeyConfigured);
      setFormError('');
      void queryClient.invalidateQueries({ queryKey: ['ai-settings'] });
      toast.success(t('aiSettingsSaved'));
    },
    onError: (error) => {
      if (isApiError(error) && error.code === 'secrets_key_missing') {
        setFormError(t('aiSettingsSecretsKeyMissing'));
        return;
      }
      setFormError(t('aiSettingsSaveFailed'));
    },
  });

  function submit(event: FormEvent) {
    event.preventDefault();
    const payload: AISettingsInput = {
      enabled: form.enabled,
      gitAuthorEmail: form.gitAuthorEmail.trim(),
      cursorCostPerMillionUsd: Number(form.cursorCostPerMillionUsd) || 2,
    };
    if (form.cursorApiKey.trim()) {
      payload.cursorApiKey = form.cursorApiKey.trim();
    }
    saveMutation.mutate(payload);
  }

  return (
    <SettingsPanel
      id="ai-summary-settings"
      kicker={
        <>
          <Sparkles aria-hidden="true" />
          {t('aiSettingsKicker')}
        </>
      }
      title={t('aiSettingsHeading')}
      subtitle={t('aiSettingsSubtitle')}
      meta={
        settingsQuery.isLoading ? (
          <span className="sync-pill">{t('loading')}</span>
        ) : settingsQuery.isError ? (
          <span className="sync-pill warning-pill">{t('aiSettingsLoadFailed')}</span>
        ) : null
      }
    >
      <form className="settings-form" noValidate onSubmit={submit}>
        <SettingsCard>
          {formError ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {formError}
            </div>
          ) : null}

          <div className="settings-toggle-row">
            <input
              checked={form.enabled}
              id="ai-settings-enabled"
              onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))}
              type="checkbox"
            />
            <label htmlFor="ai-settings-enabled">{t('aiSettingsEnabled')}</label>
          </div>

          <div className="client-form-grid settings-field-grid">
            <label className={fieldClass()} htmlFor="ai-git-author-email">
              <span>{t('aiSettingsGitAuthorEmail')}</span>
              <input
                id="ai-git-author-email"
                onChange={(event) => setForm((current) => ({ ...current, gitAuthorEmail: event.target.value }))}
                placeholder={t('aiSettingsGitAuthorEmailPlaceholder')}
                type="email"
                value={form.gitAuthorEmail}
              />
            </label>

            <label className={fieldClass()} htmlFor="ai-cursor-api-key">
              <span>{t('aiSettingsCursorApiKey')}</span>
              <input
                autoComplete="off"
                id="ai-cursor-api-key"
                onChange={(event) => setForm((current) => ({ ...current, cursorApiKey: event.target.value }))}
                placeholder={
                  cursorApiKeyConfigured
                    ? t('aiSettingsCursorApiKeyConfiguredPlaceholder')
                    : t('aiSettingsCursorApiKeyPlaceholder')
                }
                type="password"
                value={form.cursorApiKey}
              />
              {cursorApiKeyConfigured ? (
                <span className="field-hint">{t('aiSettingsCursorApiKeyConfiguredHint')}</span>
              ) : null}
            </label>

            <label className={fieldClass()} htmlFor="ai-cursor-cost-per-million">
              <span>{t('aiSettingsCursorCostPerMillion')}</span>
              <input
                id="ai-cursor-cost-per-million"
                min="0"
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    cursorCostPerMillionUsd: Number(event.target.value),
                  }))
                }
                step="0.01"
                type="number"
                value={form.cursorCostPerMillionUsd}
              />
              <span className="field-hint">{t('aiSettingsCursorCostPerMillionHint')}</span>
            </label>
          </div>
        </SettingsCard>
        <div className="settings-card-actions">
          <button disabled={saveMutation.isPending} type="submit">
            <Save aria-hidden="true" />
            {t('save')}
          </button>
        </div>
      </form>
    </SettingsPanel>
  );
}
