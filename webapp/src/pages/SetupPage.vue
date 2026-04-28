<template>
  <div class="setup-page" :class="{ 'manage-mode': isManageMode }">
    <div v-if="loading" class="setup-loading">
      <div class="setup-loading-card">
        <div class="setup-loading-spinner" aria-hidden="true"></div>
        <div class="setup-loading-text">{{ t('common.loading') }}</div>
      </div>
    </div>

    <div v-else-if="loadError" class="setup-loading">
      <div class="setup-loading-card">
        <div class="setup-loading-text">{{ loadError }}</div>
        <button class="setup-primary-btn" type="button" @click="loadConfig">
          {{ t('common.retry') }}
        </button>
      </div>
    </div>

    <div v-else class="setup-surface">
      <div v-if="!isManageMode" class="setup-lang">
        <div class="sidebar-language-compact" role="group" :aria-label="t('app.sidebar.language')" :key="currentLocale">
          <button
            v-for="option in languageOptions"
            :key="option.value"
            class="sidebar-language-btn"
            :class="{ active: option.value === currentLocale }"
            type="button"
            :aria-pressed="option.value === currentLocale"
            :aria-label="option.label"
            @click="currentLocale = option.value"
          >
            <i :class="['language-icon', option.icon]" aria-hidden="true"></i>
            <span>{{ option.shortLabel }}</span>
          </button>
        </div>
      </div>
      <div class="setup-grid">
        <aside class="setup-rail">
        <a
          v-if="!isManageMode"
          class="setup-brand brand"
          href="https://github.com/likaia/nginxpulse/"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Open NginxPulse GitHub repository"
        >
          <div class="brand-mark" aria-hidden="true">
            <span class="brand-initials">NP</span>
            <svg class="brand-pulse" viewBox="0 0 32 16" role="presentation" aria-hidden="true">
              <path
                d="M1 8H7L10 3L14 13L18 8H31"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              ></path>
            </svg>
          </div>
          <div class="brand-text">
            <div class="brand-title">NginxPulse</div>
            <div class="brand-sub">{{ t(subtitleKey) }}</div>
          </div>
        </a>
        <ol class="setup-steps" role="list">
          <li
            v-for="(step, index) in steps"
            :key="step.key"
            class="setup-step"
            :class="{ active: currentStep === index, done: currentStep > index }"
          >
            <span class="setup-step-index">{{ index + 1 }}</span>
            <div class="setup-step-text">
              <div class="setup-step-title">{{ step.title }}</div>
              <div class="setup-step-desc">{{ step.desc }}</div>
            </div>
          </li>
        </ol>
        </aside>

        <section class="setup-content">
          <div v-if="configReadonly" class="setup-alert warning setup-readonly-banner">
            <div class="setup-alert-title">{{ t('setup.readOnlyTitle') }}</div>
            <div class="setup-hint">{{ t('setup.readOnly') }}</div>
            <div class="setup-hint">{{ t('setup.readOnlyEnvHint') }}</div>
            <div class="setup-inline-actions">
              <button class="ghost-button" type="button" @click="copyConfig">
                {{ t('setup.actions.copyConfig') }}
              </button>
              <span v-if="copyStatus" class="setup-hint">{{ copyStatus }}</span>
            </div>
          </div>
          <div class="setup-scroll">
            <div v-if="currentStepErrors.length" class="setup-toast" role="alert" aria-live="assertive">
              <div class="setup-toast-title">
                <i class="ri-error-warning-line" aria-hidden="true"></i>
                <span>{{ t('setup.validationTitle') }}</span>
                <span class="setup-toast-count">{{ currentStepErrors.length }}</span>
              </div>
              <div class="setup-toast-desc">{{ currentStepErrors[0].message }}</div>
              <div v-if="currentStepErrors.length > 1" class="setup-toast-more">
                +{{ currentStepErrors.length - 1 }}
              </div>
            </div>
            <transition name="setup-fade" mode="out-in">
              <div :key="currentStep" class="card setup-card" data-anim>
            <header class="setup-card-header">
              <div>
                <div class="setup-card-title">{{ steps[currentStep].title }}</div>
                <div class="setup-card-sub">{{ steps[currentStep].desc }}</div>
              </div>
              <div class="setup-card-chip">{{ t('setup.stepLabel', { value: currentStep + 1, total: steps.length }) }}</div>
            </header>

            <div v-if="currentStep === 0" class="setup-section">
              <div
                v-for="(site, index) in websiteDrafts"
                :key="`site-${index}`"
                class="setup-site-card"
              >
                <div class="setup-site-header">
                  <div class="setup-site-title">{{ t('setup.websiteBlock', { value: index + 1 }) }}</div>
                  <button
                    v-if="websiteDrafts.length > 1"
                    class="ghost-button"
                    type="button"
                    @click="removeWebsite(index)"
                  >
                    {{ t('setup.actions.remove') }}
                  </button>
                </div>
                <div class="setup-field-grid">
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.websiteName') }}</label>
                    <input v-model.trim="site.name" class="setup-input" type="text" />
                    <div v-if="fieldError(`websites[${index}].name`)" class="setup-error">
                      {{ fieldError(`websites[${index}].name`) }}
                    </div>
                  </div>
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.domains') }}</label>
                    <input v-model.trim="site.domainsInput" class="setup-input" type="text" :placeholder="t('setup.placeholders.domains')" />
                  </div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.logPath') }}</label>
                  <input v-model.trim="site.logPath" class="setup-input" type="text" :placeholder="t('setup.placeholders.logPath')" />
                  <div class="setup-hint">{{ t('setup.hints.logPath') }}</div>
                  <div v-if="fieldError(`websites[${index}].logPath`)" class="setup-error">
                    {{ fieldError(`websites[${index}].logPath`) }}
                  </div>
                </div>

                <button
                  class="setup-advanced-toggle"
                  type="button"
                  :aria-expanded="advancedOpen.website[index]"
                  @click="toggleWebsiteAdvanced(index)"
                >
                  <span>{{ advancedOpen.website[index] ? t('setup.actions.collapse') : t('setup.actions.advanced') }}</span>
                  <i class="ri-arrow-down-s-line" :class="{ flipped: advancedOpen.website[index] }" aria-hidden="true"></i>
                </button>

                <div v-if="advancedOpen.website[index]" class="setup-advanced">
                  <div class="setup-field-grid">
                    <div class="setup-field">
                      <label class="setup-label">{{ t('setup.fields.logType') }}</label>
                      <Dropdown
                        v-model="site.logType"
                        class="setup-dropdown"
                        :options="logTypeOptionsFor(site.logType)"
                        optionLabel="label"
                        optionValue="value"
                        :placeholder="t('setup.placeholders.logType')"
                      />
                    </div>
                    <div class="setup-field">
                      <label class="setup-label">{{ t('setup.fields.timeLayout') }}</label>
                      <input
                        v-model.trim="site.timeLayout"
                        class="setup-input"
                        type="text"
                        :placeholder="t('setup.placeholders.timeLayout')"
                      />
                    </div>
                  </div>
                  <div class="setup-feature-card setup-feature-card-compact">
                    <div class="setup-feature-icon">
                      <i class="ri-radar-line" aria-hidden="true"></i>
                    </div>
                    <div class="setup-feature-body">
                      <div class="setup-feature-title">{{ t('setup.fields.autoDiscoverHosts') }}</div>
                      <div class="setup-feature-desc">{{ t('setup.hints.autoDiscoverHosts') }}</div>
                    </div>
                    <button
                      class="setup-switch"
                      type="button"
                      :class="{ active: site.autoDiscoverHosts }"
                      :aria-pressed="site.autoDiscoverHosts"
                      @click="site.autoDiscoverHosts = !site.autoDiscoverHosts"
                    >
                      <span class="setup-switch-dot"></span>
                    </button>
                  </div>
                  <div class="setup-parse-group">
                    <div class="setup-parse-head">
                      <div class="setup-parse-title">{{ t('setup.logValidation.groupTitle') }}</div>
                      <div class="setup-parse-actions">
                        <span
                          class="setup-parse-status"
                          :class="`is-${site.logValidationStatus}`"
                          :title="site.logValidationMessage"
                        >
                          <span class="setup-parse-status-dot" aria-hidden="true"></span>
                          {{ logValidationStatusLabel(site.logValidationStatus) }}
                        </span>
                        <button
                          class="ghost-button setup-validate-btn"
                          type="button"
                          :disabled="!canValidateLogRule(site)"
                          :title="canValidateLogRule(site) ? '' : t('setup.logValidation.needRule')"
                          @click="openLogValidation(index)"
                        >
                          {{ t('setup.actions.validateLogRule') }}
                        </button>
                      </div>
                    </div>
                    <div class="setup-field">
                      <label class="setup-label">{{ t('setup.fields.logFormat') }}</label>
                      <input
                        v-model.trim="site.logFormat"
                        class="setup-input"
                        type="text"
                        :placeholder="t('setup.placeholders.logFormat')"
                        @input="markLogValidationDirty(site)"
                      />
                    </div>
                    <div class="setup-field">
                      <label class="setup-label">{{ t('setup.fields.logRegex') }}</label>
                      <input
                        v-model.trim="site.logRegex"
                        class="setup-input"
                        type="text"
                        :placeholder="t('setup.placeholders.logRegex')"
                        @input="markLogValidationDirty(site)"
                      />
                    </div>
                  </div>
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.sourcesJson') }}</label>
                    <textarea v-model.trim="site.sourcesJson" class="setup-textarea" rows="6" :placeholder="t('setup.placeholders.sourcesJson')"></textarea>
                    <div class="setup-hint">{{ t('setup.hints.sourcesJson') }}</div>
                    <div v-if="fieldError(`websites[${index}].sources`)" class="setup-error">
                      {{ fieldError(`websites[${index}].sources`) }}
                    </div>
                  </div>
                  <div class="setup-field setup-toggle">
                    <label class="setup-label">{{ t('setup.fields.whitelistEnable') }}</label>
                    <button
                      class="setup-switch"
                      type="button"
                      :class="{ active: site.whitelistEnabled }"
                      :aria-pressed="site.whitelistEnabled"
                      @click="site.whitelistEnabled = !site.whitelistEnabled"
                    >
                      <span class="setup-switch-dot"></span>
                    </button>
                  </div>
                  <div v-if="fieldError(`websites[${index}].whitelist`)" class="setup-error">
                    {{ fieldError(`websites[${index}].whitelist`) }}
                  </div>
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.whitelistIps') }}</label>
                    <textarea v-model.trim="site.whitelistIPsText" class="setup-textarea" rows="3" :placeholder="t('setup.placeholders.whitelistIps')"></textarea>
                    <div class="setup-hint">{{ t('setup.hints.whitelistIps') }}</div>
                    <div v-if="fieldError(`websites[${index}].whitelist.ips`)" class="setup-error">
                      {{ fieldError(`websites[${index}].whitelist.ips`) }}
                    </div>
                  </div>
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.whitelistCities') }}</label>
                    <textarea v-model.trim="site.whitelistCitiesText" class="setup-textarea" rows="3" :placeholder="t('setup.placeholders.whitelistCities')"></textarea>
                  </div>
                  <div class="setup-field setup-toggle">
                    <label class="setup-label">{{ t('setup.fields.whitelistNonMainland') }}</label>
                    <button
                      class="setup-switch"
                      type="button"
                      :class="{ active: site.whitelistNonMainland }"
                      :aria-pressed="site.whitelistNonMainland"
                      @click="site.whitelistNonMainland = !site.whitelistNonMainland"
                    >
                      <span class="setup-switch-dot"></span>
                    </button>
                  </div>
                </div>
              </div>

              <button class="ghost-button setup-add-btn" type="button" @click="addWebsite">
                <i class="ri-add-line" aria-hidden="true"></i>
                {{ t('setup.actions.addWebsite') }}
              </button>
            </div>

            <div v-else-if="currentStep === 1" class="setup-section">
              <div class="setup-field">
                <label class="setup-label">{{ t('setup.fields.databaseDsn') }}</label>
                <input
                  v-model.trim="databaseDraft.dsn"
                  class="setup-input"
                  type="text"
                  :placeholder="t('setup.placeholders.databaseDsn', { at: '@' })"
                />
                <div v-if="fieldError('database.dsn')" class="setup-error">
                  {{ fieldError('database.dsn') }}
                </div>
              </div>

              <button
                class="setup-advanced-toggle"
                type="button"
                :aria-expanded="advancedOpen.database"
                @click="advancedOpen.database = !advancedOpen.database"
              >
                <span>{{ advancedOpen.database ? t('setup.actions.collapse') : t('setup.actions.advanced') }}</span>
                <i class="ri-arrow-down-s-line" :class="{ flipped: advancedOpen.database }" aria-hidden="true"></i>
              </button>

              <div v-if="advancedOpen.database" class="setup-advanced">
                <div class="setup-field-grid">
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.dbMaxOpen') }}</label>
                    <input v-model.trim="databaseDraft.maxOpenConns" class="setup-input" type="number" min="0" />
                  </div>
                  <div class="setup-field">
                    <label class="setup-label">{{ t('setup.fields.dbMaxIdle') }}</label>
                    <input v-model.trim="databaseDraft.maxIdleConns" class="setup-input" type="number" min="0" />
                  </div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.dbConnLifetime') }}</label>
                  <input v-model.trim="databaseDraft.connMaxLifetime" class="setup-input" type="text" />
                </div>
              </div>
            </div>

            <div v-else-if="currentStep === 2" class="setup-section">
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.serverPort') }}</label>
                  <input v-model.trim="serverPort" class="setup-input" type="text" :placeholder="t('setup.placeholders.serverPort')" />
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.webBasePath') }}</label>
                  <input
                    v-model.trim="systemDraft.webBasePath"
                    class="setup-input"
                    type="text"
                    :placeholder="t('setup.placeholders.webBasePath')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.webBasePath') }}</div>
                  <div v-if="fieldError('system.webBasePath')" class="setup-error">
                    {{ fieldError('system.webBasePath') }}
                  </div>
                </div>
              </div>
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.taskInterval') }}</label>
                  <input v-model.trim="systemDraft.taskInterval" class="setup-input" type="text" />
                  <div class="setup-hint">{{ t('setup.hints.taskInterval') }}</div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.backfillMaxDurationPerRun') }}</label>
                  <input
                    v-model.trim="systemDraft.backfillMaxDurationPerRun"
                    class="setup-input"
                    type="text"
                    :placeholder="t('setup.placeholders.backfillMaxDurationPerRun')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.backfillMaxDurationPerRun') }}</div>
                </div>
              </div>
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.parseBatchSize') }}</label>
                  <input v-model.trim="systemDraft.parseBatchSize" class="setup-input" type="number" min="1" />
                  <div class="setup-hint">{{ t('setup.hints.parseBatchSize') }}</div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.backfillMaxBytesPerRun') }}</label>
                  <input
                    v-model.trim="systemDraft.backfillMaxBytesPerRun"
                    class="setup-input"
                    type="number"
                    min="1"
                    :placeholder="t('setup.placeholders.backfillMaxBytesPerRun')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.backfillMaxBytesPerRun') }}</div>
                </div>
              </div>
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.logRetentionDays') }}</label>
                  <input v-model.trim="systemDraft.logRetentionDays" class="setup-input" type="number" min="1" />
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.ipGeoCacheLimit') }}</label>
                  <input v-model.trim="systemDraft.ipGeoCacheLimit" class="setup-input" type="number" min="1" />
                </div>
              </div>
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.httpSourceTimeout') }}</label>
                  <input
                    v-model.trim="systemDraft.httpSourceTimeout"
                    class="setup-input"
                    type="text"
                    :placeholder="t('setup.placeholders.httpSourceTimeout')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.httpSourceTimeout') }}</div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.language') }}</label>
                  <Dropdown
                    v-model="systemDraft.language"
                    class="setup-dropdown"
                    :options="languageOptions"
                    optionLabel="label"
                    optionValue="value"
                  />
                </div>
              </div>
              <div class="setup-field-grid">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.accessKeys') }}</label>
                  <input v-model.trim="systemDraft.accessKeysText" class="setup-input" type="text" :placeholder="t('setup.placeholders.accessKeys')" />
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.accessKeyExpireDays') }}</label>
                  <input
                    v-model.trim="systemDraft.accessKeyExpireDays"
                    class="setup-input"
                    type="number"
                    min="1"
                    :placeholder="t('setup.placeholders.accessKeyExpireDays')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.accessKeyExpireDays') }}</div>
                </div>
              </div>
              <div class="setup-feature-grid">
                <div class="setup-feature-card">
                  <div class="setup-feature-icon">
                    <i class="ri-rocket-2-line" aria-hidden="true"></i>
                  </div>
                  <div class="setup-feature-body">
                    <div class="setup-feature-title">{{ t('setup.fields.demoMode') }}</div>
                    <div class="setup-feature-desc">{{ t('setup.hints.demoMode') }}</div>
                  </div>
                  <button
                    class="setup-switch"
                    type="button"
                    :class="{ active: systemDraft.demoMode }"
                    :aria-pressed="systemDraft.demoMode"
                    @click="systemDraft.demoMode = !systemDraft.demoMode"
                  >
                    <span class="setup-switch-dot"></span>
                  </button>
                </div>
	                <div class="setup-feature-card">
	                  <div class="setup-feature-icon">
	                    <i class="ri-smartphone-line" aria-hidden="true"></i>
	                  </div>
                  <div class="setup-feature-body">
                    <div class="setup-feature-title">{{ t('setup.fields.mobilePwaEnabled') }}</div>
                    <div class="setup-feature-desc">{{ t('setup.hints.mobilePwaEnabled') }}</div>
                  </div>
                  <button
                    class="setup-switch"
                    type="button"
                    :class="{ active: systemDraft.mobilePwaEnabled }"
                    :aria-pressed="systemDraft.mobilePwaEnabled"
                    @click="systemDraft.mobilePwaEnabled = !systemDraft.mobilePwaEnabled"
                  >
                    <span class="setup-switch-dot"></span>
	                  </button>
	                </div>
	              </div>

	              <div class="setup-feature-card setup-feature-card-wide">
	                <div class="setup-feature-icon">
	                  <i class="ri-server-line" aria-hidden="true"></i>
	                </div>
	                <div class="setup-feature-body">
	                  <div class="setup-feature-title">{{ t('setup.fields.serverStatusEnabled') }}</div>
	                  <div class="setup-feature-desc">{{ t('setup.hints.serverStatusEnabled') }}</div>
	                </div>
	                <button
	                  class="setup-switch"
	                  type="button"
	                  :class="{ active: systemDraft.serverStatusEnabled }"
	                  :aria-pressed="systemDraft.serverStatusEnabled"
	                  @click="systemDraft.serverStatusEnabled = !systemDraft.serverStatusEnabled"
	                >
	                  <span class="setup-switch-dot"></span>
	                </button>
	              </div>

	              <div v-if="systemDraft.serverStatusEnabled" class="setup-advanced setup-server-status-fields">
	                <div class="setup-field-grid">
	                  <div class="setup-field">
	                    <label class="setup-label">{{ t('setup.fields.serverStatusMetricsUrl') }}</label>
	                    <input
	                      v-model.trim="systemDraft.serverStatusMetricsUrl"
	                      class="setup-input"
	                      type="url"
	                      :placeholder="t('setup.placeholders.serverStatusMetricsUrl')"
	                    />
	                    <div class="setup-hint">{{ t('setup.hints.serverStatusMetricsUrl') }}</div>
	                    <div v-if="fieldError('system.serverStatus.metricsUrl')" class="setup-error">
	                      {{ fieldError('system.serverStatus.metricsUrl') }}
	                    </div>
	                  </div>
	                  <div class="setup-field">
	                    <label class="setup-label">{{ t('setup.fields.serverStatusDisksUrl') }}</label>
	                    <input
	                      v-model.trim="systemDraft.serverStatusDisksUrl"
	                      class="setup-input"
	                      type="url"
	                      :placeholder="t('setup.placeholders.serverStatusDisksUrl')"
	                    />
	                    <div class="setup-hint">{{ t('setup.hints.serverStatusDisksUrl') }}</div>
	                    <div v-if="fieldError('system.serverStatus.disksUrl')" class="setup-error">
	                      {{ fieldError('system.serverStatus.disksUrl') }}
	                    </div>
	                  </div>
	                </div>
	                <div class="setup-field-grid">
	                  <div class="setup-field">
	                    <label class="setup-label">{{ t('setup.fields.serverStatusTimeout') }}</label>
	                    <input
	                      v-model.trim="systemDraft.serverStatusTimeout"
	                      class="setup-input"
	                      type="text"
	                      :placeholder="t('setup.placeholders.serverStatusTimeout')"
	                    />
	                    <div class="setup-hint">{{ t('setup.hints.serverStatusTimeout') }}</div>
	                    <div v-if="fieldError('system.serverStatus.timeout')" class="setup-error">
	                      {{ fieldError('system.serverStatus.timeout') }}
	                    </div>
	                  </div>
	                  <div class="setup-field">
	                    <label class="setup-label">{{ t('setup.fields.serverStatusRefreshInterval') }}</label>
	                    <input
	                      v-model.trim="systemDraft.serverStatusRefreshInterval"
	                      class="setup-input"
	                      type="text"
	                      :placeholder="t('setup.placeholders.serverStatusRefreshInterval')"
	                    />
	                    <div class="setup-hint">{{ t('setup.hints.serverStatusRefreshInterval') }}</div>
	                    <div v-if="fieldError('system.serverStatus.refreshInterval')" class="setup-error">
	                      {{ fieldError('system.serverStatus.refreshInterval') }}
	                    </div>
	                  </div>
	                </div>
	              </div>

	              <button
                class="setup-advanced-toggle"
                type="button"
                :aria-expanded="advancedOpen.system"
                @click="advancedOpen.system = !advancedOpen.system"
              >
                <span>{{ advancedOpen.system ? t('setup.actions.collapse') : t('setup.actions.advanced') }}</span>
                <i class="ri-arrow-down-s-line" :class="{ flipped: advancedOpen.system }" aria-hidden="true"></i>
              </button>

              <div v-if="advancedOpen.system" class="setup-advanced">
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.logDestination') }}</label>
                  <input v-model.trim="systemDraft.logDestination" class="setup-input" type="text" />
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.alertPushJson') }}</label>
                  <textarea
                    v-model.trim="systemDraft.alertPushJson"
                    class="setup-textarea"
                    rows="8"
                    :placeholder="t('setup.placeholders.alertPushJson')"
                  ></textarea>
                  <div class="setup-hint">{{ t('setup.hints.alertPushJson') }}</div>
                  <div v-if="fieldError('system.alertPush')" class="setup-error">
                    {{ fieldError('system.alertPush') }}
                  </div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.alertPushTestMessage') }}</label>
                  <input
                    v-model.trim="alertPushTestMessage"
                    class="setup-input"
                    type="text"
                    :placeholder="t('setup.placeholders.alertPushTestMessage')"
                  />
                  <div class="setup-hint">{{ t('setup.hints.alertPushTestMessage') }}</div>
                  <div class="setup-inline-actions">
                    <button
                      class="ghost-button"
                      type="button"
                      :disabled="alertPushTesting"
                      @click="runAlertPushTest"
                    >
                      {{ alertPushTesting ? t('setup.actions.testingAlertPush') : t('setup.actions.testAlertPush') }}
                    </button>
                  </div>
                  <div v-if="alertPushTestError" class="setup-error">
                    {{ alertPushTestError }}
                  </div>
                </div>
                <div v-if="alertPushTestResult" class="setup-alert" :class="alertPushTestResult.success ? 'success' : 'warning'">
                  <div class="setup-alert-title">
                    {{ t('setup.alertPushTest.summary', { success: alertPushTestResult.succeeded, total: alertPushTestResult.tested }) }}
                  </div>
                  <ul class="setup-alert-list">
                    <li v-for="(item, channel) in alertPushTestResult.results" :key="`alert-test-${channel}`">
                      {{ alertChannelLabel(channel) }}: {{ item.success ? t('setup.alertPushTest.ok') : (item.error || t('setup.alertPushTest.failed')) }}
                    </li>
                  </ul>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.statusCodeInclude') }}</label>
                  <input v-model.trim="pvDraft.statusCodeIncludeText" class="setup-input" type="text" :placeholder="t('setup.placeholders.statusCodeInclude')" />
                  <div v-if="fieldError('pvFilter.statusCodeInclude')" class="setup-error">
                    {{ fieldError('pvFilter.statusCodeInclude') }}
                  </div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.excludePatterns') }}</label>
                  <textarea v-model.trim="pvDraft.excludePatternsText" class="setup-textarea" rows="4"></textarea>
                  <div v-if="fieldError('pvFilter.excludePatterns')" class="setup-error">
                    {{ fieldError('pvFilter.excludePatterns') }}
                  </div>
                </div>
                <div class="setup-field">
                  <label class="setup-label">{{ t('setup.fields.excludeIps') }}</label>
                  <textarea v-model.trim="pvDraft.excludeIPsText" class="setup-textarea" rows="3"></textarea>
                </div>
              </div>
            </div>

            <div v-else class="setup-section">
              <div class="setup-review">
                <div class="setup-review-block">
                  <div class="setup-review-title">{{ t('setup.review.summary') }}</div>
                  <div class="setup-review-item" v-for="(site, index) in websiteDrafts" :key="`review-${index}`">
                    <div class="setup-review-label">{{ site.name || t('setup.review.unnamed') }}</div>
                    <div class="setup-review-value">{{ site.logPath || t('setup.review.noPath') }}</div>
                  </div>
                </div>
                <div class="setup-review-block">
                  <div class="setup-review-title">{{ t('setup.review.database') }}</div>
                  <div class="setup-review-value">{{ databaseDraft.dsn || t('setup.review.emptyDsn') }}</div>
                </div>
              </div>

              <div v-if="validationWarnings.length" class="setup-alert warning">
                <div class="setup-alert-title">{{ t('setup.warningTitle') }}</div>
                <ul class="setup-alert-list">
                  <li v-for="(item, idx) in validationWarnings" :key="`${item.field}-warn-${idx}`">
                    {{ item.message }}
                  </li>
                </ul>
              </div>

              <div class="setup-field">
                <label class="setup-label">{{ t('setup.review.jsonPreview') }}</label>
                <textarea class="setup-textarea" rows="10" readonly :value="configPreview"></textarea>
              </div>

              <div v-if="saveSuccess" class="setup-alert success">
                <div class="setup-alert-title">{{ t(savedLabelKey) }}</div>
                <div class="setup-hint">{{ t(restartHintKey) }}</div>
                <div v-if="autoRefreshSeconds > 0" class="setup-hint">
                  {{ t('setup.autoRefreshHint', { seconds: autoRefreshSeconds }) }}
                </div>
              </div>
              <div v-if="saveError" class="setup-alert">
                <div class="setup-alert-title">{{ t('common.requestFailed') }}</div>
                <div class="setup-hint">{{ saveError }}</div>
              </div>
            </div>
              </div>
            </transition>
          </div>

        <div class="setup-footer">
          <button
            class="ghost-button"
            type="button"
            :disabled="currentStep === 0 || saving || nextLoading"
            @click="prevStep"
          >
            {{ t('setup.actions.prev') }}
          </button>
          <button
            v-if="currentStep < steps.length - 1"
            class="setup-primary-btn"
            type="button"
            :disabled="saving || nextLoading"
            @click="nextStep"
          >
            {{ nextLoading ? t('setup.actions.nexting') : t('setup.actions.next') }}
          </button>
          <button
            v-else
            class="setup-primary-btn"
            type="button"
            :disabled="saving || configReadonly"
            @click="saveAll"
          >
            {{ saving ? t('setup.actions.saving') : t(saveLabelKey) }}
          </button>
        </div>
        <div v-if="configReadonly" class="setup-readonly">
          {{ t('setup.readOnly') }}
        </div>
        </section>
      </div>
    </div>
  </div>

  <Dialog
    v-model:visible="logValidationVisible"
    :header="t('setup.logValidation.title')"
    modal
    :draggable="false"
    class="setup-log-validate-dialog"
  >
    <div class="setup-log-validate-sub">
      {{ t('setup.logValidation.subtitle', { value: logValidationSiteLabel }) }}
    </div>
    <div class="setup-log-validate-body">
      <div class="setup-log-validate-input">
        <label class="setup-label">{{ t('setup.logValidation.sampleLabel') }}</label>
        <textarea
          v-model.trim="logValidationSample"
          class="setup-textarea setup-log-validate-textarea"
          rows="7"
          :placeholder="t('setup.logValidation.samplePlaceholder')"
        ></textarea>
        <div class="setup-hint">{{ t('setup.logValidation.sampleHint') }}</div>
      </div>
      <div class="setup-log-validate-result">
        <div class="setup-log-validate-summary" :class="`is-${logValidationResult.status}`">
          <span class="setup-parse-status-dot" aria-hidden="true"></span>
          <span>{{ logValidationResultLabel }}</span>
        </div>
        <div v-if="logValidationLoading" class="setup-log-validate-loading">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="logValidationResult.status === 'error'" class="setup-log-validate-error">
          {{ logValidationResult.message }}
        </div>
        <div v-else-if="logValidationResult.status === 'success'" class="setup-log-validate-success">
          <div class="setup-log-validate-source">
            {{ logValidationResult.source === 'logRegex'
              ? t('setup.logValidation.sourceRegex')
              : t('setup.logValidation.sourceFormat') }}
          </div>
          <div class="setup-log-validate-line">
            <span class="setup-log-validate-line-label">{{ t('setup.logValidation.matchedLine') }}</span>
            <code>{{ logValidationResult.matchedLine }}</code>
          </div>
          <div class="setup-log-validate-table-wrap">
            <table class="setup-log-validate-table">
              <thead>
                <tr>
                  <th>{{ t('setup.logValidation.field') }}</th>
                  <th>{{ t('setup.logValidation.value') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in logValidationResult.fields" :key="item.name">
                  <td class="setup-log-validate-key">{{ item.name }}</td>
                  <td class="setup-log-validate-value">{{ item.value }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="setup-log-validate-empty">
          {{ t('setup.logValidation.resultIdle') }}
        </div>
      </div>
    </div>
    <div class="setup-log-validate-actions">
      <button class="ghost-button" type="button" @click="logValidationVisible = false">
        {{ t('common.close') }}
      </button>
      <button class="setup-primary-btn" type="button" :disabled="logValidationLoading" @click="runLogValidation">
        {{ logValidationLoading ? t('common.loading') : t('setup.logValidation.run') }}
      </button>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import Dialog from 'primevue/dialog';
import Dropdown from 'primevue/dropdown';
import { fetchConfig, restartSystem, saveConfig, testAlertPush, validateConfig } from '@/api';
import { normalizeLocale, setLocale } from '@/i18n';
import type { AlertPushTestResponse, ConfigPayload, FieldError, SourceConfig } from '@/api/types';

type LogValidationStatus = 'idle' | 'success' | 'error';

type LogValidationResult = {
  status: LogValidationStatus;
  message: string;
  fields: Array<{ name: string; value: string }>;
  matchedLine: string;
  source: '' | 'logRegex' | 'logFormat';
};

interface WebsiteDraft {
  name: string;
  logPath: string;
  domainsInput: string;
  logType: string;
  logFormat: string;
  logRegex: string;
  autoDiscoverHosts: boolean;
  logValidationStatus: LogValidationStatus;
  logValidationMessage: string;
  timeLayout: string;
  sourcesJson: string;
  whitelistEnabled: boolean;
  whitelistIPsText: string;
  whitelistCitiesText: string;
  whitelistNonMainland: boolean;
}

const props = withDefaults(defineProps<{ mode?: 'setup' | 'manage' }>(), {
  mode: 'setup',
});

const { t, locale } = useI18n({ useScope: 'global' });

const isManageMode = computed(() => props.mode === 'manage');
const subtitleKey = computed(() => (isManageMode.value ? 'setup.manageSubtitle' : 'setup.subtitle'));
const saveLabelKey = computed(() =>
  isManageMode.value ? 'setup.actions.saveManage' : 'setup.actions.save'
);
const savedLabelKey = computed(() =>
  isManageMode.value ? 'setup.actions.savedManage' : 'setup.actions.saved'
);
const restartHintKey = computed(() =>
  isManageMode.value ? 'setup.restartManageHint' : 'setup.restartDockerHint'
);

const languageOptions = computed(() => {
  const _locale = locale.value;
  return [
    { value: 'zh-CN', label: t('language.zh'), shortLabel: t('language.zhShort'), icon: 'ri-translate-2' },
    { value: 'en-US', label: t('language.en'), shortLabel: t('language.enShort'), icon: 'ri-global-line' },
  ];
});

const currentLocale = computed({
  get: () => normalizeLocale(locale.value),
  set: (value: string) => setLocale(normalizeLocale(value)),
});

const steps = computed(() => [
  {
    key: 'website',
    title: t('setup.steps.website.title'),
    desc: t('setup.steps.website.desc'),
  },
  {
    key: 'database',
    title: t('setup.steps.database.title'),
    desc: t('setup.steps.database.desc'),
  },
  {
    key: 'system',
    title: t('setup.steps.system.title'),
    desc: t('setup.steps.system.desc'),
  },
  {
    key: 'review',
    title: t('setup.steps.review.title'),
    desc: t('setup.steps.review.desc'),
  },
]);

const currentStep = ref(0);
const loading = ref(true);
const loadError = ref('');
const saving = ref(false);
const saveSuccess = ref(false);
const saveError = ref('');
const nextLoading = ref(false);
const autoRefreshSeconds = ref(0);
let autoRefreshTimer: number | null = null;
const AUTO_REFRESH_SECONDS = 5;
const validationErrors = ref<FieldError[]>([]);
const validationWarnings = ref<FieldError[]>([]);
const fieldErrors = ref<Record<string, string>>({});
const configReadonly = ref(false);
const copyStatus = ref('');
const defaultLogPath = ref('');
const alertPushTestMessage = ref('');
const alertPushTesting = ref(false);
const alertPushTestResult = ref<AlertPushTestResponse | null>(null);
const alertPushTestError = ref('');
let copyStatusTimer: number | null = null;

const serverPort = ref(':8089');
const databaseDraft = reactive({
  driver: 'postgres',
  dsn: '',
  maxOpenConns: '10',
  maxIdleConns: '5',
  connMaxLifetime: '30m',
});
const systemDraft = reactive({
  logDestination: 'file',
  taskInterval: '1m',
  backfillMaxDurationPerRun: '8s',
  backfillMaxBytesPerRun: String(32 * 1024 * 1024),
  httpSourceTimeout: '2m',
  logRetentionDays: '30',
  parseBatchSize: '100',
  ipGeoCacheLimit: '1000000',
  alertPushJson: '',
  demoMode: false,
  mobilePwaEnabled: false,
  serverStatusEnabled: false,
  serverStatusMockEnabled: false,
  serverStatusMetricsUrl: '',
  serverStatusDisksUrl: '',
  serverStatusTimeout: '5s',
  serverStatusRefreshInterval: '30s',
  accessKeysText: '',
  accessKeyExpireDays: '7',
  language: 'zh-CN',
  webBasePath: '',
});
const pvDraft = reactive({
  statusCodeIncludeText: '',
  excludePatternsText: '',
  excludeIPsText: '',
});

const websiteDrafts = ref<WebsiteDraft[]>([createWebsiteDraft()]);
const advancedOpen = reactive<{ website: Record<number, boolean>; database: boolean; system: boolean }>({
  website: {},
  database: false,
  system: false,
});

const currentStepErrors = computed(() => filterErrorsForStep(validationErrors.value, currentStep.value));

const configPreview = computed(() => {
  const { config } = buildConfig(false);
  return JSON.stringify(config, null, 2);
});

const logValidationVisible = ref(false);
const logValidationIndex = ref<number | null>(null);
const logValidationSample = ref('');
const logValidationLoading = ref(false);
const logValidationResult = ref<LogValidationResult>(createLogValidationResult());

const activeLogValidationSite = computed(() => {
  if (logValidationIndex.value === null) {
    return null;
  }
  return websiteDrafts.value[logValidationIndex.value] || null;
});

const logValidationSiteLabel = computed(() => {
  if (logValidationIndex.value === null) {
    return '';
  }
  const site = activeLogValidationSite.value;
  if (!site) {
    return String(logValidationIndex.value + 1);
  }
  const name = site.name.trim();
  return name || t('setup.logValidation.siteFallback', { value: logValidationIndex.value + 1 });
});

const logValidationResultLabel = computed(() => {
  if (logValidationResult.value.status === 'success') {
    return t('setup.logValidation.resultSuccess');
  }
  if (logValidationResult.value.status === 'error') {
    return t('setup.logValidation.resultFailed');
  }
  return t('setup.logValidation.resultIdle');
});

function createWebsiteDraft(prefillLogPath = ''): WebsiteDraft {
  return {
    name: '',
    logPath: prefillLogPath,
    domainsInput: '',
    logType: 'nginx',
    logFormat: '',
    logRegex: '',
    autoDiscoverHosts: false,
    logValidationStatus: 'idle',
    logValidationMessage: '',
    timeLayout: '',
    sourcesJson: '',
    whitelistEnabled: false,
    whitelistIPsText: '',
    whitelistCitiesText: '',
    whitelistNonMainland: false,
  };
}

const ipAliases = ['ip', 'remote_addr', 'real_client_ip', 'client_ip', 'http_x_forwarded_for'];
const timeAliases = ['time', 'time_local', 'time_iso8601'];
const statusAliases = ['status'];
const urlAliases = ['url', 'request_uri', 'uri', 'path'];
const requestAliases = ['request', 'request_line'];

type LogTypeOption = {
  value: string;
  label: string;
};

const baseLogTypeOptions: LogTypeOption[] = [
  { value: 'nginx', label: 'Nginx' },
  { value: 'apache', label: 'Apache httpd' },
  { value: 'iis', label: 'IIS (W3C Extended)' },
  { value: 'haproxy', label: 'HAProxy' },
  { value: 'traefik', label: 'Traefik' },
  { value: 'envoy', label: 'Envoy' },
  { value: 'tengine', label: 'Tengine' },
  { value: 'nginx-ingress', label: 'NGINX Ingress Controller' },
  { value: 'traefik-ingress', label: 'Traefik Ingress' },
  { value: 'haproxy-ingress', label: 'HAProxy Ingress' },
  { value: 'nginx-proxy-manager', label: 'Nginx Proxy Manager' },
  { value: 'safeline', label: 'SafeLine WAF' },
  { value: 'zoraxy', label: 'Zoraxy' },
  { value: 'caddy', label: 'Caddy' },
];

function logTypeOptionsFor(currentValue: string) {
  const current = currentValue.trim();
  if (!current) {
    return baseLogTypeOptions;
  }
  if (baseLogTypeOptions.some((option) => option.value === current)) {
    return baseLogTypeOptions;
  }
  return [...baseLogTypeOptions, { value: current, label: `${current} (custom)` }];
}

function normalizePort(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }
  if (trimmed.includes(':')) {
    return trimmed;
  }
  return `:${trimmed}`;
}

function addWebsite() {
  websiteDrafts.value.push(createWebsiteDraft());
}

function removeWebsite(index: number) {
  websiteDrafts.value.splice(index, 1);
}

function toggleWebsiteAdvanced(index: number) {
  advancedOpen.website[index] = !advancedOpen.website[index];
}

function createLogValidationResult(): LogValidationResult {
  return {
    status: 'idle',
    message: '',
    fields: [],
    matchedLine: '',
    source: '',
  };
}

function canValidateLogRule(site: WebsiteDraft) {
  return Boolean(site.logFormat.trim() || site.logRegex.trim());
}

function logValidationStatusLabel(status: LogValidationStatus) {
  if (status === 'success') {
    return t('setup.logValidation.statusSuccess');
  }
  if (status === 'error') {
    return t('setup.logValidation.statusError');
  }
  return t('setup.logValidation.statusIdle');
}

function markLogValidationDirty(site: WebsiteDraft) {
  if (site.logValidationStatus !== 'idle' || site.logValidationMessage) {
    site.logValidationStatus = 'idle';
    site.logValidationMessage = '';
  }
}

function openLogValidation(index: number) {
  logValidationIndex.value = index;
  logValidationVisible.value = true;
  logValidationSample.value = '';
  logValidationResult.value = createLogValidationResult();
}

function runLogValidation() {
  const site = activeLogValidationSite.value;
  if (!site) {
    return;
  }

  logValidationLoading.value = true;
  try {
    const sampleLines = logValidationSample.value
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
    if (sampleLines.length === 0) {
      throw new Error(t('setup.logValidation.errorNoSample'));
    }

    const { regex, groupNames, source } = buildValidationRegex(site);
    const match = findMatchingLine(regex, sampleLines);
    if (!match) {
      throw new Error(t('setup.logValidation.errorNoMatch'));
    }

    const { line, result } = match;
    const groups = result.groups || {};
    const fields = groupNames.map((name) => {
      const raw = groups[name];
      const value = raw === undefined || String(raw).trim() === '' ? t('common.none') : String(raw);
      return { name, value };
    });

    logValidationResult.value = {
      status: 'success',
      message: t('setup.logValidation.resultSuccess'),
      fields,
      matchedLine: line,
      source,
    };
    site.logValidationStatus = 'success';
    site.logValidationMessage = t('setup.logValidation.resultSuccess');
  } catch (err) {
    const message = err instanceof Error ? err.message : t('common.requestFailed');
    logValidationResult.value = {
      status: 'error',
      message,
      fields: [],
      matchedLine: '',
      source: '',
    };
    const skipSummary =
      message === t('setup.logValidation.errorNoSample') || message === t('setup.logValidation.errorNoRule');
    if (!skipSummary) {
      site.logValidationStatus = 'error';
      site.logValidationMessage = message;
    }
  } finally {
    logValidationLoading.value = false;
  }
}

function buildValidationRegex(site: WebsiteDraft): {
  regex: RegExp;
  groupNames: string[];
  source: 'logRegex' | 'logFormat';
} {
  const logRegex = site.logRegex.trim();
  const logFormat = site.logFormat.trim();
  if (!logRegex && !logFormat) {
    throw new Error(t('setup.logValidation.errorNoRule'));
  }

  let pattern = '';
  let source: 'logRegex' | 'logFormat' = 'logRegex';
  if (logRegex) {
    pattern = ensureAnchors(normalizeRegexPattern(logRegex));
  } else {
    pattern = buildRegexFromFormat(logFormat);
    source = 'logFormat';
  }

  const groupNames = extractGroupNames(pattern);
  const validationError = validateLogPattern(groupNames);
  if (validationError) {
    throw new Error(validationError);
  }

  try {
    return {
      regex: new RegExp(pattern),
      groupNames,
      source,
    };
  } catch (err) {
    const message = err instanceof Error ? err.message : t('common.requestFailed');
    throw new Error(message);
  }
}

function findMatchingLine(regex: RegExp, lines: string[]) {
  for (const line of lines) {
    const result = regex.exec(line);
    if (result) {
      return { line, result };
    }
  }
  return null;
}

function normalizeRegexPattern(pattern: string) {
  return pattern.replace(/\(\?P<([a-zA-Z_][\w]*)>/g, '(?<$1>');
}

function ensureAnchors(pattern: string) {
  let trimmed = pattern.trim();
  if (!trimmed) {
    return trimmed;
  }
  if (!trimmed.startsWith('^')) {
    trimmed = `^${trimmed}`;
  }
  if (!trimmed.endsWith('$')) {
    trimmed = `${trimmed}$`;
  }
  return trimmed;
}

function extractGroupNames(pattern: string) {
  const names: string[] = [];
  const seen = new Set<string>();
  const regex = /\(\?<([a-zA-Z_][\w]*)>/g;
  let match = regex.exec(pattern);
  while (match) {
    const name = match[1];
    if (!seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
    match = regex.exec(pattern);
  }
  return names;
}

function validateLogPattern(groupNames: string[]) {
  if (groupNames.length === 0) {
    return t('setup.logValidation.errorNoGroups');
  }
  if (!hasAnyField(groupNames, ipAliases)) {
    return t('setup.logValidation.errorMissingIp');
  }
  if (!hasAnyField(groupNames, timeAliases)) {
    return t('setup.logValidation.errorMissingTime');
  }
  if (!hasAnyField(groupNames, statusAliases)) {
    return t('setup.logValidation.errorMissingStatus');
  }
  if (!hasAnyField(groupNames, urlAliases) && !hasAnyField(groupNames, requestAliases)) {
    return t('setup.logValidation.errorMissingUrl');
  }
  return '';
}

function hasAnyField(groupNames: string[], aliases: string[]) {
  return groupNames.some((name) => aliases.includes(name));
}

function buildRegexFromFormat(format: string) {
  if (!format.trim()) {
    throw new Error(t('setup.logValidation.errorEmptyFormat'));
  }
  const varPattern = /\$\w+/g;
  const matches = Array.from(format.matchAll(varPattern));
  if (matches.length === 0) {
    throw new Error(t('setup.logValidation.errorNoVars'));
  }

  let builder = '';
  let last = 0;
  const used = new Set<string>();
  matches.forEach((match) => {
    const index = match.index ?? 0;
    const literal = format.slice(last, index);
    builder += escapeRegExp(literal);
    const varName = match[0].slice(1);
    const quoted = isQuotedTokenBoundary(literal, format.slice(index + match[0].length));
    builder += tokenRegexForVar(varName, used, quoted);
    last = index + match[0].length;
  });
  builder += escapeRegExp(format.slice(last));
  return `^${builder}$`;
}

function tokenRegexForVar(name: string, used: Set<string>, quoted: boolean) {
  const addGroup = (group: string, pattern: string) => {
    if (used.has(group)) {
      return pattern;
    }
    used.add(group);
    return `(?<${group}>${pattern})`;
  };

  let commaListPattern = '[^,\\s]+(?:,\\s*[^,\\s]+)*';
  let optionalTokenPattern = '\\S*';
  let requiredTokenPattern = '\\S+';
  if (quoted) {
    commaListPattern = '(?:[^,\\s]+(?:,\\s*[^,\\s]+)*)?';
    optionalTokenPattern = '[^"]*';
    requiredTokenPattern = '[^"]+';
  }

  switch (name) {
    case 'remote_addr':
      return addGroup('ip', requiredTokenPattern);
    case 'real_client_ip':
      return addGroup('real_client_ip', requiredTokenPattern);
    case 'http_x_forwarded_for':
      return addGroup('http_x_forwarded_for', commaListPattern);
    case 'remote_user':
      return addGroup('user', optionalTokenPattern);
    case 'time_local':
      return addGroup('time', '[^]]+');
    case 'time_iso8601':
      return addGroup('time', requiredTokenPattern);
    case 'request':
      return addGroup('request', requiredTokenPattern);
    case 'request_method':
      return addGroup('method', requiredTokenPattern);
    case 'request_uri':
    case 'uri':
      return addGroup('url', requiredTokenPattern);
    case 'args':
      return addGroup('args', optionalTokenPattern);
    case 'query_string':
      return addGroup('query_string', optionalTokenPattern);
    case 'status':
      return addGroup('status', '\\d{3}');
    case 'body_bytes_sent':
    case 'bytes_sent':
      return addGroup('bytes', '\\d+');
    case 'http_referer':
      return addGroup('referer', optionalTokenPattern);
    case 'http_user_agent':
      return addGroup('ua', optionalTokenPattern);
    case 'host':
    case 'http_host':
      return addGroup('host', requiredTokenPattern);
    case 'server_name':
      return addGroup('server_name', requiredTokenPattern);
    case 'scheme':
      return addGroup('scheme', requiredTokenPattern);
    case 'request_length':
      return addGroup('request_length', '\\d+');
    case 'remote_port':
      return addGroup('remote_port', '\\d+');
    case 'connection':
      return addGroup('connection', '\\d+');
    case 'request_time':
      return addGroup('request_time', '\\d+(?:\\.\\d+)?');
    case 'request_time_msec':
      return addGroup('request_time_msec', '\\d+(?:\\.\\d+)?');
    case 'upstream_addr':
      return addGroup('upstream_addr', commaListPattern);
    case 'upstream_status':
      return addGroup('upstream_status', commaListPattern);
    case 'upstream_response_time':
      return addGroup('upstream_response_time', commaListPattern);
    case 'upstream_connect_time':
      return addGroup('upstream_connect_time', commaListPattern);
    case 'upstream_header_time':
      return addGroup('upstream_header_time', commaListPattern);
    case 'request_id':
      return addGroup('request_id', requiredTokenPattern);
    default:
      return optionalTokenPattern;
  }
}

function isQuotedTokenBoundary(prefix: string, suffix: string) {
  const prefixTrim = prefix.replace(/[ \t\r\n]+$/, '');
  if (!prefixTrim.endsWith('"')) {
    return false;
  }
  const suffixTrim = suffix.replace(/^[ \t\r\n]+/, '');
  return suffixTrim.startsWith('"');
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function fieldError(field: string) {
  return fieldErrors.value[field];
}

function splitList(value: string) {
  return value
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function validateURLField(value: string, field: string, errors: FieldError[]) {
  const trimmed = value.trim();
  if (!trimmed) {
    errors.push({ field, message: t('setup.errors.required') });
    return;
  }
  try {
    const parsed = new URL(trimmed);
    if (!['http:', 'https:'].includes(parsed.protocol)) {
      errors.push({ field, message: t('setup.errors.httpUrlRequired') });
    }
  } catch {
    errors.push({ field, message: t('setup.errors.invalidUrl') });
  }
}

function parseOptionalInt(
  value: string | number | null | undefined,
  field: string,
  errors: FieldError[],
  allowZero: boolean
): number | undefined {
  if (value === null || value === undefined) {
    return undefined;
  }
  const trimmed = (typeof value === 'number' ? String(value) : value).trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed)) {
    errors.push({ field, message: t('setup.errors.invalidNumber') });
    return undefined;
  }
  if (!allowZero && parsed <= 0) {
    errors.push({ field, message: t('setup.errors.positiveNumber') });
  }
  if (allowZero && parsed < 0) {
    errors.push({ field, message: t('setup.errors.nonNegativeNumber') });
  }
  return Math.floor(parsed);
}

function parseOptionalJSONObject(
  value: string,
  field: string,
  errors: FieldError[]
): Record<string, any> | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
      errors.push({ field, message: t('setup.errors.objectJsonRequired') });
      return undefined;
    }
    return parsed as Record<string, any>;
  } catch (err) {
    const message = err instanceof Error ? err.message : t('setup.errors.invalidJson');
    errors.push({ field, message: t('setup.errors.parseJson', { message }) });
    return undefined;
  }
}

function parseIntList(value: string, field: string, errors: FieldError[]) {
  const items = splitList(value);
  if (items.length === 0) {
    errors.push({ field, message: t('setup.errors.required') });
    return [] as number[];
  }
  const result: number[] = [];
  for (const item of items) {
    const parsed = Number(item);
    if (!Number.isFinite(parsed)) {
      errors.push({ field, message: t('setup.errors.invalidNumber') });
      continue;
    }
    result.push(Math.floor(parsed));
  }
  return result;
}

function buildConfig(collectErrors = true): { config: ConfigPayload; errors: FieldError[] } {
  const errors: FieldError[] = [];
  const websites = websiteDrafts.value.map((site, index) => {
    const sourcesJson = site.sourcesJson.trim();
    let sources: SourceConfig[] | undefined;
    if (sourcesJson) {
      try {
        const parsed = JSON.parse(sourcesJson);
        if (Array.isArray(parsed)) {
          sources = parsed;
        } else if (collectErrors) {
          errors.push({ field: `websites[${index}].sources`, message: t('setup.errors.sourcesArray') });
        }
      } catch (err) {
        if (collectErrors) {
          const message = err instanceof Error ? err.message : t('setup.errors.invalidJson');
          errors.push({ field: `websites[${index}].sources`, message: t('setup.errors.parseJson', { message }) });
        }
      }
    }

    if (collectErrors) {
      if (!site.name.trim()) {
        errors.push({ field: `websites[${index}].name`, message: t('setup.errors.required') });
      }
      if (!site.logPath.trim() && (!sources || sources.length === 0)) {
        errors.push({ field: `websites[${index}].logPath`, message: t('setup.errors.logPathRequired') });
      }
    }

    const whitelistIPs = splitList(site.whitelistIPsText);
    const whitelistCities = splitList(site.whitelistCitiesText);
    const whitelistEnabled = site.whitelistEnabled;
    const whitelistNonMainland = site.whitelistNonMainland;
    if (collectErrors && whitelistEnabled && whitelistIPs.length === 0 && whitelistCities.length === 0 && !whitelistNonMainland) {
      errors.push({ field: `websites[${index}].whitelist`, message: t('setup.errors.whitelistEmpty') });
    }

    const whitelist =
      whitelistEnabled || whitelistIPs.length > 0 || whitelistCities.length > 0 || whitelistNonMainland
        ? {
            enabled: whitelistEnabled,
            ips: whitelistIPs,
            cities: whitelistCities,
            nonMainland: whitelistNonMainland,
          }
        : undefined;

    return {
      name: site.name.trim(),
      logPath: site.logPath.trim(),
      domains: splitList(site.domainsInput),
      logType: site.logType.trim(),
      logFormat: site.logFormat.trim(),
      logRegex: site.logRegex.trim(),
      timeLayout: site.timeLayout.trim(),
      sources,
      whitelist,
      autoDiscoverHosts: site.autoDiscoverHosts,
    };
  });

  const statusCodes = parseIntList(pvDraft.statusCodeIncludeText, 'pvFilter.statusCodeInclude', errors);
  const excludePatterns = splitList(pvDraft.excludePatternsText);
  if (collectErrors && excludePatterns.length === 0) {
    errors.push({ field: 'pvFilter.excludePatterns', message: t('setup.errors.required') });
  }

  const webBasePath = systemDraft.webBasePath.trim();
  if (collectErrors && webBasePath) {
    if (webBasePath.includes('/')) {
      errors.push({ field: 'system.webBasePath', message: t('setup.errors.webBasePathSingleSegment') });
    } else if (!/^[a-zA-Z0-9_-]+$/.test(webBasePath)) {
      errors.push({ field: 'system.webBasePath', message: t('setup.errors.webBasePathInvalid') });
    }
  }
  if (collectErrors && systemDraft.serverStatusEnabled && !systemDraft.serverStatusMockEnabled) {
    validateURLField(systemDraft.serverStatusMetricsUrl, 'system.serverStatus.metricsUrl', errors);
    validateURLField(systemDraft.serverStatusDisksUrl, 'system.serverStatus.disksUrl', errors);
  }
  if (collectErrors && systemDraft.serverStatusEnabled) {
    if (!systemDraft.serverStatusTimeout.trim()) {
      errors.push({ field: 'system.serverStatus.timeout', message: t('setup.errors.required') });
    }
    if (!systemDraft.serverStatusRefreshInterval.trim()) {
      errors.push({ field: 'system.serverStatus.refreshInterval', message: t('setup.errors.required') });
    }
  }

  const config: ConfigPayload = {
    websites,
    system: {
      logDestination: systemDraft.logDestination.trim(),
      taskInterval: systemDraft.taskInterval.trim(),
      backfillMaxDurationPerRun: systemDraft.backfillMaxDurationPerRun.trim(),
      backfillMaxBytesPerRun: parseOptionalInt(systemDraft.backfillMaxBytesPerRun, 'system.backfillMaxBytesPerRun', errors, false),
      httpSourceTimeout: systemDraft.httpSourceTimeout.trim(),
      logRetentionDays: parseOptionalInt(systemDraft.logRetentionDays, 'system.logRetentionDays', errors, false),
      parseBatchSize: parseOptionalInt(systemDraft.parseBatchSize, 'system.parseBatchSize', errors, false),
      ipGeoCacheLimit: parseOptionalInt(systemDraft.ipGeoCacheLimit, 'system.ipGeoCacheLimit', errors, false),
      alertPush: parseOptionalJSONObject(systemDraft.alertPushJson, 'system.alertPush', errors),
      demoMode: systemDraft.demoMode,
      mobilePwaEnabled: systemDraft.mobilePwaEnabled,
      accessKeys: splitList(systemDraft.accessKeysText),
      accessKeyExpireDays: parseOptionalInt(systemDraft.accessKeyExpireDays, 'system.accessKeyExpireDays', errors, false),
      language: systemDraft.language,
      webBasePath,
      serverStatus: {
        enabled: systemDraft.serverStatusEnabled,
        mockEnabled: systemDraft.serverStatusMockEnabled,
        metricsUrl: systemDraft.serverStatusMetricsUrl.trim(),
        disksUrl: systemDraft.serverStatusDisksUrl.trim(),
        timeout: systemDraft.serverStatusTimeout.trim() || '5s',
        refreshInterval: systemDraft.serverStatusRefreshInterval.trim() || '30s',
      },
    },
    server: {
      Port: normalizePort(serverPort.value),
    },
    database: {
      driver: databaseDraft.driver,
      dsn: databaseDraft.dsn.trim(),
      maxOpenConns: parseOptionalInt(databaseDraft.maxOpenConns, 'database.maxOpenConns', errors, true),
      maxIdleConns: parseOptionalInt(databaseDraft.maxIdleConns, 'database.maxIdleConns', errors, true),
      connMaxLifetime: databaseDraft.connMaxLifetime.trim(),
    },
    pvFilter: {
      statusCodeInclude: statusCodes,
      excludePatterns,
      excludeIPs: splitList(pvDraft.excludeIPsText),
    },
  };

  if (collectErrors && !databaseDraft.dsn.trim()) {
    errors.push({ field: 'database.dsn', message: t('setup.errors.required') });
  }

  return { config, errors };
}

function filterErrorsForStep(errors: FieldError[], step: number) {
  if (step >= steps.value.length - 1) {
    return errors;
  }
  const prefixes =
    step === 0
      ? ['websites', 'config']
      : step === 1
        ? ['database']
        : ['system', 'server', 'pvFilter'];
  return errors.filter((item) => {
    if (!item.field) {
      return true;
    }
    return prefixes.some((prefix) => item.field.startsWith(prefix));
  });
}

function applyErrors(errors: FieldError[]) {
  const map: Record<string, string> = {};
  errors.forEach((item) => {
    if (item.field && !map[item.field]) {
      map[item.field] = item.message;
    }
  });
  fieldErrors.value = map;
}

async function validateStep(step: number, remote: boolean) {
  saveError.value = '';
  const { config, errors: localErrors } = buildConfig(true);
  let remoteErrors: FieldError[] = [];
  let warnings: FieldError[] = [];

  if (remote) {
    try {
      const result = await validateConfig(config);
      remoteErrors = result.errors || [];
      warnings = result.warnings || [];
    } catch (err) {
      const message = err instanceof Error ? err.message : t('common.requestFailed');
      remoteErrors = [{ field: '', message }];
    }
  }

  const errors = [...localErrors, ...remoteErrors];
  validationErrors.value = errors;
  validationWarnings.value = warnings;
  applyErrors(errors);
  return filterErrorsForStep(errors, step).length === 0;
}

async function nextStep() {
  if (nextLoading.value) {
    return;
  }
  nextLoading.value = true;
  try {
    const remote = currentStep.value === 0;
    const ok = await validateStep(currentStep.value, remote);
    if (!ok) {
      return;
    }
    currentStep.value += 1;
  } finally {
    nextLoading.value = false;
  }
}

function prevStep() {
  currentStep.value = Math.max(0, currentStep.value - 1);
}

async function saveAll() {
  const ok = await validateStep(steps.value.length - 1, true);
  if (!ok) {
    return;
  }
  const { config } = buildConfig(false);
  const redirectPath = buildWebRootPath(config.system?.webBasePath || '');
  saving.value = true;
  saveError.value = '';
  try {
    const result = await saveConfig(config);
    saveSuccess.value = Boolean(result.success);
    if (saveSuccess.value) {
      try {
        await restartSystem();
      } catch (err) {
        console.warn('触发重启失败:', err);
      }
      startAutoRefresh(redirectPath);
    }
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : t('common.requestFailed');
  } finally {
    saving.value = false;
  }
}

async function copyConfig() {
  if (!configPreview.value) {
    return;
  }
  if (copyStatusTimer) {
    window.clearTimeout(copyStatusTimer);
    copyStatusTimer = null;
  }
  try {
    await navigator.clipboard.writeText(configPreview.value);
    copyStatus.value = t('setup.copySuccess');
  } catch (err) {
    const textarea = document.createElement('textarea');
    textarea.value = configPreview.value;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(textarea);
    copyStatus.value = ok ? t('setup.copySuccess') : t('setup.copyFailed');
  }
  copyStatusTimer = window.setTimeout(() => {
    copyStatus.value = '';
    copyStatusTimer = null;
  }, 2000);
}

function startAutoRefresh(redirectPath = '/') {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer);
  }
  autoRefreshSeconds.value = AUTO_REFRESH_SECONDS;
  autoRefreshTimer = window.setInterval(() => {
    autoRefreshSeconds.value -= 1;
    if (autoRefreshSeconds.value <= 0) {
      window.clearInterval(autoRefreshTimer as number);
      autoRefreshTimer = null;
      window.location.assign(redirectPath);
    }
  }, 1000);
}

