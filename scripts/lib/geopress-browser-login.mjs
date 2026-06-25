import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));

export async function runBrowserLogin(config) {
  const args = parseArgs(process.argv.slice(2));
  const action = required(args, 'action');
  const sessionId = required(args, 'session-id');
  const profileDir = path.resolve(required(args, 'profile-dir'));
  const loginUrl = args['login-url'] || config.loginUrl;
  const qrSelector = args['qr-selector'] || config.qrSelector || 'canvas,img,svg';
  const qrWaitMs = Number(args['qr-wait-ms'] || config.qrWaitMs || 15000);
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
    const runtime = { ...config, action, sessionId, profileDir, loginUrl, qrSelector, qrWaitMs, stateFile, watchTimeoutMs, pollMs };
    if (action === 'start') {
      const result = await startLogin(page, runtime);
      console.log(JSON.stringify(result));
    } else if (action === 'watch') {
      await watchLogin(page, runtime);
    } else if (action === 'complete') {
      const result = await completeLogin(page, runtime);
      console.log(JSON.stringify(result));
    } else {
      throw new Error(`Unsupported action: ${action}`);
    }
  } finally {
    await context.close();
  }
}

async function startLogin(page, config) {
  await page.goto(config.loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);

  const alreadyLoggedIn = await isLoggedIn(page, config);
  if (alreadyLoggedIn) {
    const status = await loginStatus(page, config);
    return baseResult(page, config, { alreadyLoggedIn: true, qrScreenshotData: '', status });
  }

  if (!(await findVisibleQR(page, config.qrSelector, { throwOnMissing: false }))) {
    await switchToQRLogin(page, config);
  }
  const qr = await waitForVisibleQR(page, config.qrSelector, config.qrWaitMs);
  const image = await qr.screenshot({ type: 'png' });
  return baseResult(page, config, {
    alreadyLoggedIn: false,
    qrScreenshotData: `data:image/png;base64,${image.toString('base64')}`,
    qrSelector: config.qrSelector,
    status: await loginStatus(page, config),
  });
}

async function completeLogin(page, config) {
  await page.goto(config.loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => undefined);
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  const status = await loginStatus(page, config);
  return {
    sessionId: config.sessionId,
    pageUrl: status.pageUrl,
    profileDir: config.profileDir,
    stateFile: config.stateFile,
    loggedIn: status.loggedIn,
    completedAt: new Date().toISOString(),
    rawStatus: status,
  };
}

async function watchLogin(page, config) {
  const initial = await startLogin(page, config);
  await writeLoginState(config, {
    ...initial,
    loggedIn: initial.alreadyLoggedIn,
    lastCheckedAt: new Date().toISOString(),
  });
  process.stdout.write(`${JSON.stringify(initial)}\n`);

  if (initial.alreadyLoggedIn) {
    return;
  }

  const deadline = Date.now() + config.watchTimeoutMs;
  while (Date.now() < deadline) {
    await page.waitForTimeout(config.pollMs);
    const status = await loginStatus(page, config);
    const nextState = {
      sessionId: config.sessionId,
      loginUrl: config.loginUrl,
      pageUrl: status.pageUrl,
      profileDir: config.profileDir,
      platform: config.platform,
      loggedIn: status.loggedIn,
      lastCheckedAt: new Date().toISOString(),
      rawStatus: status,
    };
    await writeLoginState(config, nextState);
    if (status.loggedIn) {
      await writeLoginState(config, {
        ...nextState,
        completedAt: new Date().toISOString(),
      });
      return;
    }
  }

  await writeLoginState(config, {
    sessionId: config.sessionId,
    loginUrl: config.loginUrl,
    pageUrl: page.url(),
    profileDir: config.profileDir,
    platform: config.platform,
    loggedIn: false,
    timedOut: true,
    lastCheckedAt: new Date().toISOString(),
    rawStatus: await loginStatus(page, config),
  });
}

async function switchToQRLogin(page, config) {
  if (await findVisibleQR(page, config.qrSelector, { throwOnMissing: false })) {
    return;
  }

  const texts = config.qrSwitchTexts || ['扫码登录', '二维码登录', '扫一扫登录', 'APP 登录', 'APP扫码登录'];
  for (const frame of activeFrames(page)) {
    for (const text of texts) {
      const locator = frame.getByText(text, { exact: false }).first();
      if (await locator.isVisible({ timeout: 1000 }).catch(() => false)) {
        await locator.click().catch(() => undefined);
        await page.waitForTimeout(1200);
        if (await findVisibleQR(page, config.qrSelector, { throwOnMissing: false })) {
          return;
        }
      }
    }
  }

  const selectors = config.qrSwitchSelectors || [
    '[class*="qrcode"]',
    '[class*="qr-code"]',
    '[class*="qrCode"]',
    '[class*="scan"]',
    'img[alt*="扫码"]',
  ];
  for (const frame of activeFrames(page)) {
    for (const selector of selectors) {
      const locator = frame.locator(selector).first();
      if (await locator.isVisible({ timeout: 1000 }).catch(() => false)) {
        await locator.click().catch(() => undefined);
        await page.waitForTimeout(1200);
        if (await findVisibleQR(page, config.qrSelector, { throwOnMissing: false })) {
          return;
        }
      }
    }
  }
}

