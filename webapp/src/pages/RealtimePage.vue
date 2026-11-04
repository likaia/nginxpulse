<template>
  <header class="page-header">
    <div class="page-title">
      <span class="title-chip">{{ t('realtime.title') }}</span>
      <p class="title-sub">{{ t('realtime.subtitle') }}</p>
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
              id="realtime-website-selector"
              label=""
            />
          </div>
          <div class="realtime-range toolbar-pill">
            <button
              v-for="option in windowOptions"
              :key="option"
              class="realtime-range-btn"
              :class="{ active: currentWindow === option }"
              @click="setWindow(option)"
            >
              {{ t('realtime.minutes', { value: option }) }}
            </button>
          </div>
        </template>
        <template #utility>
          <SystemNotifications />
          <ThemeToggle />
        </template>
      </HeaderToolbar>
    </div>
  </header>

  <section class="realtime-grid">
    <div class="card realtime-card realtime-overview">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon blue"><i class="ri-radar-line"></i></span>
          <span>{{ activeTitle }}</span>
        </div>
      </div>
      <template v-if="hasRealtimeOverviewData">
        <div class="realtime-metric">
          <div class="realtime-value">{{ formatCount(activeCount) }}</div>
          <div class="realtime-mini-bars">
            <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
          </div>
        </div>
        <div class="realtime-subtitle">{{ deviceSubtitle }}</div>
        <div class="realtime-device-cards">
          <div class="realtime-device-card">
            <div class="realtime-device-icon"><i class="ri-computer-line"></i></div>
            <div class="realtime-device-label">{{ t('realtime.pc') }}</div>
            <div class="realtime-device-count">{{ formatCount(deviceStats.pc.count) }}</div>
            <div class="realtime-device-rate">{{ formatPercent(deviceStats.pc.percent) }}</div>
          </div>
          <div class="realtime-device-card">
            <div class="realtime-device-icon"><i class="ri-smartphone-line"></i></div>
            <div class="realtime-device-label">{{ t('realtime.mobile') }}</div>
            <div class="realtime-device-count">{{ formatCount(deviceStats.mobile.count) }}</div>
            <div class="realtime-device-rate">{{ formatPercent(deviceStats.mobile.percent) }}</div>
          </div>
          <div class="realtime-device-card">
            <div class="realtime-device-icon"><i class="ri-apple-line"></i></div>
            <div class="realtime-device-label">{{ t('overview.deviceApple') }}</div>
            <div class="realtime-device-count">{{ formatCount(deviceStats.apple.count) }}</div>
            <div class="realtime-device-rate">{{ formatPercent(deviceStats.apple.percent) }}</div>
          </div>
          <div class="realtime-device-card">
            <div class="realtime-device-icon"><i class="ri-shield-line"></i></div>
            <div class="realtime-device-label">{{ t('realtime.other') }}</div>
            <div class="realtime-device-count">{{ formatCount(deviceStats.other.count) }}</div>
            <div class="realtime-device-rate">{{ formatPercent(deviceStats.other.percent) }}</div>
          </div>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-radar-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.overviewEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.overviewEmptyText') }}</div>
      </div>
    </div>

    <div class="card realtime-card">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon blue"><i class="ri-compass-3-line"></i></span>
          {{ t('realtime.referer') }}
        </div>
        <button
          type="button"
          class="realtime-sort-btn"
          :class="{ asc: sortOrders.referer === 'asc' }"
          :aria-label="getSortAriaLabel('referer')"
          :title="getSortAriaLabel('referer')"
          @click="toggleSortOrder('referer')"
        >
          <span class="realtime-sort-label">{{ getSortLabel(sortOrders.referer) }}</span>
          <span class="realtime-sort-icons" aria-hidden="true">
            <i class="ri-arrow-up-s-line"></i>
            <i class="ri-arrow-down-s-line"></i>
          </span>
        </button>
      </div>
      <template v-if="hasRefererData">
        <div class="realtime-top">
          <span class="realtime-rank" :class="{ asc: sortOrders.referer === 'asc' }">{{ getRankLabel(sortOrders.referer) }}</span>
          <div class="realtime-top-title">{{ topReferer.name }}</div>
          <div class="realtime-top-meta">
            <span class="realtime-top-count">{{ formatCount(topReferer.count) }}</span>
            <span class="realtime-top-rate">{{ formatPercent(topReferer.percent) }}</span>
          </div>
        </div>
        <div class="realtime-mini-bars">
          <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table realtime-table">
            <thead>
              <tr>
                <th>{{ t('realtime.referer') }}</th>
                <th class="realtime-count-col">{{ t('realtime.topVisitors') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in refererItems" :key="item.name">
                <td :title="item.name">{{ item.name }}</td>
                <td class="realtime-count-col">{{ formatCount(item.count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-compass-3-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.refererEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.refererEmptyText') }}</div>
      </div>
    </div>

    <div class="card realtime-card">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon orange"><i class="ri-pages-line"></i></span>
          {{ t('realtime.pages') }}
        </div>
        <button
          type="button"
          class="realtime-sort-btn"
          :class="{ asc: sortOrders.page === 'asc' }"
          :aria-label="getSortAriaLabel('page')"
          :title="getSortAriaLabel('page')"
          @click="toggleSortOrder('page')"
        >
          <span class="realtime-sort-label">{{ getSortLabel(sortOrders.page) }}</span>
          <span class="realtime-sort-icons" aria-hidden="true">
            <i class="ri-arrow-up-s-line"></i>
            <i class="ri-arrow-down-s-line"></i>
          </span>
        </button>
      </div>
      <template v-if="hasPageData">
        <div class="realtime-top">
          <span class="realtime-rank" :class="{ asc: sortOrders.page === 'asc' }">{{ getRankLabel(sortOrders.page) }}</span>
          <div class="realtime-top-title">{{ topPage.name }}</div>
          <div class="realtime-top-meta">
            <span class="realtime-top-count">{{ formatCount(topPage.count) }}</span>
            <span class="realtime-top-rate">{{ formatPercent(topPage.percent) }}</span>
          </div>
        </div>
        <div class="realtime-mini-bars">
          <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table realtime-table">
            <thead>
              <tr>
                <th>{{ t('realtime.pages') }}</th>
                <th class="realtime-count-col">{{ t('realtime.viewCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in pageItems" :key="item.name">
                <td :title="item.name">{{ item.name }}</td>
                <td class="realtime-count-col">{{ formatCount(item.count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-pages-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.pagesEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.pagesEmptyText') }}</div>
      </div>
    </div>

    <div class="card realtime-card">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon orange"><i class="ri-login-circle-line"></i></span>
          {{ t('realtime.entryPages') }}
        </div>
        <button
          type="button"
          class="realtime-sort-btn"
          :class="{ asc: sortOrders.entry === 'asc' }"
          :aria-label="getSortAriaLabel('entry')"
          :title="getSortAriaLabel('entry')"
          @click="toggleSortOrder('entry')"
        >
          <span class="realtime-sort-label">{{ getSortLabel(sortOrders.entry) }}</span>
          <span class="realtime-sort-icons" aria-hidden="true">
            <i class="ri-arrow-up-s-line"></i>
            <i class="ri-arrow-down-s-line"></i>
          </span>
        </button>
      </div>
      <template v-if="hasEntryData">
        <div class="realtime-top">
          <span class="realtime-rank" :class="{ asc: sortOrders.entry === 'asc' }">{{ getRankLabel(sortOrders.entry) }}</span>
          <div class="realtime-top-title">{{ topEntry.name }}</div>
          <div class="realtime-top-meta">
            <span class="realtime-top-count">{{ formatCount(topEntry.count) }}</span>
            <span class="realtime-top-rate">{{ formatPercent(topEntry.percent) }}</span>
          </div>
        </div>
        <div class="realtime-mini-bars">
          <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table realtime-table">
            <thead>
              <tr>
                <th>{{ t('realtime.entryPages') }}</th>
                <th class="realtime-count-col">{{ t('realtime.entryCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in entryItems" :key="item.name">
                <td :title="item.name">{{ item.name }}</td>
                <td class="realtime-count-col">{{ formatCount(item.count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-login-circle-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.entryEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.entryEmptyText') }}</div>
      </div>
    </div>

    <div class="card realtime-card">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon green"><i class="ri-global-line"></i></span>
          {{ t('realtime.browser') }}
        </div>
        <button
          type="button"
          class="realtime-sort-btn"
          :class="{ asc: sortOrders.browser === 'asc' }"
          :aria-label="getSortAriaLabel('browser')"
          :title="getSortAriaLabel('browser')"
          @click="toggleSortOrder('browser')"
        >
          <span class="realtime-sort-label">{{ getSortLabel(sortOrders.browser) }}</span>
          <span class="realtime-sort-icons" aria-hidden="true">
            <i class="ri-arrow-up-s-line"></i>
            <i class="ri-arrow-down-s-line"></i>
          </span>
        </button>
      </div>
      <template v-if="hasBrowserData">
        <div class="realtime-top">
          <span class="realtime-rank" :class="{ asc: sortOrders.browser === 'asc' }">{{ getRankLabel(sortOrders.browser) }}</span>
          <div class="realtime-top-title">{{ topBrowser.name }}</div>
          <div class="realtime-top-meta">
            <span class="realtime-top-count">{{ formatCount(topBrowser.count) }}</span>
            <span class="realtime-top-rate">{{ formatPercent(topBrowser.percent) }}</span>
          </div>
        </div>
        <div class="realtime-mini-bars">
          <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table realtime-table">
            <thead>
              <tr>
                <th>{{ t('realtime.browser') }}</th>
                <th class="realtime-count-col">{{ t('realtime.topVisitors') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in browserItems" :key="item.name">
                <td>{{ item.name }}</td>
                <td class="realtime-count-col">{{ formatCount(item.count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-global-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.browserEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.browserEmptyText') }}</div>
      </div>
    </div>

    <div class="card realtime-card">
      <div class="realtime-card-header">
        <div class="realtime-card-title">
          <span class="section-icon blue"><i class="ri-map-pin-2-line"></i></span>
          {{ t('realtime.location') }}
        </div>
        <button
          type="button"
          class="realtime-sort-btn"
          :class="{ asc: sortOrders.city === 'asc' }"
          :aria-label="getSortAriaLabel('city')"
          :title="getSortAriaLabel('city')"
          @click="toggleSortOrder('city')"
        >
          <span class="realtime-sort-label">{{ getSortLabel(sortOrders.city) }}</span>
          <span class="realtime-sort-icons" aria-hidden="true">
            <i class="ri-arrow-up-s-line"></i>
            <i class="ri-arrow-down-s-line"></i>
          </span>
        </button>
      </div>
      <template v-if="hasCityData">
        <div class="realtime-top">
          <span class="realtime-rank" :class="{ asc: sortOrders.city === 'asc' }">{{ getRankLabel(sortOrders.city) }}</span>
          <div class="realtime-top-title">{{ topCity.name }}</div>
          <div class="realtime-top-meta">
            <span class="realtime-top-count">{{ formatCount(topCity.count) }}</span>
            <span class="realtime-top-rate">{{ formatPercent(topCity.percent) }}</span>
          </div>
        </div>
        <div class="realtime-mini-bars">
          <span v-for="(bar, index) in activeBars" :key="index" :class="{ active: bar.active }" :style="{ height: `${bar.height}px` }"></span>
        </div>
        <div class="table-wrapper">
          <table class="ranking-table realtime-table">
            <thead>
              <tr>
                <th>{{ t('realtime.location') }}</th>
                <th class="realtime-count-col">{{ t('realtime.topVisitors') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in cityItems" :key="item.name">
                <td>{{ item.name }}</td>
                <td class="realtime-count-col">{{ formatCount(item.count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div v-else class="realtime-empty-state">
        <span class="realtime-empty-state-icon"><i class="ri-map-pin-2-line"></i></span>
        <div class="realtime-empty-state-title">{{ t('realtime.locationEmptyTitle') }}</div>
        <div class="realtime-empty-state-text">{{ t('realtime.locationEmptyText') }}</div>
      </div>
    </div>
  </section>

  <ParsingOverlay :website-id="currentWebsiteId" @finished="loadRealtime" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { fetchRealtimeStats, fetchWebsites } from '@/api';
import type { RealtimeSeriesItem, RealtimeStats, WebsiteInfo } from '@/api/types';
import { formatBrowserLabel, formatLocationLabel, formatRefererLabel, normalizeDeviceCategory } from '@/i18n/mappings';
import { normalizeLocale } from '@/i18n';
import { getUserPreference, saveUserPreference } from '@/utils';
import ParsingOverlay from '@/components/ParsingOverlay.vue';
import HeaderToolbar from '@/components/HeaderToolbar.vue';
import SystemNotifications from '@/components/SystemNotifications.vue';
import ThemeToggle from '@/components/ThemeToggle.vue';
import WebsiteSelect from '@/components/WebsiteSelect.vue';

type SortOrder = 'desc' | 'asc';
type SortableCardKey = 'referer' | 'page' | 'entry' | 'browser' | 'city';

const websites = ref<WebsiteInfo[]>([]);
const websitesLoading = ref(true);
const currentWebsiteId = ref('');
const windowOptions = [5, 15, 30];
const currentWindow = ref(30);

const activeCount = ref(0);
const activeSeries = ref<number[]>([]);
const deviceBreakdown = ref<RealtimeSeriesItem[]>([]);
const referers = ref<RealtimeSeriesItem[]>([]);
const pages = ref<RealtimeSeriesItem[]>([]);
const entryPages = ref<RealtimeSeriesItem[]>([]);
const browsers = ref<RealtimeSeriesItem[]>([]);
const locations = ref<RealtimeSeriesItem[]>([]);

let refreshTimer: number | null = null;

const { t, n, locale } = useI18n({ useScope: 'global' });
const currentLocale = computed(() => normalizeLocale(locale.value));
const sortOrders = ref<Record<SortableCardKey, SortOrder>>({
  referer: 'desc',
  page: 'desc',
  entry: 'desc',
  browser: 'desc',
  city: 'desc',
});

const activeTitle = computed(() => t('realtime.activeTitle', { value: currentWindow.value }));
const deviceSubtitle = computed(() => t('realtime.deviceSubtitle', { value: currentWindow.value }));

const activeBars = computed(() => {
  const values = Array.isArray(activeSeries.value) && activeSeries.value.length
    ? activeSeries.value
    : new Array(currentWindow.value).fill(0);
  const maxVal = Math.max(1, ...values);
  return values.map((value) => {
    const ratio = value / maxVal;
    return {
      height: Math.max(6, Math.round(ratio * 24)),
      active: value > 0,
    }
  });
});

const deviceStats = computed(() => {
  const breakdown = deviceBreakdown.value || [];
  const totals = { desktop: 0, mobile: 0, apple: 0, other: 0 };
  breakdown.forEach((item) => {
    const category = normalizeDeviceCategory(item.name);
    totals[category] += item.count || 0;
  });
  const total = totals.desktop + totals.mobile + totals.apple + totals.other;
  return {
    pc: { count: totals.desktop, percent: total ? totals.desktop / total : 0 },
    mobile: { count: totals.mobile, percent: total ? totals.mobile / total : 0 },
    apple: { count: totals.apple, percent: total ? totals.apple / total : 0 },
    other: { count: totals.other, percent: total ? totals.other / total : 0 },
  };
});

const refererItemsRaw = computed(() =>
  (referers.value || []).map((item) => ({
    ...item,
    name: formatRefererLabel(item.name, currentLocale.value, t),
  }))
);
const pageItemsRaw = computed(() => pages.value || []);
const entryItemsRaw = computed(() => entryPages.value || []);
const browserItemsRaw = computed(() =>
  (browsers.value || []).map((item) => ({
    ...item,
    name: formatBrowserLabel(item.name, t),
  }))
);
const cityItemsRaw = computed(() =>
  (locations.value || []).map((item) => ({
    ...item,
    name: formatLocationLabel(item.name, currentLocale.value, t),
  }))
);

const refererItems = computed(() => sortItemsByCount(refererItemsRaw.value, sortOrders.value.referer));
const pageItems = computed(() => sortItemsByCount(pageItemsRaw.value, sortOrders.value.page));
const entryItems = computed(() => sortItemsByCount(entryItemsRaw.value, sortOrders.value.entry));
const browserItems = computed(() => sortItemsByCount(browserItemsRaw.value, sortOrders.value.browser));
const cityItems = computed(() => sortItemsByCount(cityItemsRaw.value, sortOrders.value.city));

const topReferer = computed(() => getTopItem(refererItems.value));
const topPage = computed(() => getTopItem(pageItems.value));
const topEntry = computed(() => getTopItem(entryItems.value));
const topBrowser = computed(() => getTopItem(browserItems.value));
const topCity = computed(() => getTopItem(cityItems.value));
const hasRealtimeOverviewData = computed(() =>
  activeCount.value > 0 ||
  activeSeries.value.some((value) => Number(value || 0) > 0) ||
  deviceStats.value.pc.count + deviceStats.value.mobile.count + deviceStats.value.apple.count + deviceStats.value.other.count > 0
);
const hasRefererData = computed(() => refererItems.value.length > 0);
const hasPageData = computed(() => pageItems.value.length > 0);
const hasEntryData = computed(() => entryItems.value.length > 0);
const hasBrowserData = computed(() => browserItems.value.length > 0);
const hasCityData = computed(() => cityItems.value.length > 0);

onMounted(() => {
  initWindowFromPreference();
  loadWebsites();
});

onBeforeUnmount(() => {
  stopAutoRefresh();
});

watch(currentWebsiteId, (value) => {
  if (value) {
    saveUserPreference('selectedWebsite', value);
  }
  loadRealtime();
  restartAutoRefresh();
});

watch(currentWindow, (value) => {
  saveUserPreference('realtimeWindow', String(value));
  loadRealtime();
  restartAutoRefresh();
});

function initWindowFromPreference() {
  const queryWindow = getWindowFromQuery();
  const savedWindow = parseInt(getUserPreference('realtimeWindow', '30'), 10);
  const preferred = Number.isFinite(queryWindow) ? queryWindow : savedWindow;
  if (windowOptions.includes(preferred)) {
    currentWindow.value = preferred;
  }
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

async function loadRealtime() {
  if (!currentWebsiteId.value) {
    return;
  }
  try {
    const data: RealtimeStats = await fetchRealtimeStats(currentWebsiteId.value, currentWindow.value);
    activeCount.value = data.activeCount || 0;
    activeSeries.value = data.activeSeries || [];
    deviceBreakdown.value = data.deviceBreakdown || [];
    referers.value = data.referers || [];
    pages.value = data.pages || [];
    entryPages.value = data.entryPages || [];
    browsers.value = data.browsers || [];
    locations.value = data.locations || [];
  } catch (error) {
    console.error('加载实时数据失败:', error);
  }
}

function startAutoRefresh() {
  if (refreshTimer) {
    return;
  }
  refreshTimer = window.setInterval(() => {
    loadRealtime();
  }, 30000);
}

function stopAutoRefresh() {
  if (!refreshTimer) {
    return;
  }
  window.clearInterval(refreshTimer);
  refreshTimer = null;
}

function restartAutoRefresh() {
  stopAutoRefresh();
  if (currentWebsiteId.value) {
    startAutoRefresh();
  }
}

function setWindow(value: number) {
  if (currentWindow.value === value) {
    return;
  }
  currentWindow.value = value;
}

function toggleSortOrder(key: SortableCardKey) {
  sortOrders.value[key] = sortOrders.value[key] === 'desc' ? 'asc' : 'desc';
}

function getSortLabel(order: SortOrder) {
  return order === 'desc' ? t('realtime.sortDesc') : t('realtime.sortAsc');
}

function getSortFieldLabel(key: SortableCardKey) {
  switch (key) {
    case 'referer':
      return t('realtime.referer');
    case 'page':
      return t('realtime.pages');
    case 'entry':
      return t('realtime.entryPages');
    case 'browser':
      return t('realtime.browser');
    default:
      return t('realtime.location');
  }
}

function getSortAriaLabel(key: SortableCardKey) {
  return t('realtime.sortActionLabel', {
    field: getSortFieldLabel(key),
    order: getSortLabel(sortOrders.value[key]),
  });
}

function getRankLabel(order: SortOrder) {
  return order === 'desc' ? t('realtime.rankTop') : t('realtime.rankBottom');
}

function getWindowFromQuery() {
  const params = new URLSearchParams(window.location.search || '');
  const raw = params.get('window');
  if (!raw) {
    return Number.NaN;
  }
  const parsed = parseInt(raw, 10);
  return Number.isFinite(parsed) ? parsed : Number.NaN;
}

function getTopItem(items: RealtimeSeriesItem[] = []) {
  if (!items.length) {
    return { name: '-', count: 0, percent: 0 };
  }
  return items[0];
}

function sortItemsByCount(items: RealtimeSeriesItem[] = [], order: SortOrder = 'desc') {
  const direction = order === 'asc' ? 1 : -1;
  return [...items].sort((a, b) => {
    if (a.count !== b.count) {
      return (a.count - b.count) * direction;
    }
    if (a.percent !== b.percent) {
      return (a.percent - b.percent) * direction;
    }
    return String(a.name || '').localeCompare(String(b.name || ''), currentLocale.value);
  });
}

function formatCount(value: number) { return n(Number(value || 0)); }

function formatPercent(value: number) { return n(Number(value || 0), 'percent'); }
</script>

<style scoped lang="scss">
:global(.realtime-page) {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.realtime-range {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.realtime-range-btn {
  border: none;
  background: transparent;
  padding: 6px 14px;
  border-radius: var(--radius-pill);
  font-weight: 600;
  font-size: 12px;
  color: var(--muted);
  cursor: pointer;
}

.realtime-range-btn.active {
  background: linear-gradient(135deg, var(--primary), var(--primary-strong));
  color: white;
  box-shadow: 0 8px 16px rgba(var(--primary-color-rgb), 0.28);
}

.realtime-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
}

.realtime-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 260px;
}

.realtime-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.realtime-card-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 700;
}

.realtime-sort-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--border);
  background: var(--panel-muted);
  border-radius: var(--radius-pill);
  padding: 4px 10px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.realtime-sort-btn:hover {
  border-color: rgba(var(--primary-color-rgb), 0.42);
  color: var(--text);
}

.realtime-sort-btn:focus-visible {
  outline: none;
  border-color: rgba(var(--primary-color-rgb), 0.55);
  box-shadow: 0 0 0 3px rgba(var(--primary-color-rgb), 0.12);
}

.realtime-sort-btn.asc {
  border-color: rgba(var(--primary-color-rgb), 0.42);
  background: rgba(var(--primary-color-rgb), 0.12);
  color: var(--primary);
}

.realtime-sort-label {
  min-width: 2.6em;
  text-align: left;
}

.realtime-sort-icons {
  display: inline-flex;
  flex-direction: column;
  line-height: 0.72;
}

.realtime-sort-icons i {
  font-size: 12px;
  color: rgba(var(--primary-color-rgb), 0.35);
}

.realtime-sort-btn .realtime-sort-icons i:last-child {
  color: var(--primary);
}

.realtime-sort-btn.asc .realtime-sort-icons i:first-child {
  color: var(--primary);
}

.realtime-sort-btn.asc .realtime-sort-icons i:last-child {
  color: rgba(var(--primary-color-rgb), 0.35);
}

.realtime-metric {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.realtime-value {
  font-size: 28px;
  font-weight: 700;
}

.realtime-mini-bars {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  gap: 4px;
  height: 28px;
  align-items: end;
}

.realtime-mini-bars span {
  display: block;
  width: 100%;
  min-height: 6px;
  border-radius: var(--radius-2xs);
  background: var(--panel-muted);
}

.realtime-mini-bars span.active {
  background: linear-gradient(180deg, rgba(30, 123, 255, 0.65), rgba(30, 123, 255, 0.15));
}

.realtime-subtitle {
  font-size: 12px;
  color: var(--muted);
}

.realtime-device-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.realtime-device-card {
  background: var(--panel-muted);
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

:global(body.dark-mode) .realtime-device-card {
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.04), var(--panel-muted));
}

.realtime-device-icon {
  font-size: 18px;
  color: var(--primary);
}

.realtime-device-label {
  font-size: 12px;
  color: var(--muted);
}

.realtime-device-count {
  font-size: 18px;
  font-weight: 700;
}

.realtime-device-rate {
  font-size: 12px;
  color: var(--muted);
}

.realtime-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.realtime-rank {
  padding: 4px 8px;
  border-radius: var(--radius-pill);
  background: rgba(245, 158, 11, 0.18);
  color: var(--accent);
  font-size: 11px;
  font-weight: 700;
}

.realtime-rank.asc {
  background: rgba(var(--primary-color-rgb), 0.18);
  color: var(--primary);
}

.realtime-top-title {
  flex: 1;
  font-weight: 600;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.realtime-top-meta {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.realtime-top-count {
  font-size: 20px;
  font-weight: 700;
}

.realtime-top-rate {
  font-size: 12px;
  color: var(--muted);
}

.realtime-table th,
.realtime-table td {
  padding: 10px 12px;
}

.realtime-count-col {
  text-align: right;
}

.realtime-empty-row {
  background: transparent !important;
  box-shadow: none !important;
  border: none !important;
}

.realtime-empty-row td {
  background: transparent !important;
}

.realtime-empty-state {
  flex: 1;
  min-height: 180px;
  border-radius: var(--radius-lg);
  border: 1px dashed rgba(var(--primary-color-rgb), 0.18);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.08), transparent 58%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(247, 250, 255, 0.86));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
  text-align: center;
}

.realtime-empty-state-icon {
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

.realtime-empty-state-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.realtime-empty-state-text {
  max-width: 240px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--muted);
}

:global(body.dark-mode) .realtime-empty-state {
  border-color: rgba(148, 163, 184, 0.26);
  background:
    radial-gradient(circle at top, rgba(var(--primary-color-rgb), 0.12), transparent 56%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.68), rgba(15, 23, 42, 0.56));
  color: var(--text);
}

:global(body.dark-mode) .realtime-empty-state-icon {
  color: #8bb8ff;
  background: rgba(var(--primary-color-rgb), 0.16);
}

:global(body.dark-mode) .realtime-empty-state-title {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .realtime-empty-state-text {
  color: rgba(148, 163, 184, 0.92);
}

@media (max-width: 1200px) {
  .realtime-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .realtime-grid {
    grid-template-columns: 1fr;
  }

  .realtime-device-cards {
    grid-template-columns: 1fr;
  }
}
</style>
