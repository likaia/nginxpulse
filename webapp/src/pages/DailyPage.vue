<template>
  <header class="page-header">
    <div class="page-title">
      <span class="title-chip">{{ t('daily.title') }}</span>
      <p class="title-sub">{{ t('daily.subtitle') }}</p>
    </div>
    <div class="header-actions header-actions-tech">
      <HeaderToolbar class="header-toolbar-tech">
        <template #primary>
          <div class="site-select-pill">
            <span class="site-label">{{ t('common.website') }}</span>
            <WebsiteSelect
              v-model="currentWebsiteId"
              class="website-select-compact"
              :websites="websites"
              :loading="websitesLoading"
              id="daily-website-selector"
              label=""
            />
          </div>
          <div class="site-select-pill">
            <span class="site-label">{{ t('common.date') }}</span>
            <div class="daily-date-control">
              <button
                type="button"
                class="daily-date-nav-btn"
                :title="t('daily.prevDate')"
                :aria-label="t('daily.prevDate')"
                @click="goToPrevDate"
              >
                <i class="ri-arrow-left-s-line" aria-hidden="true"></i>
              </button>
              <DatePicker
                v-model="currentDate"
                class="daily-date-picker toolbar-date-picker"
                dateFormat="yy-mm-dd"
                updateModelType="string"
                :maxDate="maxDate"
                :showClear="false"
                showButtonBar
                :showIcon="true"
              />
              <button
                type="button"
                class="daily-date-nav-btn"
                :title="t('daily.nextDate')"
                :aria-label="t('daily.nextDate')"
                :disabled="!canGoNextDay"
                @click="goToNextDate"
              >
                <i class="ri-arrow-right-s-line" aria-hidden="true"></i>
              </button>
            </div>
          </div>
        </template>
        <template #utility>
          <SystemNotifications />
          <ThemeToggle />
        </template>
      </HeaderToolbar>
    </div>
  </header>

  <section class="daily-url-filter-row" :class="{ active: activeDailyURLFilter }">
    <div class="daily-url-filter-copy">
      <span class="daily-url-filter-label">
        <i class="ri-links-line" aria-hidden="true"></i>
        {{ t('common.url') }}
      </span>
      <span class="daily-url-filter-status">
        {{ activeDailyURLFilter ? t('daily.urlSearchActive', { value: activeDailyURLFilter }) : t('daily.urlSearchAll') }}
      </span>
    </div>
    <div class="daily-url-filter-controls">
      <label class="daily-url-filter" for="daily-url-filter">
        <i class="ri-search-line" aria-hidden="true"></i>
        <input
          id="daily-url-filter"
          v-model="dailyURLInput"
          type="search"
          :placeholder="t('daily.urlSearchPlaceholder')"
          autocomplete="off"
          @keyup.enter="applyDailyURLFilter"
        />
      </label>
    </div>
  </section>

  <section class="daily-kpi-grid">
    <div class="card daily-kpi-card">
      <div class="daily-kpi-header">
        <div>
          <div class="daily-kpi-title">{{ t('daily.pv') }}</div>
          <div class="daily-kpi-date">{{ currentDate }}</div>
        </div>
        <span class="daily-kpi-icon orange"><i class="ri-pages-line"></i></span>
      </div>
      <div class="daily-kpi-value">{{ kpiMetrics.pv.valueText }}</div>
      <div class="daily-kpi-compare">
        <span>{{ t('common.comparePrev') }}</span>
        <span class="daily-kpi-delta" :class="kpiMetrics.pv.deltaClass">{{ kpiMetrics.pv.deltaText }}</span>
        <span class="daily-kpi-rate" :class="kpiMetrics.pv.rateClass">{{ kpiMetrics.pv.rateText }}</span>
      </div>
    </div>
    <div class="card daily-kpi-card">
      <div class="daily-kpi-header">
        <div>
          <div class="daily-kpi-title">{{ t('daily.uv') }}</div>
          <div class="daily-kpi-date">{{ currentDate }}</div>
        </div>
        <span class="daily-kpi-icon green"><i class="ri-user-3-line"></i></span>
      </div>
      <div class="daily-kpi-value">{{ kpiMetrics.uv.valueText }}</div>
      <div class="daily-kpi-compare">
        <span>{{ t('common.comparePrev') }}</span>
        <span class="daily-kpi-delta" :class="kpiMetrics.uv.deltaClass">{{ kpiMetrics.uv.deltaText }}</span>
        <span class="daily-kpi-rate" :class="kpiMetrics.uv.rateClass">{{ kpiMetrics.uv.rateText }}</span>
      </div>
    </div>
    <div class="card daily-kpi-card">
      <div class="daily-kpi-header">
        <div>
          <div class="daily-kpi-title">{{ t('daily.session') }}</div>
          <div class="daily-kpi-date">{{ currentDate }}</div>
        </div>
        <span class="daily-kpi-icon blue"><i class="ri-chat-3-line"></i></span>
      </div>
      <div class="daily-kpi-value">{{ kpiMetrics.session.valueText }}</div>
      <div class="daily-kpi-compare">
        <span>{{ t('common.comparePrev') }}</span>
        <span class="daily-kpi-delta" :class="kpiMetrics.session.deltaClass">{{ kpiMetrics.session.deltaText }}</span>
        <span class="daily-kpi-rate" :class="kpiMetrics.session.rateClass">{{ kpiMetrics.session.rateText }}</span>
      </div>
    </div>
    <div class="card daily-kpi-card">
      <div class="daily-kpi-header">
        <div>
          <div class="daily-kpi-title">{{ t('daily.bounce') }}</div>
          <div class="daily-kpi-date">{{ currentDate }}</div>
        </div>
        <span class="daily-kpi-icon purple"><i class="ri-run-line"></i></span>
      </div>
      <div class="daily-kpi-value">{{ kpiMetrics.bounce.valueText }}</div>
      <div class="daily-kpi-compare">
        <span>{{ t('common.comparePrev') }}</span>
        <span class="daily-kpi-delta" :class="kpiMetrics.bounce.deltaClass">{{ kpiMetrics.bounce.deltaText }}</span>
        <span class="daily-kpi-rate" :class="kpiMetrics.bounce.rateClass">{{ kpiMetrics.bounce.rateText }}</span>
      </div>
    </div>
    <div class="card daily-kpi-card">
      <div class="daily-kpi-header">
        <div>
          <div class="daily-kpi-title">{{ t('daily.duration') }}</div>
          <div class="daily-kpi-date">{{ currentDate }}</div>
        </div>
        <span class="daily-kpi-icon teal"><i class="ri-time-line"></i></span>
      </div>
      <div class="daily-kpi-value">{{ kpiMetrics.duration.valueText }}</div>
      <div class="daily-kpi-compare">
        <span>{{ t('common.comparePrev') }}</span>
        <span class="daily-kpi-delta" :class="kpiMetrics.duration.deltaClass">{{ kpiMetrics.duration.deltaText }}</span>
        <span class="daily-kpi-rate" :class="kpiMetrics.duration.rateClass">{{ kpiMetrics.duration.rateText }}</span>
      </div>
    </div>
  </section>

  <section class="daily-mini-grid">
    <div class="card daily-mini-card">
      <div class="daily-mini-title">{{ t('daily.ipAvg') }}</div>
      <div class="daily-mini-body">
        <div class="daily-mini-metric">
          <div class="daily-mini-label">{{ t('daily.changeRate') }}</div>
          <div class="daily-mini-value" :class="ipAvg.rateClass">{{ ipAvg.rateText }}</div>
        </div>
        <div class="daily-mini-meta">
          <div>{{ t('daily.yesterday') }} <span>{{ ipAvg.currentText }}</span></div>
          <div>{{ t('daily.prevDay') }} <span>{{ ipAvg.prevText }}</span></div>
        </div>
      </div>
    </div>
    <div class="card daily-mini-card">
      <div class="daily-mini-title">{{ t('daily.uvAvg') }}</div>
      <div class="daily-mini-body">
        <div class="daily-mini-metric">
          <div class="daily-mini-label">{{ t('daily.changeRate') }}</div>
          <div class="daily-mini-value" :class="ipAvg.rateClass">{{ ipAvg.rateText }}</div>
        </div>
        <div class="daily-mini-meta">
          <div>{{ t('daily.yesterday') }} <span>{{ ipAvg.currentText }}</span></div>
          <div>{{ t('daily.prevDay') }} <span>{{ ipAvg.prevText }}</span></div>
        </div>
      </div>
    </div>
    <div class="card daily-trend-card">
      <div class="daily-trend-header">
        <div class="daily-trend-title">{{ t('daily.trendTitle') }}</div>
        <div class="daily-trend-sub">{{ trendSummary }}</div>
      </div>
      <div class="daily-trend-chart">
        <canvas v-show="hasTrafficTrendData" ref="ipChartRef"></canvas>
        <div v-if="!hasTrafficTrendData" class="daily-empty-state">
          <span class="daily-empty-state-icon"><i class="ri-line-chart-line"></i></span>
          <div class="daily-empty-state-title">{{ t('daily.trendEmptyTitle') }}</div>
          <div class="daily-empty-state-text">{{ t('daily.trendEmptyText') }}</div>
        </div>
      </div>
    </div>
  </section>

  <section class="card daily-dev-section">
    <div class="daily-section-header daily-dev-header">
      <div class="daily-section-title">
        <span class="section-icon danger"><i class="ri-pulse-line"></i></span>
        {{ t('daily.devSectionTitle') }}
      </div>
      <div class="daily-dev-pill-row">
        <span v-for="pill in developerSummaryPills" :key="pill.label" class="daily-dev-pill">
          <span class="daily-dev-pill-label">{{ pill.label }}</span>
          <strong>{{ pill.value }}</strong>
        </span>
      </div>
    </div>
    <div class="daily-dev-subtitle">{{ t('daily.devSectionSubtitle') }}</div>
    <button
      v-if="developerDigest.compact"
      type="button"
      class="daily-dev-status-strip"
      :class="developerDigest.tone"
      @click="goToLogsByDigest(developerDigest.summaryQuery)"
    >
      <span class="daily-dev-status-dot" :class="developerDigest.tone" aria-hidden="true"></span>
      <div class="daily-dev-status-copy">
        <div class="daily-dev-status-title">{{ developerDigest.title }}</div>
        <div class="daily-dev-status-text">{{ developerDigest.summary }}</div>
      </div>
      <span class="daily-dev-status-meta" :class="developerDigest.tone">{{ developerDigest.label }}</span>
    </button>
    <div v-else class="daily-dev-digest" :class="developerDigest.tone">
      <div class="daily-dev-digest-head">
        <div class="daily-dev-digest-title">{{ developerDigest.title }}</div>
        <span class="daily-dev-digest-badge" :class="developerDigest.tone">{{ developerDigest.label }}</span>
      </div>
      <div class="daily-dev-digest-lines">
        <button
          v-for="line in developerDigest.lines"
          :key="line.text"
          type="button"
          class="daily-dev-digest-line"
          @click="goToLogsByDigest(line.query)"
        >
          {{ line.text }}
        </button>
      </div>
    </div>

    <div class="daily-dev-card-grid">
      <div v-for="card in developerCards" :key="card.key" class="daily-dev-card">
        <div class="daily-dev-card-header">
          <div>
            <div class="daily-dev-card-title">{{ card.title }}</div>
            <div class="daily-dev-card-subtitle">{{ card.subtitle }}</div>
          </div>
          <span class="daily-dev-card-icon" :class="card.iconClass">
            <i :class="card.icon"></i>
          </span>
        </div>
        <div class="daily-dev-card-value">{{ card.valueText }}</div>
        <div class="daily-dev-card-meta">
          <span>{{ t('common.comparePrev') }}</span>
          <span :class="card.deltaClass">{{ card.deltaText }}</span>
        </div>
        <div class="daily-dev-card-detail">{{ card.detailText }}</div>
      </div>
    </div>

    <div class="daily-dev-grid">
      <div class="daily-dev-chart-card">
        <div class="daily-dev-block-header">
          <div class="daily-dev-block-title">{{ t('daily.devTrendTitle') }}</div>
          <div class="daily-dev-block-sub">{{ t('daily.devTrendSubtitle') }}</div>
        </div>
        <div class="daily-dev-chart">
          <div v-if="hasDeveloperTrendData" class="daily-dev-pulse-board" role="img" :aria-label="developerTrendAriaLabel">
            <div
              v-for="day in developerTrendDays"
              :key="day.label"
              class="daily-dev-pulse-day"
              :class="day.tone"
              :title="day.title"
            >
              <div class="daily-dev-pulse-label">{{ day.label }}</div>
              <div class="daily-dev-pulse-meter" aria-hidden="true">
                <span class="daily-dev-pulse-gridline top"></span>
                <span class="daily-dev-pulse-gridline mid"></span>
                <span class="daily-dev-pulse-stack">
                  <span
                    v-if="day.status4xx > 0"
                    class="daily-dev-pulse-bar status4xx"
                    :style="{ height: day.status4xxHeight }"
                  ></span>
                  <span
                    v-if="day.status5xx > 0"
                    class="daily-dev-pulse-bar status5xx"
                    :style="{ height: day.status5xxHeight }"
                  ></span>
                </span>
              </div>
              <div class="daily-dev-pulse-meta">
                <strong>{{ day.totalText }}</strong>
                <span>{{ day.latencyText }}</span>
              </div>
            </div>
          </div>
          <div v-if="!hasDeveloperTrendData" class="daily-empty-state">
            <span class="daily-empty-state-icon"><i class="ri-pulse-line"></i></span>
            <div class="daily-empty-state-title">{{ t('daily.devTrendEmptyTitle') }}</div>
            <div class="daily-empty-state-text">{{ t('daily.devTrendEmptyText') }}</div>
          </div>
        </div>
      </div>

      <div class="daily-dev-table-card">
        <div class="daily-dev-block-header">
          <div class="daily-dev-block-title">{{ t('daily.devIssueTitle') }}</div>
          <div class="daily-dev-block-sub">{{ t('daily.devIssueSubtitle') }}</div>
        </div>
        <div class="daily-dev-issue-list">
          <div v-if="!developerIssueRows.length" class="daily-empty-state compact">
            <span class="daily-empty-state-icon"><i class="ri-checkbox-circle-line"></i></span>
            <div class="daily-empty-state-title">{{ t('daily.devIssueEmpty') }}</div>
          </div>
          <article
            v-else
            v-for="(row, index) in developerIssueRows"
            :key="row.url"
            class="daily-dev-issue-card"
            :class="{ critical: row.errors5xx > 0, watch: row.slowRequests > 0 && row.errors5xx <= 0 }"
          >
            <button type="button" class="daily-dev-issue-main" @click="goToLogsByIssue(row.url, 'all')">
              <span class="daily-dev-issue-rank">{{ index + 1 }}</span>
              <span class="daily-dev-issue-copy">
                <span class="daily-dev-issue-url" :title="row.url">{{ row.url }}</span>
                <span class="daily-dev-issue-meta">{{ t('daily.devIssueRequestCount', { value: row.requestsText }) }}</span>
              </span>
            </button>
            <div class="daily-dev-issue-metrics">
              <button
                type="button"
                class="daily-dev-issue-metric danger"
                :aria-label="t('daily.devView5xx')"
                @click="goToLogsByIssue(row.url, '5xx')"
              >
                <span>{{ t('daily.devIssue5xx') }}</span>
                <strong>{{ row.errors5xxText }}</strong>
              </button>
              <button
                type="button"
                class="daily-dev-issue-metric latency"
                :aria-label="t('daily.devViewLatency')"
                @click="goToLogsByIssue(row.url, 'latency')"
              >
                <span>{{ t('logs.duration') }}</span>
                <strong>{{ row.avgRequestTimeText }}</strong>
                <small>{{ t('daily.devMaxDuration', { value: row.maxRequestTimeText }) }}</small>
              </button>
              <button
                type="button"
                class="daily-dev-issue-metric slow"
                :aria-label="t('daily.devViewSlow')"
                @click="goToLogsByIssue(row.url, 'slow')"
              >
                <span>{{ t('daily.devSlowCount') }}</span>
                <strong>{{ row.slowRequestsText }}</strong>
              </button>
            </div>
          </article>
        </div>
        <div class="daily-summary-strip">{{ developerIssueSummary }}</div>
      </div>
    </div>
  </section>

  <section class="card daily-section">
    <div class="daily-section-header">
      <div class="daily-section-title">
        <span class="section-icon blue"><i class="ri-compass-3-line"></i></span>
        {{ t('daily.sourceAnalysis') }}
      </div>
    </div>
    <div class="daily-section-body daily-source-grid">
      <div class="daily-donut-card">
        <div class="daily-donut">
          <canvas v-show="hasSourceDistributionData" ref="sourceChartRef"></canvas>
          <div v-if="!hasSourceDistributionData" class="daily-empty-state">
            <span class="daily-empty-state-icon"><i class="ri-share-forward-line"></i></span>
            <div class="daily-empty-state-title">{{ t('daily.sourceEmptyTitle') }}</div>
            <div class="daily-empty-state-text">{{ t('daily.sourceEmptyText') }}</div>
          </div>
        </div>
        <div class="daily-summary-cards">
          <div class="daily-summary-card search">
            <div class="daily-summary-label">{{ t('daily.searchEngine') }}</div>
            <div class="daily-summary-value">{{ sourceCards.search.countText }}</div>
            <div class="daily-summary-rate" :class="sourceCards.search.rateClass">{{ sourceCards.search.rateText }}</div>
          </div>
          <div class="daily-summary-card direct">
            <div class="daily-summary-label">{{ t('daily.direct') }}</div>
            <div class="daily-summary-value">{{ sourceCards.direct.countText }}</div>
            <div class="daily-summary-rate" :class="sourceCards.direct.rateClass">{{ sourceCards.direct.rateText }}</div>
          </div>
          <div class="daily-summary-card external">
            <div class="daily-summary-label">{{ t('daily.external') }}</div>
            <div class="daily-summary-value">{{ sourceCards.external.countText }}</div>
            <div class="daily-summary-rate" :class="sourceCards.external.rateClass">{{ sourceCards.external.rateText }}</div>
          </div>
        </div>
      </div>
      <div class="daily-table-card">
        <div class="daily-tab-bar">
          <button class="daily-tab" :class="{ active: sourceTab === 'referer' }" @click="sourceTab = 'referer'">{{ t('daily.refererTop') }}</button>
          <button class="daily-tab" :class="{ active: sourceTab === 'search' }" @click="sourceTab = 'search'">{{ t('daily.searchEngine') }}</button>
          <button class="daily-tab" :class="{ active: sourceTab === 'ip' }" @click="sourceTab = 'ip'">{{ t('daily.visitorIpTop') }}</button>
        </div>
        <div v-if="sourceTab === 'ip'" class="daily-source-inline">
          <span class="daily-source-filter-label">{{ t('daily.ipSourceFilter') }}</span>
          <button class="daily-source-link" :class="{ active: sourceIPTab === 'all' }" @click="sourceIPTab = 'all'">
            {{ t('daily.ipSourceAll') }}
          </button>
          <span class="daily-source-divider">/</span>
          <button class="daily-source-link" :class="{ active: sourceIPTab === 'search' }" @click="sourceIPTab = 'search'">
            {{ t('daily.ipSourceSearch') }}
          </button>
          <span class="daily-source-divider">/</span>
          <button class="daily-source-link" :class="{ active: sourceIPTab === 'direct' }" @click="sourceIPTab = 'direct'">
            {{ t('daily.ipSourceDirect') }}
          </button>
          <span class="daily-source-divider">/</span>
          <button class="daily-source-link" :class="{ active: sourceIPTab === 'external' }" @click="sourceIPTab = 'external'">
            {{ t('daily.ipSourceExternal') }}
          </button>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table" v-show="sourceTab === 'referer'">
            <thead>
              <tr>
                <th>{{ t('daily.refererTop') }}</th>
                <th>{{ t('daily.ipCount') }}</th>
                <th>{{ t('daily.ipDelta') }}</th>
                <th>{{ t('daily.changeRate') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!refererRows.length">
                <td colspan="4">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in refererRows" :key="row.label">
                <td :title="row.label">{{ row.label }}</td>
                <td>{{ row.valueText }}</td>
                <td :class="row.deltaClass">{{ row.deltaText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
              </tr>
            </tbody>
          </table>
          <table class="ranking-table" v-show="sourceTab === 'search'">
            <thead>
              <tr>
                <th>{{ t('daily.searchEngine') }}</th>
                <th>{{ t('daily.ipCount') }}</th>
                <th>{{ t('daily.ipDelta') }}</th>
                <th>{{ t('daily.changeRate') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!searchRows.length">
                <td colspan="4">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in searchRows" :key="row.label">
                <td :title="row.label">{{ row.label }}</td>
                <td>{{ row.valueText }}</td>
                <td :class="row.deltaClass">{{ row.deltaText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
              </tr>
            </tbody>
          </table>
          <table class="ranking-table" v-show="sourceTab === 'ip'">
            <thead>
              <tr>
                <th>{{ sourceIPTableTitle }}</th>
                <th>{{ t('daily.ipCount') }}</th>
                <th>{{ t('daily.ipDelta') }}</th>
                <th>{{ t('daily.changeRate') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!sourceIPRows.length">
                <td colspan="4">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in sourceIPRows" :key="row.label" class="daily-ip-row" @click="goToLogsByIP(row.label)">
                <td class="daily-ip-cell">
                  <span class="daily-ip-main">
                    <span class="daily-ip-label" :title="row.label">{{ row.label }}</span>
                    <i class="ri-information-line daily-ip-info" aria-hidden="true"></i>
                  </span>
                  <div class="daily-ip-popover" role="tooltip">
                    <div class="daily-ip-popover-line">
                      <span>{{ t('daily.ipSourceShare') }}</span>
                      <strong>{{ row.sourceShareText }}</strong>
                    </div>
                    <div class="daily-ip-popover-line">
                      <span>{{ t('daily.ipRegion') }}</span>
                      <strong>{{ row.regionText }}</strong>
                    </div>
                  </div>
                </td>
                <td>{{ row.valueText }}</td>
                <td :class="row.deltaClass">{{ row.deltaText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="daily-summary-strip">{{ sourceSummaryText }}</div>
      </div>
    </div>
  </section>

  <section class="daily-dual-grid">
    <div class="card daily-section">
    <div class="daily-section-header">
      <div class="daily-section-title">
        <span class="section-icon orange"><i class="ri-pages-line"></i></span>
        {{ t('daily.contentAnalysis') }}
      </div>
    </div>
    <div class="table-wrapper">
      <table class="ranking-table">
        <thead>
          <tr>
            <th>{{ t('daily.topPages') }}</th>
            <th>{{ t('daily.ipContribution') }}</th>
            <th>{{ t('daily.change') }}</th>
            <th>{{ t('daily.pvContribution') }}</th>
            <th>{{ t('daily.change') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!contentRows.length">
            <td colspan="5">{{ t('common.noData') }}</td>
          </tr>
            <tr v-else v-for="row in contentRows" :key="row.label">
              <td :title="row.label">{{ row.label }}</td>
              <td>{{ row.uvText }}</td>
              <td :class="row.uvRateClass">{{ row.uvRateText }}</td>
              <td>{{ row.pvText }}</td>
              <td :class="row.pvRateClass">{{ row.pvRateText }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div class="card daily-section">
    <div class="daily-section-header">
      <div class="daily-section-title">
        <span class="section-icon green"><i class="ri-user-heart-line"></i></span>
        {{ t('daily.visitorAnalysis') }}
      </div>
    </div>
      <div class="daily-visitor-grid">
        <div class="daily-donut">
          <canvas v-show="hasVisitorDistributionData" ref="visitorChartRef"></canvas>
          <div v-if="!hasVisitorDistributionData" class="daily-empty-state">
            <span class="daily-empty-state-icon"><i class="ri-user-search-line"></i></span>
            <div class="daily-empty-state-title">{{ t('daily.visitorEmptyTitle') }}</div>
            <div class="daily-empty-state-text">{{ t('daily.visitorEmptyText') }}</div>
          </div>
        </div>
        <div class="daily-visitor-table">
          <table class="ranking-table">
            <thead>
              <tr>
                <th>{{ t('daily.visitor') }}</th>
                <th>{{ t('common.percentage') }}</th>
                <th>{{ t('common.change') }}</th>
                <th>{{ t('daily.avgDuration') }}</th>
                <th>{{ t('daily.avgPageview') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!visitorRows.length">
                <td colspan="5">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in visitorRows" :key="row.label">
                <td>{{ row.label }}</td>
                <td>{{ row.shareText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
                <td>{{ row.durationText }}</td>
                <td>{{ row.avgPvText }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </section>

  <section class="card daily-section">
    <div class="daily-section-header">
      <div class="daily-section-title">
        <span class="section-icon blue"><i class="ri-device-line"></i></span>
        {{ t('daily.deviceAnalysis') }}
      </div>
    </div>
    <div class="daily-device-grid">
      <div class="daily-device-left">
        <div class="daily-donut">
          <canvas v-show="hasDeviceDistributionData" ref="deviceChartRef"></canvas>
          <div v-if="!hasDeviceDistributionData" class="daily-empty-state">
            <span class="daily-empty-state-icon"><i class="ri-smartphone-line"></i></span>
            <div class="daily-empty-state-title">{{ t('daily.deviceEmptyTitle') }}</div>
            <div class="daily-empty-state-text">{{ t('daily.deviceEmptyText') }}</div>
          </div>
        </div>
        <div class="daily-device-cards">
          <div class="daily-device-card">
            <div class="daily-device-icon"><i class="ri-computer-line"></i></div>
            <div class="daily-device-label">{{ t('daily.devicePc') }}</div>
            <div class="daily-device-value">{{ deviceCards.pc }}</div>
          </div>
          <div class="daily-device-card">
            <div class="daily-device-icon"><i class="ri-apple-line"></i></div>
            <div class="daily-device-label">{{ t('daily.deviceIos') }}</div>
            <div class="daily-device-value">{{ deviceCards.ios }}</div>
          </div>
          <div class="daily-device-card">
            <div class="daily-device-icon"><i class="ri-android-line"></i></div>
            <div class="daily-device-label">{{ t('daily.deviceAndroid') }}</div>
            <div class="daily-device-value">{{ deviceCards.android }}</div>
          </div>
          <div class="daily-device-card">
            <div class="daily-device-icon"><i class="ri-device-line"></i></div>
            <div class="daily-device-label">{{ t('overview.deviceApple') }}</div>
            <div class="daily-device-value">{{ deviceCards.apple }}</div>
          </div>
        </div>
      </div>
      <div class="daily-device-list">
        <div class="daily-list-title">{{ t('daily.cityTop') }}</div>
        <div class="table-wrapper">
          <table class="ranking-table">
            <thead>
              <tr>
                <th>{{ t('daily.city') }}</th>
                <th>{{ t('common.percentage') }}</th>
                <th>{{ t('common.change') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!cityRows.length">
                <td colspan="3">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in cityRows" :key="row.label">
                <td>{{ row.label }}</td>
                <td>{{ row.shareText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div class="daily-device-list">
        <div class="daily-list-title">{{ t('daily.browserTop') }}</div>
        <div class="table-wrapper">
          <table class="ranking-table">
            <thead>
              <tr>
                <th>{{ t('common.browser') }}</th>
                <th>{{ t('common.percentage') }}</th>
                <th>{{ t('common.change') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!browserRows.length">
                <td colspan="3">{{ t('common.noData') }}</td>
              </tr>
              <tr v-else v-for="row in browserRows" :key="row.label">
                <td>{{ row.label }}</td>
                <td>{{ row.shareText }}</td>
                <td :class="row.rateClass">{{ row.rateText }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </section>

  <ParsingOverlay :website-id="currentWebsiteId" @finished="loadDailyReport" />
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import {
  fetchDeveloperDailyStats,
  fetchBrowserStats,
  fetchDeviceStats,
  fetchLocationStats,
  fetchOSStats,
  fetchOverallStats,
  fetchRefererIPBatchStats,
  fetchRefererStats,
  fetchSessionSummary,
  fetchTimeSeriesStats,
  fetchUrlStats,
  fetchWebsites,
} from '@/api';
import type {
  DeveloperDailyStats,
  DeveloperMetric,
  RefererIPBatchStats,
  RefererIPGroupStats,
  SimpleSeriesStats,
  TimeSeriesStats,
  WebsiteInfo,
} from '@/api/types';
import {
  formatBrowserLabel,
  formatDeviceLabel,
  formatLocationLabel,
  formatRefererLabel,
  isDirectReferer,
  normalizeDeviceCategory,
} from '@/i18n/mappings';
import { normalizeLocale } from '@/i18n';
import { formatDate, formatTraffic, getUserPreference, saveUserPreference } from '@/utils';
import { Chart } from '@/utils/chartjs';
import ParsingOverlay from '@/components/ParsingOverlay.vue';
import HeaderToolbar from '@/components/HeaderToolbar.vue';
import SystemNotifications from '@/components/SystemNotifications.vue';
import ThemeToggle from '@/components/ThemeToggle.vue';
import WebsiteSelect from '@/components/WebsiteSelect.vue';

const websites = ref<WebsiteInfo[]>([]);
const websitesLoading = ref(true);
const currentWebsiteId = ref('');
const currentDate = ref(formatDate(new Date()));
const maxDate = new Date();
const router = useRouter();
const { t, n, locale } = useI18n({ useScope: 'global' });
const currentLocale = computed(() => normalizeLocale(locale.value));
const canGoNextDay = computed(() => currentDate.value < formatDate(maxDate));

const overall = ref<Record<string, any> | null>(null);
const sessionSummary = ref<Record<string, any> | null>(null);
const sessionSummaryPrev = ref<Record<string, any> | null>(null);
const developerDaily = ref<DeveloperDailyStats | null>(null);
const dailyURLInput = ref('');
const activeDailyURLFilter = ref('');
const timeSeries = ref<TimeSeriesStats | null>(null);
const refererStats = ref<SimpleSeriesStats | null>(null);
const refererPrev = ref<SimpleSeriesStats | null>(null);
const urlStats = ref<SimpleSeriesStats | null>(null);
const urlPrev = ref<SimpleSeriesStats | null>(null);
const deviceStats = ref<SimpleSeriesStats | null>(null);
const devicePrev = ref<SimpleSeriesStats | null>(null);
const osStats = ref<SimpleSeriesStats | null>(null);
const osPrev = ref<SimpleSeriesStats | null>(null);
const browserStats = ref<SimpleSeriesStats | null>(null);
const browserPrev = ref<SimpleSeriesStats | null>(null);
const cityStats = ref<SimpleSeriesStats | null>(null);
const cityPrev = ref<SimpleSeriesStats | null>(null);

type SourceIPKind = 'all' | 'search' | 'direct' | 'external';

const refererIPBatch = ref<RefererIPBatchStats | null>(null);
const refererIPBatchPrev = ref<RefererIPBatchStats | null>(null);

const sourceTab = ref<'referer' | 'search' | 'ip'>('referer');
const sourceIPTab = ref<SourceIPKind>('all');

const ipChartRef = ref<HTMLCanvasElement | null>(null);
const sourceChartRef = ref<HTMLCanvasElement | null>(null);
const visitorChartRef = ref<HTMLCanvasElement | null>(null);
const deviceChartRef = ref<HTMLCanvasElement | null>(null);

let ipChart: Chart | null = null;
let sourceChart: Chart | null = null;
let visitorChart: Chart | null = null;
let deviceChart: Chart | null = null;

let dailyRequestId = 0;
let dailyURLDebounceTimer: ReturnType<typeof window.setTimeout> | null = null;

const trendSummary = computed(() => {
  if (!timeSeries.value || !timeSeries.value.labels) {
    return t('daily.trendPending');
  }
  const data = timeSeries.value.visitors || [];
  if (!data.length) {
    return t('daily.trendEmpty');
  }
  const maxIndex = data.indexOf(Math.max(...data));
  const minIndex = data.indexOf(Math.min(...data));
  return t('daily.trendSummary', { max: maxIndex, min: minIndex });
});

const developerSummary = computed(() => developerDaily.value?.summary || emptyDeveloperSummary());

const developerSummaryPills = computed(() => {
  const summary = developerSummary.value;
  return [
    {
      label: t('daily.devTotalRequests'),
      value: formatNumber(summary.totalRequests),
    },
    {
      label: t('daily.devAvgRequestSize'),
      value: formatMetricValue(summary.avgRequestSizeBytes, 'traffic'),
    },
  ];
});

const developerDigest = computed(() => {
  const summary = developerSummary.value;
  const lines: Array<{ text: string; query: Record<string, string> }> = [];
  let severityScore = 0;
  const stableQuery = buildDigestLogsQuery({
    sortField: 'timestamp',
    sortOrder: 'desc',
  });

  const status5xxShare = summary.status5xx.shareCurrent ?? 0;
  const status5xxDelta = summary.status5xx.delta || 0;
  if (summary.status5xx.current > 0 || status5xxDelta > 0) {
    lines.push({
      text: t('daily.devDigest5xx', {
        count: formatMetricValue(summary.status5xx, 'count'),
        share: formatMetricShare(summary.status5xx),
        delta: formatMetricDelta(summary.status5xx, 'count'),
      }),
      query: buildDigestLogsQuery({
        statusClass: '5xx',
        sortField: 'status_code',
        sortOrder: 'desc',
      }),
    });
    severityScore += status5xxShare >= 0.03 || status5xxDelta >= 10 ? 2 : 1;
  }

  const latencyDelta = summary.avgRequestTimeMs.delta || 0;
  if (summary.avgRequestTimeMs.current > 0 && latencyDelta > 0) {
    lines.push({
      text: t('daily.devDigestLatency', {
        current: formatMetricValue(summary.avgRequestTimeMs, 'ms'),
        delta: formatMetricDelta(summary.avgRequestTimeMs, 'ms'),
      }),
      query: buildDigestLogsQuery({
        sortField: 'request_time_ms',
        sortOrder: 'desc',
      }),
    });
    severityScore += latencyDelta >= 200 ? 2 : 1;
  }

  const upstreamDelta = summary.avgUpstreamTimeMs.delta || 0;
  if (summary.avgUpstreamTimeMs.current > 0 && upstreamDelta > 0) {
    lines.push({
      text: t('daily.devDigestUpstream', {
        current: formatMetricValue(summary.avgUpstreamTimeMs, 'ms'),
        delta: formatMetricDelta(summary.avgUpstreamTimeMs, 'ms'),
      }),
      query: buildDigestLogsQuery({
        sortField: 'upstream_response_time_ms',
        sortOrder: 'desc',
      }),
    });
    severityScore += upstreamDelta >= 150 ? 2 : 1;
  }

  const topIssue = developerIssueRows.value[0];
  if (topIssue) {
    lines.push({
      text: t('daily.devDigestIssue', {
        url: topIssue.url,
        errors: topIssue.errors5xxText,
        duration: topIssue.avgRequestTimeText,
      }),
      query: buildDigestLogsQuery({
        filter: topIssue.url,
        sortField: 'request_time_ms',
        sortOrder: 'desc',
      }),
    });
    severityScore += topIssue.errors5xx > 0 ? 2 : 1;
  }

  if (!lines.length) {
    return {
      compact: true,
      tone: 'stable',
      title: t('daily.devStatusTitle'),
      label: t('daily.devDigestStableLabel'),
      summary: t('daily.devDigestStableLine'),
      summaryQuery: stableQuery,
      lines: [],
    };
  }

  const tone = severityScore >= 4 ? 'critical' : 'watch';
  return {
    compact: false,
    tone,
    title: t('daily.devDigestTitle'),
    label: tone === 'critical' ? t('daily.devDigestCriticalLabel') : t('daily.devDigestWatchLabel'),
    summary: '',
    summaryQuery: stableQuery,
    lines,
  };
});

const developerCards = computed(() => {
  const summary = developerSummary.value;
  const threshold = developerDaily.value?.slowThresholdMs || 0;
  return [
    {
      key: 'status5xx',
      title: t('daily.dev5xxTitle'),
      subtitle: t('daily.dev5xxSubtitle'),
      icon: 'ri-alarm-warning-line',
      iconClass: 'critical',
      valueText: formatMetricValue(summary.status5xx, 'count'),
      deltaText: formatMetricDelta(summary.status5xx, 'count'),
      deltaClass: deltaClass(summary.status5xx.delta || 0),
      detailText: t('daily.devRateDetail', { value: formatMetricShare(summary.status5xx) }),
    },
    {
      key: 'status4xx',
      title: t('daily.dev4xxTitle'),
      subtitle: t('daily.dev4xxSubtitle'),
      icon: 'ri-error-warning-line',
      iconClass: 'warning',
      valueText: formatMetricValue(summary.status4xx, 'count'),
      deltaText: formatMetricDelta(summary.status4xx, 'count'),
      deltaClass: deltaClass(summary.status4xx.delta || 0),
      detailText: t('daily.devRateDetail', { value: formatMetricShare(summary.status4xx) }),
    },
    {
      key: 'avgRequestTimeMs',
      title: t('daily.devLatencyTitle'),
      subtitle: t('daily.devLatencySubtitle'),
      icon: 'ri-timer-flash-line',
      iconClass: 'latency',
      valueText: formatMetricValue(summary.avgRequestTimeMs, 'ms'),
      deltaText: formatMetricDelta(summary.avgRequestTimeMs, 'ms'),
      deltaClass: deltaClass(summary.avgRequestTimeMs.delta || 0),
      detailText: t('daily.devRateDetail', { value: formatMetricRate(summary.avgRequestTimeMs) }),
    },
    {
      key: 'avgUpstreamTimeMs',
      title: t('daily.devUpstreamTitle'),
      subtitle: t('daily.devUpstreamSubtitle'),
      icon: 'ri-exchange-funds-line',
      iconClass: 'upstream',
      valueText: formatMetricValue(summary.avgUpstreamTimeMs, 'ms'),
      deltaText: formatMetricDelta(summary.avgUpstreamTimeMs, 'ms'),
      deltaClass: deltaClass(summary.avgUpstreamTimeMs.delta || 0),
      detailText: t('daily.devRateDetail', { value: formatMetricRate(summary.avgUpstreamTimeMs) }),
    },
    {
      key: 'slowRequestRate',
      title: t('daily.devSlowTitle'),
      subtitle: t('daily.devSlowSubtitle', { value: formatMs(threshold) }),
      icon: 'ri-speed-up-line',
      iconClass: 'slow',
      valueText: formatMetricValue(summary.slowRequestRate, 'percent'),
      deltaText: formatMetricDelta(summary.slowRequestRate, 'percent-point'),
      deltaClass: deltaClass(summary.slowRequestRate.delta || 0),
      detailText: t('daily.devSlowDetail', {
        count: formatMetricValue(summary.slowRequests, 'count'),
        threshold: formatMs(threshold),
      }),
    },
  ];
});

const developerIssueRows = computed(() =>
  (developerDaily.value?.topIssues || []).map((item) => ({
    ...item,
    requestsText: formatNumber(item.requests),
    errors5xxText: formatNumber(item.errors5xx),
    avgRequestTimeText: formatMs(item.avgRequestTimeMs),
    slowRequestsText: formatNumber(item.slowRequests),
    maxRequestTimeText: formatMs(item.maxRequestTimeMs),
  }))
);

const developerIssueSummary = computed(() => {
  const first = developerIssueRows.value[0];
  if (!first) {
    return t('daily.devIssueEmpty');
  }
  return t('daily.devIssueSummary', {
    url: first.url,
    errors: first.errors5xxText,
    duration: first.avgRequestTimeText,
    slow: first.slowRequestsText,
  });
});

const kpiMetrics = computed(() => {
  const current = overall.value || {};
  const prev = current.compare?.previous || {};
  const currentSession = sessionSummary.value || {};
  const prevSession = sessionSummaryPrev.value || {};

  return {
    pv: buildMetric(current.pv || 0, prev.pv || 0),
    uv: buildMetric(current.uv || 0, prev.uv || 0),
    session: buildMetric(current.sessionCount || 0, prev.sessionCount || 0),
    bounce: buildPercentMetric(currentSession.bounceRate || 0, prevSession.bounceRate || 0),
    duration: buildDurationMetric(currentSession.avgDurationSeconds || 0, prevSession.avgDurationSeconds || 0),
  }
});

const ipAvg = computed(() => {
  const current = overall.value || {};
  const prev = current.compare?.previous || {};
  const avg = current.uv > 0 ? current.pv / current.uv : 0;
  const prevAvg = prev.uv > 0 ? prev.pv / prev.uv : 0;
  const rate = calcRate(avg, prevAvg);
  return {
    currentText: formatNumber(avg),
    prevText: formatNumber(prevAvg),
    rateText: formatRate(rate),
    rateClass: rateClass(rate),
  }
});

const sourceGroups = computed(() => groupReferers(refererStats.value));
const sourcePrevGroups = computed(() => groupReferers(refererPrev.value));

const sourceCards = computed(() => {
  const current = sourceGroups.value;
  const prev = sourcePrevGroups.value;
  return {
    search: buildSourceCard(current.search, prev.search),
    direct: buildSourceCard(current.direct, prev.direct),
    external: buildSourceCard(current.external, prev.external),
  }
});

const refererRows = computed(() => buildRefererRows(refererStats.value, refererPrev.value, false));
const searchRows = computed(() => buildRefererRows(refererStats.value, refererPrev.value, true));
const sourceIPRows = computed(() =>
  buildIPRows(
    getSourceIPGroup(refererIPBatch.value, sourceIPTab.value),
    getSourceIPGroup(refererIPBatchPrev.value, sourceIPTab.value)
  )
);
const sourceIPScopeLabel = computed(() => {
  if (sourceIPTab.value === 'search') {
    return t('daily.ipSourceSearch');
  }
  if (sourceIPTab.value === 'direct') {
    return t('daily.ipSourceDirect');
  }
  if (sourceIPTab.value === 'external') {
    return t('daily.ipSourceExternal');
  }
  return t('daily.ipSourceAll');
});
const sourceIPTableTitle = computed(() => `${sourceIPScopeLabel.value} · ${t('daily.visitorIp')}`);
const sourceSummary = computed(() => buildSourceSummary(refererStats.value, refererPrev.value));
const sourceIPSummary = computed(() => buildSourceIPSummary(sourceIPRows.value, sourceIPScopeLabel.value));
const sourceSummaryText = computed(() => (sourceTab.value === 'ip' ? sourceIPSummary.value : sourceSummary.value));
const contentRows = computed(() => buildContentRows(urlStats.value, urlPrev.value));
const visitorRows = computed(() => buildVisitorRows(overall.value, sessionSummary.value));
const hasTrafficTrendData = computed(() => hasPositiveSeriesValues(timeSeries.value?.visitors));
const hasDeveloperTrendData = computed(() => {
  const trend = developerDaily.value?.trend;
  if (!trend) {
    return false;
  }
  return [
    trend.status4xx,
    trend.status5xx,
    trend.avgRequestTimeMs,
    trend.avgUpstreamTimeMs,
  ].some((series) => hasPositiveSeriesValues(series));
});
const developerTrendDays = computed(() => {
  const trend = developerDaily.value?.trend;
  const labels = trend?.labels || [];
  const status4xx = trend?.status4xx || [];
  const status5xx = trend?.status5xx || [];
  const avgRequestTimeMs = trend?.avgRequestTimeMs || [];
  const avgUpstreamTimeMs = trend?.avgUpstreamTimeMs || [];
  const totals = labels.map((_, idx) => Number(status4xx[idx] || 0) + Number(status5xx[idx] || 0));
  const maxTotal = Math.max(1, ...totals);

  return labels.map((label, idx) => {
    const current4xx = Number(status4xx[idx] || 0);
    const current5xx = Number(status5xx[idx] || 0);
    const total = current4xx + current5xx;
    const latency = Math.max(Number(avgRequestTimeMs[idx] || 0), Number(avgUpstreamTimeMs[idx] || 0));
    const totalHeight = total > 0 ? Math.max(18, Math.round((total / maxTotal) * 132)) : 6;
    const status5xxShare = total > 0 ? current5xx / total : 0;
    const status5xxHeight = current5xx > 0 ? Math.max(5, Math.round(totalHeight * status5xxShare)) : 0;
    const status4xxHeight = Math.max(total > 0 ? 8 : 0, totalHeight - status5xxHeight);
    const tone = current5xx > 0 ? 'critical' : total > 0 ? 'watch' : 'quiet';

    return {
      label,
      status4xx: current4xx,
      status5xx: current5xx,
      status4xxHeight: `${status4xxHeight}px`,
      status5xxHeight: `${status5xxHeight}px`,
      totalText: formatNumber(total),
      latencyText: formatMs(latency),
      tone,
      title: t('daily.devTrendDayTitle', {
        date: label,
        errors: formatNumber(total),
        status5xx: formatNumber(current5xx),
        latency: formatMs(latency),
      }),
    };
  });
});
const developerTrendAriaLabel = computed(() => {
  const days = developerTrendDays.value;
  if (!days.length) {
    return t('daily.devTrendEmptyTitle');
  }
  return days.map((day) => day.title).join('; ');
});
const hasSourceDistributionData = computed(() => Object.values(sourceGroups.value).some((value) => value > 0));
const hasVisitorDistributionData = computed(() => {
  const current = overall.value || {};
  return (current.newVisitorCount || 0) + (current.returningVisitorCount || 0) > 0;
});
const deviceDistribution = computed(() => ({
  pc: getDeviceCount(deviceStats.value, 'desktop'),
  apple: getDeviceCount(deviceStats.value, 'apple'),
  ios: getOsCount(osStats.value, ['ios', 'iphone', 'ipad']),
  android: getOsCount(osStats.value, ['android']),
}));
const hasDeviceDistributionData = computed(() =>
  Object.values(deviceDistribution.value).some((value) => Number(value || 0) > 0)
);
const deviceCards = computed(() => buildDeviceCards(deviceStats.value, osStats.value));
const browserRows = computed(() =>
  buildSimpleRows(browserStats.value, browserPrev.value, (label) => formatBrowserLabel(label, t))
);
const cityRows = computed(() =>
  buildSimpleRows(cityStats.value, cityPrev.value, (label) => formatLocationLabel(label, currentLocale.value, t))
);

onMounted(() => {
  initDateFromQuery();
  loadWebsites();
});

onBeforeUnmount(() => {
  if (dailyURLDebounceTimer) {
    window.clearTimeout(dailyURLDebounceTimer);
    dailyURLDebounceTimer = null;
  }
  destroyCharts();
});

watch(currentWebsiteId, (value) => {
  if (value) {
    saveUserPreference('selectedWebsite', value);
  }
  loadDailyReport();
});

watch(currentDate, (value) => {
  const normalizedDate = normalizeDate(value);
  if (normalizedDate !== value) {
    currentDate.value = normalizedDate;
    return;
  }
  if (value) {
    saveUserPreference('dailyReportDate', value);
  }
  loadDailyReport();
});

watch(dailyURLInput, () => {
  scheduleDailyURLFilter();
});

watch(
  timeSeries,
  async (stats) => {
    if (!stats) {
      renderTrend(null);
      return;
    }
    await nextTick();
    renderTrend(stats);
  },
  { flush: 'post' }
);

watch(
  sourceGroups,
  async (groups) => {
    await nextTick();
    renderSourceDonut(groups);
  },
  { flush: 'post' }
);

watch(
  visitorRows,
  async () => {
    await nextTick();
    renderVisitorDonut();
  },
  { flush: 'post' }
);

watch(
  deviceCards,
  async () => {
    await nextTick();
    renderDeviceDonut();
  },
  { flush: 'post' }
);

function initDateFromQuery() {
  const queryDate = getDateFromQuery();
  const savedDate = getUserPreference('dailyReportDate', '');
  const defaultDate = normalizeDate(queryDate || savedDate || formatDate(new Date()));
  currentDate.value = defaultDate;
  if (queryDate) {
    saveUserPreference('dailyReportDate', defaultDate);
  }
}

function goToPrevDate() {
  currentDate.value = shiftDate(currentDate.value, -1);
}

function goToNextDate() {
  if (!canGoNextDay.value) {
    return;
  }
  currentDate.value = shiftDate(currentDate.value, 1);
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

async function loadDailyReport() {
  if (!currentWebsiteId.value || !currentDate.value) {
    overall.value = null;
    sessionSummary.value = null;
    sessionSummaryPrev.value = null;
    developerDaily.value = null;
    timeSeries.value = null;
    refererStats.value = null;
    refererPrev.value = null;
    refererIPBatch.value = null;
    refererIPBatchPrev.value = null;
    urlStats.value = null;
    urlPrev.value = null;
    deviceStats.value = null;
    devicePrev.value = null;
    osStats.value = null;
    osPrev.value = null;
    browserStats.value = null;
    browserPrev.value = null;
    cityStats.value = null;
    cityPrev.value = null;
    return;
  }

  const requestId = ++dailyRequestId;
  const dateStr = currentDate.value;
  const prevDate = shiftDate(dateStr, -1);
  const urlFilter = activeDailyURLFilter.value;

  try {
    const requests = await Promise.allSettled([
      fetchOverallStats(currentWebsiteId.value, dateStr, undefined, urlFilter),
      fetchSessionSummary(currentWebsiteId.value, dateStr, urlFilter),
      fetchSessionSummary(currentWebsiteId.value, prevDate, urlFilter),
      fetchDeveloperDailyStats(currentWebsiteId.value, dateStr, urlFilter),
      fetchTimeSeriesStats(currentWebsiteId.value, prevDate, 'hourly', urlFilter),
      fetchRefererStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchRefererStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchRefererIPBatchStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchRefererIPBatchStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchUrlStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchUrlStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchDeviceStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchDeviceStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchOSStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchOSStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchBrowserStats(currentWebsiteId.value, dateStr, 10, urlFilter),
      fetchBrowserStats(currentWebsiteId.value, prevDate, 10, urlFilter),
      fetchLocationStats(currentWebsiteId.value, dateStr, 'city', 10, urlFilter),
      fetchLocationStats(currentWebsiteId.value, prevDate, 'city', 10, urlFilter),
    ]);

    if (requestId !== dailyRequestId) {
      return;
    }

    const pickResult = <T>(index: number, label: string): T | null => {
      const result = requests[index];
      if (result.status === 'fulfilled') {
        return result.value as T;
      }
      console.error(`加载日报分块失败: ${label}`, result.reason);
      return null;
    };

    overall.value = pickResult<Record<string, any>>(0, 'overall');
    sessionSummary.value = pickResult<Record<string, any>>(1, 'session_summary');
    sessionSummaryPrev.value = pickResult<Record<string, any>>(2, 'session_summary_prev');
    developerDaily.value = pickResult<DeveloperDailyStats>(3, 'developer_daily');
    timeSeries.value = pickResult<TimeSeriesStats>(4, 'timeseries');
    refererStats.value = pickResult<SimpleSeriesStats>(5, 'referer');
    refererPrev.value = pickResult<SimpleSeriesStats>(6, 'referer_prev');
    refererIPBatch.value = pickResult<RefererIPBatchStats>(7, 'referer_ip_batch');
    refererIPBatchPrev.value = pickResult<RefererIPBatchStats>(8, 'referer_ip_batch_prev');
    urlStats.value = pickResult<SimpleSeriesStats>(9, 'url');
    urlPrev.value = pickResult<SimpleSeriesStats>(10, 'url_prev');
    deviceStats.value = pickResult<SimpleSeriesStats>(11, 'device');
    devicePrev.value = pickResult<SimpleSeriesStats>(12, 'device_prev');
    osStats.value = pickResult<SimpleSeriesStats>(13, 'os');
    osPrev.value = pickResult<SimpleSeriesStats>(14, 'os_prev');
    browserStats.value = pickResult<SimpleSeriesStats>(15, 'browser');
    browserPrev.value = pickResult<SimpleSeriesStats>(16, 'browser_prev');
    cityStats.value = pickResult<SimpleSeriesStats>(17, 'location_city');
    cityPrev.value = pickResult<SimpleSeriesStats>(18, 'location_city_prev');
  } catch (error) {
    console.error('加载日报失败:', error);
    overall.value = null;
    sessionSummary.value = null;
    sessionSummaryPrev.value = null;
    developerDaily.value = null;
    timeSeries.value = null;
    refererStats.value = null;
    refererPrev.value = null;
    refererIPBatch.value = null;
    refererIPBatchPrev.value = null;
    urlStats.value = null;
    urlPrev.value = null;
    deviceStats.value = null;
    devicePrev.value = null;
    osStats.value = null;
    osPrev.value = null;
    browserStats.value = null;
    browserPrev.value = null;
    cityStats.value = null;
    cityPrev.value = null;
  }
}

function scheduleDailyURLFilter() {
  if (dailyURLDebounceTimer) {
    window.clearTimeout(dailyURLDebounceTimer);
  }
  dailyURLDebounceTimer = window.setTimeout(() => {
    dailyURLDebounceTimer = null;
    applyDailyURLFilter();
  }, 450);
}

function applyDailyURLFilter() {
  if (dailyURLDebounceTimer) {
    window.clearTimeout(dailyURLDebounceTimer);
    dailyURLDebounceTimer = null;
  }
  const normalized = dailyURLInput.value.trim();
  if (normalized === activeDailyURLFilter.value) {
    return;
  }
  activeDailyURLFilter.value = normalized;
  loadDailyReport();
}

function renderTrend(stats: TimeSeriesStats | null) {
  if (ipChart) {
    ipChart.destroy();
    ipChart = null;
  }
  if (!ipChartRef.value || !stats) {
    return;
  }
  if (!hasPositiveSeriesValues(stats.visitors)) {
    return;
  }
  const ctx = ipChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }
  const gradient = ctx.createLinearGradient(0, 0, 0, ipChartRef.value.height || 160);
  gradient.addColorStop(0, 'rgba(30, 123, 255, 0.35)');
  gradient.addColorStop(1, 'rgba(30, 123, 255, 0.05)');
  ipChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: stats.labels,
      datasets: [
        {
          label: t('daily.ipTraffic'),
          data: stats.visitors,
          borderColor: '#1e7bff',
          backgroundColor: gradient,
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
      scales: {
        x: { ticks: { color: '#94a3b8', maxTicksLimit: 12 }, grid: { display: false } },
        y: { ticks: { color: '#94a3b8' }, grid: { color: 'rgba(148, 163, 184, 0.2)' } },
      },
      plugins: { legend: { display: false }, tooltip: { mode: 'index', intersect: false } },
    },
  });
}

function renderSourceDonut(groups: Record<string, number>) {
  if (sourceChart) {
    sourceChart.destroy();
    sourceChart = null;
  }
  if (!sourceChartRef.value || !Object.values(groups).some((value) => value > 0)) {
    return;
  }
  const ctx = sourceChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }
  sourceChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: [t('daily.searchEngine'), t('daily.direct'), t('daily.external')],
      datasets: [
        {
          data: [groups.search, groups.direct, groups.external],
          backgroundColor: ['#1e7bff', '#ff8a3d', '#22c55e'],
          borderWidth: 0,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '60%',
      plugins: { legend: { display: false } },
    },
  });
}

function renderVisitorDonut() {
  if (visitorChart) {
    visitorChart.destroy();
    visitorChart = null;
  }
  if (!visitorChartRef.value) {
    return;
  }
  const current = overall.value || {};
  if ((current.newVisitorCount || 0) + (current.returningVisitorCount || 0) <= 0) {
    return;
  }
  const ctx = visitorChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }
  visitorChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: [t('daily.newVisitor'), t('daily.oldVisitor')],
      datasets: [
        {
          data: [current.newVisitorCount || 0, current.returningVisitorCount || 0],
          backgroundColor: ['#1e7bff', '#22c55e'],
          borderWidth: 0,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '65%',
      plugins: { legend: { position: 'bottom' } },
    },
  });
}

function renderDeviceDonut() {
  if (deviceChart) {
    deviceChart.destroy();
    deviceChart = null;
  }
  if (!deviceChartRef.value) {
    return;
  }
  const pcCount = getDeviceCount(deviceStats.value, 'desktop');
  const iosCount = getOsCount(osStats.value, ['ios', 'iphone', 'ipad']);
  const androidCount = getOsCount(osStats.value, ['android']);
  const appleCount = getDeviceCount(deviceStats.value, 'apple');
  if (pcCount + iosCount + androidCount + appleCount <= 0) {
    return;
  }
  const ctx = deviceChartRef.value.getContext('2d');
  if (!ctx) {
    return;
  }
  deviceChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: [t('daily.devicePc'), t('daily.deviceIos'), t('daily.deviceAndroid'), t('overview.deviceApple')],
      datasets: [
        {
          data: [pcCount, iosCount, androidCount, appleCount],
          backgroundColor: ['#1e7bff', '#22c55e', '#ff8a3d', '#64748b'],
          borderWidth: 0,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '60%',
      plugins: { legend: { position: 'bottom' } },
    },
  });
}

function destroyCharts() {
  if (ipChart) {
    ipChart.destroy();
    ipChart = null;
  }
  if (sourceChart) {
    sourceChart.destroy();
    sourceChart = null;
  }
  if (visitorChart) {
    visitorChart.destroy();
    visitorChart = null;
  }
  if (deviceChart) {
    deviceChart.destroy();
    deviceChart = null;
  }
}

function getSourceIPGroup(stats: RefererIPBatchStats | null, kind: SourceIPKind): RefererIPGroupStats | null {
  if (!stats) {
    return null;
  }
  return stats[kind];
}

function buildMetric(current: number, prev: number) {
  const delta = current - prev;
  const rate = calcRate(current, prev);
  return {
    valueText: formatNumber(current),
    deltaText: formatSigned(delta),
    rateText: formatRate(rate),
    deltaClass: deltaClass(delta),
    rateClass: rateClass(rate),
  }
}

function buildPercentMetric(current: number, prev: number) {
  const delta = current - prev;
  const rate = calcRate(current, prev);
  return {
    valueText: formatPercent(current),
    deltaText: formatSignedPercent(delta),
    rateText: formatRate(rate),
    deltaClass: deltaClass(delta),
    rateClass: rateClass(rate),
  }
}

function buildDurationMetric(current: number, prev: number) {
  const delta = current - prev;
  const rate = calcRate(current, prev);
  return {
    valueText: formatDuration(current),
    deltaText: formatSignedDuration(delta),
    rateText: formatRate(rate),
    deltaClass: deltaClass(delta),
    rateClass: rateClass(rate),
  }
}

function buildSourceCard(current: number, prev: number) {
  const rate = calcRate(current, prev);
  return {
    countText: formatNumber(current),
    rateText: formatRate(rate),
    rateClass: rateClass(rate),
  }
}

function buildRefererRows(stats: SimpleSeriesStats | null, prevStats: SimpleSeriesStats | null, searchOnly: boolean) {
  const prevMap = buildStatMap(prevStats);
  const keys = stats?.key || [];
  const uvs = stats?.uv || [];
  const rows = keys
    .map((key, idx) => ({ rawLabel: key, label: formatRefererLabel(key, currentLocale.value, t), value: uvs[idx] || 0 }))
    .filter((item) => (searchOnly ? isSearchEngine(item.rawLabel) : true));

  return rows.map((item) => {
    const prev = prevMap[item.rawLabel] || 0;
    const delta = item.value - prev;
    const rate = calcRate(item.value, prev);
    return {
      label: item.label,
      valueText: formatNumber(item.value),
      deltaText: formatSigned(delta),
      deltaClass: deltaClass(delta),
      rateText: formatRate(rate),
      rateClass: rateClass(rate),
    }
  });
}

function buildIPRows(stats: RefererIPGroupStats | null, prevStats: RefererIPGroupStats | null) {
  const prevMap = buildSeriesMap(prevStats?.key || [], prevStats?.uv || []);
  const keys = stats?.key || [];
  const uvs = stats?.uv || [];
  const shares = stats?.share || [];
  const domestics = stats?.domestic || [];
  const globals = stats?.global || [];
  const total = stats?.total_uv || 0;

  return keys.map((key, idx) => {
    const value = uvs[idx] || 0;
    const prev = prevMap[key] || 0;
    const delta = value - prev;
    const rate = calcRate(value, prev);
    const share = shares[idx] ?? (total > 0 ? value / total : 0);
    return {
      label: key,
      valueText: formatNumber(value),
      deltaText: formatSigned(delta),
      deltaClass: deltaClass(delta),
      rateText: formatRate(rate),
      rateClass: rateClass(rate),
      sourceShareText: formatPercent(share),
      regionText: formatRegion(domestics[idx], globals[idx]),
    };
  });
}

function buildContentRows(stats: SimpleSeriesStats | null, prevStats: SimpleSeriesStats | null) {
  const prevPV = buildStatMap(prevStats, 'pv');
  const prevUV = buildStatMap(prevStats, 'uv');

  const keys = stats?.key || [];
  const pvs = stats?.pv || [];
  const uvs = stats?.uv || [];

  return keys.map((key, idx) => {
    const pv = pvs[idx] || 0;
    const uv = uvs[idx] || 0;
    const pvRate = calcRate(pv, prevPV[key] || 0);
    const uvRate = calcRate(uv, prevUV[key] || 0);
    return {
      label: key,
      pvText: formatNumber(pv),
      uvText: formatNumber(uv),
      pvRateText: formatRate(pvRate),
      uvRateText: formatRate(uvRate),
      pvRateClass: rateClass(pvRate),
      uvRateClass: rateClass(uvRate),
    }
  });
}

function buildVisitorRows(overallData: Record<string, any> | null, sessionData: Record<string, any> | null) {
  if (!overallData) {
    return [];
  }
  const newCount = overallData.newVisitorCount || 0;
  const returningCount = overallData.returningVisitorCount || 0;
  const total = newCount + returningCount;

  const prevNew = overallData.prevNewVisitorCount || 0;
  const prevReturning = overallData.prevReturningVisitorCount || 0;
  const avgDuration = sessionData?.avgDurationSeconds || 0;
  const avgPV = overallData.uv ? overallData.pv / overallData.uv : 0;

  const rows = [
    { label: t('daily.newVisitor'), count: newCount, prev: prevNew },
    { label: t('daily.oldVisitor'), count: returningCount, prev: prevReturning },
  ];

  return rows.map((item) => {
    const share = total > 0 ? item.count / total : 0;
    const rate = calcRate(item.count, item.prev);
    return {
      label: item.label,
      shareText: formatPercent(share),
      rateText: formatRate(rate),
      rateClass: rateClass(rate),
      durationText: formatDuration(avgDuration),
      avgPvText: formatNumber(avgPV),
    }
  });
}

function buildDeviceCards(deviceData: SimpleSeriesStats | null, osData: SimpleSeriesStats | null) {
  const pcCount = getDeviceCount(deviceData, 'desktop');
  const iosCount = getOsCount(osData, ['ios', 'iphone', 'ipad']);
  const androidCount = getOsCount(osData, ['android']);
  return {
    pc: formatNumber(pcCount),
    ios: formatNumber(iosCount),
    android: formatNumber(androidCount),
    apple: formatNumber(getDeviceCount(deviceData, 'apple')),
  }
}

function buildSimpleRows(
  stats: SimpleSeriesStats | null,
  prevStats: SimpleSeriesStats | null,
  formatLabel?: (label: string) => string
) {
  const prevMap = buildStatMap(prevStats);
  const keys = stats?.key || [];
  const uvs = stats?.uv || [];
  if (!keys.length) {
    return [];
  }

  const total = uvs.reduce((sum, val) => sum + val, 0);
  return keys.map((key, idx) => {
    const value = uvs[idx] || 0;
    const share = total > 0 ? value / total : 0;
    const prev = prevMap[key] || 0;
    const rate = calcRate(value, prev);
    return {
      label: formatLabel ? formatLabel(key) : key,
      shareText: formatPercent(share),
      rateText: formatRate(rate),
      rateClass: rateClass(rate),
    }
  });
}

function buildSourceSummary(stats: SimpleSeriesStats | null, prevStats: SimpleSeriesStats | null) {
  if (!stats || !stats.key) {
    return t('daily.sourceEmpty');
  }
  const prevMap = buildStatMap(prevStats);
  const keys = stats.key || [];
  const uvs = stats.uv || [];
  if (!keys.length) {
    return t('daily.sourceEmpty');
  }
  const diffs = keys.map((key, idx) => ({
    key,
    rate: calcRate(uvs[idx] || 0, prevMap[key] || 0),
  }));
  const rising = diffs.reduce((acc, item) => (item.rate !== null && item.rate > (acc.rate ?? -Infinity) ? item : acc), {
    rate: -Infinity,
  });
  const falling = diffs.reduce((acc, item) => (item.rate !== null && item.rate < (acc.rate ?? Infinity) ? item : acc), {
    rate: Infinity,
  });
  const risingLabel = rising.key ? formatRefererLabel(rising.key, currentLocale.value, t) : t('common.none');
  const fallingLabel = falling.key ? formatRefererLabel(falling.key, currentLocale.value, t) : t('common.none');
  return t('daily.sourceSummary', {
    rising: risingLabel,
    risingRate: formatRate(rising.rate),
    falling: fallingLabel,
    fallingRate: formatRate(falling.rate),
  });
}

function buildSourceIPSummary(
  rows: Array<{
    label: string;
    valueText: string;
    deltaText: string;
    rateText: string;
    sourceShareText: string;
    regionText: string;
  }>,
  scope: string
) {
  if (!rows.length) {
    return t('daily.sourceEmpty');
  }
  const topRow = rows[0];
  return t('daily.ipTopSummary', {
    scope,
    ip: topRow.label,
    count: topRow.valueText,
    share: topRow.sourceShareText,
    region: topRow.regionText,
    delta: topRow.deltaText,
    rate: topRow.rateText,
  });
}

function emptyDeveloperMetric(): DeveloperMetric {
  return {
    current: 0,
    previous: 0,
    delta: 0,
    changeRate: 0,
  };
}

function emptyDeveloperSummary() {
  return {
    totalRequests: 0,
    avgRequestSizeBytes: emptyDeveloperMetric(),
    status5xx: emptyDeveloperMetric(),
    status4xx: emptyDeveloperMetric(),
    avgRequestTimeMs: emptyDeveloperMetric(),
    avgUpstreamTimeMs: emptyDeveloperMetric(),
    slowRequests: emptyDeveloperMetric(),
    slowRequestRate: emptyDeveloperMetric(),
  };
}

function formatMetricValue(metric: DeveloperMetric, type: 'count' | 'ms' | 'traffic' | 'percent') {
  const value = Number(metric?.current || 0);
  switch (type) {
    case 'ms':
      return formatMs(value);
    case 'traffic':
      return formatTraffic(Math.round(value));
    case 'percent':
      return formatPercent(value);
    default:
      return formatNumber(Math.round(value));
  }
}

function formatMetricDelta(metric: DeveloperMetric, type: 'count' | 'ms' | 'percent-point') {
  const value = Number(metric?.delta || 0);
  switch (type) {
    case 'ms':
      return formatSignedMs(value);
    case 'percent-point':
      return formatSignedPercent(value);
    default:
      return formatSigned(Math.round(value));
  }
}

function formatMetricRate(metric: DeveloperMetric) {
  return formatRate(metric?.changeRate ?? null);
}

function formatMetricShare(metric: DeveloperMetric) {
  if (metric?.shareCurrent === undefined || metric?.shareCurrent === null) {
    return t('common.none');
  }
  return formatPercent(metric.shareCurrent);
}

function formatRegion(domestic?: string, global?: string) {
  const domesticText = normalizeRegionValue(domestic);
  const globalText = normalizeRegionValue(global);
  if (globalText && domesticText && globalText !== domesticText) {
    return `${globalText} · ${domesticText}`;
  }
  return domesticText || globalText || t('common.none');
}

function normalizeRegionValue(value?: string) {
  const normalized = String(value || '').trim();
  if (!normalized || normalized === '-') {
    return '';
  }
  return normalized;
}

function calcRate(current: number, prev: number) {
  if (prev === 0) {
    if (current === 0) {
      return 0;
    }
    return null;
  }
  return (current - prev) / prev;
}

function hasPositiveSeriesValues(values?: Array<number | null | undefined>) {
  return Array.isArray(values) && values.some((value) => Number(value || 0) > 0);
}

function formatNumber(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  return n(Number(value));
}

function formatPercent(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  return n(value, 'percent');
}

function formatRate(rate: number | null) {
  if (rate === null) {
    return t('common.none');
  }
  return `${rate >= 0 ? '+' : ''}${(rate * 100).toFixed(2)}%`;
}

function formatSigned(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  return `${value >= 0 ? '+' : ''}${value}`;
}

function formatSignedPercent(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  return `${value >= 0 ? '+' : ''}${(value * 100).toFixed(2)}%`;
}

function formatSignedDuration(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  const prefix = value >= 0 ? '+' : '-';
  return `${prefix}${formatDuration(Math.abs(value))}`;
}

function formatMs(value: number) {
  const normalized = Number(value || 0);
  if (normalized >= 1000) {
    return `${(normalized / 1000).toFixed(normalized >= 10000 ? 1 : 2)} s`;
  }
  return `${Math.round(normalized)} ms`;
}

function formatSignedMs(value: number) {
  if (value === null || value === undefined) {
    return t('common.none');
  }
  const prefix = value >= 0 ? '+' : '-';
  return `${prefix}${formatMs(Math.abs(value))}`;
}

function formatDuration(seconds: number) {
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

function deltaClass(delta: number) {
  if (delta > 0) return 'trend-up';
  if (delta < 0) return 'trend-down';
  return 'trend-flat';
}

function rateClass(rate: number | null) {
  if (rate === null) return 'trend-flat';
  if (rate > 0) return 'trend-up';
  if (rate < 0) return 'trend-down';
  return 'trend-flat';
}

function buildStatMap(stats: SimpleSeriesStats | null, field: 'uv' | 'pv' = 'uv') {
  const map: Record<string, number> = {};
  if (!stats || !stats.key) {
    return map;
  }
  const values = stats[field] || [];
  stats.key.forEach((key, idx) => {
    map[key] = values[idx] || 0;
  });
  return map;
}

function buildSeriesMap(keys: string[], values: number[]) {
  const map: Record<string, number> = {};
  keys.forEach((key, idx) => {
    map[key] = values[idx] || 0;
  });
  return map;
}

function getDeviceCount(stats: SimpleSeriesStats | null, category: 'desktop' | 'mobile' | 'apple' | 'other') {
  if (!stats || !stats.key) {
    return 0;
  }
  let total = 0;
  const values = stats.uv || [];
  stats.key.forEach((key, idx) => {
    if (normalizeDeviceCategory(key) === category) {
      total += values[idx] || 0;
    }
  });
  return total;
}

function getOsCount(stats: SimpleSeriesStats | null, keywords: string[]) {
  if (!stats || !stats.key) {
    return 0;
  }
  let total = 0;
  const values = stats.uv || [];
  stats.key.forEach((key, idx) => {
    const lower = String(key || '').toLowerCase();
    if (keywords.some((word) => lower.includes(word))) {
      total += values[idx] || 0;
    }
  });
  return total;
}

function groupReferers(stats: SimpleSeriesStats | null) {
  const keys = stats?.key || [];
  const uvs = stats?.uv || [];
  const groups = { search: 0, direct: 0, external: 0 };
  keys.forEach((key, idx) => {
    const value = uvs[idx] || 0;
    if (isDirectReferer(key)) {
      groups.direct += value;
    } else if (isSearchEngine(key)) {
      groups.search += value;
    } else {
      groups.external += value;
    }
  });
  return groups;
}

const searchEngines = ['baidu.', 'google.', 'bing.', 'sogou.', '360.', 'so.com', 'yahoo.', 'duckduckgo.', 'yandex.'];

function isSearchEngine(value: string) {
  const lower = value.toLowerCase();
  return searchEngines.some((engine) => lower.includes(engine));
}

function shiftDate(dateStr: string, offsetDays: number) {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return formatDate(maxDate);
  }
  date.setDate(date.getDate() + offsetDays);
  return normalizeDate(formatDate(date));
}

function normalizeDate(dateStr: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateStr || '')) {
    return formatDate(maxDate);
  }
  const today = formatDate(maxDate);
  if (dateStr > today) {
    return today;
  }
  return dateStr;
}

function getDateFromQuery() {
  const params = new URLSearchParams(window.location.search || '');
  const raw = params.get('date');
  if (!raw) {
    return '';
  }
  if (raw === 'today') {
    return formatDate(new Date());
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    return raw;
  }
  return '';
}

function goToLogsByIP(ip: string) {
  const normalizedIP = String(ip || '').trim();
  if (!normalizedIP || !currentWebsiteId.value) {
    return;
  }
  const { start, end } = buildDayTimeRange(currentDate.value);
  router.push({
    name: 'logs',
    query: {
      id: currentWebsiteId.value,
      ipFilter: normalizedIP,
      timeStart: start,
      timeEnd: end,
    },
  });
}

function goToLogsByURL(url: string) {
  const normalizedURL = String(url || '').trim();
  if (!normalizedURL || !currentWebsiteId.value) {
    return;
  }
  const { start, end } = buildDayTimeRange(currentDate.value);
  router.push({
    name: 'logs',
    query: {
      id: currentWebsiteId.value,
      urlFilter: normalizedURL,
      timeStart: start,
      timeEnd: end,
    },
  });
}

function goToLogsByIssue(url: string, mode: 'all' | '5xx' | 'latency' | 'slow' | 'compare') {
  const normalizedURL = String(url || '').trim();
  if (!normalizedURL || !currentWebsiteId.value) {
    return;
  }

  const extra: Record<string, string> = {
    filter: normalizedURL,
  };

  switch (mode) {
    case '5xx':
      extra.statusClass = '5xx';
      extra.sortField = 'timestamp';
      extra.sortOrder = 'desc';
      break;
    case 'latency':
      extra.sortField = 'request_time_ms';
      extra.sortOrder = 'desc';
      break;
    case 'slow':
      extra.sortField = 'request_time_ms';
      extra.sortOrder = 'desc';
      break;
    case 'compare':
      extra.sortField = 'upstream_response_time_ms';
      extra.sortOrder = 'desc';
      break;
    default:
      extra.sortField = 'timestamp';
      extra.sortOrder = 'desc';
      break;
  }

  goToLogsByDigest(buildDigestLogsQuery(extra));
}

function goToLogsByDigest(query: Record<string, string>) {
  if (!currentWebsiteId.value) {
    return;
  }
  router.push({
    name: 'logs',
    query,
  });
}

function buildDigestLogsQuery(extra: Record<string, string>) {
  const { start, end } = buildDayTimeRange(currentDate.value);
  return {
    id: currentWebsiteId.value,
    timeStart: start,
    timeEnd: end,
    ...extra,
  };
}

function buildDayTimeRange(dateStr: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateStr || '')) {
    const fallback = formatDate(new Date());
    return {
      start: `${fallback} 00:00`,
      end: `${fallback} 23:59`,
    };
  }
  return {
    start: `${dateStr} 00:00`,
    end: `${dateStr} 23:59`,
  };
}
</script>

<style scoped lang="scss">
:global(.daily-page) {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.daily-date-picker {
  min-width: 190px;
}

.daily-date-control {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.daily-date-nav-btn {
  width: var(--toolbar-item-height);
  height: var(--toolbar-item-height);
  border-radius: var(--radius-md);
  border: 1px solid rgba(var(--primary-color-rgb), 0.2);
  background: linear-gradient(180deg, rgba(var(--primary-color-rgb), 0.1), rgba(var(--primary-color-rgb), 0.04));
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease;
}

.daily-date-nav-btn i {
  font-size: 18px;
}

.daily-date-nav-btn:hover:not(:disabled) {
  border-color: rgba(var(--primary-color-rgb), 0.48);
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.14);
  box-shadow: 0 0 0 3px rgba(var(--primary-color-rgb), 0.12);
}

.daily-date-nav-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.daily-kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.daily-kpi-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 150px;
}

.daily-kpi-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.daily-kpi-title {
  font-weight: 600;
  font-size: 14px;
}

.daily-kpi-date {
  color: var(--muted);
  font-size: 12px;
  margin-top: 2px;
}

.daily-kpi-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  background: var(--panel-muted);
  color: var(--primary);
}

.daily-kpi-icon.orange {
  color: var(--accent);
  background: rgba(245, 158, 11, 0.12);
}

.daily-kpi-icon.green {
  color: var(--green);
  background: rgba(34, 197, 94, 0.12);
}

.daily-kpi-icon.blue {
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.12);
}

.daily-kpi-icon.purple {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}

.daily-kpi-icon.teal {
  color: #14b8a6;
  background: rgba(20, 184, 166, 0.12);
}

.daily-kpi-value {
  font-size: 28px;
  font-weight: 700;
}

:deep(.header-toolbar.header-toolbar-tech .toolbar-date-picker .p-inputtext) {
  height: var(--toolbar-item-height);
  border-radius: 12px 0 0 12px;
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-right: 0;
  box-shadow: inset 0 0 0 1px transparent;
}
:deep(.p-datepicker-dropdown) {
  border-start-end-radius: 12px;
  border-end-end-radius: 12px;
}

:global(body.dark-mode) :deep(.header-toolbar.header-toolbar-tech .toolbar-date-picker .p-inputtext) {
  background: rgba(15, 23, 42, 0.64);
  border-color: rgba(148, 163, 184, 0.26);
}

.daily-kpi-compare {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--muted);
}

.daily-kpi-delta,
.daily-kpi-rate {
  font-weight: 600;
}

.daily-empty-state {
  height: 100%;
  min-height: 180px;
  border-radius: var(--radius-lg);
  border: 1px dashed rgba(var(--primary-color-rgb), 0.18);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.08), transparent 58%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(247, 250, 255, 0.86));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
  text-align: center;
}

.daily-empty-state.compact {
  min-height: 132px;
  padding: 18px;
}

.daily-empty-state-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.12);
}

.daily-empty-state-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.daily-empty-state-text {
  max-width: 240px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--muted);
}

.trend-up {
  color: #ef4444;
}

.trend-down {
  color: #16a34a;
}

.trend-flat {
  color: var(--muted);
}

.section-icon.danger {
  color: #ef4444;
  background: transparent;
  box-shadow: none;
}

.daily-dev-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  background:
    radial-gradient(circle at top right, rgba(239, 68, 68, 0.08), transparent 28%),
    radial-gradient(circle at bottom left, rgba(30, 123, 255, 0.1), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 255, 0.88));
}

.daily-dev-header {
  gap: 12px;
  flex-wrap: wrap;
}

.daily-dev-subtitle {
  margin-top: -8px;
  font-size: 13px;
  color: var(--muted);
}

.daily-url-filter-row {
  min-height: 56px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--primary-color-rgb), 0.12);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.78), rgba(248, 250, 255, 0.66));
  display: grid;
  grid-template-columns: minmax(180px, 0.8fr) minmax(280px, 1.2fr);
  align-items: center;
  gap: 14px;
  padding: 10px 14px;
}

.daily-url-filter-row.active {
  border-color: rgba(var(--primary-color-rgb), 0.28);
  box-shadow: 0 10px 24px rgba(30, 123, 255, 0.08);
}

.daily-url-filter-copy {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.daily-url-filter-label {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
  color: var(--text);
}

.daily-url-filter-label i {
  font-size: 16px;
  color: var(--primary);
}

.daily-url-filter-controls {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.daily-url-filter {
  width: 100%;
  min-width: 0;
  height: 40px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(var(--primary-color-rgb), 0.16);
  background: rgba(255, 255, 255, 0.78);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  color: var(--muted);
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.daily-url-filter:focus-within {
  border-color: rgba(var(--primary-color-rgb), 0.42);
  box-shadow: 0 0 0 3px rgba(var(--primary-color-rgb), 0.12);
  background: rgba(255, 255, 255, 0.94);
}

.daily-url-filter i {
  flex: 0 0 auto;
  font-size: 16px;
}

.daily-url-filter input {
  width: 100%;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text);
  font-size: 13px;
}

.daily-url-filter input::placeholder {
  color: var(--muted);
}

.daily-url-filter-status {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--muted);
}

.daily-dev-status-strip {
  width: 100%;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.86), rgba(249, 250, 251, 0.8));
  border-radius: 18px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  text-align: left;
  cursor: pointer;
  color: var(--text);
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.daily-dev-status-strip:hover {
  border-color: rgba(var(--primary-color-rgb), 0.26);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
}

.daily-dev-status-strip.stable {
  border-color: rgba(34, 197, 94, 0.18);
  background: linear-gradient(180deg, rgba(240, 253, 244, 0.82), rgba(255, 255, 255, 0.78));
}

.daily-dev-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  flex: 0 0 auto;
  background: rgba(var(--primary-color-rgb), 0.42);
}

.daily-dev-status-dot.stable {
  background: #22c55e;
}

.daily-dev-status-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.daily-dev-status-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--muted);
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

.daily-dev-status-text {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.5;
}

.daily-dev-status-meta {
  margin-left: auto;
  padding-left: 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
  white-space: nowrap;
}

.daily-dev-status-meta.stable {
  color: #15803d;
}

.daily-dev-digest {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px 18px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--primary-color-rgb), 0.14);
  background: rgba(255, 255, 255, 0.76);
}

