<template>
  <v-dialog
    :model-value="modelValue"
    max-width="720"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="dialog-card settings-dialog-card">
      <div class="settings-dialog-header">
        <div>
          <div class="settings-dialog-title">{{ t('dialogs.settings.title') }}</div>
          <div class="settings-dialog-subtitle">{{ t('dialogs.settings.hint') }}</div>
        </div>
      </div>

      <v-card-text class="settings-dialog-body">
        <section class="settings-section">
          <div class="settings-section-title">{{ t('dialogs.settings.sections.workspace') }}</div>
          <div class="settings-path-field">
            <div class="settings-field-label">{{ t('dialogs.settings.codexHomePath') }}</div>
            <v-text-field
              v-model="path"
              :aria-label="t('dialogs.settings.codexHomePath')"
              hide-details
              placeholder="C:\\Users\\You\\.codex"
            />
          </div>
        </section>

        <section class="settings-section">
          <div class="settings-section-title">{{ t('dialogs.settings.sections.behavior') }}</div>
          <div class="settings-row">
            <div class="settings-row-copy">
              <div class="settings-row-title">{{ t('dialogs.settings.minimizeToTrayOnClose') }}</div>
              <div class="settings-row-description">{{ t('dialogs.settings.minimizeToTrayOnCloseHint') }}</div>
            </div>
            <v-switch
              v-model="minimizeToTray"
              color="primary"
              density="compact"
              hide-details
              inset
              class="settings-row-switch"
              :aria-label="t('dialogs.settings.minimizeToTrayOnClose')"
            />
          </div>
          <div class="settings-row">
            <div class="settings-row-copy">
              <div class="settings-row-title">{{ t('dialogs.settings.restartCodexAfterSwitch') }}</div>
              <div class="settings-row-description">{{ t('dialogs.settings.restartCodexAfterSwitchHint') }}</div>
            </div>
            <v-switch
              v-model="restartCodex"
              color="primary"
              density="compact"
              hide-details
              inset
              class="settings-row-switch"
              :aria-label="t('dialogs.settings.restartCodexAfterSwitch')"
            />
          </div>
        </section>

        <section class="settings-section">
          <div class="settings-section-title">{{ t('dialogs.settings.sections.maintenance') }}</div>
          <div class="settings-row settings-row-retention">
            <div class="settings-row-copy">
              <div class="settings-row-title">{{ t('dialogs.settings.archivedSessionRetentionDays') }}</div>
              <div class="settings-row-description">{{ t('dialogs.settings.archivedSessionRetentionDaysHint') }}</div>
            </div>
            <v-text-field
              v-model.number="archivedSessionRetentionDays"
              type="number"
              min="1"
              step="1"
              hide-details
              class="settings-retention-field"
              :aria-label="t('dialogs.settings.archivedSessionRetentionDays')"
              :suffix="t('dialogs.settings.archivedSessionRetentionDaysSuffix')"
            />
          </div>
        </section>
      </v-card-text>

      <v-card-actions class="dialog-actions settings-dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">
          {{ t('dialogs.confirm.cancel') }}
        </v-btn>
        <v-btn color="primary" :loading="loading" @click="submit">{{ t('common.save') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

import { useI18n } from '../i18n';

type SettingsPayload = {
  codexHomePath: string;
  minimizeToTrayOnClose: boolean;
  restartCodexAfterSwitch: boolean;
  archivedSessionRetentionDays: number;
};

const props = defineProps<{
  modelValue: boolean;
  loading: boolean;
  codexHomePath: string;
  minimizeToTrayOnClose: boolean;
  restartCodexAfterSwitch: boolean;
  archivedSessionRetentionDays: number;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  save: [SettingsPayload];
}>();

const { t } = useI18n();

const path = ref('');
const minimizeToTray = ref(true);
const restartCodex = ref(false);
const archivedSessionRetentionDays = ref(30);

watch(
  () => props.codexHomePath,
  (value) => {
    path.value = value;
  },
  { immediate: true },
);

watch(
  () => props.minimizeToTrayOnClose,
  (value) => {
    minimizeToTray.value = value;
  },
  { immediate: true },
);

watch(
  () => props.restartCodexAfterSwitch,
  (value) => {
    restartCodex.value = value;
  },
  { immediate: true },
);

watch(
  () => props.archivedSessionRetentionDays,
  (value) => {
    archivedSessionRetentionDays.value = normalizeRetentionDays(value);
  },
  { immediate: true },
);

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      path.value = props.codexHomePath;
      minimizeToTray.value = props.minimizeToTrayOnClose;
      restartCodex.value = props.restartCodexAfterSwitch;
      archivedSessionRetentionDays.value = normalizeRetentionDays(props.archivedSessionRetentionDays);
    }
  },
);

