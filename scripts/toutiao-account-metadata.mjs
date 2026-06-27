#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const profileDir = path.resolve(required(args, 'profile-dir'));
const metadataUrl = args['metadata-url'] || 'https://mp.toutiao.com/profile_v4/';
const chromePath = args['chrome-path'] || undefined;
const outputFile = args.output ? path.resolve(args.output) : '';
const debugDir = args['debug-dir'] ? path.resolve(args['debug-dir']) : '';
const timeoutMs = Number(args['timeout-ms'] || 45000);
const settleMs = Number(args['settle-ms'] || 3000);
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

const endpointMatchers = {
  userInfo: /\/mp\/agw\/creator_center\/user_info\b/,
  homeMerge: /\/mp\/fe_api\/home\/merge_v2\b/,
  worksList: /\/api\/feed\/mp_provider\/v1\/\b/,
};

let context;
try {
  const playwright = await importPlaywright();
  await mkdir(profileDir, { recursive: true });
  context = await playwright.chromium.launchPersistentContext(profileDir, {
    executablePath: chromePath,
    headless,
    viewport: { width: 1440, height: 960 },
    locale: 'zh-CN',
    args: chromiumArgs,
  });

  const page = context.pages()[0] ?? (await context.newPage());
  const result = await collectMetadata(page);
  await writeResult(result);
  process.stdout.write(`${JSON.stringify(result)}\n`);
  if (!result.ok) {
    process.exitCode = 2;
  }
} catch (error) {
  const result = {
    ok: false,
    platform: 'toutiao',
    status: isProfileInUseError(error) ? 'profile_in_use' : 'metadata_collection_failed',
    profileDir,
    metadataUrl,
    capturedAt: new Date().toISOString(),
    error: error instanceof Error ? error.message : String(error),
  };
  await writeResult(result).catch(() => undefined);
  process.stderr.write(`${JSON.stringify(result)}\n`);
  process.exitCode = 1;
} finally {
  await context?.close().catch(() => undefined);
}

async function collectMetadata(page) {
  const collector = createResponseCollector(page);
  await page.goto(metadataUrl, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(settleMs);

  const loginState = await getLoginState(page);
  if (!loginState.loggedIn) {
    const result = {
      ok: false,
      platform: 'toutiao',
      status: 'not_logged_in',
      profileDir,
      metadataUrl,
      pageUrl: page.url(),
      title: await safeTitle(page),
      capturedAt: new Date().toISOString(),
      loginState,
      diagnostics: collector.diagnostics(),
    };
    await writeDebug(page, result, collector);
    return result;
  }

  await clickFirstVisible(page, ['作品管理', '主页', '数据', '粉丝数据']);
  await page.waitForTimeout(Math.min(settleMs, 2500));
  await waitForSelectedEndpoints(collector, 10000);

  const userInfo = collector.latest('userInfo')?.json || {};
  const homeMerge = collector.latest('homeMerge')?.json || {};
  const worksList = collector.latest('worksList')?.json || {};
  const visible = await collectVisibleState(page);
  const metadata = buildMetadata(userInfo, homeMerge, worksList, visible);
  const ok = Boolean(metadata.displayName || metadata.userId || metadata.mediaId);
  const result = {
    ok,
    platform: 'toutiao',
    status: ok ? 'collected' : 'account_metadata_not_found',
    profileDir,
    metadataUrl,
    pageUrl: page.url(),
    title: await safeTitle(page),
    capturedAt: new Date().toISOString(),
    dataSource: 'browser_context_request',
    selectors: {
      profileShellText: 'body text contains 头条号/创作/作品管理',
      accountLink: 'a[href*="toutiao.com/c/user/"]',
      observedEndpoints: Object.fromEntries(Object.entries(endpointMatchers).map(([key, value]) => [key, String(value)])),
    },
    loginState,
    metadata,
    diagnostics: collector.diagnostics(),
  };
  await writeDebug(page, result, collector);
  return result;
}

function createResponseCollector(page) {
  const events = [];
  page.on('response', async (response) => {
    const url = response.url();
    if (!/https:\/\/[^/]*toutiao\.com\//i.test(url)) {
      return;
    }
    const key = endpointKey(url);
    if (!key) {
      return;
    }
    const item = {
      key,
      url,
      status: response.status(),
      method: response.request().method(),
      capturedAt: new Date().toISOString(),
      json: null,
      bodySample: '',
    };
    try {
      const text = await response.text();
      item.bodySample = sanitizeText(text).slice(0, 2400);
      item.json = sanitizeValue(JSON.parse(text));
    } catch {
      // Some responses are streaming or compressed in a way Playwright cannot expose here.
    }
    events.push(item);
  });
  return {
    latest(key) {
      for (let index = events.length - 1; index >= 0; index -= 1) {
        if (events[index].key === key) {
          return events[index];
        }
      }
      return null;
    },
    diagnostics() {
      return {
        observedEndpoints: events.map((event) => ({
          key: event.key,
          url: event.url,
          status: event.status,
          method: event.method,
          capturedAt: event.capturedAt,
        })),
      };
    },
    debugEvents() {
      return events.map((event) => ({
        key: event.key,
        url: event.url,
        status: event.status,
        method: event.method,
        capturedAt: event.capturedAt,
        json: event.json,
        bodySample: event.bodySample,
      }));
    },
  };
}

function endpointKey(url) {
  for (const [key, matcher] of Object.entries(endpointMatchers)) {
    if (matcher.test(url)) {
      return key;
    }
  }
  return '';
}

async function waitForSelectedEndpoints(collector, timeoutMsValue) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMsValue) {
    if (collector.latest('userInfo') && collector.latest('homeMerge')) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 300));
  }
}