function buildWebRootPath(basePath: string) {
  const normalized = basePath.trim().replace(/^\/+|\/+$/g, '');
  return normalized ? `/${normalized}/` : '/';
}

function alertChannelLabel(channel: string) {
  const normalized = channel.trim().toLowerCase();
  if (normalized === 'feishu') {
    return 'Feishu';
  }
  if (normalized === 'dingtalk') {
    return 'DingTalk';
  }
  if (normalized === 'wecom') {
    return 'WeCom';
  }
  if (normalized === 'email') {
    return t('setup.alertPushTest.channelEmail');
  }
  return channel;
}

async function runAlertPushTest() {
  if (alertPushTesting.value) {
    return;
  }

  alertPushTesting.value = true;
  alertPushTestError.value = '';
  alertPushTestResult.value = null;
  try {
    const parseErrors: FieldError[] = [];
    const alertPush = parseOptionalJSONObject(systemDraft.alertPushJson, 'system.alertPush', parseErrors);
    if (parseErrors.length > 0) {
      throw new Error(parseErrors[0].message);
    }

    alertPushTestResult.value = await testAlertPush({
      alertPush,
      message: alertPushTestMessage.value.trim(),
    });
  } catch (err) {
    alertPushTestError.value = err instanceof Error ? err.message : t('common.requestFailed');
  } finally {
    alertPushTesting.value = false;
  }
}