function submit() {
  emit('save', {
    codexHomePath: path.value,
    minimizeToTrayOnClose: minimizeToTray.value,
    restartCodexAfterSwitch: restartCodex.value,
    archivedSessionRetentionDays: normalizeRetentionDays(archivedSessionRetentionDays.value),
  });
}

function normalizeRetentionDays(value: number) {
  if (!Number.isFinite(value) || value < 1) {
    return 30;
  }
  return Math.floor(value);
}
</script>

<style scoped>
.settings-dialog-card {
  overflow: hidden;
  border: 1px solid rgba(34, 73, 64, 0.14);
  border-radius: 8px !important;
  background:
    linear-gradient(180deg, rgba(255, 253, 248, 0.98), rgba(248, 243, 236, 0.96)) !important;
  box-shadow: 0 28px 80px rgba(34, 44, 39, 0.18) !important;
}

.settings-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px 17px;
  border-bottom: 1px solid rgba(34, 73, 64, 0.1);
  background:
    linear-gradient(135deg, rgba(255, 252, 246, 0.98), rgba(239, 246, 239, 0.82));
}

.settings-dialog-title {
  color: #17362f;
  font-size: 22px;
  font-weight: 760;
  line-height: 1.1;
}

.settings-dialog-subtitle {
  margin-top: 8px;
  max-width: 560px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.55;
}

.settings-dialog-body {
  display: grid;
  gap: 20px;
  padding: 18px 24px 10px !important;
}

.settings-section {
  display: grid;
  gap: 10px;
}

.settings-section-title {
  color: rgba(25, 52, 45, 0.62);
  font-size: 11px;
  font-weight: 780;
  letter-spacing: 0.11em;
  line-height: 1.2;
  text-transform: uppercase;
}

.settings-path-field {
  display: grid;
  gap: 8px;
}

.settings-field-label,
.settings-row-title {
  color: #17362f;
  font-size: 14px;
  font-weight: 690;
  line-height: 1.35;
}

.settings-path-field :deep(.v-field),
.settings-retention-field :deep(.v-field) {
  border-radius: 8px !important;
  background: rgba(255, 255, 255, 0.56);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.66);
}

.settings-path-field :deep(.v-field__input),
.settings-retention-field :deep(.v-field__input) {
  min-height: 44px;
  color: #20332d;
  font-size: 14px;
}

.settings-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  min-height: 74px;
  padding: 13px 0;
  border-top: 1px solid rgba(34, 73, 64, 0.09);
}

.settings-section-title + .settings-row {
  border-top: 0;
  padding-top: 2px;
}

.settings-row-copy {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.settings-row-description {
  max-width: 520px;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 1.55;
}

.settings-row-switch {
  justify-self: end;
  margin: 0;
}

.settings-row-switch :deep(.v-selection-control) {
  min-height: 34px;
}

.settings-retention-field {
  width: 138px;
  justify-self: end;
}

.settings-retention-field :deep(input) {
  text-align: right;
}

.settings-dialog-actions {
  padding: 14px 24px 20px !important;
  border-top: 1px solid rgba(34, 73, 64, 0.1);
  background: rgba(255, 250, 244, 0.72);
}

.settings-dialog-actions :deep(.v-btn) {
  min-width: 76px;
  height: 36px;
  letter-spacing: 0;
  text-transform: none;
}

@media (max-width: 640px) {
  .settings-dialog-header {
    padding: 20px 18px 15px;
  }

  .settings-dialog-body {
    gap: 16px;
    padding: 16px 18px 8px !important;
  }

  .settings-row,
  .settings-row-retention {
    grid-template-columns: 1fr;
    gap: 10px;
    min-height: 0;
  }

  .settings-row-switch,
  .settings-retention-field {
    justify-self: start;
  }

  .settings-retention-field {
    width: min(100%, 180px);
  }

  .settings-dialog-actions {
    padding: 12px 18px 18px !important;
  }
}
</style>