async function clickFirstVisible(page, labels) {
  for (const label of labels) {
    const locator = page.getByText(label, { exact: false }).first();
    if (await locator.isVisible({ timeout: 800 }).catch(() => false)) {
      await locator.click({ timeout: 1500 }).catch(() => undefined);
      return label;
    }
  }
  return '';
}

async function collectVisibleState(page) {
  return page.evaluate(() => {
    const bodyText = textOf(document.body).slice(0, 6000);
    const accountLink = Array.from(document.querySelectorAll('a[href*="toutiao.com/c/user/"]')).map((link) => link.href).find(Boolean) || '';
    const avatarUrl = Array.from(document.querySelectorAll('img'))
      .map((image) => image.currentSrc || image.src || '')
      .find((src) => /user-avatar|avatar|toutiaostatic/i.test(src)) || '';
    const metrics = {};
    const metricPattern = /(粉丝数|总阅读(?:\(播放\))?量|累计收益|作品数|内容数)\s*([0-9.,万亿-]+(?:元)?)/g;
    for (const match of bodyText.matchAll(metricPattern)) {
      metrics[match[1]] = {
        raw: match[0],
        valueText: match[2],
        value: parseMetricValue(match[2]),
      };
    }
    return { bodyText, accountLink, avatarUrl, metrics };

    function textOf(element) {
      return (element?.innerText || element?.textContent || '').replace(/\s+/g, ' ').trim();
    }

    function parseMetricValue(valueText) {
      const normalized = String(valueText || '').replace(/,/g, '').replace(/元$/, '').trim();
      if (!normalized || normalized === '-') {
        return null;
      }
      const match = normalized.match(/^(-?\d+(?:\.\d+)?)(万|亿|%)?$/);
      if (!match) {
        return null;
      }
      const number = Number(match[1]);
      if (!Number.isFinite(number)) {
        return null;
      }
      if (match[2] === '亿') {
        return number * 100000000;
      }
      if (match[2] === '万') {
        return number * 10000;
      }
      return number;
    }
  }).catch(() => ({ bodyText: '', accountLink: '', avatarUrl: '', metrics: {} }));
}