.daily-dev-digest.critical {
  border-color: rgba(239, 68, 68, 0.22);
  background: linear-gradient(180deg, rgba(254, 242, 242, 0.96), rgba(255, 255, 255, 0.8));
}

.daily-dev-digest.watch {
  border-color: rgba(245, 158, 11, 0.24);
  background: linear-gradient(180deg, rgba(255, 251, 235, 0.96), rgba(255, 255, 255, 0.8));
}

.daily-dev-digest.stable {
  border-color: rgba(34, 197, 94, 0.18);
  background: linear-gradient(180deg, rgba(240, 253, 244, 0.96), rgba(255, 255, 255, 0.8));
}

.daily-dev-digest-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.daily-dev-digest-title {
  font-size: 14px;
  font-weight: 700;
}

.daily-dev-digest-badge {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.daily-dev-digest-badge.critical {
  color: #b91c1c;
  background: rgba(239, 68, 68, 0.14);
}

.daily-dev-digest-badge.watch {
  color: #b45309;
  background: rgba(245, 158, 11, 0.16);
}

.daily-dev-digest-badge.stable {
  color: #15803d;
  background: rgba(34, 197, 94, 0.14);
}

.daily-dev-digest-lines {
  display: grid;
  gap: 8px;
}

.daily-dev-digest-line {
  position: relative;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
  padding-left: 16px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
}

.daily-dev-digest-line::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: rgba(var(--primary-color-rgb), 0.5);
}

