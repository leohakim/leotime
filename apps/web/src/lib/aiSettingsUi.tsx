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
    };
    if (form.cursorApiKey.trim()) {
      payload.cursorApiKey = form.cursorApiKey.trim();
    }
    saveMutation.mutate(payload);
  }

  return (
    <section className="panel-section" id="ai-summary-settings">
      <div className="panel-section-heading">
        <span className="section-kicker">
          <Sparkles aria-hidden="true" />
          {t('aiSettingsKicker')}
        </span>
        <h2>{t('aiSettingsHeading')}</h2>
        <p>{t('aiSettingsSubtitle')}</p>
      </div>

      {settingsQuery.isLoading ? <p>{t('loading')}</p> : null}
      {settingsQuery.isError ? <p role="alert">{t('aiSettingsLoadFailed')}</p> : null}

      <form className="profile-form" noValidate onSubmit={submit}>
        {formError ? (
          <div className="form-alert" role="alert">
            <CircleAlert aria-hidden="true" />
            {formError}
          </div>
        ) : null}

        <label className="checkbox-field">
          <input
            checked={form.enabled}
            onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))}
            type="checkbox"
          />
          <span>{t('aiSettingsEnabled')}</span>
        </label>

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
              cursorApiKeyConfigured ? t('aiSettingsCursorApiKeyConfiguredPlaceholder') : t('aiSettingsCursorApiKeyPlaceholder')
            }
            type="password"
            value={form.cursorApiKey}
          />
          {cursorApiKeyConfigured ? <small>{t('aiSettingsCursorApiKeyConfiguredHint')}</small> : null}
        </label>

        <div className="profile-form-actions">
          <button disabled={saveMutation.isPending} type="submit">
            <Save aria-hidden="true" />
            {t('save')}
          </button>
        </div>
      </form>
    </section>
  );
}
