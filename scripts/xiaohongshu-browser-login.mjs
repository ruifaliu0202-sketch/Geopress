#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const action = required(args, 'action');
const sessionId = required(args, 'session-id');
const profileDir = path.resolve(required(args, 'profile-dir'));
const loginUrl = args['login-url'] || 'https://creator.xiaohongshu.com/login';
const qrSelector = args['qr-selector'] || 'canvas,img';
const chromePath = args['chrome-path'] || undefined;
const stateFile = args['state-file'] ? path.resolve(args['state-file']) : path.join(profileDir, 'geopress-login-state.json');
const watchTimeoutMs = Number(args['watch-timeout-ms'] || 5 * 60 * 1000);
const pollMs = Number(args['poll-ms'] || 1000);
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

const playwright = await importPlaywright();
await mkdir(profileDir, { recursive: true });

const context = await playwright.chromium.launchPersistentContext(profileDir, {
  executablePath: chromePath,
  headless,
  viewport: { width: 1280, height: 900 },
  locale: 'zh-CN',
  args: chromiumArgs,
});

try {
  const page = context.pages()[0] ?? (await context.newPage());
  if (action === 'start') {
    const result = await startLogin(page);
    console.log(JSON.stringify(result));
  } else if (action === 'watch') {
    await watchLogin(page);
  } else if (action === 'complete') {
    const result = await completeLogin(page);
    console.log(JSON.stringify(result));
  } else {
    throw new Error(`Unsupported action: ${action}`);
  }
} finally {
  await context.close();
}

async function startLogin(page) {
  await page.goto(loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);

  const alreadyLoggedIn = await isLoggedIn(page);
  if (alreadyLoggedIn) {
    const status = await loginStatus(page);
    return baseResult(page, { alreadyLoggedIn: true, qrScreenshotData: '', status });
  }

  await switchToQRLogin(page);
  const qr = await findVisibleQR(page);
  const image = await qr.screenshot({ type: 'png' });
  return baseResult(page, {
    alreadyLoggedIn: false,
    qrScreenshotData: `data:image/png;base64,${image.toString('base64')}`,
    qrSelector,
    status: await loginStatus(page),
  });
}

async function completeLogin(page) {
  await page.goto(loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => undefined);
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  const status = await loginStatus(page);
  return {
    sessionId,
    pageUrl: status.pageUrl,
    profileDir,
    loggedIn: status.loggedIn,
    completedAt: new Date().toISOString(),
    rawStatus: status,
  };
}

async function watchLogin(page) {
  const initial = await startLogin(page);
  await writeLoginState({
    ...initial,
    loggedIn: initial.alreadyLoggedIn,
    lastCheckedAt: new Date().toISOString(),
  });
  process.stdout.write(`${JSON.stringify(initial)}\n`);

  if (initial.alreadyLoggedIn) {
    return;
  }

  const deadline = Date.now() + watchTimeoutMs;
  while (Date.now() < deadline) {
    await page.waitForTimeout(pollMs);
    const status = await loginStatus(page);
    const nextState = {
      sessionId,
      loginUrl,
      pageUrl: status.pageUrl,
      profileDir,
      loggedIn: status.loggedIn,
      lastCheckedAt: new Date().toISOString(),
      rawStatus: status,
    };
    await writeLoginState(nextState);
    if (status.loggedIn) {
      await writeLoginState({
        ...nextState,
        completedAt: new Date().toISOString(),
      });
      return;
    }
  }

  await writeLoginState({
    sessionId,
    loginUrl,
    pageUrl: page.url(),
    profileDir,
    loggedIn: false,
    timedOut: true,
    lastCheckedAt: new Date().toISOString(),
    rawStatus: await loginStatus(page),
  });
}

async function switchToQRLogin(page) {
  const bodyText = await page.locator('body').innerText({ timeout: 5000 }).catch(() => '');
  if (/APP扫一扫登录|扫码即同意/.test(bodyText)) {
    return;
  }

  const switcher = page.locator('.sso-login-wrapper img').first();
  if (!(await switcher.isVisible({ timeout: 5000 }).catch(() => false))) {
    return;
  }
  await switcher.click();
  await page.waitForTimeout(1200);
}

async function findVisibleQR(page) {
  const candidates = page.locator(qrSelector);
  const count = await candidates.count();
  for (let index = 0; index < count; index += 1) {
    const candidate = candidates.nth(index);
    if (!(await candidate.isVisible().catch(() => false))) {
      continue;
    }
    const box = await candidate.boundingBox().catch(() => null);
    if (!box || box.width < 80 || box.height < 80) {
      continue;
    }
    return candidate;
  }
  throw new Error(`No visible Xiaohongshu login QR element found with selector: ${qrSelector}`);
}

async function isLoggedIn(page) {
  return (await loginStatus(page)).loggedIn;
}

async function loginStatus(page) {
  const cookies = await page.context().cookies();
  const names = new Set(cookies.map((cookie) => cookie.name));
  const bodyText = await page.locator('body').innerText({ timeout: 3000 }).catch(() => '');
  const pageUrl = page.url();
  const title = await safeTitle(page);

  if (names.has('web_session') || names.has('customerClientId') || names.has('id_token')) {
    return {
      loggedIn: true,
      pageUrl,
      title,
      cookieNames: [...names].sort(),
      bodyText: bodyText.slice(0, 1000),
      reason: 'login_cookie_present',
    };
  }

  if (!/login|signin|passport/i.test(pageUrl)) {
    if (!/登录|扫码|验证码/.test(bodyText)) {
      return {
        loggedIn: true,
        pageUrl,
        title,
        cookieNames: [...names].sort(),
        bodyText: bodyText.slice(0, 1000),
        reason: 'left_login_page',
      };
    }
  }

  return {
    loggedIn: false,
    pageUrl,
    title,
    cookieNames: [...names].sort(),
    bodyText: bodyText.slice(0, 1000),
    reason: 'login_not_confirmed',
  };
}

function baseResult(page, overrides) {
  return {
    sessionId,
    loginUrl,
    pageUrl: page.url(),
    profileDir,
    startedAt: new Date().toISOString(),
    rawStatus: {
      url: page.url(),
    },
    ...overrides,
  };
}

async function safeTitle(page) {
  return page.title().catch(() => '');
}

async function writeLoginState(value) {
  await mkdir(path.dirname(stateFile), { recursive: true });
  await writeFile(stateFile, JSON.stringify({ ...value, stateFile }, null, 2), 'utf8');
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