.daily-dev-digest-line:hover {
  color: var(--primary);
}

.daily-dev-pill-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.daily-dev-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid rgba(var(--primary-color-rgb), 0.14);
  background: rgba(var(--primary-color-rgb), 0.06);
  color: var(--text);
  font-size: 12px;
}

.daily-dev-pill-label {
  color: var(--muted);
}

.daily-dev-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(208px, 1fr));
  gap: 12px;
}

.daily-dev-card {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 164px;
  padding: 16px 16px 14px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--primary-color-rgb), 0.1);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.84), rgba(248, 250, 255, 0.9));
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.03);
}

.daily-dev-card::after {
  content: '';
  position: absolute;
  inset: auto -16% -44% auto;
  width: 104px;
  height: 104px;
  border-radius: 999px;
  background: rgba(var(--primary-color-rgb), 0.05);
  pointer-events: none;
}

.daily-dev-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.daily-dev-card-title {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.daily-dev-card-subtitle {
  margin-top: 4px;
  font-size: 11px;
  line-height: 1.45;
  color: var(--muted);
}

.daily-dev-card-icon {
  width: 32px;
  height: 32px;
  border-radius: 11px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 15px;
  border: 1px solid transparent;
}

.daily-dev-card-icon.critical {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.12);
}

.daily-dev-card-icon.warning {
  color: #f97316;
  background: rgba(249, 115, 22, 0.08);
  border-color: rgba(249, 115, 22, 0.12);
}

