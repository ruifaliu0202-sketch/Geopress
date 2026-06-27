#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const profileDir = path.resolve(required(args, 'profile-dir'));
const metadataUrl = args['metadata-url'] || 'https://mp.sohu.com/mpfe/v4/';
const chromePath = args['chrome-path'] || undefined;
const outputFile = args.output ? path.resolve(args.output) : '';
const debugDir = args['debug-dir'] ? path.resolve(args['debug-dir']) : '';
const timeoutMs = Number(args['timeout-ms'] || 45000);
const settleMs = Number(args['settle-ms'] || 3000);
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const sohuUserAgent =
  args['user-agent'] ||
  process.env.GEOPRESS_SOHU_USER_AGENT ||
  'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.7827.102 Safari/537.36';
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

const endpointMatchers = {
  accountList: /\/mpbp\/bp\/account\/listV2\b/,
  accountInfo: /\/mpbp\/bp\/account\/info\b/,
  registerInfo: /\/mpbp\/bp\/account\/register-info\b/,
  userCheck: /\/mpbp\/bp\/account\/check\/user\b/,
  commonAuth: /\/mpbp\/bp\/account\/common\/auth\b/,
  operatorAuthority: /\/mpbp\/bp\/account\/operator\/authority\b/,
};

let context;
try {
  const playwright = await importPlaywright();
  await mkdir(profileDir, { recursive: true });
  context = await playwright.chromium.launchPersistentContext(profileDir, {
    executablePath: chromePath,
    headless,
    viewport: { width: 1440, height: 1000 },
    locale: 'zh-CN',
    userAgent: sohuUserAgent,
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
    platform: 'sohu',
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

  const visible = await collectVisibleState(page);
  const loginState = await getLoginState(page, visible);
  if (!loginState.loggedIn) {
    const result = {
      ok: false,
      platform: 'sohu',
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

  await clickFirstVisible(page, ['账号信息', '数据分析', '内容分析', '我的内容']);
  await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => undefined);
  await page.waitForTimeout(Math.min(settleMs, 2500));
  await waitForSelectedEndpoints(collector, 10000);

  const accountInfo = dataOf(collector.latest('accountInfo')?.json) || {};
  const accountList = dataOf(collector.latest('accountList')?.json) || {};
  const registerInfo = dataOf(collector.latest('registerInfo')?.json) || {};
  const userCheck = dataOf(collector.latest('userCheck')?.json) || {};
  const commonAuth = dataOf(collector.latest('commonAuth')?.json) || {};
  const operatorAuthority = dataOf(collector.latest('operatorAuthority')?.json) || {};
  const metadata = buildMetadata({ accountInfo, accountList, registerInfo, userCheck, commonAuth, operatorAuthority, visible });
  const ok = Boolean(metadata.displayName || metadata.accountId);
  const result = {
    ok,
    platform: 'sohu',
    status: ok ? 'collected' : 'account_metadata_not_found',
    profileDir,
    metadataUrl,
    pageUrl: page.url(),
    title: await safeTitle(page),
    capturedAt: new Date().toISOString(),
    dataSource: 'browser_context_request',
    selectors: {
      profileShellText: 'body text contains 搜狐号/内容管理/数据分析/账号信息',
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
    if (!/https:\/\/[^/]*sohu\.com\//i.test(url)) {
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
      // The diagnostic event remains useful even if the response body is unavailable.
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
    if (collector.latest('accountInfo') && collector.latest('registerInfo')) {
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
    const accountName = bodyText.match(/实名认证\s+\S+\s+\d+\s+([^\n\r]+?)\s+我的内容/)?.[1] || '';
    return { bodyText, accountName };

    function textOf(element) {
      return (element?.innerText || element?.textContent || '').replace(/[ \t]+/g, ' ').trim();
    }
  }).catch(() => ({ bodyText: '', accountName: '' }));
}

function buildMetadata({ accountInfo, accountList, registerInfo, userCheck, commonAuth, operatorAuthority, visible }) {
  const listAccount = firstListAccount(accountList);
  const registerAccount = objectAt(registerInfo, ['account']) || {};
  const org = objectAt(registerInfo, ['org']) || {};
  const user = objectAt(registerInfo, ['user']) || {};
  const source = accountInfo.id ? accountInfo : listAccount.id ? listAccount : registerAccount;
  const accountId = firstNonEmpty(source.id, listAccount.id, registerAccount.id);
  const orgId = firstNonEmpty(source.orgId, listAccount.orgId, registerAccount.orgId, org.id);
  const avatarUrl = normalizeURL(firstNonEmpty(source.avatar, listAccount.avatar));
  const rights = source.rights && typeof source.rights === 'object' ? source.rights : {};
  return {
    displayName: firstNonEmpty(source.nickName, listAccount.nickName, visible.accountName),
    avatarUrl,
    accountId,
    orgId,
    userId: firstNonEmpty(user.id),
    userCode: firstNonEmpty(registerInfo.userCode, userCheck.userCode, user.userCode),
    profileUrl: firstNonEmpty(source.homePage, listAccount.homePage),
    accountType: numberOrZero(firstNonNull(source.accountType, listAccount.accountType, registerAccount.accountType)),
    accountTypeName: firstNonEmpty(source.accountTypeName, listAccount.accountTypeName, registerAccount.accountTypeName, org.orgTypeDesc),
    statusCode: numberOrZero(firstNonNull(source.status, listAccount.status, registerAccount.status, userCheck.code)),
    statusName: firstNonEmpty(source.statusName, listAccount.statusName, registerAccount.statusName, userCheck.des),
    registerStatus: numberOrNull(registerInfo.registerStatus),
    securityScore: numberOrNull(firstNonNull(source.securityScore, listAccount.securityScore)),
    riskLevel: numberOrNull(firstNonNull(source.riskLevel, listAccount.riskLevel)),
    newUser: Boolean(source.newUser ?? listAccount.newUser),
    admin: Boolean(registerInfo.admin),
    hasOperator: Boolean(operatorAuthority.hasOperator),
    userIsOperator: Boolean(operatorAuthority.userIsOperator),
    orgStatusName: firstNonEmpty(accountList.data?.[0]?.orgStatusName, org.statusName),
    locationName: firstNonEmpty(org.locationName),
    provinceName: firstNonEmpty(org.provinceName),
    cityName: firstNonEmpty(org.cityName),
    visibleText: visible.bodyText,
    visibleAccountName: visible.accountName,
    rightsSummary: summarizeRights(rights),
    commonAuth,
    userCheck,
    accountInfo,
    listAccount,
    registerInfo,
  };
}

function firstListAccount(accountList) {
  const group = Array.isArray(accountList.data) ? accountList.data[0] : null;
  const accountInfos = group && Array.isArray(group.accountInfos) ? group.accountInfos : [];
  const managerAccounts = group && Array.isArray(group.managerAccounts) ? group.managerAccounts : [];
  return accountInfos[0] || managerAccounts[0] || {};
}

function summarizeRights(rights) {
  const result = {};
  for (const [key, item] of Object.entries(rights || {})) {
    if (!item || typeof item !== 'object') {
      continue;
    }
    result[key] = {
      code: item.code || key,
      value: numberOrNull(item.value),
      name: item.name || '',
      info: item.info || '',
    };
  }
  return result;
}

async function getLoginState(page, visible) {
  const cookies = await page.context().cookies();
  const cookieNames = cookies.map((cookie) => cookie.name).sort();
  const names = new Set(cookieNames);
  const pageUrl = page.url();
  const bodyText = visible.bodyText || '';
  const onLoginPage = /\/login\b|passport|sso/i.test(pageUrl);
  const hasLoginText = /账号登录|手机登录|验证码|注册账号/.test(bodyText);
  const hasShellText = /搜狐号|我的内容|发布内容|数据分析|账号信息|个人中心/.test(bodyText);
  const loggedInByCookie = names.has('ppinf') || names.has('pprdig') || names.has('SUV') || names.has('gidinf');
  return {
    loggedIn: loggedInByCookie && !onLoginPage && hasShellText && !hasLoginText,
    pageUrl,
    cookieNames,
    reason: loggedInByCookie && hasShellText ? 'creator_cookie_present' : 'login_not_confirmed',
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

function dataOf(value) {
  return value && typeof value === 'object' && value.data && typeof value.data === 'object' ? value.data : {};
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

function normalizeURL(value) {
  const text = String(value || '').trim();
  if (!text) {
    return '';
  }
  if (text.startsWith('//')) {
    return `https:${text}`;
  }
  return text;
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
  return /cookie|token|session|authorization|credential|csrf|signature|secret|password|pprdig|ppinf|passport/i.test(key);
}

function sanitizeText(value) {
  return String(value || '')
    .replace(/(ppinf|pprdig|cookie|token|session|authorization|credential|csrf|signature|secret|password)["'=:\s]+[^"',\s&}]+/gi, '$1=[redacted]')
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
