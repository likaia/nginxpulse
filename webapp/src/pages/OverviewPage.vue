<template>
  <div class="overview-page">
    <header class="page-header">
      <div class="page-title">
        <span class="title-chip">{{ t('overview.title') }}</span>
        <p class="title-sub">{{ t('overview.subtitle') }}</p>
      </div>
      <div class="header-actions header-actions-tech">
        <HeaderToolbar class="header-toolbar-tech">
          <template #primary>
            <div class="traffic-pill">
              <span class="traffic-label">{{ t('common.traffic') }}</span>
              <span class="traffic-value">{{ trafficText }}</span>
            </div>
            <div class="site-select-pill">
              <span class="site-label">{{ t('common.website') }}</span>
              <WebsiteSelect
                v-model="currentWebsiteId"
                class="website-select-compact"
                :websites="websites"
                :loading="websitesLoading"
                id="website-selector"
                label=""
              />
            </div>
          </template>
          <template #utility>
            <button
              type="button"
              class="auto-refresh-toggle"
              :class="{ inactive: !autoRefreshAllowed, active: autoRefreshEnabled }"
              :title="autoRefreshAllowed ? t('overview.autoRefreshHint') : t('overview.autoRefreshTodayOnly')"
              :disabled="!autoRefreshAllowed"
              :aria-pressed="autoRefreshEnabled"
              @click="toggleAutoRefresh"
            >
              <i class="ri-refresh-line" aria-hidden="true"></i>
              <span class="auto-refresh-text">{{ t('overview.autoRefresh') }}</span>
              <span v-if="!autoRefreshAllowed" class="auto-refresh-hint">{{ t('overview.autoRefreshTodayOnly') }}</span>
            </button>
            <SystemNotifications />
            <ThemeToggle />
          </template>
        </HeaderToolbar>
        <div class="select-group sr-only">
          <label class="select-label" for="date-range">{{ t('common.date') }}</label>
          <Dropdown
            inputId="date-range"
            v-model="dateRange"
            class="date-range-dropdown"
            :options="dateRangeOptions"
            optionLabel="label"
            optionValue="value"
          />
        </div>
      </div>
    </header>

    <section class="overview-grid">
      <div class="card live-card" data-anim>
        <div class="live-card-header">
          <span class="live-chip">{{ t('overview.liveVisitors') }}</span>
        </div>
        <div class="live-card-body">
          <div class="live-value">{{ liveVisitorText }}</div>
          <div class="live-sub">{{ t('overview.liveStatus') }}</div>
          <RouterLink class="ghost-link" to="/realtime?window=5">{{ t('overview.viewRealtime') }}</RouterLink>
        </div>
      </div>
      <div class="card metrics-card" data-anim>
        <div class="metrics-head">
          <div>
            <div class="metrics-title">{{ t('overview.metricsTitle') }}</div>
            <div class="metrics-sub">{{ t('overview.metricsSub') }}</div>
          </div>
          <div class="metrics-range">
            <span class="metrics-range-label">{{ t('common.date') }}</span>
            <div class="range-tabs inline">
              <button
                v-for="tab in rangeTabs"
                :key="tab.value"
                class="range-tab"
                :class="{ active: dateRange === tab.value }"
                @click="setRange(tab.value)"
              >
                {{ tab.label }}
              </button>
            </div>
            <div class="metrics-date-picker-wrap" :class="{ active: isSpecificDateSelected }">
              <DatePicker
                v-model="specificDateValue"
                class="metrics-date-picker"
                dateFormat="yy-mm-dd"
                updateModelType="string"
                :manualInput="false"
                :maxDate="maxSelectableDate"
                showButtonBar
                :showClear="true"
                :showIcon="true"
                :placeholder="t('overview.specificDate')"
              />
            </div>
          </div>
        </div>
        <div class="metrics-grid">
          <div class="metric-tile status-tile">
            <div class="metric-header">
              <div class="metric-label">{{ t('overview.statusHits') }}</div>
              <button class="link-button metric-detail" @click="openDetail('metric-status')">{{ t('overview.detail') }}</button>
            </div>
            <div class="metric-value">{{ statusMetrics.total }}</div>
            <div class="metric-sub">
              <span class="metric-sub-label">{{ statusMetrics.prevLabel }}</span>
              <span class="metric-sub-group">
                <span class="metric-sub-value">{{ statusMetrics.prevTotal }}</span>
                <span class="metric-delta-inline" :class="statusMetrics.deltaClass">{{ statusMetrics.deltaText }}</span>
              </span>
            </div>
            <div class="metric-sub"><span class="metric-sub-label">2xx</span><span class="metric-sub-value">{{ statusMetrics.s2xx }}</span></div>
            <div class="metric-sub"><span class="metric-sub-label">3xx</span><span class="metric-sub-value">{{ statusMetrics.s3xx }}</span></div>
            <div class="metric-sub"><span class="metric-sub-label">4xx</span><span class="metric-sub-value">{{ statusMetrics.s4xx }}</span></div>
            <div class="metric-sub"><span class="metric-sub-label">5xx</span><span class="metric-sub-value">{{ statusMetrics.s5xx }}</span></div>
          </div>
          <div class="metric-tile" data-metric="pv">
            <div class="metric-header">
              <div class="metric-label">{{ t('overview.pv') }}</div>
              <button class="link-button metric-detail" @click="openDetail('metric-pv')">{{ t('overview.detail') }}</button>
            </div>
            <div class="metric-value">{{ metricTiles.pv.current }}</div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.prev }}</span><span class="metric-sub-value">{{ metricTiles.pv.prev }}</span></div>
            <div class="metric-sub">
              <span class="metric-sub-label with-hint">
                {{ metricLabels.forecast }}
                <span class="metric-hint">
                  <button
                    type="button"
                    class="metric-hint-btn"
                    :aria-label="forecastHintAria"
                  >
                    <i class="ri-information-line" aria-hidden="true"></i>
                  </button>
                  <span class="metric-hint-popover" role="tooltip">{{ forecastHintText }}</span>
                </span>
              </span>
              <span class="metric-sub-value trend" :class="metricTiles.pv.deltaClass">{{ metricTiles.pv.forecast }}</span>
              <span class="metric-delta-inline" :class="metricTiles.pv.deltaClass">{{ metricTiles.pv.deltaText }}</span>
            </div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.sameTime }}</span><span class="metric-sub-value">{{ metricTiles.pv.sameTime }}</span></div>
          </div>
          <div class="metric-tile" data-metric="uv">
            <div class="metric-header">
              <div class="metric-label">{{ t('overview.uv') }}</div>
              <button class="link-button metric-detail" @click="openDetail('metric-uv')">{{ t('overview.detail') }}</button>
            </div>
            <div class="metric-value">{{ metricTiles.uv.current }}</div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.prev }}</span><span class="metric-sub-value">{{ metricTiles.uv.prev }}</span></div>
            <div class="metric-sub">
              <span class="metric-sub-label with-hint">
                {{ metricLabels.forecast }}
                <span class="metric-hint">
                  <button
                    type="button"
                    class="metric-hint-btn"
                    :aria-label="forecastHintAria"
                  >
                    <i class="ri-information-line" aria-hidden="true"></i>
                  </button>
                  <span class="metric-hint-popover" role="tooltip">{{ forecastHintText }}</span>
                </span>
              </span>
              <span class="metric-sub-value trend" :class="metricTiles.uv.deltaClass">{{ metricTiles.uv.forecast }}</span>
              <span class="metric-delta-inline" :class="metricTiles.uv.deltaClass">{{ metricTiles.uv.deltaText }}</span>
            </div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.sameTime }}</span><span class="metric-sub-value">{{ metricTiles.uv.sameTime }}</span></div>
          </div>
          <div class="metric-tile" data-metric="session">
            <div class="metric-header">
              <div class="metric-label">{{ t('overview.session') }}</div>
              <button class="link-button metric-detail" @click="openDetail('metric-session')">{{ t('overview.detail') }}</button>
            </div>
            <div class="metric-value">{{ metricTiles.session.current }}</div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.prev }}</span><span class="metric-sub-value">{{ metricTiles.session.prev }}</span></div>
            <div class="metric-sub">
              <span class="metric-sub-label with-hint">
                {{ metricLabels.forecast }}
                <span class="metric-hint">
                  <button
                    type="button"
                    class="metric-hint-btn"
                    :aria-label="forecastHintAria"
                  >
                    <i class="ri-information-line" aria-hidden="true"></i>
                  </button>
                  <span class="metric-hint-popover" role="tooltip">{{ forecastHintText }}</span>
                </span>
              </span>
              <span class="metric-sub-value trend" :class="metricTiles.session.deltaClass">{{ metricTiles.session.forecast }}</span>
              <span class="metric-delta-inline" :class="metricTiles.session.deltaClass">{{ metricTiles.session.deltaText }}</span>
            </div>
            <div class="metric-sub"><span class="metric-sub-label">{{ metricLabels.sameTime }}</span><span class="metric-sub-value">{{ metricTiles.session.sameTime }}</span></div>
          </div>
        </div>
      </div>
    </section>

    <section v-if="serverStatusVisible" class="server-status-grid">
      <div class="card server-status-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon green"><i class="ri-server-line"></i></span>
            {{ t('overview.serverStatusTitle') }}
          </div>
          <div class="server-status-head-meta">
            <span class="server-status-chip" :class="serverStatusClass">
              <span class="server-status-dot" aria-hidden="true"></span>
              {{ serverStatusLabel }}
            </span>
            <span v-if="serverStatusUpdatedAt" class="server-status-updated">{{ serverStatusUpdatedAt }}</span>
          </div>
        </div>
        <div v-if="serverStatusLoading && !serverStatus" class="server-status-skeleton" aria-busy="true">
          <div class="server-status-tile-grid">
            <div
              v-for="index in 5"
              :key="`server-status-skeleton-${index}`"
              class="server-status-tile server-status-skeleton-tile"
            >
              <span class="server-status-skeleton-icon"></span>
              <span class="server-status-skeleton-line short"></span>
              <span class="server-status-skeleton-line value"></span>
              <span class="server-status-skeleton-line"></span>
            </div>
          </div>
          <div class="server-status-skeleton-disk">
            <span class="server-status-skeleton-line short"></span>
            <span class="server-status-skeleton-line value"></span>
            <span class="server-status-skeleton-line"></span>
          </div>
        </div>
        <div v-else-if="serverStatusError" class="server-status-error">
          <i class="ri-error-warning-line" aria-hidden="true"></i>
          <span>{{ serverStatusError }}</span>
        </div>
        <template v-else>
          <div class="server-status-dashboard">
            <div class="server-status-tile-grid">
              <div class="server-status-tile health" :class="serverHealthTone">
                <div class="server-status-tile-icon">
                  <i class="ri-shield-check-line" aria-hidden="true"></i>
                </div>
                <div class="server-status-tile-main">
                  <span>{{ t('overview.serverStatusHealthScore') }}</span>
                  <strong>{{ serverHealthScore }}%</strong>
                  <small>{{ serverStatusLabel }}</small>
                </div>
              </div>

              <div
                v-for="item in serverStatusSensorRows"
                :key="item.key"
                class="server-status-tile"
                :class="[item.tone, { 'is-fan': item.key === 'cpu-fan' }]"
                :style="item.style"
              >
                <div class="server-status-tile-icon">
                  <span v-if="item.key === 'cpu-fan'" class="server-fan-icon" aria-hidden="true">
                    <span class="server-fan-blade"></span>
                    <span class="server-fan-blade"></span>
                    <span class="server-fan-blade"></span>
                  </span>
                  <i v-else :class="item.icon" aria-hidden="true"></i>
                </div>
                <div class="server-status-tile-main">
                  <span class="server-status-tile-label">
                    {{ item.label }}
                    <em v-if="item.key === 'cpu-fan'">RPM</em>
                  </span>
                  <div v-if="item.key === 'cpu-fan'" class="server-fan-values">
                    <span>
                      <small>{{ t('overview.serverStatusFanCpuShort') }}</small>
                      <strong>{{ item.cpuValue }}</strong>
                    </span>
                    <span>
                      <small>{{ t('overview.serverStatusFanChassisShort') }}</small>
                      <strong>{{ item.chassisValue }}</strong>
                    </span>
                  </div>
                  <strong v-else>{{ item.value }}</strong>
                  <div class="server-status-tile-track" aria-hidden="true">
                    <span></span>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="primaryServerDisk" class="server-disk-summary" :class="diskTone(primaryServerDisk)">
              <div class="server-disk-summary-main">
                <div class="server-disk-summary-icon">
                  <i class="ri-hard-drive-3-line" aria-hidden="true"></i>
                </div>
                <div>
                  <div class="server-disk-summary-title">{{ t('overview.serverStatusDiskSmart') }}</div>
                  <div class="server-disk-summary-sub">{{ serverDiskSummaryText }}</div>
                </div>
              </div>

              <div class="server-disk-summary-stats">
                <span v-for="stat in serverDiskSummaryStats" :key="stat.key">
                  <small>{{ stat.label }}</small>
                  <strong>{{ stat.value }}</strong>
                </span>
              </div>

              <button
                v-if="serverDisks.length"
                type="button"
                class="server-disk-view-all"
                @click="serverDiskDialogVisible = true"
              >
                <i class="ri-list-check-3" aria-hidden="true"></i>
                <span>{{ serverDiskActionText }}</span>
              </button>
            </div>
          </div>
          <div v-if="serverStatusMessages.length" class="server-status-messages">
            <span v-for="message in serverStatusMessages" :key="message">{{ message }}</span>
          </div>
        </template>
      </div>
    </section>

    <section class="trend-grid">
      <div class="card trend-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon blue"><i class="ri-line-chart-line"></i></span>
            {{ t('overview.trend') }}
          </div>
          <div class="card-actions">
            <div class="view-toggle">
              <button
                class="data-view-toggle-btn"
                :class="{ active: chartView === 'hourly' }"
                @click="setChartView('hourly')"
              >
                {{ t('overview.byHour') }}
              </button>
              <button
                class="data-view-toggle-btn"
                :class="{ active: chartView === 'daily', disabled: dailyViewDisabled }"
                :disabled="dailyViewDisabled"
                @click="setChartView('daily')"
              >
                {{ t('overview.byDay') }}
              </button>
            </div>
          </div>
        </div>
        <div class="chart-wrap">
          <canvas v-show="hasVisitsTrendData" ref="visitsChartRef"></canvas>
          <div v-if="!chartError && !hasVisitsTrendData" class="overview-empty-state">
            <span class="overview-empty-state-icon"><i class="ri-line-chart-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.trendEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.trendEmptyText') }}</div>
          </div>
          <div v-if="chartError" class="chart-error-message">{{ chartError }}</div>
        </div>
      </div>
      <div class="card new-old-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon green"><i class="ri-user-heart-line"></i></span>
            {{ t('overview.newOld') }}
          </div>
        </div>
        <div class="chart-mini">
          <canvas v-show="hasNewOldData" ref="newOldChartRef"></canvas>
          <div v-if="!hasNewOldData" class="overview-empty-state overview-empty-state-compact">
            <span class="overview-empty-state-icon"><i class="ri-user-heart-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.newOldEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.newOldEmptyText') }}</div>
          </div>
        </div>
        <div class="mini-cards">
          <div class="mini-card blue">
            <div class="mini-label">{{ t('overview.newVisitor') }}</div>
            <div class="mini-value">{{ newOldStats.newCountText }}</div>
            <div class="mini-percent">{{ newOldStats.newRate }}</div>
          </div>
          <div class="mini-card orange">
            <div class="mini-label">{{ t('overview.oldVisitor') }}</div>
            <div class="mini-value">{{ newOldStats.oldCountText }}</div>
            <div class="mini-percent">{{ newOldStats.oldRate }}</div>
          </div>
        </div>
      </div>
    </section>

    <section class="list-grid">
      <div class="card list-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon blue"><i class="ri-compass-3-line"></i></span>
            {{ t('overview.referer') }}
          </div>
          <button class="link-button" @click="openDetail('referer')">{{ t('overview.detail') }}</button>
        </div>
        <div class="table-wrapper">
          <table v-if="overviewLoading || refererRows.length > 0" class="ranking-table">
            <thead>
              <tr>
                <th class="domain-col">{{ t('overview.refererSite') }}</th>
                <th class="visitor-col">{{ t('common.visitors') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="overviewLoading">
                <td colspan="2">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else v-for="row in refererRows" :key="row.label">
                <td class="item-path" :title="row.label">{{ row.label }}</td>
                <td class="item-count">
                  <div class="bar-container">
                    <span class="bar-label">{{ formatCount(row.value) }}</span>
                    <div class="bar">
                      <div class="bar-fill" :style="{ width: `${row.percent}%` }"></div>
                      <span class="bar-percentage">{{ row.percent }}%</span>
                    </div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-else class="overview-empty-state overview-empty-state-table">
            <span class="overview-empty-state-icon"><i class="ri-compass-3-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.refererEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.refererEmptyText') }}</div>
          </div>
        </div>
      </div>
      <div class="card list-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon orange"><i class="ri-pages-line"></i></span>
            {{ t('overview.topPage') }}
          </div>
          <button class="link-button" @click="openDetail('url')">{{ t('overview.detail') }}</button>
        </div>
        <div class="table-wrapper">
          <table v-if="overviewLoading || urlRows.length > 0" class="ranking-table">
            <thead>
              <tr>
                <th class="url-col">{{ t('common.url') }}</th>
                <th class="pv-col">{{ t('common.viewCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="overviewLoading">
                <td colspan="2">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else v-for="row in urlRows" :key="row.label">
                <td class="item-path" :title="row.label">{{ row.label }}</td>
                <td class="item-count">
                  <div class="bar-container">
                    <span class="bar-label">{{ formatCount(row.value) }}</span>
                    <div class="bar">
                      <div class="bar-fill" :style="{ width: `${row.percent}%` }"></div>
                      <span class="bar-percentage">{{ row.percent }}%</span>
                    </div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-else class="overview-empty-state overview-empty-state-table">
            <span class="overview-empty-state-icon"><i class="ri-pages-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.topPageEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.topPageEmptyText') }}</div>
          </div>
        </div>
      </div>
      <div class="card list-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon green"><i class="ri-door-open-line"></i></span>
            {{ t('overview.entryPage') }}
          </div>
          <button class="link-button" @click="openDetail('entry')">{{ t('overview.detail') }}</button>
        </div>
        <div class="table-wrapper">
          <table v-if="overviewLoading || entryRows.length > 0" class="ranking-table">
            <thead>
              <tr>
                <th class="url-col">{{ t('common.url') }}</th>
                <th class="pv-col">{{ t('common.entryCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="overviewLoading">
                <td colspan="2">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else v-for="row in entryRows" :key="row.label">
                <td class="item-path" :title="row.label">{{ row.label }}</td>
                <td class="item-count">
                  <div class="bar-container">
                    <span class="bar-label">{{ formatCount(row.value) }}</span>
                    <div class="bar">
                      <div class="bar-fill" :style="{ width: `${row.percent}%` }"></div>
                      <span class="bar-percentage">{{ row.percent }}%</span>
                    </div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-else class="overview-empty-state overview-empty-state-table">
            <span class="overview-empty-state-icon"><i class="ri-door-open-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.entryPageEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.entryPageEmptyText') }}</div>
          </div>
        </div>
      </div>
    </section>

    <section class="geo-device-grid">
      <div class="card geo-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon blue"><i class="ri-map-pin-2-line"></i></span>
            {{ t('overview.region') }}
          </div>
          <div class="card-actions">
            <div class="view-toggle">
              <button
                class="data-map-toggle-btn"
                :class="{ active: mapView === 'china' }"
                @click="setMapView('china')"
              >
                {{ t('overview.domestic') }}
              </button>
              <button
                class="data-map-toggle-btn"
                :class="{ active: mapView === 'world' }"
                @click="setMapView('world')"
              >
                {{ t('overview.global') }}
              </button>
            </div>
            <button class="link-button" @click="openDetail('geo')">{{ t('overview.detail') }}</button>
          </div>
        </div>
        <div v-if="geoPending" class="geo-pending-note">
          {{ t('overview.geoPendingNotice') }}
        </div>
        <div v-if="hasGeoData" class="geo-content">
          <div class="map-container">
            <div id="geo-map" ref="geoMapRef"></div>
          </div>
          <div class="geo-list">
            <table class="ranking-table">
              <thead>
                <tr>
                  <th class="region-col">{{ t('common.province') }}</th>
                  <th class="visitor-col">{{ t('common.visitors') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="geoRows.length === 0">
                  <td colspan="2">{{ t('common.noData') }}</td>
                </tr>
                <tr v-else v-for="row in geoRows" :key="row.label">
                  <td class="item-path" :title="row.label">{{ row.label }}</td>
                  <td class="item-count">
                    <div class="bar-container">
                      <span class="bar-label">{{ formatCount(row.value) }}</span>
                      <div class="bar">
                        <div class="bar-fill" :style="{ width: `${row.percent}%` }"></div>
                        <span class="bar-percentage">{{ row.percent }}%</span>
                      </div>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="overview-empty-state geo-empty-state">
          <span class="overview-empty-state-icon"><i class="ri-map-pin-2-line"></i></span>
          <div class="overview-empty-state-title">{{ t('overview.geoEmptyTitle') }}</div>
          <div class="overview-empty-state-text">{{ t('overview.geoEmptyText') }}</div>
        </div>
      </div>
      <div class="card device-card" data-anim>
        <div class="card-header">
          <div class="card-title">
            <span class="card-icon blue"><i class="ri-device-line"></i></span>
            {{ t('overview.deviceAnalysis') }}
          </div>
          <button class="link-button" @click="openDetail('device')">{{ t('overview.detail') }}</button>
        </div>
        <div class="device-chart">
          <canvas v-show="hasDeviceData" ref="deviceChartRef"></canvas>
          <div v-if="!hasDeviceData" class="overview-empty-state overview-empty-state-compact">
            <span class="overview-empty-state-icon"><i class="ri-device-line"></i></span>
            <div class="overview-empty-state-title">{{ t('overview.deviceEmptyTitle') }}</div>
            <div class="overview-empty-state-text">{{ t('overview.deviceEmptyText') }}</div>
          </div>
        </div>
        <div class="device-cards">
          <div class="device-mini blue">
            <div class="device-label">{{ t('overview.deviceDesktop') }}</div>
            <div class="device-value">{{ deviceTotals.desktopText }}</div>
            <div class="device-percent">{{ deviceTotals.desktopRate }}</div>
          </div>
          <div class="device-mini orange">
            <div class="device-label">{{ t('overview.deviceMobile') }}</div>
            <div class="device-value">{{ deviceTotals.mobileText }}</div>
            <div class="device-percent">{{ deviceTotals.mobileRate }}</div>
          </div>
          <div class="device-mini apple">
            <div class="device-label">{{ t('overview.deviceApple') }}</div>
            <div class="device-value">{{ deviceTotals.appleText }}</div>
            <div class="device-percent">{{ deviceTotals.appleRate }}</div>
          </div>
          <div class="device-mini green">
            <div class="device-label">{{ t('overview.deviceOther') }}</div>
            <div class="device-value">{{ deviceTotals.otherText }}</div>
            <div class="device-percent">{{ deviceTotals.otherRate }}</div>
          </div>
        </div>
      </div>
    </section>

    <Dialog
      v-model:visible="serverDiskDialogVisible"
      modal
      :header="t('overview.serverStatusAllDisksTitle')"
      class="server-disk-dialog"
      :style="{ width: 'min(1120px, calc(100vw - 32px))' }"
    >
      <div class="server-disk-dialog-body">
        <div class="server-disk-dialog-toolbar">
          <div class="server-disk-dialog-summary">
            <span>{{ serverDiskDialogSummary }}</span>
            <span>{{ t('overview.serverStatusUpdatedAt', { value: serverStatusUpdatedAt }) }}</span>
          </div>
          <div class="server-disk-sort-tabs" role="tablist" :aria-label="t('overview.serverStatusDiskSort')">
            <button
              v-for="option in serverDiskSortOptions"
              :key="option.value"
              type="button"
              class="server-disk-sort-tab"
              :class="{ active: serverDiskSortKey === option.value }"
              :aria-pressed="serverDiskSortKey === option.value"
              @click="serverDiskSortKey = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <div class="server-disk-table-wrap">
          <table class="ranking-table server-disk-table">
            <thead>
              <tr>
                <th class="server-disk-status-col">{{ t('overview.serverStatusDiskColStatus') }}</th>
                <th class="server-disk-device-col">{{ t('overview.serverStatusDiskColDevice') }}</th>
                <th>{{ t('overview.serverStatusDiskColCapacity') }}</th>
                <th>{{ t('overview.serverStatusDiskColTemp') }}</th>
                <th>{{ t('overview.serverStatusDiskColHealth') }}</th>
                <th>{{ t('overview.serverStatusDiskColIo') }}</th>
                <th>{{ t('overview.serverStatusDiskColPower') }}</th>
                <th>{{ t('overview.serverStatusDiskColErrors') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="disk in serverDiskDialogRows" :key="disk.key" :class="disk.tone">
                <td>
                  <span class="server-disk-status-pill" :class="disk.tone">
                    <i :class="disk.icon" aria-hidden="true"></i>
                    {{ disk.status }}
                  </span>
                </td>
                <td>
                  <div class="server-disk-table-main">
                    <span :title="disk.model">{{ disk.model }}</span>
                    <small :title="disk.path">{{ disk.path }}</small>
                  </div>
                </td>
                <td>
                  <div class="server-disk-table-stack">
                    <span>{{ disk.capacity }}</span>
                    <small>{{ disk.type }}</small>
                  </div>
                </td>
                <td>{{ disk.temperature }}</td>
                <td>{{ disk.health }}</td>
                <td>
                  <div class="server-disk-table-stack">
                    <span>{{ t('overview.serverStatusWritten') }} {{ disk.written }}</span>
                    <small>{{ t('overview.serverStatusRead') }} {{ disk.read }}</small>
                  </div>
                </td>
                <td>{{ disk.powerOn }}</td>
                <td>
                  <div class="server-disk-table-stack">
                    <span>{{ t('overview.serverStatusErrors') }} {{ disk.mediaErrors }}</span>
                    <small>{{ t('overview.serverStatusUnsafeShort', { value: disk.unsafe }) }}</small>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </Dialog>

    <div
      class="detail-overlay"
      :class="{ open: detailOpen, modal: detailLayout === 'modal' }"
      :aria-hidden="detailOpen ? 'false' : 'true'"
      @click.self="closeDetail"
    >
      <div
        class="detail-panel"
        :class="{ modal: detailLayout === 'modal', 'show-logs': detailMode === 'logs' }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="detail-title"
      >
        <div class="detail-header">
          <div>
            <div class="detail-title" id="detail-title">{{ detailTitle }}</div>
            <div class="detail-sub" id="detail-subtitle">{{ detailSubtitle }}</div>
          </div>
          <button
            class="detail-close"
            type="button"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="closeDetail"
          ></button>
        </div>
        <div class="detail-body">
          <div class="detail-filters" v-if="detailMode === 'logs'" aria-hidden="false">
            <div
              v-for="(section, sectionIndex) in detailFilterLayout"
              :key="sectionIndex"
              :class="section.className || 'detail-filter-section'"
            >
              <template v-for="(item, itemIndex) in section.items" :key="itemIndex">
                <div
                  v-if="typeof item === 'string' && detailFilterFields[detailLogScope][item]"
                  class="detail-filter-group"
                >
                  <template v-if="detailFilterFields[detailLogScope][item].type === 'checkbox'">
                    <Checkbox
                      v-model="detailFilterState[detailLogScope][item]"
                      binary
                      :inputId="`detail-${detailLogScope}-${item}`"
                    />
                    <label :for="`detail-${detailLogScope}-${item}`">
                      {{ detailFilterFields[detailLogScope][item].label }}
                    </label>
                  </template>
                  <template v-else>
                    <span class="detail-filter-label">{{ detailFilterFields[detailLogScope][item].label }}</span>
                    <Dropdown
                      v-if="detailFilterFields[detailLogScope][item].type === 'select'"
                      v-model="detailFilterState[detailLogScope][item]"
                      class="detail-filter-select"
                      :options="getDetailFieldOptions(detailLogScope, item)"
                      optionLabel="label"
                      optionValue="value"
                    />
                    <DatePicker
                      v-else-if="detailFilterFields[detailLogScope][item].inputType === 'datetime-local'"
                      v-model="detailFilterState[detailLogScope][item]"
                      class="detail-filter-datepicker detail-filter-datetime"
                      dateFormat="yy-mm-dd"
                      updateModelType="string"
                      showTime
                      hourFormat="24"
                      showButtonBar
                      :showClear="true"
                    />
                    <InputNumber
                      v-else-if="detailFilterFields[detailLogScope][item].inputType === 'number'"
                      v-model="detailFilterState[detailLogScope][item]"
                      class="detail-filter-input"
                      :min="detailFilterFields[detailLogScope][item].min"
                      :max="detailFilterFields[detailLogScope][item].max"
                      :step="1"
                      :useGrouping="false"
                      :minFractionDigits="0"
                      :maxFractionDigits="0"
                      :placeholder="detailFilterFields[detailLogScope][item].placeholder"
                    />
                    <InputText
                      v-else
                      v-model="detailFilterState[detailLogScope][item]"
                      class="detail-filter-input"
                      :type="detailFilterFields[detailLogScope][item].inputType || 'text'"
                      :placeholder="detailFilterFields[detailLogScope][item].placeholder"
                    />
                  </template>
                </div>
                <div v-else-if="typeof item === 'object' && item.type === 'range'" class="detail-filter-group detail-filter-range">
                  <span class="detail-filter-label">{{ item.label || '' }}</span>
                  <DatePicker
                    v-model="detailFilterState[detailLogScope][item.startKey]"
                    class="detail-filter-datepicker detail-filter-datetime"
                    dateFormat="yy-mm-dd"
                    updateModelType="string"
                    showTime
                    hourFormat="24"
                    showButtonBar
                    :showClear="true"
                  />
                  <span class="detail-filter-divider">{{ item.divider || t('common.to') }}</span>
                  <DatePicker
                    v-model="detailFilterState[detailLogScope][item.endKey]"
                    class="detail-filter-datepicker detail-filter-datetime"
                    dateFormat="yy-mm-dd"
                    updateModelType="string"
                    showTime
                    hourFormat="24"
                    showButtonBar
                    :showClear="true"
                  />
                </div>
                <Button v-else-if="typeof item === 'object' && item.type === 'apply'" severity="primary" @click="applyDetailFilters">
                  {{ item.label || t('common.apply') }}
                </Button>
              </template>
            </div>
          </div>
          <div class="detail-ip-notice" v-if="detailMode === 'logs' && detailIpParsing">
            {{ t('logs.ipParsing', { progress: detailIpParsingProgressLabel }) }}
          </div>
          <div class="detail-ip-notice" v-if="detailMode === 'logs' && detailParsingPending">
            {{ t('logs.backfillParsing', { progress: detailParsingPendingProgressLabel }) }}
          </div>
          <div class="detail-ip-notice" v-if="detailMode === 'logs' && (detailIpGeoParsing || detailIpGeoPending)">
            {{ detailIpGeoParsingMessage }}
          </div>
          <div class="detail-list">
            <div
              v-if="!showDetailLoading && !detailError && ((detailMode !== 'logs' && detailRankingRows.length === 0) || (detailMode === 'logs' && detailLogRows.length === 0))"
              class="detail-empty-state"
            >
              <span class="detail-empty-state-icon">
                <i :class="detailMode === 'logs' ? 'ri-file-search-line' : 'ri-bar-chart-box-line'"></i>
              </span>
              <div class="detail-empty-state-title">
                {{ detailMode === 'logs' ? t('overview.detailLogsEmptyTitle') : t('overview.detailEmptyTitle') }}
              </div>
              <div class="detail-empty-state-text">
                {{ detailMode === 'logs' ? t('overview.detailLogsEmptyText') : t('overview.detailEmptyText') }}
              </div>
            </div>
            <div v-else class="table-wrapper">
              <table class="ranking-table" :class="{ 'detail-logs': detailMode === 'logs' }">
                <thead>
                  <tr>
                    <th v-for="column in detailColumns" :key="column.label" :class="column.className">
                      {{ column.label }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="showDetailLoading">
                    <td :colspan="detailColumns.length">{{ t('common.loading') }}</td>
                  </tr>
                  <tr v-else-if="detailError">
                    <td :colspan="detailColumns.length">{{ t('common.requestFailed') }}</td>
                  </tr>
                  <template v-else-if="detailMode !== 'logs'">
                    <tr v-for="row in detailRankingRows" :key="row.label">
                      <td class="item-path" :title="row.label">{{ row.label }}</td>
                      <td class="item-count">
                        <div class="bar-container">
                          <span class="bar-label">{{ formatCount(row.value) }}</span>
                          <div class="bar">
                            <div class="bar-fill" :style="{ width: `${row.percent}%` }"></div>
                            <span class="bar-percentage">{{ row.percent }}%</span>
                          </div>
                        </div>
                      </td>
                    </tr>
                  </template>
                  <template v-else>
                    <tr v-for="(row, rowIndex) in detailLogRows" :key="rowIndex">
                      <td
                        v-for="(cell, cellIndex) in row.cells"
                        :key="cellIndex"
                        :class="cell.className"
                        :title="cell.title"
                      >
                        {{ cell.value }}
                      </td>
                    </tr>
                  </template>
                </tbody>
                <tfoot v-if="detailMode === 'logs' && detailLogRows.length > 0">
                  <tr class="detail-load-row">
                    <td :colspan="detailColumns.length">
                      <Button outlined :disabled="detailLoadMoreDisabled" @click="loadMoreDetail">
                        {{ detailLoadMoreText }}
                      </Button>
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>

    <ParsingOverlay :website-id="currentWebsiteId" @finished="refreshAll" />
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import Dialog from 'primevue/dialog';
import type { EChartsType } from 'echarts/core';
import {
  fetchBrowserStats,
  fetchDeviceStats,
  fetchLocationStats,
  fetchLogs,
  fetchOSStats,
  fetchOverallStats,
  fetchRefererStats,
  fetchServerStatus,
  fetchSessions,
  fetchTimeSeriesStats,
  fetchUrlStats,
  fetchWebsites,
} from '@/api';
import type { ServerDiskStatus, ServerStatusResponse, SimpleSeriesStats, TimeSeriesStats, WebsiteInfo } from '@/api/types';
import { formatTraffic, getInitialServerStatusEnabled, getUserPreference, saveUserPreference } from '@/utils';
import { Chart } from '@/utils/chartjs';
import ParsingOverlay from '@/components/ParsingOverlay.vue';
import HeaderToolbar from '@/components/HeaderToolbar.vue';
import SystemNotifications from '@/components/SystemNotifications.vue';
import ThemeToggle from '@/components/ThemeToggle.vue';
import WebsiteSelect from '@/components/WebsiteSelect.vue';
import {
  chinaProvinceAlias,
  chinaProvinceMap,
  formatBrowserLabel,
  formatDeviceLabel,
  formatLocationLabel,
  formatOSLabel,
  formatRefererLabel,
  normalizeChinaProvinceName,
  normalizeDeviceCategory,
} from '@/i18n/mappings';
import { normalizeLocale } from '@/i18n';

type ThemeContext = {
  isDark: { value: boolean };
};

type EChartsCore = typeof import('echarts/core');
type ServerDiskSortKey = 'risk' | 'temperature' | 'capacity' | 'name';

let echartsCore: EChartsCore | null = null;
let geoMapRuntimePromise: Promise<EChartsCore> | null = null;

const theme = inject<ThemeContext>('theme', null);
const setLiveVisitorCount = inject<((value: number | null) => void) | null>('setLiveVisitorCount', null);
const { t, n, locale } = useI18n({ useScope: 'global' });
const currentLocale = computed(() => normalizeLocale(locale.value));

const websites = ref<WebsiteInfo[]>([]);
const websitesLoading = ref(true);
const currentWebsiteId = ref('');
const dateRange = ref('today');
const chartView = ref<'hourly' | 'daily'>('hourly');
const mapView = ref<'china' | 'world'>('china');
const overviewLoading = ref(false);
const chartError = ref('');
const autoRefreshEnabled = ref(getUserPreference('overviewAutoRefresh', 'false') === 'true');
const autoRefreshAllowed = computed(() => dateRange.value === 'today');
const maxSelectableDate = new Date();

function toggleAutoRefresh() {
  if (!autoRefreshAllowed.value) {
    return;
  }
  autoRefreshEnabled.value = !autoRefreshEnabled.value;
}

const overall = ref<Record<string, any> | null>(null);
const timeSeriesData = ref<TimeSeriesStats | null>(null);
const urlStats = ref<SimpleSeriesStats | null>(null);
const refererStats = ref<SimpleSeriesStats | null>(null);
const browserStats = ref<SimpleSeriesStats | null>(null);
const osStats = ref<SimpleSeriesStats | null>(null);
const deviceStats = ref<SimpleSeriesStats | null>(null);
const geoData = ref<Array<{ name: string; value: number; percentage: number }>>([]);
const geoPending = ref(false);
const bootServerStatusEnabled = getInitialServerStatusEnabled();
const serverStatus = ref<ServerStatusResponse | null>(readCachedServerStatus(bootServerStatusEnabled));
const initialServerStatusEnabled = ref<boolean | null>(bootServerStatusEnabled ?? (serverStatus.value?.enabled ? true : null));
const serverStatusLoading = ref(false);
const serverStatusError = ref('');
const serverDiskDialogVisible = ref(false);
const serverDiskSortKey = ref<ServerDiskSortKey>('risk');

const visitsChartRef = ref<HTMLCanvasElement | null>(null);
const newOldChartRef = ref<HTMLCanvasElement | null>(null);
const deviceChartRef = ref<HTMLCanvasElement | null>(null);
const geoMapRef = ref<HTMLDivElement | null>(null);

let visitsChart: Chart | null = null;
let newOldChart: Chart | null = null;
let deviceChart: Chart | null = null;
let geoMapChart: EChartsType | null = null;

let overviewRequestId = 0;
let chartRequestId = 0;
let mapRequestId = 0;
let autoRefreshTimer: number | null = null;
let serverStatusTimer: number | null = null;

const DETAIL_LIMIT = 50;
const DETAIL_LOG_PAGE_SIZE = 30;
const AUTO_REFRESH_INTERVAL = 3000;
const SERVER_STATUS_CACHE_KEY = 'nginxpulse:serverStatusSnapshot';

const dateRangeOptions = computed(() => [
  { value: 'today', label: t('common.today') },
  { value: 'yesterday', label: t('common.yesterday') },
  { value: 'week', label: t('common.week') },
  { value: 'last7days', label: t('common.last7Days') },
  { value: 'month', label: t('overview.month') },
  { value: 'last30days', label: t('common.last30Days') },
]);

const rangeTabs = computed(() => [
  { value: 'today', label: t('overview.todayShort') },
  { value: 'yesterday', label: t('overview.yesterdayShort') },
  { value: 'last7days', label: t('overview.last7DaysShort') },
  { value: 'last30days', label: t('overview.last30DaysShort') },
]);
const specificDateValue = computed<string | null>({
  get: () => (isSpecificDateValue(dateRange.value) ? dateRange.value : null),
  set: (value) => {
    if (typeof value === 'string' && isSpecificDateValue(value)) {
      dateRange.value = value;
      return;
    }
    if (isSpecificDateValue(dateRange.value)) {
      dateRange.value = 'today';
    }
  },
});
const isSpecificDateSelected = computed(() => isSpecificDateValue(dateRange.value));

const trafficText = computed(() => formatTraffic(overall.value?.traffic ?? 0));

const liveVisitorText = computed(() => {
  const value = overall.value?.activeVisitorCount;
  return Number.isFinite(value) ? n(Number(value)) : t('common.none');
});

const metricLabels = computed(() => getMetricCompareLabels(dateRange.value));
const forecastRateWeight = computed(() => getForecastRateWeight(dateRange.value));
const FORECAST_EXAMPLE_PROGRESS = 0.4;
const FORECAST_EXAMPLE_CURRENT = 4000;
const FORECAST_EXAMPLE_PROGRESS_PROJECTED = Math.round(FORECAST_EXAMPLE_CURRENT / FORECAST_EXAMPLE_PROGRESS);
const FORECAST_EXAMPLE_RATE_PROJECTED = 11200;
const forecastExampleFinalProjected = computed(() =>
  Math.round(
    FORECAST_EXAMPLE_RATE_PROJECTED * forecastRateWeight.value
      + FORECAST_EXAMPLE_PROGRESS_PROJECTED * (1 - forecastRateWeight.value)
  )
);
const forecastHintText = computed(() =>
  [
    t('overview.forecastHint', {
      rateWeight: n(forecastRateWeight.value, 'percent'),
      progressWeight: n(1 - forecastRateWeight.value, 'percent'),
    }),
    t('overview.forecastHintExample', {
      progress: n(FORECAST_EXAMPLE_PROGRESS, 'percent'),
      current: n(FORECAST_EXAMPLE_CURRENT),
      progressProjected: n(FORECAST_EXAMPLE_PROGRESS_PROJECTED),
      rateProjected: n(FORECAST_EXAMPLE_RATE_PROJECTED),
      finalProjected: n(forecastExampleFinalProjected.value),
    }),
  ].join('\n')
);
const forecastHintAria = computed(() => t('overview.forecastHintAria'));

const statusMetrics = computed(() => {
  const hits = overall.value?.statusCodeHits;
  const prevHits = overall.value?.statusCodeHitsPrevious;
  const prevLabel = metricLabels.value.prev;
  if (!hits) {
    return {
      total: t('common.none'),
      s2xx: t('common.none'),
      s3xx: t('common.none'),
      s4xx: t('common.none'),
      s5xx: t('common.none'),
      prevLabel,
      prevTotal: t('common.none'),
      deltaText: t('common.none'),
      deltaClass: '',
    }
  }
  const s2xx = Number(hits.s2xx) || 0;
  const s3xx = Number(hits.s3xx) || 0;
  const s4xx = Number(hits.s4xx) || 0;
  const s5xx = Number(hits.s5xx) || 0;
  const total = s2xx + s3xx + s4xx + s5xx;

  let prevTotal = null;
  if (prevHits) {
    const prev2xx = Number(prevHits.s2xx) || 0;
    const prev3xx = Number(prevHits.s3xx) || 0;
    const prev4xx = Number(prevHits.s4xx) || 0;
    const prev5xx = Number(prevHits.s5xx) || 0;
    prevTotal = prev2xx + prev3xx + prev4xx + prev5xx;
  }

  const delta = buildDeltaTextFromTotals(total, prevTotal);

  return {
    total: formatCount(total),
    s2xx: formatCount(s2xx),
    s3xx: formatCount(s3xx),
    s4xx: formatCount(s4xx),
    s5xx: formatCount(s5xx),
    prevLabel,
    prevTotal: prevTotal === null ? t('common.none') : formatCount(prevTotal),
    deltaText: delta.text,
    deltaClass: delta.className,
  }
});

const metricTiles = computed(() => {
  const current = overall.value || {};
  const compare = current.compare || {};
  return {
    pv: buildMetricTile('pv', current, compare),
    uv: buildMetricTile('uv', current, compare),
    session: buildMetricTile('session', current, compare),
  }
});

const newOldStats = computed(() => {
  const safe = overall.value || {};
  const newCount = Math.max(0, safe.newVisitorCount || 0);
  const oldCount = Math.max(0, safe.returningVisitorCount || 0);
  const total = newCount + oldCount;
  const newRate = total ? ((newCount / total) * 100).toFixed(2) + '%' : '0%';
  const oldRate = total ? ((oldCount / total) * 100).toFixed(2) + '%' : '0%';
  return {
    newCount,
    oldCount,
    newCountText: formatCount(newCount),
    oldCountText: formatCount(oldCount),
    newRate,
    oldRate,
    prevNew: Math.max(0, safe.prevNewVisitorCount || 0),
    prevOld: Math.max(0, safe.prevReturningVisitorCount || 0),
    labels: getCompareLabels(dateRange.value),
  }
});

const deviceTotals = computed(() => {
  const stats = deviceStats.value;
  const totals = { desktop: 0, mobile: 0, apple: 0, other: 0 };

  if (stats?.key && stats.uv) {
    stats.key.forEach((label, index) => {
      const value = stats.uv[index] || 0;
      const category = normalizeDeviceCategory(String(label));
      if (category === 'desktop') {
        totals.desktop += value;
      } else if (category === 'mobile') {
        totals.mobile += value;
      } else if (category === 'apple') {
        totals.apple += value;
      } else {
        totals.other += value;
      }
    });
  }

  const total = totals.desktop + totals.mobile + totals.apple + totals.other;
  const desktopRate = total ? ((totals.desktop / total) * 100).toFixed(2) + '%' : '0%';
  const mobileRate = total ? ((totals.mobile / total) * 100).toFixed(2) + '%' : '0%';
  const otherRate = total ? ((totals.other / total) * 100).toFixed(2) + '%' : '0%';
  const appleRate = total ? ((totals.apple / total) * 100).toFixed(2) + '%' : '0%';

  return {
    desktop: totals.desktop,
    mobile: totals.mobile,
    apple: totals.apple,
    other: totals.other,
    desktopText: formatCount(totals.desktop),
    mobileText: formatCount(totals.mobile),
    appleText: formatCount(totals.apple),
    otherText: formatCount(totals.other),
    desktopRate,
    mobileRate,
    appleRate,
    otherRate,
  }
});

const refererRows = computed(() =>
  buildRankingRows(refererStats.value, false, (label) => formatRefererLabel(label, currentLocale.value, t))
);
const urlRows = computed(() => buildRankingRows(urlStats.value, true));
const entryRows = computed(() => buildRankingRows(overall.value?.entryPages, false));
const geoRows = computed(() => buildGeoRows(geoData.value));
const hasVisitsTrendData = computed(() => hasTimeSeriesData(timeSeriesData.value));
const hasNewOldData = computed(() => newOldStats.value.newCount + newOldStats.value.oldCount > 0);
const hasGeoData = computed(() => geoRows.value.length > 0);
const hasDeviceData = computed(() => deviceTotals.value.desktop + deviceTotals.value.mobile + deviceTotals.value.apple + deviceTotals.value.other > 0);
const serverStatusVisible = computed(() => {
  if (serverStatus.value) {
    return Boolean(serverStatus.value.enabled);
  }
  return initialServerStatusEnabled.value === true;
});
const serverDisks = computed(() => serverStatus.value?.disks || []);
const riskSortedServerDisks = computed(() =>
  [...serverDisks.value].sort((a, b) => diskRiskScore(b) - diskRiskScore(a))
);
const primaryServerDisk = computed(() => riskSortedServerDisks.value[0] || null);
const hottestServerDisk = computed(() =>
  [...serverDisks.value].sort((a, b) => (numberValue(b.temperature_celsius) ?? -1) - (numberValue(a.temperature_celsius) ?? -1))[0] || null
);
const serverDiskSummaryText = computed(() => {
  const count = serverStatus.value?.disk_count ?? serverDisks.value.length;
  const worst = primaryServerDisk.value;
  if (!worst) {
    return t('overview.serverStatusDiskCount', { value: count });
  }
  const diskName = worst.model || worst.name || t('overview.serverStatusDiskFallback');
  if (count <= 1) {
    return t('overview.serverStatusDiskSingleSummary', {
      count,
      disk: diskName,
    });
  }
  if (serverDisks.value.every((disk) => diskTone(disk) === 'good')) {
    return t('overview.serverStatusDiskAllHealthySummary', { count });
  }
  return t('overview.serverStatusDiskAttentionSummary', {
    count,
    disk: diskName,
  });
});
const serverDiskActionText = computed(() => {
  if (serverDisks.value.length <= 1) {
    return t('overview.serverStatusViewDiskDetail');
  }
  return t('overview.serverStatusViewAllDisks', { count: serverDisks.value.length });
});
const serverDiskSummaryStats = computed(() => {
  const disk = primaryServerDisk.value;
  if (!disk) {
    return [];
  }
  return [
    {
      key: 'temp',
      label: t('overview.serverStatusDiskColTemp'),
      value: formatTemperature(numberValue(disk.temperature_celsius)),
    },
    {
      key: 'health',
      label: t('overview.serverStatusDiskColHealth'),
      value: formatDiskHealth(disk),
    },
    {
      key: 'written',
      label: t('overview.serverStatusWritten'),
      value: formatOptionalTraffic(disk.data_units_written_bytes),
    },
  ];
});
const serverDiskDialogSummary = computed(() => {
  const riskyCount = serverDisks.value.filter((disk) => diskTone(disk) !== 'good').length;
  return t('overview.serverStatusAllDisksSummary', {
    count: serverDisks.value.length,
    risky: riskyCount,
  });
});
const serverDiskSortOptions = computed<Array<{ label: string; value: ServerDiskSortKey }>>(() => [
  { label: t('overview.serverStatusDiskSortRisk'), value: 'risk' },
  { label: t('overview.serverStatusDiskSortTemp'), value: 'temperature' },
  { label: t('overview.serverStatusDiskSortCapacity'), value: 'capacity' },
  { label: t('overview.serverStatusDiskSortName'), value: 'name' },
]);
const serverDiskDialogRows = computed(() =>
  sortServerDisks(serverDisks.value, serverDiskSortKey.value).map((disk, index) => ({
    key: `${disk.path || disk.name || disk.model || 'disk'}-${index}`,
    tone: diskTone(disk),
    icon: diskTone(disk) === 'danger'
      ? 'ri-error-warning-line'
      : diskTone(disk) === 'warn'
        ? 'ri-alert-line'
        : 'ri-checkbox-circle-line',
    status: formatDiskStatus(disk),
    model: disk.model || disk.name || t('overview.serverStatusDiskFallback'),
    path: disk.path || disk.smartctl_path || disk.name || t('common.none'),
    type: disk.type ? String(disk.type).toUpperCase() : t('common.none'),
    capacity: disk.size_bytes ? formatOptionalTraffic(disk.size_bytes) : t('common.none'),
    temperature: formatTemperature(numberValue(disk.temperature_celsius)),
    health: formatDiskHealth(disk),
    written: formatOptionalTraffic(disk.data_units_written_bytes),
    read: formatOptionalTraffic(disk.data_units_read_bytes),
    powerOn: formatServerHours(disk.power_on_hours),
    mediaErrors: formatCount(Number(disk.media_errors ?? 0)),
    unsafe: formatCount(Number(disk.unsafe_shutdowns ?? 0)),
  }))
);
const serverStatusClass = computed(() => `is-${serverStatus.value?.status || 'ok'}`);
const serverStatusLabel = computed(() => {
  switch (serverStatus.value?.status) {
    case 'ok':
      return t('overview.serverStatusOk');
    case 'warning':
      return t('overview.serverStatusWarning');
    case 'partial':
      return t('overview.serverStatusPartial');
    case 'error':
      return t('overview.serverStatusErrorState');
    default:
      return t('common.none');
  }
});
const serverStatusUpdatedAt = computed(() => formatServerStatusTime(serverStatus.value?.updated_at));
const serverHealthScore = computed(() => {
  const metrics = serverStatus.value?.metrics || {};
  const disks = serverDisks.value;
  const cpuTemp = metricNumber(metrics, 'cpu_temp_celsius');
  const diskTemp = maxDiskTemperature(disks);
  const nvmeTemp = Math.max(
    metricNumber(metrics, 'nvme_temp_celsius') ?? 0,
    diskTemp ?? 0
  ) || null;
  const diskRemaining = minDiskRemaining(disks);
  const mediaErrors = disks.reduce((total, disk) => total + Number(disk.media_errors || 0), 0);
  let score = diskRemaining ?? 100;
  if (disks.some((disk) => disk.health_passed === false)) {
    score = Math.min(score, 35);
  }
  if ((serverStatus.value?.errors || []).length > 0) {
    score -= 15;
  }
  score -= temperaturePenalty(cpuTemp);
  score -= temperaturePenalty(nvmeTemp);
  score -= Math.min(25, mediaErrors * 4);
  return clampNumber(Math.round(score), 0, 100);
});
const serverHealthTone = computed(() => {
  if (serverHealthScore.value < 50 || serverStatus.value?.status === 'error') {
    return 'danger';
  }
  if (serverHealthScore.value < 80 || ['warning', 'partial'].includes(serverStatus.value?.status || '')) {
    return 'warn';
  }
  return 'good';
});
const serverStatusSensorRows = computed(() => {
  const metrics = serverStatus.value?.metrics || {};
  const cpuFan = metricNumber(metrics, 'cpu_fan_rpm');
  const chassisFan = metricNumber(metrics, 'chassis_fan1_rpm');
  const fanLoad = maxMetricNumber(cpuFan, chassisFan);
  const diskTemp = maxDiskTemperature(serverDisks.value);
  const nvmeTemp = Math.max(metricNumber(metrics, 'nvme_temp_celsius') ?? 0, diskTemp ?? 0) || null;
  const hottestDisk = hottestServerDisk.value;
  return [
    {
      key: 'cpu-temp',
      icon: 'ri-cpu-line',
      label: t('overview.serverStatusCpuTemp'),
      value: formatTemperature(metricNumber(metrics, 'cpu_temp_celsius')),
      sub: t('overview.serverStatusNormalRange'),
      tone: temperatureTone(metricNumber(metrics, 'cpu_temp_celsius')),
      style: sensorStyle(temperaturePercent(metricNumber(metrics, 'cpu_temp_celsius'))),
    },
    {
      key: 'board-temp',
      icon: 'ri-dashboard-3-line',
      label: t('overview.serverStatusBoardTempLabel'),
      value: formatTemperature(metricNumber(metrics, 'board_temp_celsius')),
      sub: t('overview.serverStatusBoardTempHint'),
      tone: temperatureTone(metricNumber(metrics, 'board_temp_celsius')),
      style: sensorStyle(temperaturePercent(metricNumber(metrics, 'board_temp_celsius'))),
    },
    {
      key: 'nvme-temp',
      icon: 'ri-hard-drive-3-line',
      label: t('overview.serverStatusNvmeTemp'),
      value: formatTemperature(nvmeTemp),
      sub: hottestDisk
        ? t('overview.serverStatusHottestDisk', { disk: hottestDisk.name || hottestDisk.model || t('overview.serverStatusDiskFallback') })
        : t('overview.serverStatusDiskCount', { value: serverStatus.value?.disk_count ?? serverStatus.value?.disks?.length ?? 0 }),
      tone: temperatureTone(nvmeTemp),
      style: sensorStyle(temperaturePercent(nvmeTemp)),
    },
    {
      key: 'cpu-fan',
      icon: 'ri-fan-line',
      label: t('overview.serverStatusFanSpeed'),
      value: '',
      cpuValue: formatFanRPMValue(cpuFan),
      chassisValue: formatFanRPMValue(chassisFan),
      sub: '',
      tone: 'neutral',
      style: fanSensorStyle(cpuFan ?? chassisFan, fanLoad),
    },
  ];
});
const serverStatusMessages = computed(() => {
  const messages = [...(serverStatus.value?.errors || [])];
  const missing = serverStatus.value?.missing_metrics || [];
  if (missing.length > 0) {
    messages.push(t('overview.serverStatusMissingMetrics', { value: missing.join(', ') }));
  }
  return messages;
});

const dailyViewDisabled = computed(() => ['today', 'yesterday'].includes(dateRange.value) || isSpecificDateValue(dateRange.value));

const isDark = computed(() => theme?.isDark.value ?? false);

const detailOpen = ref(false);
const detailConfig = ref<DetailConfig | null>(null);
const detailLogScope = ref<'status' | 'pv' | 'uv' | 'session'>('status');
const detailLoading = ref(false);
const detailError = ref(false);
const detailIpParsing = ref(false);
const detailIpParsingProgress = ref<number | null>(null);
const detailIpParsingEstimatedRemainingSeconds = ref<number | null>(null);
const detailIpGeoParsing = ref(false);
const detailIpGeoPending = ref(false);
const detailIpGeoProgress = ref<number | null>(null);
const detailIpGeoEstimatedRemainingSeconds = ref<number | null>(null);
const detailParsingPending = ref(false);
const detailParsingPendingProgress = ref<number | null>(null);
const detailIpParsingProgressText = computed(() => {
  if (detailIpParsingProgress.value === null) {
    return '';
  }
  if (detailIpParsingEstimatedRemainingSeconds.value) {
    const duration = formatDurationSeconds(detailIpParsingEstimatedRemainingSeconds.value);
    return t('parsing.progressWithRemaining', { value: detailIpParsingProgress.value, duration });
  }
  return t('parsing.progress', { value: detailIpParsingProgress.value });
});
const detailIpParsingProgressLabel = computed(() => {
  if (!detailIpParsingProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${detailIpParsingProgressText.value}）`
    : ` (${detailIpParsingProgressText.value})`;
});

const detailIpGeoProgressText = computed(() => {
  if (detailIpGeoProgress.value === null) {
    return '';
  }
  return t('parsing.progress', { value: detailIpGeoProgress.value });
});
const detailIpGeoProgressLabel = computed(() => {
  if (!detailIpGeoProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${detailIpGeoProgressText.value}）`
    : ` (${detailIpGeoProgressText.value})`;
});
const detailIpGeoRemainingLabel = computed(() => {
  if (detailIpGeoEstimatedRemainingSeconds.value === null) {
    return '';
  }
  return formatDurationSeconds(detailIpGeoEstimatedRemainingSeconds.value);
});
const detailIpGeoParsingMessage = computed(() => {
  if (detailIpGeoProgressLabel.value && detailIpGeoRemainingLabel.value) {
    return t('logs.ipGeoParsingProgress', {
      progress: detailIpGeoProgressLabel.value,
      remaining: detailIpGeoRemainingLabel.value,
    });
  }
  if (detailIpGeoProgressLabel.value) {
    return t('logs.ipGeoParsingProgressOnly', { progress: detailIpGeoProgressLabel.value });
  }
  return t('logs.ipGeoParsing');
});

const detailParsingPendingProgressText = computed(() => {
  if (detailParsingPendingProgress.value === null) {
    return '';
  }
  return t('parsing.progress', { value: detailParsingPendingProgress.value });
});
const detailParsingPendingProgressLabel = computed(() => {
  if (!detailParsingPendingProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${detailParsingPendingProgressText.value}）`
    : ` (${detailParsingPendingProgressText.value})`;
});
const detailLoadState = ref<'ready' | 'loading' | 'done' | 'error'>('ready');
const detailHasMore = ref(false);
const detailPage = ref(1);
const detailRankingRows = ref<Array<{ label: string; value: number; percent: number }>>([]);
const detailLogRows = ref<Array<{ cells: Array<{ value: string; className?: string; title?: string }> }>>([]);

let detailRequestId = 0;
let latestOverall: Record<string, any> | null = null;
let latestOverallKey = '';

const detailMode = computed(() => (detailConfig.value?.mode === 'logs' ? 'logs' : 'table'));
const detailLayout = computed(() => detailConfig.value?.layout || 'panel');
const detailTitle = computed(() => detailConfig.value?.title || t('overview.detail'));
const detailSubtitle = computed(() => buildDetailSubtitle());
const detailColumns = computed(() => buildDetailColumns(detailConfig.value));
const showDetailLoading = computed(
  () => detailLoading.value && (detailMode.value === 'logs' ? detailLogRows.value.length === 0 : detailRankingRows.value.length === 0)
);
const detailLoadMoreText = computed(() => {
  switch (detailLoadState.value) {
    case 'loading':
      return t('common.loading');
    case 'done':
      return t('overview.loadMoreDone');
    case 'error':
      return t('overview.loadMoreRetry');
    default:
      return t('overview.loadMore');
  }
});
const detailLoadMoreDisabled = computed(() => detailLoadState.value === 'loading' || detailLoadState.value === 'done');

const DETAIL_FILTER_DEFAULTS = {
  status: {
    statusClass: 'all',
    statusCode: null,
    excludeInternal: false,
    ipFilter: '',
  },
  pv: {
    timeStart: '',
    timeEnd: '',
    locationFilter: '',
    urlFilter: '',
    excludeInternal: false,
    ipFilter: '',
  },
  uv: {
    isNew: 'all',
    timeStart: '',
    timeEnd: '',
    ipFilter: '',
  },
  session: {
    ipFilter: '',
    deviceFilter: 'all',
    browserFilter: 'all',
    osFilter: 'all',
  },
}

const detailFilterFields = computed(() => ({
  status: {
    statusClass: {
      type: 'select',
      label: t('overview.statusCode'),
      options: [
        { value: 'all', label: t('common.all') },
        { value: '2xx', label: '2xx' },
        { value: '3xx', label: '3xx' },
        { value: '4xx', label: '4xx' },
        { value: '5xx', label: '5xx' },
      ],
    },
    statusCode: {
      type: 'input',
      label: t('overview.exact'),
      inputType: 'number',
      min: 100,
      max: 599,
      placeholder: t('overview.statusCodePlaceholder'),
    },
    excludeInternal: {
      type: 'checkbox',
      label: t('overview.excludeInternal'),
    },
    ipFilter: {
      type: 'input',
      label: t('common.ip'),
      inputType: 'text',
      placeholder: t('overview.ipPlaceholder'),
    },
  },
  pv: {
    timeStart: {
      type: 'input',
      label: t('overview.visitTime'),
      inputType: 'datetime-local',
    },
    timeEnd: {
      type: 'input',
      inputType: 'datetime-local',
    },
    locationFilter: {
      type: 'input',
      label: t('overview.ipLocation'),
      inputType: 'text',
      placeholder: t('overview.locationPlaceholder'),
    },
    urlFilter: {
      type: 'input',
      label: t('overview.visitLink'),
      inputType: 'text',
      placeholder: t('overview.urlPlaceholder'),
    },
    excludeInternal: {
      type: 'checkbox',
      label: t('overview.excludeInternal'),
    },
    ipFilter: {
      type: 'input',
      label: t('common.ip'),
      inputType: 'text',
      placeholder: t('overview.ipPlaceholder'),
    },
  },
  uv: {
    isNew: {
      type: 'select',
      label: t('overview.isNewVisitor'),
      options: [
        { value: 'all', label: t('common.all') },
        { value: 'new', label: t('overview.newVisitor') },
        { value: 'returning', label: t('overview.oldVisitor') },
      ],
    },
    timeStart: {
      type: 'input',
      label: t('overview.visitTime'),
      inputType: 'datetime-local',
    },
    timeEnd: {
      type: 'input',
      inputType: 'datetime-local',
    },
    ipFilter: {
      type: 'input',
      label: t('common.ip'),
      inputType: 'text',
      placeholder: t('overview.ipPlaceholder'),
    },
  },
  session: {
    ipFilter: {
      type: 'input',
      label: t('common.ip'),
      inputType: 'text',
      placeholder: t('overview.ipPlaceholder'),
    },
    deviceFilter: {
      type: 'select',
      label: t('overview.deviceType'),
      options: [{ value: 'all', label: t('common.all') }],
    },
    browserFilter: {
      type: 'select',
      label: t('common.browser'),
      options: [{ value: 'all', label: t('common.all') }],
    },
    osFilter: {
      type: 'select',
      label: t('common.os'),
      options: [{ value: 'all', label: t('common.all') }],
    },
  },
}));

const detailFilterLayout = computed(() => DETAIL_FILTER_LAYOUTS.value[detailLogScope.value] || []);
const detailFilterState = reactive({
  status: { ...DETAIL_FILTER_DEFAULTS.status },
  pv: { ...DETAIL_FILTER_DEFAULTS.pv },
  uv: { ...DETAIL_FILTER_DEFAULTS.uv },
  session: { ...DETAIL_FILTER_DEFAULTS.session },
});

const sessionFilterOptions = reactive({
  deviceFilter: [] as Array<{ value: string; label: string }>,
  browserFilter: [] as Array<{ value: string; label: string }>,
  osFilter: [] as Array<{ value: string; label: string }>,
});

function resetSessionFilterOptions() {
  sessionFilterOptions.deviceFilter = [{ value: 'all', label: t('common.all') }];
  sessionFilterOptions.browserFilter = [{ value: 'all', label: t('common.all') }];
  sessionFilterOptions.osFilter = [{ value: 'all', label: t('common.all') }];
}

resetSessionFilterOptions();

const DETAIL_FILTER_LAYOUTS = computed(() => ({
  status: [
    {
      className: 'detail-filter-section',
      items: ['statusClass', 'statusCode', 'excludeInternal', 'ipFilter', { type: 'apply', label: t('common.apply') }],
    },
  ],
  pv: [
    {
      className: 'detail-filter-section detail-filter-pv',
      items: [
        { type: 'range', label: t('overview.visitTime'), startKey: 'timeStart', endKey: 'timeEnd' },
        'locationFilter',
        'urlFilter',
        'excludeInternal',
        'ipFilter',
        { type: 'apply', label: t('common.apply') },
      ],
    },
  ],
  uv: [
    {
      className: 'detail-filter-section detail-filter-uv',
      items: [
        'isNew',
        { type: 'range', label: t('overview.visitTime'), startKey: 'timeStart', endKey: 'timeEnd' },
        'ipFilter',
        { type: 'apply', label: t('common.apply') },
      ],
    },
  ],
  session: [
    {
      className: 'detail-filter-section',
      items: ['ipFilter', 'deviceFilter', 'browserFilter', 'osFilter', { type: 'apply', label: t('common.apply') }],
    },
  ],
}));

onMounted(() => {
  loadWebsites();
  loadServerStatus();
  initGeoMap();
  restartAutoRefresh();
});

onBeforeUnmount(() => {
  stopAutoRefresh();
  stopServerStatusRefresh();
  if (visitsChart) {
    visitsChart.destroy();
    visitsChart = null;
  }
  if (newOldChart) {
    newOldChart.destroy();
    newOldChart = null;
  }
  if (deviceChart) {
    deviceChart.destroy();
    deviceChart = null;
  }
  if (geoMapChart) {
    geoMapChart.dispose();
    geoMapChart = null;
  }
});

watch(currentWebsiteId, (value) => {
  if (value) {
    saveUserPreference('selectedWebsite', value);
  }
  closeDetail();
  refreshOverview();
  restartAutoRefresh();
});

watch(dateRange, (range) => {
  if (dailyViewDisabled.value) {
    chartView.value = 'hourly';
  }
  closeDetail();
  refreshOverview();
  restartAutoRefresh();
});

watch(autoRefreshEnabled, (value) => {
  saveUserPreference('overviewAutoRefresh', value ? 'true' : 'false');
  restartAutoRefresh();
});

watch([currentWebsiteId, dateRange, chartView], () => {
  if (!currentWebsiteId.value) {
    return;
  }
  loadTimeSeries();
});

watch([currentWebsiteId, dateRange, mapView], () => {
  if (!currentWebsiteId.value) {
    return;
  }
  loadGeoMap();
});

watch(newOldStats, (stats) => {
  renderNewOldChart(stats);
});

watch(deviceTotals, (totals) => {
  renderDeviceChart(totals);
});

watch(isDark, () => {
  if (!geoMapChart || geoData.value.length === 0) {
    return;
  }
  renderGeoMap(geoData.value);
});

watch(locale, () => {
  resetSessionFilterOptions();
  if (geoData.value.length) {
    renderGeoMap(geoData.value);
  }
});

function refreshAll() {
  if (!currentWebsiteId.value) {
    return;
  }
  refreshOverview();
  loadTimeSeries();
  loadGeoMap();
}

function setRange(range: string) {
  dateRange.value = range;
}


function setChartView(view: 'hourly' | 'daily') {
  if (view === 'daily' && dailyViewDisabled.value) {
    return;
  }
  chartView.value = view;
}

function setMapView(view: 'china' | 'world') {
  mapView.value = view;
}

async function loadWebsites() {
  websitesLoading.value = true;
  try {
    const data = await fetchWebsites();
    websites.value = data || [];
    const saved = getUserPreference('selectedWebsite', '');
    if (saved && websites.value.find((site) => site.id === saved)) {
      currentWebsiteId.value = saved;
    } else if (websites.value.length > 0) {
      currentWebsiteId.value = websites.value[0].id;
    } else {
      currentWebsiteId.value = '';
    }
  } catch (error) {
    console.error('初始化网站失败:', error);
    websites.value = [];
    currentWebsiteId.value = '';
  } finally {
    websitesLoading.value = false;
  }
}

async function refreshOverview(silent = false) {
  if (!currentWebsiteId.value) {
    return;
  }
  const requestId = ++overviewRequestId;
  if (!silent) {
    overviewLoading.value = true;
  }
  try {
    const range = dateRange.value;
    const [overallData, urlData, refererData, browserData, osData, deviceData] = await Promise.all([
      fetchOverallStats(currentWebsiteId.value, range),
      fetchUrlStats(currentWebsiteId.value, range, 10),
      fetchRefererStats(currentWebsiteId.value, range, 10),
      fetchBrowserStats(currentWebsiteId.value, range, 10),
      fetchOSStats(currentWebsiteId.value, range, 10),
      fetchDeviceStats(currentWebsiteId.value, range, 10),
    ]);

    if (requestId !== overviewRequestId) {
      return;
    }

    overall.value = overallData;
    urlStats.value = urlData;
    refererStats.value = refererData;
    browserStats.value = browserData;
    osStats.value = osData;
    deviceStats.value = deviceData;

    latestOverall = overallData;
    latestOverallKey = buildOverallKey(currentWebsiteId.value, range);

    setLiveVisitorCount?.(overallData?.activeVisitorCount ?? null);
  } catch (error) {
    console.error('加载概况数据失败:', error);
  } finally {
    if (requestId === overviewRequestId && !silent) {
      overviewLoading.value = false;
    }
  }
}

async function loadServerStatus() {
  if (serverStatusLoading.value) {
    return;
  }
  serverStatusLoading.value = true;
  try {
    const data = await fetchServerStatus();
    serverStatus.value = data;
    initialServerStatusEnabled.value = data.enabled;
    serverStatusError.value = '';
    persistServerStatus(data);
    scheduleServerStatusRefresh(data);
  } catch (error) {
    console.error('加载服务器状态失败:', error);
    serverStatusError.value = serverStatus.value ? '' : t('overview.serverStatusLoadError');
    scheduleServerStatusRefresh(serverStatus.value);
  } finally {
    serverStatusLoading.value = false;
  }
}

function readCachedServerStatus(initialEnabled: boolean | null): ServerStatusResponse | null {
  if (initialEnabled === false || typeof window === 'undefined') {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(SERVER_STATUS_CACHE_KEY);
    if (!raw) {
      return null;
    }
    const data = JSON.parse(raw) as ServerStatusResponse;
    return data?.enabled ? data : null;
  } catch {
    window.localStorage.removeItem(SERVER_STATUS_CACHE_KEY);
    return null;
  }
}

function persistServerStatus(data: ServerStatusResponse) {
  if (typeof window === 'undefined') {
    return;
  }
  if (!data.enabled) {
    window.localStorage.removeItem(SERVER_STATUS_CACHE_KEY);
    return;
  }
  window.localStorage.setItem(SERVER_STATUS_CACHE_KEY, JSON.stringify(data));
}

function scheduleServerStatusRefresh(data: ServerStatusResponse | null) {
  stopServerStatusRefresh();
  if (!data?.enabled) {
    return;
  }
  const seconds = Math.max(5, Number(data.refresh_interval_seconds || 30));
  serverStatusTimer = window.setInterval(() => {
    if (document.visibilityState !== 'visible' || serverStatusLoading.value) {
      return;
    }
    loadServerStatus();
  }, seconds * 1000);
}

function stopServerStatusRefresh() {
  if (!serverStatusTimer) {
    return;
  }
  window.clearInterval(serverStatusTimer);
  serverStatusTimer = null;
}

function startAutoRefresh() {
  if (autoRefreshTimer) {
    return;
  }
  if (!autoRefreshEnabled.value || !autoRefreshAllowed.value || !currentWebsiteId.value) {
    return;
  }
  autoRefreshTimer = window.setInterval(() => {
    if (!autoRefreshEnabled.value || !autoRefreshAllowed.value || !currentWebsiteId.value) {
      stopAutoRefresh();
      return;
    }
    if (document.visibilityState !== 'visible' || overviewLoading.value) {
      return;
    }
    refreshOverview(true);
  }, AUTO_REFRESH_INTERVAL);
}

function stopAutoRefresh() {
  if (!autoRefreshTimer) {
    return;
  }
  window.clearInterval(autoRefreshTimer);
  autoRefreshTimer = null;
}

function restartAutoRefresh() {
  stopAutoRefresh();
  startAutoRefresh();
}

async function loadTimeSeries() {
  if (!currentWebsiteId.value || !visitsChartRef.value) {
    return;
  }

  const requestId = ++chartRequestId;
  chartError.value = '';
  timeSeriesData.value = null;

  try {
    const data = await fetchTimeSeriesStats(currentWebsiteId.value, dateRange.value, chartView.value);
    if (requestId !== chartRequestId) {
      return;
    }
    timeSeriesData.value = data;
    renderVisitsChart(data);
  } catch (error: any) {
    if (error?.name === 'AbortError') {
      return;
    }
    timeSeriesData.value = null;
    chartError.value = t('overview.trendError');
  }
}

async function loadGeoMap() {
  if (!currentWebsiteId.value) {
    return;
  }

  const requestId = ++mapRequestId;
  const range = dateRange.value;

  try {
    geoPending.value = false;
    const statsData = await fetchLocationStats(
      currentWebsiteId.value,
      range,
      mapView.value === 'china' ? 'domestic' : 'global',
      99
    );

    if (requestId !== mapRequestId) {
      return;
    }

    geoPending.value = Boolean(statsData?.key?.some((name: string) => isPendingGeoName(name)));

    const rows = (statsData?.key || []).map((location: string, index: number) => ({
      name: location,
      value: statsData.uv[index],
      percentage: statsData.uv_percent[index],
    }));

    const normalizedRows =
      mapView.value === 'china'
        ? rows.map((row) => ({ ...row, name: normalizeChinaRegionName(row.name) }))
        : rows.map((row) => ({ ...row, name: normalizeWorldRegionName(row.name) }));

    const nextGeoData = normalizedRows.filter((row) => !isExcludedGeoName(row.name));
    geoData.value = nextGeoData;
    if (nextGeoData.length > 0) {
      await nextTick();
      await initGeoMap();
    }
    renderGeoMap(geoData.value);
  } catch (error: any) {
    if (error?.name === 'AbortError') {
      return;
    }
    console.error('加载地域数据失败:', error);
  }
}

async function ensureGeoMapRuntime() {
  if (echartsCore) {
    return echartsCore;
  }
  if (!geoMapRuntimePromise) {
    geoMapRuntimePromise = Promise.all([
      import('echarts/core'),
      import('echarts/charts'),
      import('echarts/components'),
      import('echarts/renderers'),
      import('@/assets/maps/china.json'),
      import('@/assets/maps/world.json'),
    ])
      .then(([core, charts, components, renderers, chinaMap, worldMap]) => {
        core.use([
          charts.MapChart,
          components.GeoComponent,
          components.TooltipComponent,
          components.VisualMapComponent,
          renderers.CanvasRenderer,
        ]);
        core.registerMap('china', chinaMap.default as any);
        core.registerMap('world', worldMap.default as any);
        echartsCore = core;
        return core;
      })
      .catch((error) => {
        geoMapRuntimePromise = null;
        throw error;
      });
  }
  return geoMapRuntimePromise;
}

async function initGeoMap() {
  if (!geoMapRef.value || geoMapChart) {
    return;
  }
  try {
    const echarts = await ensureGeoMapRuntime();
    if (!geoMapRef.value || geoMapChart) {
      return;
    }
    geoMapChart = echarts.init(geoMapRef.value);
  } catch (error) {
    console.error('初始化地域地图失败:', error);
    return;
  }
  renderGeoMap(geoData.value);
  if (currentWebsiteId.value && geoData.value.length === 0) {
    loadGeoMap();
  }
}

function renderGeoMap(data: Array<{ name: string; value: number; percentage: number }>) {
  if (!geoMapChart) {
    return;
  }
  if (!data || data.length === 0) {
    geoMapChart.clear();
    return;
  }

  const maxValue = data[0]?.value || 10;
  const inRange = isDark.value
    ? { color: ['#2a5769', '#7eb9ff'] }
    : { color: ['#e0ffff', '#006edd'] };

  if (mapView.value === 'china') {
    geoMapChart.setOption(
      {
        tooltip: {
          trigger: 'item',
          formatter: (params: any) => {
            const value = Number.isFinite(params.value) ? params.value : 0;
            return `${params.name}<br/>${t('overview.geoVisits')}: ${formatCount(value)}`;
          },
        },
        visualMap: {
          backgroundColor: 'transparent',
          min: -5,
          max: maxValue,
          left: 'left',
          bottom: '10%',
          calculable: false,
          inRange,
        },
        geo: {
          map: 'china',
          nameMap: chinaNameMap.value,
          roam: true,
          label: {
            show: false,
          },
          regions: [
            {
              name: '南海诸岛',
              selected: false,
              itemStyle: {
                areaColor: 'transparent',
                opacity: 0,
              },
            },
          ],
        },
        series: [
          {
            name: t('overview.geoVisits'),
            type: 'map',
            map: 'china',
            geoIndex: 0,
            nameMap: chinaNameMap.value,
            data,
          },
        ],
      },
      true
    );
    return;
  }

  geoMapChart.setOption(
    {
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          const value = Number.isFinite(params.value) ? params.value : 0;
          return `${params.name}<br/>${t('overview.geoVisits')}: ${formatCount(value)}`;
        },
      },
      visualMap: {
        backgroundColor: 'transparent',
        min: -5,
        max: maxValue,
        left: 'left',
        bottom: '10%',
        calculable: false,
        inRange,
      },
      series: [
        {
          name: t('overview.geoVisits'),
          type: 'map',
          map: 'world',
          nameMap: worldNameMap.value,
          roam: true,
          label: {
            show: false,
          },
          data,
        },
      ],
    },
    true
  );
}

type MetricSnapshot = { pv?: number; uv?: number; sessionCount?: number };

function buildMetricTile(metric: 'pv' | 'uv' | 'session', current: MetricSnapshot, compare: any) {
  const prev = compare.previous || {};
  const forecast = compare.forecast || {};
  const sameTime = compare.sameTime || {};

  const currentValue = getMetricValue(metric, current);
  const prevValue = getMetricValue(metric, prev);
  const delta = buildDeltaText(metric, current, prev);

  return {
    current: formatCount(currentValue),
    prev: formatCount(prevValue),
    forecast: formatCount(getMetricValue(metric, forecast)),
    sameTime: formatCount(getMetricValue(metric, sameTime)),
    deltaText: delta.text,
    deltaClass: delta.className,
  }
}

function getMetricValue(metric: 'pv' | 'uv' | 'session', source: MetricSnapshot) {
  if (!source) {
    return NaN;
  }
  switch (metric) {
    case 'pv':
      return Number(source.pv);
    case 'uv':
      return Number(source.uv);
    case 'session':
      return Number(source.sessionCount);
    default:
      return NaN;
  }
}

function renderVisitsChart(stats: TimeSeriesStats) {
  if (visitsChart) {
    visitsChart.destroy();
    visitsChart = null;
  }
  if (!visitsChartRef.value || !hasTimeSeriesData(stats)) {
    return;
  }

  const ctx = visitsChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }

  const gradientUv = ctx.createLinearGradient(0, 0, 0, visitsChartRef.value.height || 300);
  gradientUv.addColorStop(0, 'rgba(30, 123, 255, 0.35)');
  gradientUv.addColorStop(1, 'rgba(30, 123, 255, 0.02)');

  const gradientPv = ctx.createLinearGradient(0, 0, 0, visitsChartRef.value.height || 300);
  gradientPv.addColorStop(0, 'rgba(255, 138, 61, 0.35)');
  gradientPv.addColorStop(1, 'rgba(255, 138, 61, 0.02)');

  const chartConfig = {
    type: 'line' as const,
    data: {
      labels: stats.labels,
      datasets: [
        {
          label: t('overview.uv'),
          data: stats.visitors,
          borderColor: '#1e7bff',
          backgroundColor: gradientUv,
          borderWidth: 2,
          tension: 0.4,
          pointRadius: 0,
          fill: true,
        },
        {
          label: t('overview.pv'),
          data: stats.pageviews,
          borderColor: '#ff8a3d',
          backgroundColor: gradientPv,
          borderWidth: 2,
          tension: 0.4,
          pointRadius: 0,
          fill: true,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index' as const,
        intersect: false,
      },
      scales: {
        y: {
          beginAtZero: true,
          grid: {
            color: 'rgba(148, 163, 184, 0.25)',
          },
        },
        x: {
          grid: {
            display: false,
          },
          ticks: {
            callback: function (val: any, index: number) {
              const currentLabel = (this as any).getLabelForValue(val);
              const labels = (this as any).chart.data.labels as string[];
              const firstIndex = labels.indexOf(currentLabel);
              return firstIndex === index ? currentLabel : '';
            },
          },
        },
      },
      plugins: {
        tooltip: {
          callbacks: {
            label: function (context: any) {
              const index = context.dataIndex;
              const fullLabel = stats.labels[index];
              if (context.datasetIndex === 0) {
                return `${fullLabel} - ${t('overview.uv')}: ${stats.visitors[index]}`;
              }
              return `${fullLabel} - ${t('overview.pv')}: ${stats.pageviews[index]}`;
            },
          },
        },
        legend: {
          position: 'top' as const,
          align: 'end' as const,
          labels: {
            usePointStyle: true,
            boxWidth: 10,
          },
        },
      },
    },
  }

  visitsChart = new Chart(ctx, chartConfig as any);
}

function renderNewOldChart(stats: typeof newOldStats.value) {
  if (newOldChart) {
    newOldChart.destroy();
    newOldChart = null;
  }
  if (!newOldChartRef.value || stats.newCount + stats.oldCount <= 0) {
    return;
  }

  const ctx = newOldChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }

  const currentNew = stats.newCount;
  const currentOld = stats.oldCount;
  const previousNew = stats.prevNew;
  const previousOld = stats.prevOld;

  newOldChart = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: stats.labels,
      datasets: [
        {
          label: t('overview.newVisitor'),
          data: [currentNew, previousNew],
          backgroundColor: 'rgba(30, 123, 255, 0.7)',
          borderRadius: 10,
          barThickness: 28,
        },
        {
          label: t('overview.oldVisitor'),
          data: [currentOld, previousOld],
          backgroundColor: 'rgba(255, 138, 61, 0.7)',
          borderRadius: 10,
          barThickness: 28,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: 'top' as const,
          align: 'end' as const,
          labels: {
            usePointStyle: true,
            boxWidth: 8,
          },
        },
      },
      scales: {
        x: {
          grid: {
            display: false,
          },
        },
        y: {
          beginAtZero: true,
          grid: {
            color: 'rgba(148, 163, 184, 0.25)',
          },
        },
      },
    },
  });
}

function renderDeviceChart(totals: typeof deviceTotals.value) {
  if (deviceChart) {
    deviceChart.destroy();
    deviceChart = null;
  }
  if (!deviceChartRef.value || totals.desktop + totals.mobile + totals.apple + totals.other <= 0) {
    return;
  }

  const ctx = deviceChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }

  const data = [totals.desktop, totals.mobile, totals.apple, totals.other];

  deviceChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: [t('overview.deviceDesktop'), t('overview.deviceMobile'), t('overview.deviceApple'), t('overview.deviceOther')],
      datasets: [
        {
          data,
          backgroundColor: ['#1e7bff', '#ff8a3d', '#64748b', '#2ec27e'],
          borderWidth: 0,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '68%',
      plugins: {
        legend: {
          position: 'bottom' as const,
          labels: {
            usePointStyle: true,
            boxWidth: 8,
          },
        },
      },
    },
  });
}

function hasTimeSeriesData(stats: TimeSeriesStats | null) {
  if (!stats || !Array.isArray(stats.labels) || stats.labels.length === 0) {
    return false;
  }
  return [stats.visitors || [], stats.pageviews || []].some((series) =>
    Array.isArray(series) && series.some((value) => Number(value || 0) > 0)
  );
}

async function openDetail(type: string) {
  const config = buildDetailConfig(type);
  if (!config) {
    return;
  }
  detailConfig.value = config;
  detailOpen.value = true;
  detailError.value = false;
  detailIpParsing.value = false;
  detailIpParsingProgress.value = null;
  detailIpParsingEstimatedRemainingSeconds.value = null;
  detailIpGeoParsing.value = false;
  detailIpGeoPending.value = false;
  detailIpGeoProgress.value = null;
  detailIpGeoEstimatedRemainingSeconds.value = null;
  detailParsingPending.value = false;
  detailLoadState.value = 'ready';
  detailRequestId += 1;

  if (config.mode === 'logs') {
    detailLogScope.value = config.logScope || 'status';
    resetDetailFilters(detailLogScope.value);
    if (detailLogScope.value === 'session') {
      await loadSessionFilterOptions(detailRequestId);
    }
    detailPage.value = 1;
    detailHasMore.value = true;
    detailLogRows.value = [];
    await loadDetailLogs(true, detailRequestId);
    return;
  }

  detailLogScope.value = 'status';
  detailRankingRows.value = [];
  await loadDetailTable(config, detailRequestId);
}

function closeDetail() {
  detailOpen.value = false;
  detailConfig.value = null;
  detailError.value = false;
  detailLoading.value = false;
  detailLoadState.value = 'ready';
  detailHasMore.value = false;
  detailPage.value = 1;
  detailIpParsing.value = false;
  detailIpParsingProgress.value = null;
  detailIpParsingEstimatedRemainingSeconds.value = null;
  detailIpGeoParsing.value = false;
  detailIpGeoPending.value = false;
  detailIpGeoProgress.value = null;
  detailIpGeoEstimatedRemainingSeconds.value = null;
  detailParsingPending.value = false;
  detailRankingRows.value = [];
  detailLogRows.value = [];
  resetDetailFilters('status');
  resetDetailFilters('pv');
  resetDetailFilters('uv');
  resetDetailFilters('session');
}

function applyDetailFilters() {
  if (detailMode.value !== 'logs') {
    return;
  }
  detailPage.value = 1;
  detailHasMore.value = true;
  detailLogRows.value = [];
  loadDetailLogs(true, detailRequestId + 1);
}

function loadMoreDetail() {
  if (detailMode.value !== 'logs') {
    return;
  }
  if (detailLoadState.value === 'loading') {
    return;
  }
  if (detailLoadState.value === 'done') {
    return;
  }
  if (detailLoadState.value === 'error') {
    loadDetailLogs(false, detailRequestId + 1);
    return;
  }
  detailPage.value += 1;
  loadDetailLogs(false, detailRequestId + 1);
}

async function loadDetailTable(config: DetailConfig, requestId: number) {
  if (!config.fetch) {
    return;
  }
  detailLoading.value = true;
  detailError.value = false;
  try {
    const data = await config.fetch();
    if (requestId !== detailRequestId) {
      return;
    }
    detailRankingRows.value = buildRankingRows(data, config.showPv, config.formatLabel);
  } catch (error) {
    if (requestId !== detailRequestId) {
      return;
    }
    detailError.value = true;
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false;
    }
  }
}

async function loadDetailLogs(reset: boolean, requestId: number) {
  if (detailLoading.value) {
    return;
  }
  detailLoading.value = true;
  detailLoadState.value = 'loading';
  detailError.value = false;
  detailRequestId = requestId;

  const scope = detailLogScope.value;
  const range = dateRange.value;
  const state = detailFilterState[scope] || {};
  let statusCode: number | null = null;
  let statusClass = '';
  let excludeInternal = false;
  let ipFilter = '';
  let timeStart = '';
  let timeEnd = '';
  let locationFilter = '';
  let urlFilter = '';
  let pageviewOnly = false;
  let newVisitor = '';
  let distinctIp = false;

  if (scope === 'status') {
    const rawStatusCode = state.statusCode;
    if (typeof rawStatusCode === 'number' && Number.isFinite(rawStatusCode)) {
      statusCode = rawStatusCode;
    } else if (typeof rawStatusCode === 'string') {
      const parsed = parseInt(rawStatusCode.trim(), 10);
      if (Number.isFinite(parsed)) {
        statusCode = parsed;
      }
    }
    statusClass = statusCode !== null ? '' : (state.statusClass || 'all');
    excludeInternal = Boolean(state.excludeInternal);
    ipFilter = state.ipFilter || '';
  }

  if (scope === 'pv') {
    timeStart = state.timeStart || '';
    timeEnd = state.timeEnd || '';
    locationFilter = state.locationFilter || '';
    urlFilter = state.urlFilter || '';
    excludeInternal = Boolean(state.excludeInternal);
    ipFilter = state.ipFilter || '';
    pageviewOnly = true;
  }

  if (scope === 'uv') {
    timeStart = state.timeStart || '';
    timeEnd = state.timeEnd || '';
    ipFilter = state.ipFilter || '';
    newVisitor = state.isNew || 'all';
    pageviewOnly = true;
    distinctIp = true;
  }

  if (scope === 'session') {
    ipFilter = state.ipFilter || '';
  }

  try {
    if (scope === 'session') {
      const result = await fetchSessions(
        currentWebsiteId.value,
        detailPage.value,
        DETAIL_LOG_PAGE_SIZE,
        range,
        timeStart,
        timeEnd,
        ipFilter,
        normalizeSelectFilterValue(state.deviceFilter),
        normalizeSelectFilterValue(state.browserFilter),
        normalizeSelectFilterValue(state.osFilter)
      );

      if (requestId !== detailRequestId) {
        return;
      }

      const sessions = result.sessions || [];
      updateLogRows(sessions, reset, scope);
      const pages = result.pagination?.pages || 1;
      detailHasMore.value = detailPage.value < pages;
      detailLoadState.value = detailHasMore.value ? 'ready' : 'done';
      detailIpParsing.value = false;
      detailIpParsingProgress.value = null;
      detailIpParsingEstimatedRemainingSeconds.value = null;
      detailIpGeoParsing.value = false;
      detailIpGeoPending.value = false;
      detailIpGeoProgress.value = null;
      detailIpGeoEstimatedRemainingSeconds.value = null;
      return;
    }

    const result = await fetchLogs(
      currentWebsiteId.value,
      detailPage.value,
      DETAIL_LOG_PAGE_SIZE,
      'timestamp',
      'desc',
      '',
      range,
      statusClass === 'all' ? '' : statusClass,
      statusCode,
      excludeInternal,
      ipFilter,
      timeStart,
      timeEnd,
      locationFilter,
      urlFilter,
      pageviewOnly,
      newVisitor,
      distinctIp
    );

    if (requestId !== detailRequestId) {
      return;
    }

    detailIpParsing.value = Boolean(result.ip_parsing);
    detailIpParsingProgress.value = detailIpParsing.value ? normalizeProgress(result.ip_parsing_progress) : null;
    detailIpParsingEstimatedRemainingSeconds.value = detailIpParsing.value
      ? normalizeSeconds(result.ip_parsing_estimated_remaining_seconds)
      : null;
    detailIpGeoParsing.value = Boolean(result.ip_geo_parsing);
    detailIpGeoPending.value = Boolean(result.ip_geo_pending);
    detailIpGeoProgress.value = detailIpGeoParsing.value || detailIpGeoPending.value
      ? normalizeProgress(result.ip_geo_progress)
      : null;
    detailIpGeoEstimatedRemainingSeconds.value = detailIpGeoParsing.value || detailIpGeoPending.value
      ? normalizeSeconds(result.ip_geo_estimated_remaining_seconds)
      : null;
    detailParsingPending.value = Boolean(result.parsing_pending);
    detailParsingPendingProgress.value = detailParsingPending.value
      ? normalizeProgress(result.parsing_pending_progress)
      : null;
    const logs = result.logs || [];
    updateLogRows(logs, reset, scope);
    const exact = result.pagination?.exact !== false;
    const pages = result.pagination?.pages || 1;
    detailHasMore.value = exact ? detailPage.value < pages : Boolean(result.pagination?.hasMore);
    detailLoadState.value = detailHasMore.value ? 'ready' : 'done';
  } catch (error) {
    console.error('加载日志详情失败:', error);
    if (requestId !== detailRequestId) {
      return;
    }
    detailError.value = detailLogRows.value.length === 0;
    detailLoadState.value = 'error';
    detailIpParsing.value = false;
    detailIpParsingProgress.value = null;
    detailIpParsingEstimatedRemainingSeconds.value = null;
    detailIpGeoParsing.value = false;
    detailIpGeoPending.value = false;
    detailIpGeoProgress.value = null;
    detailIpGeoEstimatedRemainingSeconds.value = null;
    detailParsingPending.value = false;
    detailParsingPendingProgress.value = null;
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false;
    }
  }
}

function updateLogRows(logs: Array<Record<string, any>>, reset: boolean, scope: string) {
  const rows = logs.map((log) => buildLogRow(log, scope));
  if (reset) {
    detailLogRows.value = rows;
    return;
  }
  detailLogRows.value = detailLogRows.value.concat(rows);
}

function buildLogRow(log: Record<string, any>, scope: string) {
  const time = log.time || log.start_time || t('common.none');
  const url = log.url || t('common.none');
  const ip = log.ip || t('common.none');
  const deviceRaw = log.user_device || t('common.none');
  const statusCode = log.status_code !== undefined && log.status_code !== null ? log.status_code : t('common.none');
  const locationRaw = log.domestic_location || log.global_location || t('common.none');
  const isNew = log.is_new_visitor;
  const newLabel = isNew === true ? t('overview.newVisitor') : isNew === false ? t('overview.oldVisitor') : t('common.none');
  const durationSeconds = Number.isFinite(log.duration_seconds) ? log.duration_seconds : 0;
  const durationLabel = formatDurationSeconds(durationSeconds);
  const pageCount = log.page_count ?? t('common.none');
  const entryUrl = log.entry_url || t('common.none');
  const exitUrl = log.exit_url || t('common.none');
  const browserRaw = log.user_browser || t('common.none');
  const osRaw = log.user_os || t('common.none');
  const device = formatDeviceLabel(deviceRaw, t);
  const location = formatLocationLabel(locationRaw, currentLocale.value, t);
  const browser = formatBrowserLabel(browserRaw, t);
  const os = formatOSLabel(osRaw, t);

  if (scope === 'pv') {
    return {
      cells: [
        { value: ip },
        { value: url, className: 'item-path', title: url },
        { value: location },
        { value: time },
        { value: device },
      ],
    }
  }

  if (scope === 'uv') {
    return {
      cells: [
        { value: ip },
        { value: location },
        { value: device },
        { value: newLabel },
        { value: time },
      ],
    }
  }

  if (scope === 'session') {
    return {
      cells: [
        { value: ip },
        { value: location },
        { value: device },
        { value: browser },
        { value: os },
        { value: time },
        { value: durationLabel },
        { value: pageCount },
        { value: entryUrl, className: 'item-path', title: entryUrl },
        { value: exitUrl, className: 'item-path', title: exitUrl },
      ],
    }
  }

  return {
    cells: [
      { value: statusCode },
      { value: time },
      { value: url, className: 'item-path', title: url },
      { value: ip },
      { value: device },
    ],
  }
}

async function loadSessionFilterOptions(requestId: number) {
  if (!currentWebsiteId.value) {
    return;
  }
  try {
    const range = dateRange.value;
    const [deviceData, browserData, osData] = await Promise.all([
      fetchDeviceStats(currentWebsiteId.value, range, 20),
      fetchBrowserStats(currentWebsiteId.value, range, 20),
      fetchOSStats(currentWebsiteId.value, range, 20),
    ]);

    if (requestId !== detailRequestId) {
      return;
    }

    sessionFilterOptions.deviceFilter = buildOptionList(deviceData?.key, (label) => formatDeviceLabel(label, t));
    sessionFilterOptions.browserFilter = buildOptionList(browserData?.key, (label) => formatBrowserLabel(label, t));
    sessionFilterOptions.osFilter = buildOptionList(osData?.key, (label) => formatOSLabel(label, t));
  } catch (error) {
    console.error('加载会话筛选项失败:', error);
  }
}

function resetDetailFilters(scope: 'status' | 'pv' | 'uv' | 'session') {
  Object.assign(detailFilterState[scope], DETAIL_FILTER_DEFAULTS[scope]);
}

function getDetailFieldOptions(scope: 'status' | 'pv' | 'uv' | 'session', key: string) {
  if (scope === 'session') {
    if (key === 'deviceFilter') {
      return sessionFilterOptions.deviceFilter;
    }
    if (key === 'browserFilter') {
      return sessionFilterOptions.browserFilter;
    }
    if (key === 'osFilter') {
      return sessionFilterOptions.osFilter;
    }
  }
  return detailFilterFields.value[scope][key]?.options || [];
}

function buildDetailConfig(detailType: string): DetailConfig | null {
  const range = dateRange.value || 'today';
  switch (detailType) {
    case 'geo': {
      const geoInfo = getGeoDetailInfo();
      return {
        title: t('overview.geoDetailTitle', { label: geoInfo.label }),
        keyLabel: geoInfo.keyLabel,
        valueLabel: t('common.visitors'),
        showPv: false,
        formatLabel: (label) => formatLocationLabel(label, currentLocale.value, t),
        fetch: () => fetchLocationStats(currentWebsiteId.value, range, geoInfo.type, DETAIL_LIMIT),
      }
    }
    case 'referer':
      return {
        title: t('overview.refererDetailTitle'),
        keyLabel: t('overview.refererSite'),
        valueLabel: t('common.visitors'),
        showPv: false,
        formatLabel: (label) => formatRefererLabel(label, currentLocale.value, t),
        fetch: () => fetchRefererStats(currentWebsiteId.value, range, DETAIL_LIMIT),
      }
    case 'url':
      return {
        title: t('overview.pageDetailTitle'),
        keyLabel: t('common.url'),
        valueLabel: t('common.viewCount'),
        showPv: true,
        fetch: () => fetchUrlStats(currentWebsiteId.value, range, DETAIL_LIMIT),
      }
    case 'entry':
      return {
        title: t('overview.entryDetailTitle'),
        keyLabel: t('common.url'),
        valueLabel: t('common.entryCount'),
        showPv: false,
        fetch: async () => {
          const data = await getOverallForDetail(range);
          return data.entryPages;
        },
      }
    case 'device':
      return {
        title: t('overview.deviceDetailTitle'),
        keyLabel: t('overview.deviceType'),
        valueLabel: t('common.visitors'),
        showPv: false,
        formatLabel: (label) => formatDeviceLabel(label, t),
        fetch: () => fetchDeviceStats(currentWebsiteId.value, range, DETAIL_LIMIT),
      }
    case 'metric-status':
      return {
        title: t('overview.statusDetailTitle'),
        mode: 'logs',
        layout: 'modal',
        logScope: 'status',
        columns: [
          { label: t('overview.statusCode'), className: 'detail-status-col' },
          { label: t('overview.hitTime'), className: 'detail-time-col' },
          { label: t('overview.visitLink'), className: 'detail-url-col' },
          { label: t('overview.visitorIp'), className: 'detail-ip-col' },
          { label: t('overview.deviceType'), className: 'detail-device-col' },
        ],
      }
    case 'metric-pv':
      return {
        title: t('overview.pvDetailTitle'),
        mode: 'logs',
        layout: 'modal',
        logScope: 'pv',
        columns: [
          { label: t('overview.visitorIp'), className: 'detail-ip-col' },
          { label: t('overview.visitLink'), className: 'detail-url-col' },
          { label: t('overview.ipLocation'), className: 'detail-location-col' },
          { label: t('overview.visitTime'), className: 'detail-time-col' },
          { label: t('overview.deviceType'), className: 'detail-device-col' },
        ],
      }
    case 'metric-uv':
      return {
        title: t('overview.uvDetailTitle'),
        mode: 'logs',
        layout: 'modal',
        logScope: 'uv',
        columns: [
          { label: t('overview.visitorIp'), className: 'detail-ip-col' },
          { label: t('overview.ipLocation'), className: 'detail-location-col' },
          { label: t('overview.deviceType'), className: 'detail-device-col' },
          { label: t('overview.isNewVisitor'), className: 'detail-new-col' },
          { label: t('overview.visitTime'), className: 'detail-time-col' },
        ],
      }
    case 'metric-session':
      return {
        title: t('overview.sessionDetailTitle'),
        mode: 'logs',
        layout: 'modal',
        logScope: 'session',
        columns: [
          { label: t('overview.visitorIp'), className: 'detail-ip-col' },
          { label: t('overview.ipLocation'), className: 'detail-location-col' },
          { label: t('overview.deviceType'), className: 'detail-device-col' },
          { label: t('common.browser'), className: 'detail-browser-col' },
          { label: t('common.os'), className: 'detail-os-col' },
          { label: t('overview.sessionStart'), className: 'detail-time-col' },
          { label: t('overview.sessionDuration'), className: 'detail-duration-col' },
          { label: t('overview.sessionPages'), className: 'detail-pages-col' },
          { label: t('overview.sessionEntry'), className: 'detail-entry-col' },
          { label: t('overview.sessionExit'), className: 'detail-exit-col' },
        ],
      }
    default:
      return null;
  }
}

function buildDetailColumns(config: DetailConfig | null) {
  if (!config) {
    return [];
  }
  if (config.columns && config.columns.length > 0) {
    return config.columns;
  }
  return [
    { label: config.keyLabel || t('overview.dimension'), className: 'detail-key-col' },
    { label: config.valueLabel || t('overview.quantity'), className: 'detail-value-col' },
  ];
}

function buildDetailSubtitle() {
  const rangeLabel = getRangeLabel(dateRange.value || 'today');
  const websiteName = websites.value.find((site) => site.id === currentWebsiteId.value)?.name || '';
  if (!websiteName) {
    return rangeLabel;
  }
  return `${websiteName} · ${rangeLabel}`;
}

async function getOverallForDetail(range: string) {
  const key = buildOverallKey(currentWebsiteId.value, range);
  if (latestOverall && latestOverallKey === key) {
    return latestOverall;
  }
  const data = await fetchOverallStats(currentWebsiteId.value, range);
  latestOverall = data;
  latestOverallKey = key;
  return data;
}

function buildOverallKey(websiteId: string, range: string) { return `${websiteId || ''}:${range || ''}`; }

function getGeoDetailInfo() {
  if (mapView.value === 'world') {
    return { type: 'global', label: t('overview.global'), keyLabel: t('overview.countryRegion') };
  }
  return { type: 'domestic', label: t('overview.domestic'), keyLabel: t('common.province') };
}

function buildRankingRows(
  data: SimpleSeriesStats | undefined | null,
  usePv = false,
  formatLabel?: (label: string) => string
) {
  const safeData = data || {};
  const labels = safeData.key || [];
  const values = usePv ? safeData.pv || [] : safeData.uv || [];
  const percents = usePv ? safeData.pv_percent || [] : safeData.uv_percent || [];

  if (!labels.length) {
    return [];
  }

  return labels.map((label: string, index: number) => ({
    label: formatLabel ? formatLabel(label) : label,
    value: values[index] || 0,
    percent: percents[index] || 0,
  }));
}

function buildGeoRows(rows: Array<{ name: string; value: number; percentage: number }>) {
  return (rows || []).slice(0, 10).map((row) => ({
    label: formatLocationLabel(row.name, currentLocale.value, t),
    value: row.value || 0,
    percent: row.percentage || 0,
  }));
}

function buildDeltaTextFromTotals(currentTotal: number, prevTotal: number | null) {
  if (!Number.isFinite(currentTotal) || !Number.isFinite(prevTotal) || (prevTotal ?? 0) <= 0) {
    return { text: t('common.none'), className: '' };
  }

  const delta = ((currentTotal - (prevTotal as number)) / (prevTotal as number)) * 100;
  if (!Number.isFinite(delta)) {
    return { text: t('common.none'), className: '' };
  }

  const absDelta = Math.abs(delta).toFixed(2);
  if (Math.abs(delta) < 0.01) {
    return { text: '0.00%', className: 'flat' };
  }

  const arrow = delta > 0 ? '↑' : '↓';
  const className = delta > 0 ? 'up' : 'down';
  return { text: `${arrow} ${absDelta}%`, className };
}

function buildDeltaText(metric: 'pv' | 'uv' | 'session', current: MetricSnapshot, previous: MetricSnapshot) {
  const currentValue = getMetricValue(metric, current);
  const prevValue = getMetricValue(metric, previous);

  if (!Number.isFinite(currentValue) || !Number.isFinite(prevValue) || prevValue <= 0) {
    return { text: t('common.none'), className: '' };
  }

  const delta = ((currentValue - prevValue) / prevValue) * 100;
  if (!Number.isFinite(delta)) {
    return { text: t('common.none'), className: '' };
  }

  const absDelta = Math.abs(delta).toFixed(2);
  if (Math.abs(delta) < 0.01) {
    return { text: '0.00%', className: 'flat' };
  }

  const arrow = delta > 0 ? '↑' : '↓';
  const className = delta > 0 ? 'up' : 'down';
  return { text: `${arrow} ${absDelta}%`, className };
}

function formatCount(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return t('common.none');
  }
  return n(Number(value));
}

function numberValue(value: unknown): number | null {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function metricNumber(metrics: Record<string, unknown>, key: string): number | null {
  return numberValue(metrics[key]);
}

function maxMetricNumber(...values: Array<number | null | undefined>) {
  const finiteValues = values.filter((value): value is number => Number.isFinite(value));
  if (finiteValues.length === 0) {
    return null;
  }
  return Math.max(...finiteValues);
}

function minDiskRemaining(disks: ServerDiskStatus[]) {
  const values = disks
    .map((disk) => numberValue(disk.percentage_remaining))
    .filter((value): value is number => value !== null);
  if (values.length === 0) {
    return null;
  }
  return Math.min(...values);
}

function maxDiskTemperature(disks: ServerDiskStatus[]) {
  const values = disks
    .map((disk) => numberValue(disk.temperature_celsius))
    .filter((value): value is number => value !== null);
  if (values.length === 0) {
    return null;
  }
  return Math.max(...values);
}

function diskRiskScore(disk: ServerDiskStatus) {
  const remaining = numberValue(disk.percentage_remaining);
  const temp = numberValue(disk.temperature_celsius);
  let score = 0;
  if (disk.health_passed === false) {
    score += 1000;
  }
  if (remaining !== null) {
    score += Math.max(0, 100 - remaining) * 5;
  }
  if (temp !== null) {
    score += Math.max(0, temp - 45) * 4;
  }
  score += Number(disk.media_errors || 0) * 20;
  score += Number(disk.error_log_entries || 0) * 3;
  score += Number(disk.unsafe_shutdowns || 0);
  return score;
}

function diskTone(disk: ServerDiskStatus) {
  const remaining = numberValue(disk.percentage_remaining);
  const temp = numberValue(disk.temperature_celsius);
  if (disk.health_passed === false || (remaining !== null && remaining < 20) || Number(disk.media_errors || 0) > 0) {
    return 'danger';
  }
  if ((remaining !== null && remaining < 60) || (temp !== null && temp >= 55) || Number(disk.error_log_entries || 0) > 0) {
    return 'warn';
  }
  return 'good';
}

function formatDiskStatus(disk: ServerDiskStatus) {
  const tone = diskTone(disk);
  if (tone === 'danger') {
    return t('overview.serverStatusDiskToneDanger');
  }
  if (tone === 'warn') {
    return t('overview.serverStatusDiskToneWarn');
  }
  return t('overview.serverStatusDiskToneGood');
}

function formatDiskHealth(disk: ServerDiskStatus) {
  const remaining = numberValue(disk.percentage_remaining);
  if (remaining !== null) {
    return `${remaining}%`;
  }
  if (disk.health_passed === true) {
    return t('overview.serverStatusHealthPassed');
  }
  if (disk.health_passed === false) {
    return t('overview.serverStatusErrorState');
  }
  return t('common.none');
}

function formatDiskSubText(disk: ServerDiskStatus) {
  const parts = [
    disk.type ? String(disk.type).toUpperCase() : '',
    disk.size_bytes ? formatOptionalTraffic(disk.size_bytes) : '',
    disk.path || '',
  ].filter(Boolean);
  return parts.join(' · ');
}

function sortServerDisks(disks: ServerDiskStatus[], sortKey: ServerDiskSortKey) {
  return [...disks].sort((a, b) => {
    if (sortKey === 'temperature') {
      return compareNumbersDesc(numberValue(b.temperature_celsius), numberValue(a.temperature_celsius)) || diskRiskScore(b) - diskRiskScore(a);
    }
    if (sortKey === 'capacity') {
      return compareNumbersDesc(numberValue(b.size_bytes), numberValue(a.size_bytes)) || diskRiskScore(b) - diskRiskScore(a);
    }
    if (sortKey === 'name') {
      return diskDisplayName(a).localeCompare(diskDisplayName(b), currentLocale.value);
    }
    return diskRiskScore(b) - diskRiskScore(a);
  });
}

function compareNumbersDesc(left: number | null, right: number | null) {
  const safeLeft = left ?? Number.NEGATIVE_INFINITY;
  const safeRight = right ?? Number.NEGATIVE_INFINITY;
  return safeLeft - safeRight;
}

function diskDisplayName(disk: ServerDiskStatus) {
  return disk.model || disk.name || disk.path || '';
}

function clampNumber(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function temperaturePenalty(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  if (Number(value) >= 85) {
    return 30;
  }
  if (Number(value) >= 75) {
    return 20;
  }
  if (Number(value) >= 65) {
    return 10;
  }
  return 0;
}

function formatTemperature(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return t('common.none');
  }
  return `${Number(value).toFixed(1)}°C`;
}

function formatRPM(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return t('common.none');
  }
  return `${Math.round(Number(value))} RPM`;
}

function formatFanRPMValue(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return t('common.none');
  }
  return n(Math.round(Number(value)));
}

function formatOptionalTraffic(value: unknown) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return t('common.none');
  }
  return formatTraffic(parsed);
}

function formatServerHours(value: unknown) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return t('common.none');
  }
  return t('overview.serverStatusHours', { value: n(Math.round(parsed)) });
}

function temperatureTone(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return 'neutral';
  }
  if (Number(value) >= 75) {
    return 'danger';
  }
  if (Number(value) >= 60) {
    return 'warn';
  }
  return 'good';
}

function temperaturePercent(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return clampNumber((Number(value) / 90) * 100, 6, 100);
}

function fanPercent(value: number | null | undefined) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return clampNumber((Number(value) / 1800) * 100, 8, 100);
}

function sensorStyle(percent: number) {
  return {
    '--sensor-percent': `${clampNumber(percent, 0, 100)}%`,
  };
}

function fanSensorStyle(value: number | null | undefined, loadValue: number | null | undefined = value) {
  const rpm = Number(value);
  const duration = Number.isFinite(rpm) && rpm > 0
    ? clampNumber(1200 / rpm, 0.65, 2.4)
    : 2.4;
  return {
    ...sensorStyle(fanPercent(loadValue)),
    '--fan-duration': `${duration.toFixed(2)}s`,
  };
}

function formatServerStatusTime(value: string | undefined) {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString(currentLocale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatDurationSeconds(seconds: number) {
  if (!Number.isFinite(seconds)) {
    return t('common.none');
  }
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return t('overview.durationHoursMinutes', { hours, minutes });
  }
  if (minutes > 0) {
    return t('overview.durationMinutesSeconds', { minutes, seconds: secs });
  }
  return t('overview.durationSeconds', { seconds: secs });
}

function getCompareLabels(range: string) {
  if (isSpecificDateValue(range)) {
    return [range, getPreviousDateRangeValue(range)];
  }
  switch (range) {
    case 'today':
      return [t('overview.todayShort'), t('overview.yesterdayShort')];
    case 'yesterday':
      return [t('overview.yesterdayShort'), t('overview.dayBeforeShort')];
    case 'last7days':
      return [t('overview.last7DaysShort'), t('overview.prev7DaysShort')];
    case 'last30days':
      return [t('overview.last30DaysShort'), t('overview.prev30DaysShort')];
    case 'week':
      return [t('overview.thisWeek'), t('overview.lastWeek')];
    case 'month':
      return [t('overview.thisMonth'), t('overview.lastMonth')];
    default:
      return [t('overview.current'), t('overview.previous')];
  }
}

function getRangeLabel(range: string) {
  if (isSpecificDateValue(range)) {
    return range;
  }
  switch (range) {
    case 'today':
      return t('overview.todayShort');
    case 'yesterday':
      return t('overview.yesterdayShort');
    case 'last7days':
      return t('overview.last7DaysShort');
    case 'last30days':
      return t('overview.last30DaysShort');
    case 'week':
      return t('overview.thisWeek');
    case 'month':
      return t('overview.thisMonth');
    default:
      return t('overview.current');
  }
}

function getMetricCompareLabels(range: string) {
  if (isSpecificDateValue(range)) {
    return {
      prev: getPreviousDateRangeValue(range),
      forecast: t('overview.forecastCurrent'),
      sameTime: t('overview.sameTimePrevious'),
    };
  }
  switch (range) {
    case 'today':
      return {
        prev: t('overview.prevDay'),
        forecast: t('overview.forecastToday'),
        sameTime: t('overview.sameTimeYesterday'),
      };
    case 'yesterday':
      return {
        prev: t('overview.dayBefore'),
        forecast: t('overview.forecastYesterday'),
        sameTime: t('overview.sameTimeDayBefore'),
      };
    case 'last7days':
      return {
        prev: t('overview.prev7Days'),
        forecast: t('overview.forecastLast7Days'),
        sameTime: t('overview.sameTimePrev7Days'),
      };
    case 'last30days':
      return {
        prev: t('overview.prev30Days'),
        forecast: t('overview.forecastLast30Days'),
        sameTime: t('overview.sameTimePrev30Days'),
      };
    case 'week':
      return {
        prev: t('overview.lastWeek'),
        forecast: t('overview.forecastThisWeek'),
        sameTime: t('overview.sameTimeLastWeek'),
      };
    case 'month':
      return {
        prev: t('overview.lastMonth'),
        forecast: t('overview.forecastThisMonth'),
        sameTime: t('overview.sameTimeLastMonth'),
      };
    default:
      return {
        prev: t('overview.previous'),
        forecast: t('overview.forecastCurrent'),
        sameTime: t('overview.sameTimePrevious'),
      };
  }
}

function getForecastRateWeight(range: string) {
  if (isSpecificDateValue(range) || range === 'today' || range === 'yesterday') {
    return 0.7;
  }
  if (range === 'last7days' || range === 'week') {
    return 0.5;
  }
  return 0.3;
}

function buildOptionList(items: string[] = [], formatLabel?: (value: string) => string) {
  const options = [{ value: 'all', label: t('common.all') }];
  const seen = new Set<string>();
  (items || []).forEach((item) => {
    const label = String(item || '').trim();
    if (!label || seen.has(label)) {
      return;
    }
    seen.add(label);
    options.push({ value: label, label: formatLabel ? formatLabel(label) : label });
  });
  return options;
}

function normalizeSelectFilterValue(value: string) {
  if (!value || value === 'all') {
    return '';
  }
  return value;
}

function normalizeProgress(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null;
  }
  return Math.min(100, Math.max(0, Math.round(value)));
}

function normalizeSeconds(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null;
  }
  const normalized = Math.round(value);
  if (normalized <= 0) {
    return null;
  }
  return normalized;
}

function isSpecificDateValue(range: string) {
  return /^\d{4}-\d{2}-\d{2}$/.test(range || '');
}

function getPreviousDateRangeValue(range: string) {
  const date = parseSpecificDate(range);
  if (!date) {
    return t('overview.previous');
  }
  date.setDate(date.getDate() - 1);
  return formatDateRangeValue(date);
}

function parseSpecificDate(value: string) {
  if (!isSpecificDateValue(value)) {
    return null;
  }
  const [year, month, day] = value.split('-').map((part) => Number(part));
  const date = new Date(year, month - 1, day);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null;
  }
  return date;
}

function formatDateRangeValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

type DetailConfig = {
  title: string;
  keyLabel?: string;
  valueLabel?: string;
  showPv?: boolean;
  fetch?: () => Promise<SimpleSeriesStats>;
  formatLabel?: (label: string) => string;
  mode?: 'logs';
  layout?: 'panel' | 'modal';
  logScope?: 'status' | 'pv' | 'uv' | 'session';
  columns?: Array<{ label: string; className: string }>;
}

const zhWordNameMap: Record<string, string> = {
  Afghanistan: '阿富汗',
  Singapore: '新加坡',
  Angola: '安哥拉',
  Albania: '阿尔巴尼亚',
  'United Arab Emirates': '阿联酋',
  Argentina: '阿根廷',
  Armenia: '亚美尼亚',
  'French Southern and Antarctic Lands': '法属南半球和南极领地',
  Australia: '澳大利亚',
  Austria: '奥地利',
  Azerbaijan: '阿塞拜疆',
  Burundi: '布隆迪',
  Belgium: '比利时',
  Benin: '贝宁',
  'Burkina Faso': '布基纳法索',
  Bangladesh: '孟加拉国',
  Bulgaria: '保加利亚',
  'The Bahamas': '巴哈马',
  'Bosnia and Herzegovina': '波斯尼亚和黑塞哥维那',
  Belarus: '白俄罗斯',
  Belize: '伯利兹',
  Bermuda: '百慕大',
  Bolivia: '玻利维亚',
  Brazil: '巴西',
  Brunei: '文莱',
  Bhutan: '不丹',
  Botswana: '博茨瓦纳',
  'Central African Republic': '中非共和国',
  Canada: '加拿大',
  Switzerland: '瑞士',
  Chile: '智利',
  China: '中国',
  'Ivory Coast': '象牙海岸',
  Cameroon: '喀麦隆',
  'Democratic Republic of the Congo': '刚果民主共和国',
  'Republic of the Congo': '刚果共和国',
  Colombia: '哥伦比亚',
  'Costa Rica': '哥斯达黎加',
  Cuba: '古巴',
  'Northern Cyprus': '北塞浦路斯',
  Cyprus: '塞浦路斯',
  'Czech Republic': '捷克共和国',
  Germany: '德国',
  Djibouti: '吉布提',
  Denmark: '丹麦',
  'Dominican Republic': '多明尼加共和国',
  Algeria: '阿尔及利亚',
  Ecuador: '厄瓜多尔',
  Egypt: '埃及',
  Eritrea: '厄立特里亚',
  Spain: '西班牙',
  Estonia: '爱沙尼亚',
  Ethiopia: '埃塞俄比亚',
  Finland: '芬兰',
  Fiji: '斐',
  'Falkland Islands': '福克兰群岛',
  France: '法国',
  Gabon: '加蓬',
  'United Kingdom': '英国',
  Georgia: '格鲁吉亚',
  Ghana: '加纳',
  Guinea: '几内亚',
  Gambia: '冈比亚',
  'Guinea Bissau': '几内亚比绍',
  Greece: '希腊',
  Greenland: '格陵兰',
  Guatemala: '危地马拉',
  'French Guiana': '法属圭亚那',
  Guyana: '圭亚那',
  Honduras: '洪都拉斯',
  Croatia: '克罗地亚',
  Haiti: '海地',
  Hungary: '匈牙利',
  Indonesia: '印度尼西亚',
  India: '印度',
  Ireland: '爱尔兰',
  Iran: '伊朗',
  Iraq: '伊拉克',
  Iceland: '冰岛',
  Israel: '以色列',
  Italy: '意大利',
  Jamaica: '牙买加',
  Jordan: '约旦',
  Japan: '日本',
  Kazakhstan: '哈萨克斯坦',
  Kenya: '肯尼亚',
  Kyrgyzstan: '吉尔吉斯斯坦',
  Cambodia: '柬埔寨',
  Kosovo: '科索沃',
  Kuwait: '科威特',
  Laos: '老挝',
  Lebanon: '黎巴嫩',
  Liberia: '利比里亚',
  Libya: '利比亚',
  'Sri Lanka': '斯里兰卡',
  Lesotho: '莱索托',
  Lithuania: '立陶宛',
  Luxembourg: '卢森堡',
  Latvia: '拉脱维亚',
  Morocco: '摩洛哥',
  Moldova: '摩尔多瓦',
  Madagascar: '马达加斯加',
  Mexico: '墨西哥',
  Macedonia: '马其顿',
  Mali: '马里',
  Myanmar: '缅甸',
  Montenegro: '黑山',
  Mongolia: '蒙古',
  Mozambique: '莫桑比克',
  Mauritania: '毛里塔尼亚',
  Malawi: '马拉维',
  Malaysia: '马来西亚',
  Namibia: '纳米比亚',
  'New Caledonia': '新喀里多尼亚',
  Niger: '尼日尔',
  Nigeria: '尼日利亚',
  Nicaragua: '尼加拉瓜',
  Netherlands: '荷兰',
  Norway: '挪威',
  Nepal: '尼泊尔',
  'New Zealand': '新西兰',
  Oman: '阿曼',
  Pakistan: '巴基斯坦',
  Panama: '巴拿马',
  Peru: '秘鲁',
  Philippines: '菲律宾',
  'Papua New Guinea': '巴布亚新几内亚',
  Poland: '波兰',
  'Puerto Rico': '波多黎各',
  'North Korea': '北朝鲜',
  Portugal: '葡萄牙',
  Paraguay: '巴拉圭',
  Qatar: '卡塔尔',
  Romania: '罗马尼亚',
  Russia: '俄罗斯',
  Rwanda: '卢旺达',
  'Western Sahara': '西撒哈拉',
  'Saudi Arabia': '沙特阿拉伯',
  Sudan: '苏丹',
  'South Sudan': '南苏丹',
  Senegal: '塞内加尔',
  'Solomon Islands': '所罗门群岛',
  'Sierra Leone': '塞拉利昂',
  'El Salvador': '萨尔瓦多',
  Somaliland: '索马里兰',
  Somalia: '索马里',
  'Republic of Serbia': '塞尔维亚',
  Suriname: '苏里南',
  Slovakia: '斯洛伐克',
  Slovenia: '斯洛文尼亚',
  Sweden: '瑞典',
  Swaziland: '斯威士兰',
  Syria: '叙利亚',
  Chad: '乍得',
  Togo: '多哥',
  Thailand: '泰国',
  Tajikistan: '塔吉克斯坦',
  Turkmenistan: '土库曼斯坦',
  'East Timor': '东帝汶',
  'Trinidad and Tobago': '特里尼达和多巴哥',
  Tunisia: '突尼斯',
  Turkey: '土耳其',
  'United Republic of Tanzania': '坦桑尼亚',
  Uganda: '乌干达',
  Ukraine: '乌克兰',
  Uruguay: '乌拉圭',
  'United States': '美国',
  Uzbekistan: '乌兹别克斯坦',
  Venezuela: '委内瑞拉',
  Vietnam: '越南',
  Vanuatu: '瓦努阿图',
  'West Bank': '西岸',
  Yemen: '也门',
  'South Africa': '南非',
  Zambia: '赞比亚',
  Korea: '韩国',
  Tanzania: '坦桑尼亚',
  Zimbabwe: '津巴布韦',
  Congo: '刚果',
  'Central African Rep.': '中非',
  Serbia: '塞尔维亚',
  'Bosnia and Herz.': '波斯尼亚和黑塞哥维那',
  'Czech Rep.': '捷克',
  'W. Sahara': '西撒哈拉',
  'Lao PDR': '老挝',
  'Dem.Rep.Korea': '朝鲜',
  'Falkland Is.': '福克兰群岛',
  'Timor-Leste': '东帝汶',
  'Solomon Is.': '所罗门群岛',
  Palestine: '巴勒斯坦',
  'N. Cyprus': '北塞浦路斯',
  Aland: '奥兰群岛',
  'Fr. S. Antarctic Lands': '法属南半球和南极陆地',
  Mauritius: '毛里求斯',
  Comoros: '科摩罗',
  'Eq. Guinea': '赤道几内亚',
  'Guinea-Bissau': '几内亚比绍',
  'Dominican Rep.': '多米尼加',
  'Saint Lucia': '圣卢西亚',
  Dominica: '多米尼克',
  'Antigua and Barb.': '安提瓜和巴布达',
  'U.S. Virgin Is.': '美国原始岛屿',
  Montserrat: '蒙塞拉特',
  Grenada: '格林纳达',
  Barbados: '巴巴多斯',
  Samoa: '萨摩亚',
  Bahamas: '巴哈马',
  'Cayman Is.': '开曼群岛',
  'Faeroe Is.': '法罗群岛',
  'IsIe of Man': '马恩岛',
  Malta: '马耳他共和国',
  Jersey: '泽西',
  'Cape Verde': '佛得角共和国',
  'Turks and Caicos Is.': '特克斯和凯科斯群岛',
  'St. Vin. and Gren.': '圣文森特和格林纳丁斯',
  'Singapore Rep.': '新加坡',
  "Côte d'Ivoire": '科特迪瓦',
  'Siachen Glacier': '锡亚琴冰川',
  'Br. Indian Ocean Ter.': '英属印度洋领土',
  'Dem. Rep. Congo': '刚果民主共和国',
  'Dem. Rep. Korea': '朝鲜',
  'S. Sudan': '南苏丹',
};

const worldNameMap = computed(() => (currentLocale.value === 'zh-CN' ? zhWordNameMap : undefined));
const worldNameMapEn: Record<string, string> = Object.entries(zhWordNameMap).reduce((acc, [en, zh]) => {
  acc[zh] = en;
  return acc;
}, {} as Record<string, string>);

const chinaNameMap = computed(() => buildChinaNameMap(currentLocale.value));

function buildChinaNameMap(localeValue: string) {
  const map: Record<string, string> = {};
  Object.entries(chinaProvinceAlias).forEach(([full, short]) => {
    map[full] = localeValue === 'en-US' ? (chinaProvinceMap[short] || short) : short;
  });
  return map;
}

function normalizeChinaRegionName(name: string) {
  const normalized = normalizeChinaProvinceName(name);
  if (currentLocale.value === 'en-US') {
    return chinaProvinceMap[normalized] || normalized;
  }
  return normalized;
}

function normalizeWorldRegionName(name: string) {
  const trimmed = String(name || '').trim();
  if (!trimmed) {
    return trimmed;
  }
  if (currentLocale.value === 'en-US') {
    return worldNameMapEn[trimmed] || trimmed;
  }
  return zhWordNameMap[trimmed] || trimmed;
}

function isPendingGeoName(name: string) {
  const trimmed = String(name || '').trim();
  if (!trimmed) {
    return false;
  }
  const lower = trimmed.toLowerCase();
  return trimmed === '待解析' || trimmed === '解析中' || lower === 'pending' || lower === 'resolving';
}

function isExcludedGeoName(name: string) {
  const trimmed = String(name || '').trim();
  if (!trimmed) {
    return true;
  }
  const lower = trimmed.toLowerCase();
  return (
    isPendingGeoName(trimmed) ||
    trimmed === '国外' ||
    trimmed === '未知' ||
    lower === 'overseas' ||
    lower === 'unknown'
  );
}
</script>

<style scoped lang="scss">
.chart-error-message {
  margin-top: 20px;
  padding: 40px;
  text-align: center;
  color: #721c24;
  background-color: #f8d7da;
  border: 1px solid #f5c6cb;
  border-radius: var(--radius-2xs);
}

.overview-empty-state {
  width: 100%;
  min-height: 180px;
  box-sizing: border-box;
  border-radius: var(--radius-lg);
  border: 1px dashed rgba(var(--primary-color-rgb), 0.16);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.06), transparent 58%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(247, 250, 255, 0.84));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px 24px;
  text-align: center;
}

.overview-empty-state-compact {
  min-height: 112px;
  padding: 14px 18px;
  gap: 6px;
}

.geo-empty-state {
  min-height: 260px;
  border-color: rgba(var(--primary-color-rgb), 0.12);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.04), transparent 62%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.88), rgba(248, 250, 255, 0.8));
}

.overview-empty-state-table {
  min-height: 180px;
}

.detail-empty-state {
  min-height: 260px;
  border-radius: var(--radius-lg);
  border: 1px dashed rgba(var(--primary-color-rgb), 0.16);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.06), transparent 58%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(247, 250, 255, 0.84));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  text-align: center;
  box-sizing: border-box;
}

.detail-empty-state-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.1);
}

.detail-empty-state-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text);
}

.detail-empty-state-text {
  max-width: 320px;
  font-size: 12px;
  line-height: 1.55;
  color: var(--muted);
}

.overview-empty-state-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.1);
}

.overview-empty-state-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text);
}

.overview-empty-state-text {
  max-width: 320px;
  font-size: 12px;
  line-height: 1.55;
  color: var(--muted);
}

.chart-wrap .overview-empty-state {
  min-height: 232px;
  padding-inline: 28px;
}

.chart-mini .overview-empty-state {
  min-height: 136px;
}

.chart-mini .overview-empty-state-icon,
.device-chart .overview-empty-state-icon {
  width: 36px;
  height: 36px;
  font-size: 18px;
}

.chart-mini .overview-empty-state-title,
.device-chart .overview-empty-state-title {
  font-size: 12px;
}

.chart-mini .overview-empty-state-text,
.device-chart .overview-empty-state-text {
  max-width: 240px;
  font-size: 11px;
  line-height: 1.5;
}

.device-chart .overview-empty-state {
  min-height: 136px;
}

.geo-empty-state .overview-empty-state-icon {
  width: 42px;
  height: 42px;
  font-size: 20px;
  background: rgba(var(--primary-color-rgb), 0.08);
}

.geo-empty-state .overview-empty-state-text {
  max-width: 360px;
}

:global(body.dark-mode) .overview-empty-state {
  border-color: rgba(148, 163, 184, 0.26);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.12), transparent 56%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.68), rgba(15, 23, 42, 0.56));
  color: var(--text);
}

:global(body.dark-mode) .overview-empty-state-icon {
  color: #8bb8ff;
  background: rgba(var(--primary-color-rgb), 0.16);
}

:global(body.dark-mode) .overview-empty-state-title {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .overview-empty-state-text {
  color: rgba(148, 163, 184, 0.92);
}

:global(body.dark-mode) .geo-empty-state {
  border-color: rgba(148, 163, 184, 0.18);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.08), transparent 62%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.62), rgba(15, 23, 42, 0.5));
}

:global(body.dark-mode) .detail-empty-state {
  border-color: rgba(148, 163, 184, 0.26);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.12), transparent 56%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.68), rgba(15, 23, 42, 0.56));
  color: var(--text);
}

:global(body.dark-mode) .detail-empty-state-icon {
  color: #8bb8ff;
  background: rgba(var(--primary-color-rgb), 0.16);
}

:global(body.dark-mode) .detail-empty-state-title {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .detail-empty-state-text {
  color: rgba(148, 163, 184, 0.92);
}
</style>