.daily-dev-card-icon.latency {
  color: #1e7bff;
  background: rgba(30, 123, 255, 0.08);
  border-color: rgba(30, 123, 255, 0.12);
}

.daily-dev-card-icon.upstream {
  color: #14b8a6;
  background: rgba(20, 184, 166, 0.08);
  border-color: rgba(20, 184, 166, 0.12);
}

.daily-dev-card-icon.slow {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.08);
  border-color: rgba(139, 92, 246, 0.12);
}

.daily-dev-card-value {
  margin-top: 2px;
  font-size: 28px;
  font-weight: 780;
  letter-spacing: -0.02em;
  line-height: 1;
}

.daily-dev-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--muted);
}

.daily-dev-card-detail {
  margin-top: auto;
  padding-top: 6px;
  font-size: 11px;
  line-height: 1.45;
  color: var(--muted);
}

.daily-dev-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
  align-items: stretch;
  gap: 16px;
}

.daily-dev-chart-card,
.daily-dev-table-card {
  min-height: 100%;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--primary-color-rgb), 0.12);
  background: rgba(255, 255, 255, 0.72);
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.daily-dev-block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.daily-dev-block-title {
  font-size: 15px;
  font-weight: 700;
}

.daily-dev-block-sub {
  font-size: 12px;
  color: var(--muted);
}