async function waitForVisibleQR(page, qrSelector, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  do {
    const candidate = await findVisibleQR(page, qrSelector, { throwOnMissing: false });
    if (candidate) {
      return candidate;
    }
    await page.waitForTimeout(500);
  } while (Date.now() < deadline);

  return findVisibleQR(page, qrSelector);
}

async function findVisibleQR(page, qrSelector, options = {}) {
  const candidates = [];
  for (const frame of activeFrames(page)) {
    const locator = frame.locator(qrSelector);
    const count = await locator.count().catch(() => 0);
    for (let index = 0; index < Math.min(count, 80); index += 1) {
      const candidate = locator.nth(index);
      if (!(await candidate.isVisible().catch(() => false))) {
        continue;
      }
      const box = await candidate.boundingBox().catch(() => null);
      if (!box || box.width < 80 || box.height < 80) {
        continue;
      }
      const score = await scoreQRCandidate(candidate, box);
      if (score <= 0) {
        continue;
      }
      candidates.push({ locator: candidate, score });
    }
  }

  candidates.sort((left, right) => right.score - left.score);
  if (candidates[0]) {
    return candidates[0].locator;
  }

  if (options.throwOnMissing === false) {
    return null;
  }

  const frameURLs = activeFrames(page).map((frame) => frame.url()).filter(Boolean).join(' | ');
  throw new Error(`No visible ${page.url()} login QR element found with selector: ${qrSelector}; frames: ${frameURLs}`);
}

async function scoreQRCandidate(locator, box) {
  const meta = await locator.evaluate((element) => {
    const className = typeof element.className === 'string' ? element.className : element.getAttribute('class') || '';
    return {
      tagName: element.tagName.toLowerCase(),
      className,
      id: element.id || '',
      alt: element.getAttribute('alt') || '',
      ariaLabel: element.getAttribute('aria-label') || '',
      src: element.getAttribute('src') || '',
    };
  }).catch(() => null);
  if (!meta) {
    return 0;
  }

  const ratio = Math.max(box.width, box.height) / Math.max(1, Math.min(box.width, box.height));
  let score = 100;
  if (ratio <= 1.35) {
    score += 40;
  } else if (ratio <= 1.8) {
    score += 10;
  } else {
    score -= 30;
  }

  if (['canvas', 'img', 'svg'].includes(meta.tagName)) {
    score += 20;
  }
  if (box.width >= 140 && box.height >= 140) {
    score += 20;
  }
  if (box.width > 640 || box.height > 640) {
    score -= 50;
  }

  const hints = `${meta.className} ${meta.id} ${meta.alt} ${meta.ariaLabel} ${meta.src}`.toLowerCase();
  if (/qr|qrcode|scan|扫码|二维码/.test(hints)) {
    score += 50;
  }
  return score;
}

async function isLoggedIn(page, config) {
  return (await loginStatus(page, config)).loggedIn;
}

async function loginStatus(page, config) {
  const cookies = await page.context().cookies();
  const names = new Set(cookies.map((cookie) => cookie.name));
  const bodyText = await visiblePageText(page);
  const pageUrl = page.url();
  const title = await safeTitle(page);
  const cookieNames = [...names].sort();

  for (const cookieName of config.loggedInCookieNames || []) {
    if (names.has(cookieName)) {
      return status(true, pageUrl, title, cookieNames, bodyText, `cookie_present:${cookieName}`);
    }
  }

  const loginPattern = config.loginTextPattern || /登录|扫码|二维码|验证码|手机验证码|账号密码/;
  const loggedInPattern = config.loggedInTextPattern || /发布|发文|创作|内容管理|作品管理|数据|账号/;
  if (loggedInPattern.test(bodyText) && !loginPattern.test(bodyText)) {
    return status(true, pageUrl, title, cookieNames, bodyText, 'creator_shell_text_present');
  }

  if (!/login|signin|passport|sso/i.test(pageUrl) && !loginPattern.test(bodyText)) {
    return status(true, pageUrl, title, cookieNames, bodyText, 'left_login_page');
  }

  return status(false, pageUrl, title, cookieNames, bodyText, 'login_not_confirmed');
}

function activeFrames(page) {
  return page.frames().filter((frame) => !frame.isDetached());
}

async function visiblePageText(page) {
  const chunks = [];
  for (const frame of activeFrames(page)) {
    const text = await frame.locator('body').innerText({ timeout: 1000 }).catch(() => '');
    if (text) {
      chunks.push(text);
    }
  }
  return chunks.join('\n');
}

function status(loggedIn, pageUrl, title, cookieNames, bodyText, reason) {
  return {
    loggedIn,
    pageUrl,
    title,
    cookieNames,
    bodyText: bodyText.slice(0, 1000),
    reason,
  };
}

function baseResult(page, config, overrides) {
  return {
    sessionId: config.sessionId,
    loginUrl: config.loginUrl,
    pageUrl: page.url(),
    profileDir: config.profileDir,
    stateFile: config.stateFile,
    platform: config.platform,
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

async function writeLoginState(config, value) {
  await mkdir(path.dirname(config.stateFile), { recursive: true });
  await writeFile(config.stateFile, JSON.stringify({ ...value, stateFile: config.stateFile }, null, 2), 'utf8');
}

async function importPlaywright() {
  try {
    const mod = await import('playwright');
    return mod.default ?? mod;
  } catch (firstError) {
    const frontendModule = path.resolve(scriptDir, '..', '..', 'frontend', 'node_modules', 'playwright', 'index.js');
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
