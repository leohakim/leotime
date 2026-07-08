import { useMutation, useQueryClient } from '@tanstack/react-query';
import { BadgeDollarSign, Building2, CircleAlert, CircleCheck, Mail, Pencil, Plus, RotateCcw, Save, Trash2, X } from 'lucide-react';
import { FormEvent, useState } from 'react';
import {
  archiveClient,
  restoreClient,
  updateClient,
  type Client,
  type ClientInput,
} from '../../lib/api';
import { DirectoryInactiveHeading, FieldError, fieldClass, formatMinor, formatRateInput, hasErrors, rateToMinor } from '../../lib/crudFormUi';
import { patchClientsCache, refreshOverviewIfOnline } from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { createClient, isLocalId } from '../../lib/offline/mutations';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type ClientFormState = Omit<ClientInput, 'defaultHourlyRateMinor'> & {
  hourlyRate: string;
  active: boolean;
};

type ClientFormErrors = Partial<Record<keyof ClientFormState | 'form', string>>;

const emptyClientForm: ClientFormState = {
  name: '',
  email: '',
  taxId: '',
  billingAddress: '',
  defaultCurrency: 'EUR',
  hourlyRate: '',
  active: true,
};

export function ClientPanel({ clients, isLoading, t }: { clients: Client[]; isLoading: boolean; t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingClientId, setEditingClientId] = useState<string | null>(null);
  const [form, setForm] = useState<ClientFormState>(emptyClientForm);
  const [errors, setErrors] = useState<ClientFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createClient,
    onSuccess: (client) => {
      setForm(emptyClientForm);
      setErrors({});
      patchClientsCache(queryClient, client);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(client.id)) {
        queryClient.invalidateQueries({ queryKey: ['clients'] });
      }
      toastMutationSuccess(toast, t, 'clientCreated', client.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      clientId,
      input,
      active,
      wasActive,
    }: {
      clientId: string;
      input: ClientInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateClient(clientId, input);
      if (active && !wasActive) {
        await restoreClient(clientId);
      } else if (!active && wasActive) {
        await archiveClient(clientId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveClient,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientArchiveFailed') }));
      toast.error(t('clientArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreClient,
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  function submitClient(event: FormEvent) {
    event.preventDefault();
    const validation = validateClientForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = clientFormToInput(form);
    if (editingClientId) {
      const client = clients.find((item) => item.id === editingClientId);
      updateMutation.mutate({
        clientId: editingClientId,
        input,
        active: form.active,
        wasActive: !client?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof ClientFormState>(field: K, value: ClientFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateClientForm(next, t));
    }
  }

  function startEditing(client: Client) {
    setEditingClientId(client.id);
    setErrors({});
    setForm({
      name: client.name,
      email: client.email,
      taxId: client.taxId,
      billingAddress: client.billingAddress,
      defaultCurrency: client.defaultCurrency,
      hourlyRate: formatRateInput(client.defaultHourlyRateMinor),
      active: !client.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingClientId(null);
    setForm(emptyClientForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeClientCount = clients.filter((client) => !client.archivedAt).length;
  const activeClients = clients.filter((client) => !client.archivedAt);
  const inactiveClients = clients.filter((client) => client.archivedAt);

  function renderClientRow(client: Client, isActive: boolean) {
    return (
      <article
        className={editingClientId === client.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={client.id}
      >
        <div className="client-row-main">
          <div className="client-avatar" aria-hidden="true">
            {client.name.slice(0, 1).toUpperCase()}
          </div>
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{client.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <span className="client-contact">
              <Mail aria-hidden="true" />
              {client.email || t('noContact')}
            </span>
          </div>
        </div>
        <div className="client-row-meta">
          <span className="rate-pill">
            <BadgeDollarSign aria-hidden="true" />
            {client.defaultCurrency} {formatMinor(client.defaultHourlyRateMinor)}/h
          </span>
          {client.taxId ? <span>{client.taxId}</span> : null}
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(client)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(client.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(client.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

  return (
    <section className="clients-section" id="clients" aria-labelledby="clients-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Building2 aria-hidden="true" />
            {t('clients')}
          </span>
          <h2 id="clients-title">{t('clientDirectory')}</h2>
          <p>{t('clientPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newClient')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeClients')}</span>
              <strong>{activeClientCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {clients.length === 0 ? (
              <div className="empty-state">
                <Building2 aria-hidden="true" />
                <p>{t('noClients')}</p>
              </div>
            ) : null}
            {activeClients.map((client) => renderClientRow(client, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveClients.length} t={t} />
          {inactiveClients.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveClients.map((client) => renderClientRow(client, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitClient}>
          <div className="editor-header">
            <div>
              <span>{editingClientId ? t('editingClient') : t('newClient')}</span>
              <h3>{editingClientId ? t('clientFormEdit') : t('clientFormCreate')}</h3>
            </div>
            {editingClientId ? (
              <button className="ghost-button icon-button" type="button" onClick={cancelEditing} title={t('cancel')}>
                <X aria-hidden="true" />
              </button>
            ) : null}
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid">
            <label className={fieldClass(errors.name)} htmlFor="client-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'client-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="client-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('clientNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="client-name-error" message={errors.name} />
            </label>

            <label className={fieldClass(errors.email)} htmlFor="client-email">
              <span>{t('email')}</span>
              <input
                aria-describedby={errors.email ? 'client-email-error' : undefined}
                aria-invalid={Boolean(errors.email)}
                id="client-email"
                onChange={(event) => updateField('email', event.target.value)}
                placeholder={t('clientEmailPlaceholder')}
                type="email"
                value={form.email}
              />
              <FieldError id="client-email-error" message={errors.email} />
            </label>

            <label className={fieldClass(errors.defaultCurrency)} htmlFor="client-currency">
              <span>
                {t('defaultCurrency')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.defaultCurrency ? 'client-currency-error' : undefined}
                aria-invalid={Boolean(errors.defaultCurrency)}
                id="client-currency"
                maxLength={3}
                onChange={(event) => updateField('defaultCurrency', event.target.value.toUpperCase())}
                placeholder={t('clientCurrencyPlaceholder')}
                value={form.defaultCurrency}
              />
              <FieldError id="client-currency-error" message={errors.defaultCurrency} />
            </label>

            <label className={fieldClass(errors.hourlyRate)} htmlFor="client-rate">
              <span>{t('hourlyRate')}</span>
              <input
                aria-describedby={errors.hourlyRate ? 'client-rate-error' : undefined}
                aria-invalid={Boolean(errors.hourlyRate)}
                id="client-rate"
                inputMode="decimal"
                min="0"
                onChange={(event) => updateField('hourlyRate', event.target.value)}
                placeholder={t('clientRatePlaceholder')}
                type="text"
                value={form.hourlyRate}
              />
              <FieldError id="client-rate-error" message={errors.hourlyRate} />
            </label>

            <label className={fieldClass(errors.taxId)} htmlFor="client-tax-id">
              <span>{t('taxId')}</span>
              <input
                id="client-tax-id"
                onChange={(event) => updateField('taxId', event.target.value)}
                placeholder={t('clientTaxPlaceholder')}
                value={form.taxId}
              />
              <FieldError id="client-tax-id-error" message={errors.taxId} />
            </label>

            <label className={fieldClass(errors.billingAddress) + ' client-address-field'} htmlFor="client-address">
              <span>{t('billingAddress')}</span>
              <input
                id="client-address"
                onChange={(event) => updateField('billingAddress', event.target.value)}
                placeholder={t('clientAddressPlaceholder')}
                value={form.billingAddress}
              />
              <FieldError id="client-address-error" message={errors.billingAddress} />
            </label>

            {editingClientId ? (
              <label className="client-active-field" htmlFor="client-active">
                <input
                  checked={form.active}
                  id="client-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('clientActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingClientId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingClientId ? t('updateClient') : t('createClient')}
            </button>
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cleanForm')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

function validateClientForm(form: ClientFormState, t: Translator): ClientFormErrors {
  const errors: ClientFormErrors = {};
  const name = form.name.trim();
  const email = form.email.trim();
  const currency = form.defaultCurrency.trim().toUpperCase();
  const rate = form.hourlyRate.trim().replace(',', '.');

  if (!name) {
    errors.name = t('clientNameRequired');
  } else if (name.length < 2) {
    errors.name = t('clientNameTooShort');
  }

  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    errors.email = t('clientEmailInvalid');
  }

  if (!/^[A-Z]{3}$/.test(currency)) {
    errors.defaultCurrency = t('clientCurrencyInvalid');
  }

  if (rate && (!/^\d+(\.\d{1,2})?$/.test(rate) || Number(rate) < 0)) {
    errors.hourlyRate = t('clientRateInvalid');
  }

  return errors;
}

function clientFormToInput(form: ClientFormState): ClientInput {
  return {
    name: form.name.trim(),
    email: form.email.trim(),
    taxId: form.taxId.trim(),
    billingAddress: form.billingAddress.trim(),
    defaultCurrency: form.defaultCurrency.trim().toUpperCase() || 'EUR',
    defaultHourlyRateMinor: rateToMinor(form.hourlyRate),
  };
}