.daily-dev-chart {
  flex: 1;
  min-height: 292px;
  height: auto;
}

.daily-dev-pulse-board {
  height: 100%;
  min-height: 100%;
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 8px;
}

.daily-dev-pulse-day {
  min-width: 0;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.86), rgba(248, 250, 252, 0.78));
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 6px;
  padding: 8px;
}

.daily-dev-pulse-day.watch {
  border-color: rgba(251, 146, 60, 0.28);
  background: linear-gradient(180deg, rgba(255, 247, 237, 0.82), rgba(255, 255, 255, 0.78));
}

.daily-dev-pulse-day.critical {
  border-color: rgba(248, 113, 113, 0.32);
  background: linear-gradient(180deg, rgba(254, 242, 242, 0.88), rgba(255, 255, 255, 0.78));
}

.daily-dev-pulse-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 700;
  color: var(--muted);
  text-align: center;
}

.daily-dev-pulse-meter {
  position: relative;
  min-height: 118px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  border-radius: 12px;
  overflow: hidden;
  background:
    linear-gradient(180deg, rgba(248, 250, 252, 0.26), rgba(241, 245, 249, 0.38)),
    radial-gradient(circle at bottom, rgba(var(--primary-color-rgb), 0.07), transparent 62%);
}

.daily-dev-pulse-gridline {
  position: absolute;
  left: 10px;
  right: 10px;
  height: 1px;
  background: rgba(148, 163, 184, 0.14);
}

