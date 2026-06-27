#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const profileDir = path.resolve(required(args, 'profile-dir'));
const metadataUrl = args['metadata-url'] || 'https://creator.xiaohongshu.com/new/home';
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

const selectors = {
  accountCard: '.home-card-wrapper .personal',
  profileBase: '.home-card-wrapper .personal .base',
  avatar: '.home-card-wrapper .personal .base .avatar img',
  accountName: '.home-card-wrapper .personal .base .account-name',
  accountStatusImage: '.home-card-wrapper .personal .base .text > div:first-child img',
  accountStats: '.home-card-wrapper .personal .base .static.description-text > div',
  redAccountId: '.home-card-wrapper .personal .base .others.description-text',
  overviewBlocks: '.home-card-wrapper .statics .datas.grouped-note-data .creator-block',
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
    platform: 'xiaohongshu',
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
  await page.goto(metadataUrl, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(settleMs);

  const loginState = await getLoginState(page);
  if (!loginState.loggedIn) {
    const result = {
      ok: false,
      platform: 'xiaohongshu',
      status: 'not_logged_in',
      profileDir,
      metadataUrl,
      pageUrl: page.url(),
      title: await safeTitle(page),
      capturedAt: new Date().toISOString(),
      loginState,
    };
    await writeDebug(page, result);
    return result;
  }

  const hasAccountCard = await page.locator(selectors.profileBase).first().isVisible({ timeout: timeoutMs }).catch(() => false);
  if (!hasAccountCard) {
    const result = {
      ok: false,
      platform: 'xiaohongshu',
      status: 'account_card_not_found',
      profileDir,
      metadataUrl,
      pageUrl: page.url(),
      title: await safeTitle(page),
      capturedAt: new Date().toISOString(),
      selectors,
      loginState,
    };
    await writeDebug(page, result);
    return result;
  }

  const payload = await page.evaluate((fixedSelectors) => {
    const profileBase = document.querySelector(fixedSelectors.profileBase);
    const accountName = textOf(document.querySelector(fixedSelectors.accountName));
    const avatarElement = document.querySelector(fixedSelectors.avatar);
    const statusImageElement = document.querySelector(fixedSelectors.accountStatusImage);
    const redAccountText = textOf(document.querySelector(fixedSelectors.redAccountId));
    const redAccountId = redAccountText.match(/小红书账号[:：]\s*([^\s]+)/)?.[1] || '';
    const accountStats = {};

    for (const item of document.querySelectorAll(fixedSelectors.accountStats)) {
      const numberText = textOf(item.querySelector('.numerical'));
      const label = textOf(item).replace(numberText, '').trim();
      if (!label) {
        continue;
      }
      accountStats[label] = {
        raw: textOf(item),
        valueText: numberText,
        value: parseMetricValue(numberText),
      };
    }

    const overviewMetrics = {};
    for (const block of document.querySelectorAll(fixedSelectors.overviewBlocks)) {
      const title = textOf(block.querySelector('.title'));
      if (!title) {
        continue;
      }
      const valueText = textOf(block.querySelector('.number-container'));
      overviewMetrics[title] = {
        raw: textOf(block),
        valueText,
        value: parseMetricValue(valueText),
      };
    }

    return {
      displayName: accountName,
      avatarUrl: imageSource(avatarElement),
      accountStatusImageUrl: externalImageSource(statusImageElement),
      accountStatusAlt: statusImageElement?.getAttribute('alt') || '',
      redAccountId,
      rawRedAccountText: redAccountText,
      followingCount: accountStats['关注数']?.value ?? null,
      followerCount: accountStats['粉丝数']?.value ?? null,
      likedAndFavoritedCount: accountStats['获赞与收藏']?.value ?? null,
      accountStats,
      overviewMetrics,
      accountCardText: textOf(document.querySelector(fixedSelectors.accountCard)),
      profileBaseText: textOf(profileBase),
    };

    function imageSource(element) {
      return element?.currentSrc || element?.src || '';
    }

    function externalImageSource(element) {
      const src = imageSource(element);
      return /^https?:\/\//i.test(src) ? src : '';
    }

    function textOf(element) {
      return (element?.innerText || element?.textContent || '').replace(/\s+/g, ' ').trim();
    }

    function parseMetricValue(valueText) {
      const normalized = String(valueText || '').replace(/,/g, '').trim();
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
  }, selectors);

  const result = {
    ok: true,
    platform: 'xiaohongshu',
    status: 'collected',
    profileDir,
    metadataUrl,
    pageUrl: page.url(),
    title: await safeTitle(page),
    capturedAt: new Date().toISOString(),
    dataSource: 'browser_visible_dom',
    selectors,
    loginState,
    metadata: payload,
  };
  await writeDebug(page, result);
  return result;
}

async function getLoginState(page) {
  const cookies = await page.context().cookies();
  const cookieNames = cookies.map((cookie) => cookie.name).sort();
  const names = new Set(cookieNames);
  const pageUrl = page.url();
  const bodyText = await page.locator('body').innerText({ timeout: 3000 }).catch(() => '');
  const loggedInByCookie =
    names.has('customerClientId') ||
    names.has('access-token-creator.xiaohongshu.com') ||
    names.has('galaxy_creator_session_id') ||
    names.has('x-user-id-creator.xiaohongshu.com');
  const loginPage = /login|signin|passport/i.test(pageUrl) || /扫码|验证码|登录/.test(bodyText.slice(0, 500));
  return {
    loggedIn: loggedInByCookie && !loginPage,
    pageUrl,
    cookieNames,
    reason: loggedInByCookie && !loginPage ? 'creator_cookie_present' : 'login_not_confirmed',
  };
}

async function writeResult(result) {
  if (!outputFile) {
    return;
  }
  await mkdir(path.dirname(outputFile), { recursive: true });
  await writeFile(outputFile, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
}

async function writeDebug(page, result) {
  if (!debugDir) {
    return;
  }
  await mkdir(debugDir, { recursive: true });
  await writeFile(path.join(debugDir, 'metadata-result.json'), `${JSON.stringify(result, null, 2)}\n`, 'utf8');
  await page.screenshot({ path: path.join(debugDir, 'metadata-page.png'), fullPage: false }).catch(() => undefined);
}

async function safeTitle(page) {
  return page.title().catch(() => '');
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