async function loadConfig() {
  loading.value = true;
  loadError.value = '';
  try {
    const response = await fetchConfig();
    defaultLogPath.value = response.default_log_path || '';
    configReadonly.value = Boolean(response.readonly);
    hydrateDraft(response.config);
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : t('common.requestFailed');
  } finally {
    loading.value = false;
  }
}

function hydrateDraft(config: ConfigPayload) {
  serverPort.value = config.server?.Port || ':8089';
  databaseDraft.driver = config.database?.driver || 'postgres';
  databaseDraft.dsn = config.database?.dsn || '';
  databaseDraft.maxOpenConns = String(config.database?.maxOpenConns ?? 10);
  databaseDraft.maxIdleConns = String(config.database?.maxIdleConns ?? 5);
  databaseDraft.connMaxLifetime = config.database?.connMaxLifetime || '30m';

  systemDraft.logDestination = config.system?.logDestination || 'file';
  systemDraft.taskInterval = config.system?.taskInterval || '1m';
  systemDraft.backfillMaxDurationPerRun = config.system?.backfillMaxDurationPerRun || '8s';
  systemDraft.backfillMaxBytesPerRun = String(config.system?.backfillMaxBytesPerRun ?? 32 * 1024 * 1024);
  systemDraft.httpSourceTimeout = config.system?.httpSourceTimeout || '2m';
  systemDraft.logRetentionDays = String(config.system?.logRetentionDays ?? 30);
  systemDraft.parseBatchSize = String(config.system?.parseBatchSize ?? 100);
  systemDraft.ipGeoCacheLimit = String(config.system?.ipGeoCacheLimit ?? 1000000);
  systemDraft.alertPushJson = config.system?.alertPush ? JSON.stringify(config.system.alertPush, null, 2) : '';
  systemDraft.demoMode = Boolean(config.system?.demoMode);
  systemDraft.mobilePwaEnabled = Boolean(config.system?.mobilePwaEnabled);
  systemDraft.serverStatusEnabled = Boolean(config.system?.serverStatus?.enabled);
  systemDraft.serverStatusMockEnabled = Boolean(config.system?.serverStatus?.mockEnabled);
  systemDraft.serverStatusMetricsUrl = config.system?.serverStatus?.metricsUrl || '';
  systemDraft.serverStatusDisksUrl = config.system?.serverStatus?.disksUrl || '';
  systemDraft.serverStatusTimeout = config.system?.serverStatus?.timeout || '5s';
  systemDraft.serverStatusRefreshInterval = config.system?.serverStatus?.refreshInterval || '30s';
  systemDraft.accessKeysText = (config.system?.accessKeys || []).join(', ');
  systemDraft.accessKeyExpireDays = String(config.system?.accessKeyExpireDays ?? 7);
  systemDraft.language = config.system?.language || 'zh-CN';
  systemDraft.webBasePath = config.system?.webBasePath || '';
  alertPushTestMessage.value = '';
  alertPushTestResult.value = null;
  alertPushTestError.value = '';

  pvDraft.statusCodeIncludeText = (config.pvFilter?.statusCodeInclude || []).join(', ');
  pvDraft.excludePatternsText = (config.pvFilter?.excludePatterns || []).join('\n');
  pvDraft.excludeIPsText = (config.pvFilter?.excludeIPs || []).join(', ');

  const mapped = (config.websites || []).map((site) => ({
    name: site.name || '',
    logPath: site.logPath || '',
    domainsInput: (site.domains || []).join(', '),
    logType: site.logType || 'nginx',
    logFormat: site.logFormat || '',
    logRegex: site.logRegex || '',
    autoDiscoverHosts: Boolean(site.autoDiscoverHosts),
    logValidationStatus: 'idle' as LogValidationStatus,
    logValidationMessage: '',
    timeLayout: site.timeLayout || '',
    sourcesJson: site.sources && site.sources.length > 0 ? JSON.stringify(site.sources, null, 2) : '',
    whitelistEnabled: Boolean(site.whitelist?.enabled),
    whitelistIPsText: (site.whitelist?.ips || []).join(', '),
    whitelistCitiesText: (site.whitelist?.cities || []).join(', '),
    whitelistNonMainland: Boolean(site.whitelist?.nonMainland),
  }));
  websiteDrafts.value = mapped.length ? mapped : [createWebsiteDraft(defaultLogPath.value)];
}

onMounted(() => {
  loadConfig();
});

onBeforeUnmount(() => {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer);
    autoRefreshTimer = null;
  }
  if (copyStatusTimer) {
    window.clearTimeout(copyStatusTimer);
    copyStatusTimer = null;
  }
});
</script>
