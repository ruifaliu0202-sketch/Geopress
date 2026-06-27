#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const platform = required(args, 'platform');
const profileDir = path.resolve(required(args, 'profile-dir'));
const startUrl = required(args, 'url');
const debugDir = path.resolve(args['debug-dir'] || path.join(scriptDir, '..', 'runtime', `${platform}-metadata-probe`));
const outputFile = path.resolve(args.output || path.join(debugDir, 'probe-result.json'));
const chromePath = args['chrome-path'] || undefined;
const userAgent = args['user-agent'] || defaultUserAgent(platform);
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const settleMs = Number(args['settle-ms'] || 2500);
const maxNavClicks = Number(args['max-nav-clicks'] || 8);
const viewport = { width: 1440, height: 1000 };

const platformConfigs = {
  netease: {
    platformName: '网易号',
    hostPattern: /(^|\.)163\.com$|(^|\.)126\.net$/,
    loginPattern: /登录|扫码|二维码|验证码|网易邮箱|手机验证码/,
    shellPattern: /网易号|发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|收益/,
    navTexts: ['数据', '数据中心', '内容管理', '文章管理', '作品管理', '账号', '帐号', '首页'],
  },
  toutiao: {
    platformName: '头条号',
    hostPattern: /(^|\.)toutiao\.com$|(^|\.)bytedance\.com$|(^|\.)byteimg\.com$/,
    loginPattern: /登录|扫码|二维码|验证码|抖音|今日头条|手机验证码/,
    shellPattern: /头条号|发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|收益/,
    navTexts: ['数据', '数据中心', '内容管理', '文章管理', '作品管理', '账号', '帐号', '首页'],
  },
  sohu: {
    platformName: '搜狐号',
    hostPattern: /(^|\.)sohu\.com$|(^|\.)sohucs\.com$/,
    loginPattern: /登录|扫码|二维码|验证码|搜狐|手机验证码|图形验证码/,
    shellPattern: /搜狐号|发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|收益/,
    navTexts: ['数据', '数据中心', '内容管理', '文章管理', '作品管理', '账号', '帐号', '首页'],
  },
};

const config = platformConfigs[platform] || {
  platformName: platform,
  hostPattern: /.*/,
  loginPattern: /登录|扫码|二维码|验证码|手机验证码/,
  shellPattern: /发布|发文|创作|内容管理|作品管理|数据|账号/,
  navTexts: ['数据', '内容管理', '文章管理', '作品管理', '账号', '首页'],
};

const playwright = await importPlaywright();
await mkdir(profileDir, { recursive: true });
await mkdir(debugDir, { recursive: true });

const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

const context = await playwright.chromium.launchPersistentContext(profileDir, {
  executablePath: chromePath,
  headless,
  viewport,
  locale: 'zh-CN',
  userAgent,
  args: chromiumArgs,
});

