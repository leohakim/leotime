import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, CircleCheck, Pencil, Plus, RotateCcw, Save, Tag, Tags, Trash2, X } from 'lucide-react';
import { FormEvent, useState } from 'react';
import {
  archiveTag,
  fetchTagSummary,
  restoreTag,
  updateTag,
  type Tag as TagRecord,
  type TagInput,
} from '../../lib/api';
import { DirectoryInactiveHeading, FieldError, fieldClass, hasErrors } from '../../lib/crudFormUi';
import { confirmDestructiveAction } from '../../lib/destructiveUi';
import { patchTagsCache, refreshOverviewIfOnline } from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { createTag, isLocalId } from '../../lib/offline/mutations';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type TagFormState = {
  name: string;
  color: string;
  active: boolean;
};

type TagFormErrors = Partial<Record<keyof TagFormState | 'form', string>>;

const emptyTagForm: TagFormState = {
  name: '',
  color: '#64748b',
  active: true,
};

export function TagPanel({ isLoading, tags, t }: { isLoading: boolean; tags: TagRecord[]; t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const tagSummaryQuery = useQuery({
    queryKey: ['tag-summary'],
    queryFn: fetchTagSummary,
  });
  const [editingTagId, setEditingTagId] = useState<string | null>(null);
  const [form, setForm] = useState<TagFormState>(emptyTagForm);
  const [errors, setErrors] = useState<TagFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createTag,
    onSuccess: (tag) => {
      setForm(emptyTagForm);
      setErrors({});
      patchTagsCache(queryClient, tag);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(tag.id)) {
        queryClient.invalidateQueries({ queryKey: ['tags'] });
        queryClient.invalidateQueries({ queryKey: ['tag-summary'] });
      }
      toastMutationSuccess(toast, t, 'tagCreated', tag.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      tagId,
      input,
      active,
      wasActive,
    }: {
      tagId: string;
      input: TagInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateTag(tagId, input);
      if (active && !wasActive) {
        await restoreTag(tagId);
      } else if (!active && wasActive) {
        await archiveTag(tagId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingTagId(null);
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['tag-summary'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('tagUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['tag-summary'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('tagArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagArchiveFailed') }));
      toast.error(t('tagArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreTag,
    onSuccess: () => {
      setEditingTagId(null);
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['tag-summary'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('tagRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  function submitTag(event: FormEvent) {
    event.preventDefault();
    const validation = validateTagForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = tagFormToInput(form);
    if (editingTagId) {
      const tag = tags.find((item) => item.id === editingTagId);
      updateMutation.mutate({
        tagId: editingTagId,
        input,
        active: form.active,
        wasActive: !tag?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof TagFormState>(field: K, value: TagFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateTagForm(next, t));
    }
  }

  function startEditing(tag: TagRecord) {
    setEditingTagId(tag.id);
    setErrors({});
    setForm({
      name: tag.name,
      color: tag.color,
      active: !tag.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingTagId(null);
    setForm(emptyTagForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeTags = tags.filter((tag) => !tag.archivedAt);
  const inactiveTags = tags.filter((tag) => tag.archivedAt);
  const summaryInventory = tagSummaryQuery.data
    ? t('tagSummaryInventory')
        .replace('{active}', String(tagSummaryQuery.data.active))
        .replace('{archived}', String(tagSummaryQuery.data.archived))
    : null;

  function renderTagRow(tag: TagRecord, isActive: boolean) {
    return (
      <article
        className={editingTagId === tag.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={tag.id}
      >
        <div className="client-row-main">
          <div className="project-color-dot" style={{ backgroundColor: tag.color }} aria-hidden="true" />
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{tag.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <span className="client-contact">
              <Tag aria-hidden="true" />
              {tag.color}
            </span>
          </div>
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(tag)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              aria-label={t('archive')}
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => {
                if (confirmDestructiveAction(t('archiveTagConfirm'))) {
                  archiveMutation.mutate(tag.id);
                }
              }}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(tag.id)}
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
    <section className="clients-section tags-section" id="tags" aria-labelledby="tags-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Tags aria-hidden="true" />
            {t('tags')}
          </span>
          <h2 id="tags-title">{t('tagDirectory')}</h2>
          <p>{t('tagPanelSubtitle')}</p>
          {summaryInventory ? <p className="tag-summary-inventory">{summaryInventory}</p> : null}
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newTag')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeTags')}</span>
              <strong>{tagSummaryQuery.data?.active ?? activeTags.length}</strong>
            </div>
            <div>
              <span>{t('archivedTags')}</span>
              <strong>{tagSummaryQuery.data?.archived ?? inactiveTags.length}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {tags.length === 0 ? (
              <div className="empty-state">
                <Tags aria-hidden="true" />
                <p>{t('noTags')}</p>
              </div>
            ) : null}
            {activeTags.map((tag) => renderTagRow(tag, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveTags.length} t={t} />
          {inactiveTags.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveTags.map((tag) => renderTagRow(tag, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitTag}>
          <div className="editor-header">
            <div>
              <span>{editingTagId ? t('editingTag') : t('createTag')}</span>
              <h3>{editingTagId ? t('tagFormEdit') : t('tagFormCreate')}</h3>
            </div>
            {editingTagId ? (
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
            <label className={fieldClass(errors.name)} htmlFor="tag-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'tag-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="tag-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('tagNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="tag-name-error" message={errors.name} />
            </label>

            <label className={fieldClass(errors.color)} htmlFor="tag-color">
              <span>
                {t('tagColor')} <em>{t('required')}</em>
              </span>
              <div className="color-input-row">
                <input
                  aria-label={t('tagColor')}
                  onChange={(event) => updateField('color', event.target.value)}
                  type="color"
                  value={form.color}
                />
                <input
                  aria-describedby={errors.color ? 'tag-color-error' : undefined}
                  aria-invalid={Boolean(errors.color)}
                  id="tag-color"
                  onChange={(event) => updateField('color', event.target.value)}
                  placeholder={t('tagColorPlaceholder')}
                  value={form.color}
                />
              </div>
              <FieldError id="tag-color-error" message={errors.color} />
            </label>

            {editingTagId ? (
              <label className="client-active-field" htmlFor="tag-active">
                <input
                  checked={form.active}
                  id="tag-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('tagActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingTagId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingTagId ? t('updateTag') : t('createTag')}
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

function validateTagForm(form: TagFormState, t: Translator): TagFormErrors {
  const errors: TagFormErrors = {};
  const name = form.name.trim();
  const color = form.color.trim();

  if (!name) {
    errors.name = t('tagNameRequired');
  } else if (name.length < 2) {
    errors.name = t('tagNameTooShort');
  }

  if (!/^#[0-9a-fA-F]{6}$/.test(color)) {
    errors.color = t('tagColorInvalid');
  }

  return errors;
}

function tagFormToInput(form: TagFormState): TagInput {
  return {
    name: form.name.trim(),
    color: form.color.trim() || '#64748b',
  };
}
