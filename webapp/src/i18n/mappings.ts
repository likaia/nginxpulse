import type { AppLocale } from './index';

type TranslateFn = (key: string, params?: Record<string, unknown>) => string;

const labelKeyMap: Record<string, string> = {
  '未知': 'labels.unknown',
  unknown: 'labels.unknown',
  '待解析': 'labels.geoPending',
  '解析中': 'labels.geoPending',
  pending: 'labels.geoPending',
  resolving: 'labels.geoPending',
  '本地': 'labels.local',
  local: 'labels.local',
  '内网': 'labels.intranet',
  intranet: 'labels.intranet',
  '本地网络': 'labels.localNetwork',
  'local network': 'labels.localNetwork',
  '蜘蛛': 'labels.bot',
  bot: 'labels.bot',
  '未知浏览器': 'labels.unknownBrowser',
  'unknown browser': 'labels.unknownBrowser',
  '未知操作系统': 'labels.unknownOS',
  'unknown os': 'labels.unknownOS',
  '桌面设备': 'labels.desktopDevice',
  desktop: 'labels.desktopDevice',
  '手机': 'labels.mobileDevice',
  mobile: 'labels.mobileDevice',
  '平板': 'labels.tabletDevice',
  tablet: 'labels.tabletDevice',
  '苹果设备': 'labels.appleDevice',
  'apple device': 'labels.appleDevice',
  '其他设备': 'labels.otherDevice',
  'other device': 'labels.otherDevice',
};

const refererKeyMap: Record<string, string> = {
  '直接输入网址访问': 'labels.direct',
  direct: 'labels.direct',
  '站内访问': 'labels.internal',
  internal: 'labels.internal',
};

const directRefererAliases = new Set(['直接输入网址访问', 'direct', '-', '']);
const internalRefererAliases = new Set(['站内访问', 'internal']);

export const chinaProvinceMap: Record<string, string> = {
  北京: 'Beijing',
  天津: 'Tianjin',
  上海: 'Shanghai',
  重庆: 'Chongqing',
  河北: 'Hebei',
  山西: 'Shanxi',
  辽宁: 'Liaoning',
  吉林: 'Jilin',
  黑龙江: 'Heilongjiang',
  江苏: 'Jiangsu',
  浙江: 'Zhejiang',
  安徽: 'Anhui',
  福建: 'Fujian',
  江西: 'Jiangxi',
  山东: 'Shandong',
  河南: 'Henan',
  湖北: 'Hubei',
  湖南: 'Hunan',
  广东: 'Guangdong',
  海南: 'Hainan',
  四川: 'Sichuan',
  贵州: 'Guizhou',
  云南: 'Yunnan',
  陕西: 'Shaanxi',
  甘肃: 'Gansu',
  青海: 'Qinghai',
  台湾: 'Taiwan',
  内蒙古: 'Inner Mongolia',
  广西: 'Guangxi',
  西藏: 'Tibet',
  宁夏: 'Ningxia',
  新疆: 'Xinjiang',
  香港: 'Hong Kong',
  澳门: 'Macau',
};

export const chinaProvinceAlias: Record<string, string> = {
  北京市: '北京',
  天津市: '天津',
  上海市: '上海',
  重庆市: '重庆',
  河北省: '河北',
  山西省: '山西',
  辽宁省: '辽宁',
  吉林省: '吉林',
  黑龙江省: '黑龙江',
  江苏省: '江苏',
  浙江省: '浙江',
  安徽省: '安徽',
  福建省: '福建',
  江西省: '江西',
  山东省: '山东',
  河南省: '河南',
  湖北省: '湖北',
  湖南省: '湖南',
  广东省: '广东',
  海南省: '海南',
  四川省: '四川',
  贵州省: '贵州',
  云南省: '云南',
  陕西省: '陕西',
  甘肃省: '甘肃',
  青海省: '青海',
  台湾省: '台湾',
  内蒙古自治区: '内蒙古',
  广西壮族自治区: '广西',
  西藏自治区: '西藏',
  宁夏回族自治区: '宁夏',
  新疆维吾尔自治区: '新疆',
  香港特别行政区: '香港',
  澳门特别行政区: '澳门',
};

const chinaProvinceReverseMap: Record<string, string> = Object.entries(chinaProvinceMap).reduce(
  (acc, [zh, en]) => {
    acc[en.toLowerCase()] = zh;
    return acc;
  },
  {} as Record<string, string>
);

export function translateLabel(raw: string, t: TranslateFn) {
  const key = String(raw || '').trim();
  if (!key) {
    return raw;
  }
  const mappedKey = labelKeyMap[key] || labelKeyMap[key.toLowerCase()];
  return mappedKey ? t(mappedKey) : raw;
}

export function translateRefererLabel(raw: string, t: TranslateFn) {
  const key = String(raw || '').trim();
  if (key === '-' || key === '') {
    return t('labels.direct');
  }
  const mappedKey = refererKeyMap[key] || refererKeyMap[key.toLowerCase()];
  return mappedKey ? t(mappedKey) : raw;
}

export function isDirectReferer(value: string) {
  const normalized = String(value || '').trim().toLowerCase();
  if (internalRefererAliases.has(normalized)) {
    return true;
  }
  return directRefererAliases.has(normalized);
}

export function normalizeDeviceCategory(raw: string) {
  const value = String(raw || '').toLowerCase();
  if (value.includes('苹果') || value.includes('apple device')) {
    return 'apple';
  }
  if (value.includes('桌面') || value.includes('desktop') || value.includes('pc')) {
    return 'desktop';
  }
  if (value.includes('手机') || value.includes('移动') || value.includes('平板') || value.includes('mobile') || value.includes('tablet')) {
    return 'mobile';
  }
  return 'other';
}

export function translateLocationLabel(raw: string, locale: AppLocale, t: TranslateFn) {
  const trimmed = String(raw || '').trim();
  if (!trimmed) {
    return trimmed;
  }
  const translated = translateLabel(trimmed, t);
  if (translated !== trimmed) {
    return translated;
  }
  if (locale !== 'en-US') {
    return trimmed;
  }
  if (trimmed === '中国') {
    return 'China';
  }
  const parts = trimmed.split('·').map((part) => part.trim());
  const mapped = parts.map((part) => {
    const aliased = chinaProvinceAlias[part] || part;
    return chinaProvinceMap[aliased] || part;
  });
  return mapped.join(' · ');
}

export function normalizeChinaProvinceName(raw: string) {
  const trimmed = String(raw || '').trim();
  if (!trimmed) {
    return trimmed;
  }
  const alias = chinaProvinceAlias[trimmed] || trimmed;
  const englishLookup = chinaProvinceReverseMap[trimmed.toLowerCase()];
  return englishLookup || alias;
}

export function formatRefererLabel(raw: string, locale: AppLocale, t: TranslateFn) {
  if (locale === 'zh-CN') {
    return translateRefererLabel(raw, t);
  }
  return translateRefererLabel(raw, t);
}

export function formatDeviceLabel(raw: string, t: TranslateFn) {
  return translateLabel(raw, t);
}

export function formatBrowserLabel(raw: string, t: TranslateFn) {
  return translateLabel(raw, t);
}

export function formatOSLabel(raw: string, t: TranslateFn) {
  return translateLabel(raw, t);
}

export function formatLocationLabel(raw: string, locale: AppLocale, t: TranslateFn) {
  return translateLocationLabel(raw, locale, t);
}