function buildMetadata(userInfoEnvelope, homeMergeEnvelope, worksListEnvelope, visible) {
  const userInfo = objectAt(userInfoEnvelope, ['data']) || userInfoEnvelope || {};
  const homeData = objectAt(homeMergeEnvelope, ['data']) || {};
  const statistic = objectAt(homeData, ['statistic', 'data']) || {};
  const works = objectAt(homeData, ['works']) || {};
  const userId = firstNonEmpty(userInfo.user_id_str, userInfo.user_id, parseUserID(visible.accountLink));
  const mediaId = firstNonEmpty(userInfo.media_id, userInfo.media_id_str);
  const profileUrl = userId ? `https://www.toutiao.com/c/user/${userId}/` : visible.accountLink;
  return {
    displayName: firstNonEmpty(userInfo.name, userInfo.screen_name),
    avatarUrl: firstNonEmpty(userInfo.avatar_url, visible.avatarUrl),
    userId,
    mediaId,
    profileUrl,
    isCreator: Boolean(userInfo.is_creator),
    authType: numberOrNull(userInfo.auth_type),
    followerCount: numberOrZero(firstNonNull(userInfo.total_fans_count, statistic.total_subscribe_count, statistic.fans_data?.total, statistic.fans_data?.total_stat, statistic.fans_count_data?.total_stat, visible.metrics?.['粉丝数']?.value)),
    contentCount: numberOrZero(firstNonNull(works.total_count, statistic.thread_count, worksListEnvelope.total_number, visible.metrics?.['作品数']?.value, visible.metrics?.['内容数']?.value)),
    totalReadPlayCount: numberOrZero(firstNonNull(statistic.total_read_play_count, visible.metrics?.['总阅读(播放)量']?.value, visible.metrics?.['总阅读量']?.value)),
    yesterdayReadCount: numberOrZero(firstNonNull(statistic.yesterday_read_count)),
    yesterdayPlayCount: numberOrZero(firstNonNull(statistic.yesterday_play_count)),
    yesterdayFansCount: numberOrZero(firstNonNull(statistic.yesterday_fans_count, statistic.yesterday_fans)),
    totalIncome: numberOrZero(firstNonNull(statistic.total_income, visible.metrics?.['累计收益']?.value)),
    visibleMetrics: visible.metrics || {},
    visibleAccountLink: visible.accountLink || '',
    userInfo,
    homeStatistic: statistic,
    worksSummary: works,
    worksListSummary: {
      totalNumber: numberOrNull(worksListEnvelope.total_number),
      loginStatus: numberOrNull(worksListEnvelope.login_status),
      itemCount: Array.isArray(worksListEnvelope.data) ? worksListEnvelope.data.length : null,
    },
  };
}

async function getLoginState(page) {
  const cookies = await page.context().cookies();
  const cookieNames = cookies.map((cookie) => cookie.name).sort();
  const names = new Set(cookieNames);
  const pageUrl = page.url();
  const bodyText = await page.locator('body').innerText({ timeout: 3000 }).catch(() => '');
  const bodyHead = bodyText.slice(0, 1200);
  const onLoginPage = /\/auth\/page\/login|passport|sso|login/i.test(pageUrl);
  const hasLoginText = /扫码登录|验证码登录|登录\/注册|立即注册|账号登录/.test(bodyHead);
  const hasCreatorShell = /头条号|创作中心|作品管理|数据|总阅读|粉丝数/.test(bodyText);
  const loggedInByCookie = names.has('sessionid') || names.has('sid_guard') || names.has('uid_tt') || names.has('msToken');
  return {
    loggedIn: loggedInByCookie && !onLoginPage && !hasLoginText && hasCreatorShell,
    pageUrl,
    cookieNames,
    reason: loggedInByCookie && !onLoginPage && hasCreatorShell ? 'creator_cookie_present' : 'login_not_confirmed',
  };
}