let page;
try {
  page = context.pages()[0] ?? (await context.newPage());
  const capture = createNetworkCapture(page, config);
  const observations = [];

  await page.goto(startUrl, { waitUntil: 'domcontentloaded', timeout: 45000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(settleMs);
  observations.push(await observePage(page, 'start', debugDir));

  const clicked = new Set();
  for (const text of config.navTexts) {
    if (clicked.size >= maxNavClicks) {
      break;
    }
    const clickedKey = await clickVisibleNavText(page, text, clicked);
    if (!clickedKey) {
      continue;
    }
    clicked.add(clickedKey);
    await page.waitForLoadState('networkidle', { timeout: 12000 }).catch(() => undefined);
    await page.waitForTimeout(settleMs);
    observations.push(await observePage(page, `nav-${slug(text)}-${clicked.size}`, debugDir));
  }

  const networkEvents = await capture.flush();
  const result = {
    ok: true,
    platform,
    platformName: config.platformName,
    profileDir,
    startUrl,
    pageUrl: page.url(),
    capturedAt: new Date().toISOString(),
    loginState: inferLoginState(observations[0], config),
    observations,
    network: summarizeNetwork(networkEvents),
    networkCapturePath: path.join(debugDir, 'network-capture.json'),
  };
  await writeFile(result.networkCapturePath, JSON.stringify(sanitizeForOutput(networkEvents), null, 2), 'utf8');
  await writeFile(outputFile, JSON.stringify(sanitizeForOutput(result), null, 2), 'utf8');
  console.log(JSON.stringify(sanitizeForOutput(result)));
} finally {
  await context.close();
}

async function observePage(page, label, directory) {
  const screenshotPath = path.join(directory, `${label}-${Date.now()}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true }).catch(() => undefined);
  const bodyText = await page.locator('body').innerText({ timeout: 5000 }).catch(() => '');
  const domMetrics = extractDOMMetrics(bodyText);
  const links = await page.locator('a, [role="link"], button, [role="button"]').evaluateAll((elements) => elements
    .map((element) => {
      const rect = element.getBoundingClientRect();
      return {
        text: (element.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 80),
        href: element.href || element.getAttribute('href') || '',
        role: element.getAttribute('role') || element.tagName.toLowerCase(),
        visible: rect.width > 0 && rect.height > 0,
        x: Math.round(rect.x),
        y: Math.round(rect.y),
      };
    })
    .filter((item) => item.visible && item.text)
    .slice(0, 160)).catch(() => []);

  return {
    label,
    title: await page.title().catch(() => ''),
    pageUrl: page.url(),
    screenshotPath,
    bodyTextSample: bodyText.slice(0, 5000),
    domMetrics,
    visibleActions: links,
  };
}

async function clickVisibleNavText(page, text, clicked) {
  const locators = [
    page.getByRole('link', { name: new RegExp(escapeRegExp(text)) }).first(),
    page.getByRole('button', { name: new RegExp(escapeRegExp(text)) }).first(),
    page.locator('a, [role="link"], button, [role="button"], li, span, div').filter({ hasText: new RegExp(`^\\s*${escapeRegExp(text)}\\s*$`) }).first(),
    page.getByText(text, { exact: false }).first(),
  ];

  for (const locator of locators) {
    if (!(await locator.isVisible({ timeout: 1000 }).catch(() => false))) {
      continue;
    }
    const target = await locatorTarget(locator, text);
    const key = `${target.text}:${target.x}:${target.y}`;
    if (clicked.has(key)) {
      continue;
    }
    await locator.click().catch(() => undefined);
    return key;
  }
  return '';
}

async function locatorTarget(locator, text) {
  const box = await locator.boundingBox().catch(() => null);
  return {
    text,
    x: box ? Math.round(box.x + box.width / 2) : 0,
    y: box ? Math.round(box.y + box.height / 2) : 0,
  };
}

function createNetworkCapture(page, config) {
  const events = [];
  const pending = new Set();
  page.on('response', (response) => {
    const task = captureNetworkResponse(response, events, config).catch(() => undefined);
    pending.add(task);
    task.finally(() => pending.delete(task));
  });
  return {
    async flush() {
      const deadline = Date.now() + 3000;
      while (pending.size > 0 && Date.now() < deadline) {
        await Promise.race([
          Promise.allSettled([...pending]),
          new Promise((resolve) => setTimeout(resolve, 250)),
        ]);
      }
      return [...events];
    },
  };
}

async function captureNetworkResponse(response, events, config) {
  const url = response.url();
  if (!isPlatformResponse(url, config)) {
    return;
  }
  const request = response.request();
  const method = request.method();
  const status = response.status();
  const contentType = response.headers()['content-type'] || '';
  const interesting = /api|ajax|content|article|item|news|stat|data|metric|profile|account|user|author|publish|post|opus|doc/i.test(url);
  if (!interesting && !/json/i.test(contentType)) {
    return;
  }
  const bodyText = await response.text().catch(() => '');
  if (!bodyText || bodyText.length > 300000) {
    return;
  }
  let parsed = null;
  if (/^\s*[\[{]/.test(bodyText)) {
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      parsed = null;
    }
  }
  const numericMetrics = parsed ? extractNumericMetrics(parsed) : {};
  const idCandidates = parsed ? extractIDCandidates(parsed) : [];
  events.push({
    at: new Date().toISOString(),
    url,
    method,
    status,
    resourceType: request.resourceType(),
    contentType: contentType.split(';')[0],
    numericMetrics,
    idCandidates: idCandidates.slice(0, 30),
    bodySample: parsed ? JSON.stringify(sanitizeForOutput(parsed)).slice(0, 3000) : sanitizeText(bodyText.slice(0, 3000)),
  });
  if (events.length > 180) {
    events.splice(0, events.length - 180);
  }
}

function isPlatformResponse(value, config) {
  try {
    return config.hostPattern.test(new URL(value).hostname);
  } catch {
    return false;
  }
}

function summarizeNetwork(events) {
  return events.map((event) => ({
    url: event.url,
    method: event.method,
    status: event.status,
    contentType: event.contentType,
    metricKeys: Object.keys(event.numericMetrics || {}),
    idCandidates: event.idCandidates,
  })).slice(-80);
}

function inferLoginState(observation, config) {
  const text = observation?.bodyTextSample || '';
  const pageUrl = observation?.pageUrl || '';
  return {
    loggedIn: !/login|passport|sso/i.test(pageUrl) && config.shellPattern.test(text) && !config.loginPattern.test(text.replace(config.shellPattern, '')),
    pageUrl,
    hasLoginText: config.loginPattern.test(text),
    hasShellText: config.shellPattern.test(text),
  };
}

function extractDOMMetrics(text) {
  const result = {};
  const compact = String(text || '').replace(/\s+/g, ' ');
  const patterns = [
    ['followerCount', /(?:粉丝|粉丝数|订阅|关注者)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['followingCount', /(?:关注|关注数)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['contentCount', /(?:文章|内容|作品)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['viewCount', /(?:阅读|阅读量|浏览|展现|播放)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['likeCount', /(?:点赞|赞|获赞)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['commentCount', /(?:评论|评论数)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['shareCount', /(?:分享|转发)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
    ['favoriteCount', /(?:收藏)\s*[:：]?\s*([0-9,.]+万?|[0-9,.]+k?)/i],
  ];
  for (const [key, pattern] of patterns) {
    const match = compact.match(pattern);
    if (match) {
      result[key] = { raw: match[0], valueText: match[1], value: parseNumberText(match[1]) };
    }
  }
  return result;
}

function extractNumericMetrics(value, pathParts = []) {
  const result = {};
  const visit = (node, path) => {
    if (node == null || Object.keys(result).length > 120) {
      return;
    }
    if (Array.isArray(node)) {
      node.slice(0, 20).forEach((item, index) => visit(item, [...path, String(index)]));
      return;
    }
    if (typeof node === 'object') {
      for (const [key, child] of Object.entries(node)) {
        visit(child, [...path, key]);
      }
      return;
    }
    const key = path[path.length - 1] || '';
    if (!/(fans|follower|follow|read|view|pv|uv|impression|like|comment|share|favorite|collect|click|article|content|post|item|count|num|total|播放|阅读|粉丝|点赞|评论|分享|收藏)/i.test(key)) {
      return;
    }
    const parsed = typeof node === 'number' ? node : parseNumberText(String(node));
    if (!Number.isFinite(parsed)) {
      return;
    }
    result[path.join('.')] = parsed;
  };
  visit(value, pathParts);
  return result;
}

function extractIDCandidates(value, pathParts = []) {
  const result = [];
  const seen = new Set();
  const visit = (node, path) => {
    if (result.length >= 120 || node == null) {
      return;
    }
    if (Array.isArray(node)) {
      node.slice(0, 30).forEach((item, index) => visit(item, [...path, String(index)]));
      return;
    }
    if (typeof node === 'object') {
      for (const [key, child] of Object.entries(node)) {
        visit(child, [...path, key]);
      }
      return;
    }
    const key = path[path.length - 1] || '';
    const text = String(node).trim();
    if (!/(id|article|item|news|doc|post|opus|content|account|user|author)/i.test(key) || !/^[A-Za-z0-9_-]{6,96}$/.test(text)) {
      return;
    }
    const signature = `${path.join('.')}:${text}`;
    if (seen.has(signature)) {
      return;
    }
    seen.add(signature);
    result.push({ path: path.join('.'), key, value: text });
  };
  visit(value, pathParts);
  return result;
}

function parseNumberText(value) {
  const text = String(value || '').trim().replace(/,/g, '');
  if (!text) {
    return NaN;
  }
  const match = text.match(/([0-9]+(?:\.[0-9]+)?)(万|k|K)?/);
  if (!match) {
    return NaN;
  }
  const base = Number(match[1]);
  if (!Number.isFinite(base)) {
    return NaN;
  }
  if (match[2] === '万') {
    return Math.round(base * 10000);
  }
  if (/k/i.test(match[2] || '')) {
    return Math.round(base * 1000);
  }
  return Math.round(base);
}

function sanitizeForOutput(value, pathParts = []) {
  if (Array.isArray(value)) {
    return value.map((item, index) => sanitizeForOutput(item, [...pathParts, String(index)]));
  }
  if (value && typeof value === 'object') {
    const result = {};
    for (const [key, child] of Object.entries(value)) {
      if (isSensitiveKey(key, pathParts)) {
        result[key] = '[redacted]';
        continue;
      }
      result[key] = sanitizeForOutput(child, [...pathParts, key]);
    }
    return result;
  }
  if (typeof value === 'string') {
    return sanitizeText(value);
  }
  return value;
}

function isSensitiveKey(key, pathParts) {
  const target = [...pathParts, key].join('.').toLowerCase();
  return /cookie|token|session|authorization|credential|secret|password|passwd|csrf|xsrf|signature|sign|x-s|x-s-common|x-t/.test(target);
}

function sanitizeText(value) {
  return String(value)
    .replace(/(cookie|token|session|authorization|credential|secret|password|passwd|csrf|xsrf|signature|sign|x-s|x-s-common|x-t)(["'=:\s]+)[^"',\s}]+/gi, '$1$2[redacted]')
    .replace(/Bearer\s+[A-Za-z0-9._-]+/gi, 'Bearer [redacted]');
}

function slug(value) {
  return String(value).replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '') || 'page';
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function defaultUserAgent(value) {
  if (value === 'sohu') {
    return process.env.GEOPRESS_SOHU_USER_AGENT || 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.7827.102 Safari/537.36';
  }
  return undefined;
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