.daily-dev-pulse-gridline.top {
  top: 26%;
}

.daily-dev-pulse-gridline.mid {
  top: 58%;
}

.daily-dev-pulse-stack {
  position: relative;
  z-index: 1;
  width: min(38px, 54%);
  display: flex;
  flex-direction: column-reverse;
  align-items: stretch;
  justify-content: flex-start;
  border-radius: 999px 999px 8px 8px;
  overflow: hidden;
  box-shadow: 0 10px 22px rgba(251, 146, 60, 0.13);
}

.daily-dev-pulse-bar {
  display: block;
  min-height: 0;
}

.daily-dev-pulse-bar.status4xx {
  background: linear-gradient(180deg, rgba(251, 146, 60, 0.46), rgba(253, 186, 116, 0.2));
  border: 1px solid rgba(251, 146, 60, 0.42);
}

.daily-dev-pulse-bar.status5xx {
  background: linear-gradient(180deg, rgba(248, 113, 113, 0.72), rgba(254, 202, 202, 0.42));
  border: 1px solid rgba(239, 68, 68, 0.62);
  border-bottom: 0;
}

.daily-dev-pulse-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  font-size: 11px;
  color: var(--muted);
}

.daily-dev-pulse-meta strong {
  color: var(--text);
  font-size: 13px;
  font-weight: 800;
}