async function writeResult(result) {
  if (!outputFile) {
    return;
  }
  await mkdir(path.dirname(outputFile), { recursive: true });
  await writeFile(outputFile, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
}

async function writeDebug(page, result, collector) {
  if (!debugDir) {
    return;
  }
  await mkdir(debugDir, { recursive: true });
  await writeFile(path.join(debugDir, 'metadata-result.json'), `${JSON.stringify(result, null, 2)}\n`, 'utf8');
  await writeFile(path.join(debugDir, 'network-capture.json'), `${JSON.stringify(collector.debugEvents(), null, 2)}\n`, 'utf8');
  await page.screenshot({ path: path.join(debugDir, 'metadata-page.png'), fullPage: false }).catch(() => undefined);
}

async function safeTitle(page) {
  return page.title().catch(() => '');
}

function objectAt(value, pathItems) {
  let current = value;
  for (const item of pathItems) {
    if (!current || typeof current !== 'object') {
      return null;
    }
    current = current[item];
  }
  return current && typeof current === 'object' ? current : null;
}

function firstNonEmpty(...values) {
  for (const value of values) {
    const text = String(value ?? '').trim();
    if (text) {
      return text;
    }
  }
  return '';
}

function firstNonNull(...values) {
  for (const value of values) {
    if (value !== null && value !== undefined && value !== '') {
      return value;
    }
  }
  return null;
}

function numberOrNull(value) {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
}

function numberOrZero(value) {
  return numberOrNull(value) ?? 0;
}

function parseUserID(value) {
  return String(value || '').match(/\/c\/user\/(\d+)/)?.[1] || '';
}

function sanitizeValue(value) {
  if (Array.isArray(value)) {
    return value.slice(0, 80).map((item) => sanitizeValue(item));
  }
  if (!value || typeof value !== 'object') {
    return typeof value === 'string' ? sanitizeText(value) : value;
  }
  const result = {};
  for (const [key, item] of Object.entries(value)) {
    if (isSensitiveKey(key)) {
      result[key] = '[redacted]';
      continue;
    }
    result[key] = sanitizeValue(item);
  }
  return result;
}

function isSensitiveKey(key) {
  return /cookie|token|session|authorization|credential|csrf|signature|secret|password|x-s|x-t|msToken/i.test(key);
}

function sanitizeText(value) {
  return String(value || '')
    .replace(/(sessionid|sid_guard|uid_tt|msToken|passport_csrf_token|csrf_token|token)["'=:\s]+[^"',\s&}]+/gi, '$1=[redacted]')
    .replace(/\s+/g, ' ')
    .trim();
}

async function importPlaywright() {
  try {
    const mod = await import('playwright');
    return mod.default ?? mod;
  } catch (firstError) {
    const frontendModule = path.resolve(scriptDir, '..', 'frontend', 'node_modules', 'playwright', 'index.js');
    try {
      const mod = await import(pathToFileURL(frontendModule).href);
      return mod.default ?? mod;
    } catch {
      throw firstError;
    }
  }
}

function parseArgs(argv) {
  const result = {};
  for (let index = 0; index < argv.length; index += 1) {
    const item = argv[index];
    if (!item.startsWith('--')) {
      continue;
    }
    const key = item.slice(2);
    const next = argv[index + 1];
    if (!next || next.startsWith('--')) {
      result[key] = 'true';
      continue;
    }
    result[key] = next;
    index += 1;
  }
  return result;
}

function required(values, key) {
  const value = values[key];
  if (!value) {
    throw new Error(`Missing required argument: --${key}`);
  }
  return value;
}

function isProfileInUseError(error) {
  const message = error instanceof Error ? error.message : String(error);
  return /profile.*in use|SingletonLock|ProcessSingleton|user data directory is already in use/i.test(message);
}