.daily-dev-issue-list {
  flex: 1;
  display: grid;
  gap: 8px;
  align-content: start;
}

.daily-dev-issue-card {
  display: grid;
  grid-template-columns: minmax(190px, 1fr) minmax(260px, 1.08fr);
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.94), rgba(248, 250, 252, 0.76)),
    radial-gradient(circle at right bottom, rgba(var(--primary-color-rgb), 0.06), transparent 42%);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.04);
}

.daily-dev-issue-card.critical {
  border-color: rgba(248, 113, 113, 0.28);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(255, 247, 237, 0.66)),
    radial-gradient(circle at right bottom, rgba(248, 113, 113, 0.08), transparent 46%);
}

.daily-dev-issue-card.watch {
  border-color: rgba(251, 146, 60, 0.22);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(255, 251, 235, 0.68)),
    radial-gradient(circle at right bottom, rgba(251, 146, 60, 0.08), transparent 46%);
}

.daily-dev-issue-main,
.daily-dev-issue-metric {
  min-width: 0;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.daily-dev-issue-main {
  display: grid;
  grid-template-columns: 26px minmax(0, 1fr);
  align-items: center;
  gap: 9px;
  padding: 0;
  text-align: left;
}

.daily-dev-issue-rank {
  display: inline-flex;
  width: 26px;
  height: 26px;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: rgba(var(--primary-color-rgb), 0.1);
  color: var(--primary);
  font-size: 12px;
  font-weight: 800;
}

.daily-dev-issue-copy {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.daily-dev-issue-url {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 800;
  color: var(--text);
}

.daily-dev-issue-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
}

.daily-dev-issue-metrics {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}

.daily-dev-issue-metric {
  display: grid;
  align-content: center;
  gap: 2px;
  min-height: 48px;
  padding: 7px 9px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 12px;
  text-align: left;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.daily-dev-issue-metric span,
.daily-dev-issue-metric small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  font-weight: 700;
  color: var(--muted);
}

.daily-dev-issue-metric strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 850;
  color: var(--text);
}

.daily-dev-issue-metric.danger strong {
  color: #ef4444;
}

.daily-dev-issue-metric.latency strong {
  color: #2563eb;
}

.daily-dev-issue-metric.slow strong {
  color: #f97316;
}

.daily-dev-issue-main:hover .daily-dev-issue-url,
.daily-dev-issue-metric:hover {
  color: var(--primary);
  border-color: rgba(var(--primary-color-rgb), 0.26);
  background: rgba(var(--primary-color-rgb), 0.06);
}

:global(body.dark-mode) .daily-dev-section {
  background:
    radial-gradient(circle at top right, rgba(239, 68, 68, 0.16), transparent 28%),
    radial-gradient(circle at bottom left, rgba(30, 123, 255, 0.16), transparent 34%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.94), rgba(15, 23, 42, 0.9));
}

:global(body.dark-mode) .daily-dev-card,
:global(body.dark-mode) .daily-dev-digest,
:global(body.dark-mode) .daily-dev-chart-card,
:global(body.dark-mode) .daily-dev-table-card {
  background: rgba(15, 23, 42, 0.78);
  box-shadow: 0 10px 20px rgba(2, 6, 23, 0.24);
}

:global(body.dark-mode) .daily-dev-status-strip {
  border-color: rgba(148, 163, 184, 0.2);
  background: linear-gradient(180deg, rgba(30, 41, 59, 0.76), rgba(15, 23, 42, 0.72));
}

:global(body.dark-mode) .daily-dev-status-strip:hover {
  border-color: rgba(var(--primary-color-rgb), 0.3);
}

:global(body.dark-mode) .daily-dev-status-strip.stable {
  border-color: rgba(34, 197, 94, 0.16);
  background: linear-gradient(180deg, rgba(20, 83, 45, 0.28), rgba(15, 23, 42, 0.72));
}

:global(body.dark-mode) .daily-dev-status-meta {
  color: rgba(226, 232, 240, 0.72);
}

:global(body.dark-mode) .daily-dev-status-meta.stable {
  color: #86efac;
}

:global(body.dark-mode) .daily-dev-digest {
  border-color: rgba(148, 163, 184, 0.18);
  color: rgba(241, 245, 249, 0.94);
  box-shadow: 0 12px 24px rgba(2, 6, 23, 0.26);
}

:global(body.dark-mode) .daily-dev-digest.critical {
  border-color: rgba(239, 68, 68, 0.3);
  background: linear-gradient(180deg, rgba(69, 10, 10, 0.82), rgba(15, 23, 42, 0.92));
}

:global(body.dark-mode) .daily-dev-digest.watch {
  border-color: rgba(245, 158, 11, 0.28);
  background: linear-gradient(180deg, rgba(120, 53, 15, 0.78), rgba(15, 23, 42, 0.92));
}

:global(body.dark-mode) .daily-dev-digest.stable {
  border-color: rgba(34, 197, 94, 0.24);
  background: linear-gradient(180deg, rgba(20, 83, 45, 0.76), rgba(15, 23, 42, 0.92));
}

:global(body.dark-mode) .daily-dev-digest-title {
  color: rgba(248, 250, 252, 0.96);
}

:global(body.dark-mode) .daily-dev-digest-line {
  color: rgba(226, 232, 240, 0.94);
}

:global(body.dark-mode) .daily-dev-digest-line::before {
  background: rgba(147, 197, 253, 0.86);
}

:global(body.dark-mode) .daily-dev-digest-line:hover {
  color: #93c5fd;
}

:global(body.dark-mode) .daily-dev-digest-badge.critical {
  color: #fecaca;
  background: rgba(127, 29, 29, 0.72);
}

:global(body.dark-mode) .daily-dev-digest-badge.watch {
  color: #fde68a;
  background: rgba(120, 53, 15, 0.72);
}

:global(body.dark-mode) .daily-dev-digest-badge.stable {
  color: #bbf7d0;
  background: rgba(20, 83, 45, 0.68);
}

:global(body.dark-mode) .daily-dev-subtitle,
:global(body.dark-mode) .daily-url-filter-status,
:global(body.dark-mode) .daily-dev-pill-label,
:global(body.dark-mode) .daily-dev-block-sub,
:global(body.dark-mode) .daily-dev-issue-meta,
:global(body.dark-mode) .daily-dev-issue-metric span,
:global(body.dark-mode) .daily-dev-issue-metric small {
  color: rgba(148, 163, 184, 0.92);
}

:global(body.dark-mode) .daily-dev-issue-card {
  border-color: rgba(148, 163, 184, 0.18);
  background:
    linear-gradient(135deg, rgba(15, 23, 42, 0.76), rgba(15, 23, 42, 0.58)),
    radial-gradient(circle at right bottom, rgba(var(--primary-color-rgb), 0.12), transparent 44%);
}

:global(body.dark-mode) .daily-dev-issue-card.critical {
  border-color: rgba(248, 113, 113, 0.28);
  background:
    linear-gradient(135deg, rgba(69, 10, 10, 0.36), rgba(15, 23, 42, 0.62)),
    radial-gradient(circle at right bottom, rgba(248, 113, 113, 0.1), transparent 46%);
}

:global(body.dark-mode) .daily-dev-issue-card.watch {
  border-color: rgba(251, 146, 60, 0.24);
  background:
    linear-gradient(135deg, rgba(67, 35, 12, 0.36), rgba(15, 23, 42, 0.62)),
    radial-gradient(circle at right bottom, rgba(251, 146, 60, 0.1), transparent 46%);
}

:global(body.dark-mode) .daily-dev-issue-url,
:global(body.dark-mode) .daily-dev-issue-metric strong {
  color: rgba(241, 245, 249, 0.96);
}

:global(body.dark-mode) .daily-dev-issue-metric.danger strong {
  color: #fca5a5;
}

:global(body.dark-mode) .daily-dev-issue-metric.latency strong {
  color: #93c5fd;
}

:global(body.dark-mode) .daily-dev-issue-metric.slow strong {
  color: #fdba74;
}

:global(body.dark-mode) .daily-dev-issue-rank {
  background: rgba(var(--primary-color-rgb), 0.18);
}

:global(body.dark-mode) .daily-dev-issue-metric {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.28);
}

:global(body.dark-mode) .daily-dev-issue-main:hover .daily-dev-issue-url,
:global(body.dark-mode) .daily-dev-issue-metric:hover {
  border-color: rgba(var(--primary-color-rgb), 0.32);
  background: rgba(var(--primary-color-rgb), 0.14);
}

:global(body.dark-mode) .daily-dev-pulse-day {
  border-color: rgba(148, 163, 184, 0.18);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.76), rgba(15, 23, 42, 0.58));
}

:global(body.dark-mode) .daily-dev-pulse-day.watch {
  border-color: rgba(251, 146, 60, 0.22);
  background: linear-gradient(180deg, rgba(67, 35, 12, 0.48), rgba(15, 23, 42, 0.62));
}

:global(body.dark-mode) .daily-dev-pulse-day.critical {
  border-color: rgba(248, 113, 113, 0.28);
  background: linear-gradient(180deg, rgba(69, 10, 10, 0.46), rgba(15, 23, 42, 0.62));
}

:global(body.dark-mode) .daily-dev-pulse-meter {
  background:
    linear-gradient(180deg, rgba(30, 41, 59, 0.24), rgba(15, 23, 42, 0.44)),
    radial-gradient(circle at bottom, rgba(var(--primary-color-rgb), 0.1), transparent 62%);
}

:global(body.dark-mode) .daily-dev-pulse-meta strong {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .daily-dev-pill {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.54);
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .daily-url-filter-row {
  border-color: rgba(148, 163, 184, 0.18);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.78), rgba(15, 23, 42, 0.58));
}

:global(body.dark-mode) .daily-url-filter-row.active {
  border-color: rgba(var(--primary-color-rgb), 0.28);
  box-shadow: 0 12px 24px rgba(2, 6, 23, 0.24);
}

:global(body.dark-mode) .daily-url-filter-label {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .daily-url-filter {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.62);
}

:global(body.dark-mode) .daily-url-filter:focus-within {
  border-color: rgba(var(--primary-color-rgb), 0.36);
  background: rgba(15, 23, 42, 0.82);
}

:global(body.dark-mode) .daily-empty-state {
  border-color: rgba(148, 163, 184, 0.26);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.12), transparent 56%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.68), rgba(15, 23, 42, 0.56));
  color: var(--text);
}

:global(body.dark-mode) .daily-empty-state-icon {
  color: #8bb8ff;
  background: rgba(var(--primary-color-rgb), 0.16);
}

:global(body.dark-mode) .daily-empty-state-title {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .daily-empty-state-text {
  color: rgba(148, 163, 184, 0.92);
}

.daily-mini-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.daily-mini-card {
  grid-column: span 3;
  background: linear-gradient(135deg, rgba(var(--primary-color-rgb), 0.12), rgba(var(--primary-color-rgb), 0.03));
  min-height: 140px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.daily-mini-card:nth-of-type(2) {
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.12), rgba(14, 165, 233, 0.03));
}

.daily-mini-title {
  font-weight: 600;
  font-size: 14px;
}

.daily-mini-body {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
}

.daily-mini-metric {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.daily-mini-label {
  font-size: 12px;
  color: var(--muted);
}

.daily-mini-value {
  font-size: 24px;
  font-weight: 700;
}

.daily-mini-meta {
  font-size: 12px;
  color: var(--muted);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.daily-trend-card {
  grid-column: span 6;
  min-height: 240px;
}

.daily-trend-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.daily-trend-title {
  font-weight: 600;
}

.daily-trend-sub {
  font-size: 12px;
  color: var(--muted);
}

.daily-trend-chart {
  height: 180px;
}

.daily-source-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
  position: relative;
  z-index: 6;
}

.daily-donut-card {
  grid-column: span 4;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.daily-donut {
  height: 220px;
}

.daily-summary-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.daily-summary-card {
  border-radius: var(--radius-md);
  padding: 14px;
  border: 1px solid var(--border);
  background: var(--panel-muted);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.daily-summary-card.search {
  border-color: rgba(30, 123, 255, 0.25);
}

.daily-summary-card.direct {
  border-color: rgba(245, 158, 11, 0.2);
}

.daily-summary-card.external {
  border-color: rgba(34, 197, 94, 0.2);
}

.daily-summary-label {
  font-size: 12px;
  color: var(--muted);
}

.daily-summary-value {
  font-size: 18px;
  font-weight: 700;
}

.daily-summary-rate {
  font-size: 12px;
  font-weight: 600;
}

.daily-table-card {
  grid-column: span 8;
  position: relative;
  z-index: 8;
}

.daily-tab-bar {
  display: inline-flex;
  gap: 8px;
  padding: 4px;
  background: var(--panel-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  margin-bottom: 12px;
}

.daily-tab {
  border: none;
  background: transparent;
  padding: 6px 14px;
  border-radius: var(--radius-pill);
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
  cursor: pointer;
}

.daily-tab.active {
  background: linear-gradient(135deg, var(--primary), var(--primary-strong));
  color: white;
}

.daily-source-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  margin-left: 10px;
}

.daily-source-filter-label {
  font-size: 12px;
  color: var(--muted);
  font-weight: 600;
}

.daily-source-link {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
  cursor: pointer;
  text-decoration: none;
  text-underline-offset: 4px;
}

.daily-source-link:hover {
  color: var(--text);
}

.daily-source-link.active {
  color: var(--primary);
  text-decoration: underline;
}

.daily-source-divider {
  font-size: 12px;
  color: var(--muted);
}

.daily-ip-cell {
  position: relative;
  overflow: visible !important;
}

.daily-ip-row {
  cursor: pointer;
}

.daily-ip-main {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  width: 100%;
}

.daily-ip-label {
  display: block;
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  letter-spacing: 0.01em;
}

.daily-ip-info {
  flex: 0 0 auto;
  margin-left: 6px;
  font-size: 12px;
  color: var(--muted);
}

.daily-ip-popover {
  position: absolute;
  left: 0;
  top: calc(100% + 8px);
  min-width: 240px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid rgba(var(--primary-color-rgb), 0.22);
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.98), rgba(244, 248, 255, 0.94));
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.12);
  opacity: 0;
  transform: translateY(6px);
  pointer-events: none;
  transition: opacity 0.18s ease, transform 0.18s ease;
  z-index: 80;
}

.daily-ip-popover-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 12px;
  color: var(--muted);
}

.daily-ip-popover-line + .daily-ip-popover-line {
  margin-top: 6px;
}

.daily-ip-popover-line strong {
  color: var(--text);
  font-weight: 600;
}

.daily-ip-cell:hover .daily-ip-popover,
.daily-ip-cell:focus-within .daily-ip-popover {
  opacity: 1;
  transform: translateY(0);
}

:global(body.dark-mode) .daily-ip-popover {
  border-color: rgba(148, 163, 184, 0.35);
  background: linear-gradient(160deg, rgba(17, 24, 39, 0.96), rgba(15, 23, 42, 0.94));
  box-shadow: 0 14px 30px rgba(2, 6, 23, 0.5);
}

.daily-summary-strip {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
}

.daily-dual-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.daily-section {
  grid-column: span 6;
}

.daily-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.daily-section-title {
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}

.daily-visitor-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.daily-visitor-grid .daily-donut {
  grid-column: span 4;
}

.daily-visitor-table {
  grid-column: span 8;
}

.daily-device-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.daily-device-left {
  grid-column: span 4;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.daily-device-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.daily-device-card {
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--panel-muted);
}

.daily-device-icon {
  font-size: 18px;
  color: var(--primary);
}

.daily-device-label {
  font-size: 12px;
  color: var(--muted);
}

.daily-device-value {
  font-size: 18px;
  font-weight: 700;
}

.daily-device-list {
  grid-column: span 4;
}

.daily-list-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

@media (max-width: 1200px) {
  .daily-dev-card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .daily-dev-grid {
    grid-template-columns: 1fr;
  }

  .daily-dev-pulse-board {
    grid-template-columns: repeat(7, minmax(96px, 1fr));
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .daily-dev-chart {
    min-height: 280px;
  }

  .daily-dev-issue-card {
    grid-template-columns: minmax(180px, 1fr) minmax(260px, 1.1fr);
  }

  .daily-mini-card {
    grid-column: span 6;
  }

  .daily-trend-card {
    grid-column: span 12;
  }

  .daily-donut-card {
    grid-column: span 12;
  }

  .daily-table-card {
    grid-column: span 12;
  }

  .daily-section {
    grid-column: span 12;
  }

  .daily-visitor-grid .daily-donut,
  .daily-visitor-table,
  .daily-device-left,
  .daily-device-list {
    grid-column: span 12;
  }

  .daily-device-cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .daily-dev-card-grid {
    grid-template-columns: 1fr;
  }

  .daily-dev-block-header,
  .daily-dev-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .daily-dev-status-strip {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .daily-dev-status-meta {
    margin-left: 22px;
    padding-left: 0;
  }

  .daily-dev-issue-card {
    grid-template-columns: 1fr;
  }

  .daily-dev-chart {
    min-height: 260px;
  }

  .daily-dev-issue-metrics {
    grid-template-columns: 1fr;
  }

  .daily-dev-issue-metric {
    min-height: auto;
  }

  .daily-url-filter-row,
  .daily-url-filter-copy,
  .daily-url-filter-controls,
  .daily-url-filter {
    width: 100%;
  }

  .daily-url-filter-row {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .daily-url-filter-status {
    white-space: normal;
  }

  .daily-date-control {
    width: 100%;
  }

  .daily-date-picker {
    flex: 1;
    min-width: 0;
    width: 100%;
  }

  .daily-mini-card {
    grid-column: span 12;
  }

  .daily-device-cards {
    grid-template-columns: 1fr;
  }

  .daily-summary-cards {
    grid-template-columns: 1fr;
  }
}
</style>
