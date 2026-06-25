import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));

export async function runBrowserPublish(config) {
  const args = parseArgs(process.argv.slice(2));
  const profileDir = path.resolve(required(args, 'profile-dir'));
  const title = required(args, 'title').trim();
  const body = required(args, 'body').trim();
  const publishMode = args['publish-mode'] || 'article';
  const chromePath = args['chrome-path'] || undefined;
  const publishUrl = args['publish-url'] || config.publishUrl;
  const screenshotDir = path.resolve(args['screenshot-dir'] || path.join(scriptDir, '..', '..', 'runtime'));
  const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
  const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
  if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
    chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
  }

  if (publishMode !== 'article') {
    throw new Error(`Unsupported publish mode: ${publishMode}`);
  }

  const playwright = await importPlaywright();
  await mkdir(profileDir, { recursive: true });
  await mkdir(screenshotDir, { recursive: true });

  const context = await playwright.chromium.launchPersistentContext(profileDir, {
    executablePath: chromePath,
    headless,
    viewport: { width: 1440, height: 1000 },
    locale: 'zh-CN',
    args: chromiumArgs,
  });

  let page;
  try {
    page = context.pages()[0] ?? (await context.newPage());
    const runtime = { ...config, profileDir, title, body, publishMode, publishUrl, screenshotDir };
    const publishOutcome = await publishArticle(page, runtime);
    const screenshotPath = await saveScreenshot(page, runtime, publishOutcome.status === 'published' ? `${config.platform}-publish-verified` : `${config.platform}-publish-pending`);
    console.log(JSON.stringify({
      status: publishOutcome.status,
      message: publishOutcome.message,
      pageUrl: page.url(),
      externalUrl: publishOutcome.externalUrl || '',
      externalId: publishOutcome.externalId || '',
      screenshotPath,
      submittedAt: new Date().toISOString(),
      rawStatus: {
        ...(await pageStatus(page)),
        publishOutcome,
      },
    }));
  } catch (error) {
    const screenshotPath = page ? await saveScreenshot(page, { screenshotDir }, `${config.platform}-publish-error`).catch(() => '') : '';
    throw new Error(`${error.message}${screenshotPath ? ` (screenshot: ${screenshotPath})` : ''}`);
  } finally {
    await context.close();
  }
}

async function publishArticle(page, config) {
  await page.goto(config.publishUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);

  await ensureLoggedIn(page, config);
  await openArticleEditor(page, config);
  await fillArticleEditor(page, config);
  return clickPublish(page, config);
}

async function ensureLoggedIn(page, config) {
  const status = await pageStatus(page);
  const bodyText = status.bodyText || '';
  if (/login|signin|passport|sso/i.test(status.pageUrl)) {
    throw new Error(`${config.platformName} browser profile is not logged in`);
  }
  const loginPattern = config.loginTextPattern || /登录|扫码|二维码|验证码|手机验证码|账号密码/;
  const shellPattern = config.creatorShellPattern || /发布|发文|创作|内容管理|作品管理|数据|账号/;
  if (loginPattern.test(bodyText) && !shellPattern.test(bodyText)) {
    throw new Error(`${config.platformName} browser profile is not logged in`);
  }
}

async function openArticleEditor(page, config) {
  if (await hasEditor(page, config)) {
    return;
  }
  for (const text of config.editorEntryTexts || []) {
    const locator = page.getByText(text, { exact: false }).first();
    if (await locator.isVisible({ timeout: 2000 }).catch(() => false)) {
      await locator.click();
      await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => undefined);
      await page.waitForTimeout(1500);
      if (await hasEditor(page, config)) {
        return;
      }
    }
  }
  throw new Error(`${config.platformName} article editor did not open`);
}

async function hasEditor(page, config) {
  return (await firstVisible(page, config.titleSelectors || [])) !== null
    && (await firstVisible(page, config.bodySelectors || [])) !== null;
}

async function fillArticleEditor(page, config) {
  const titleTarget = await firstVisible(page, config.titleSelectors || []);
  if (!titleTarget) {
    throw new Error(`${config.platformName} title input was not found`);
  }
  await fillLocator(titleTarget, config.title);

  const bodyTarget = await firstVisible(page, config.bodySelectors || []);
  if (!bodyTarget) {
    throw new Error(`${config.platformName} body editor was not found`);
  }
  await fillLocator(bodyTarget, config.body);
  await page.waitForTimeout(800);
}

async function firstVisible(page, selectors) {
  for (const selector of selectors) {
    const locator = page.locator(selector).first();
    if (await locator.isVisible({ timeout: 1500 }).catch(() => false)) {
      return locator;
    }
  }
  return null;
}

async function fillLocator(locator, value) {
  await locator.click().catch(() => undefined);
  await locator.fill(value).catch(async () => {
    await locator.evaluate((element, nextValue) => {
      if ('value' in element) {
        element.value = nextValue;
        element.dispatchEvent(new Event('input', { bubbles: true }));
        element.dispatchEvent(new Event('change', { bubbles: true }));
        return;
      }
      element.textContent = nextValue;
      element.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: nextValue }));
    }, value);
  });
}

async function clickPublish(page, config) {
  const beforeURL = page.url();
  await page.keyboard.press('Escape').catch(() => undefined);
  await page.evaluate(() => {
    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur();
    }
    window.scrollTo(0, document.body.scrollHeight);
  }).catch(() => undefined);
  await page.waitForTimeout(800);

  const action = await findPublishAction(page, config);
  if (!action) {
    throw new Error(`${config.platformName} publish button was not found`);
  }
  await action.locator.click();
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  return waitForPublishOutcome(page, config, beforeURL, action.target);
}

async function findPublishAction(page, config) {
  for (const text of config.publishTexts || ['发布']) {
    const locators = [
      page.getByRole('button', { name: new RegExp(`^${escapeRegExp(text)}$`) }).last(),
      page.locator('button').filter({ hasText: new RegExp(`^${escapeRegExp(text)}$`) }).last(),
      page.locator('[role="button"]').filter({ hasText: new RegExp(`^${escapeRegExp(text)}$`) }).last(),
      page.getByText(text, { exact: true }).last(),
    ];
    for (const locator of locators) {
      if (await waitVisible(locator, 2500) && await isReasonableAction(locator)) {
        return { locator, target: await locatorTarget(locator, text) };
      }
    }
  }
  return null;
}

async function waitForPublishOutcome(page, config, beforeURL, clickTarget) {
  const deadline = Date.now() + 30000;
  let lastStatus = await pageStatus(page);
  while (Date.now() < deadline) {
    lastStatus = await pageStatus(page);
    const bodyText = lastStatus.bodyText || '';
    const successText = firstIncluded(bodyText, config.successTexts || []);
    if (successText) {
      return {
        status: 'published',
        message: `${config.platformName}已确认提交：${successText}`,
        beforeURL,
        afterURL: lastStatus.pageUrl,
        clickTarget,
        matchedText: successText,
        externalUrl: extractLikelyExternalURL(lastStatus.pageUrl),
      };
    }

    const blockingText = firstIncluded(bodyText, config.blockingTexts || []);
    if (blockingText) {
      throw new Error(`${config.platformName} publish was blocked: ${blockingText}`);
    }

    const leftEditor = lastStatus.pageUrl !== beforeURL && !isEditorText(bodyText, config);
    if (leftEditor) {
      return {
        status: 'published',
        message: `已点击${config.platformName}发布按钮并离开编辑页，平台已接收提交。`,
        beforeURL,
        afterURL: lastStatus.pageUrl,
        clickTarget,
        leftEditor: true,
        externalUrl: extractLikelyExternalURL(lastStatus.pageUrl),
      };
    }

    await page.waitForTimeout(1000);
  }

  return {
    status: 'submitted_pending_verification',
    message: `已尝试点击${config.platformName}发布按钮，但未检测到明确成功提示，请人工核对。`,
    beforeURL,
    afterURL: lastStatus.pageUrl,
    clickTarget,
  };
}

function isEditorText(bodyText, config) {
  return (config.editorTexts || ['标题', '正文', '发布']).some((value) => bodyText.includes(value));
}

function firstIncluded(value, candidates) {
  return candidates.find((item) => value.includes(item)) || '';
}

function extractLikelyExternalURL(pageUrl) {
  if (/\/(article|a|i|item|news|profile|publish)\//i.test(pageUrl)) {
    return pageUrl;
  }
  return '';
}

async function waitVisible(locator, timeout) {
  try {
    await locator.waitFor({ state: 'visible', timeout });
    return true;
  } catch {
    return false;
  }
}

async function isReasonableAction(locator) {
  const box = await locator.boundingBox().catch(() => null);
  if (!box) {
    return false;
  }
  return box.width > 0 && box.height > 0 && box.width <= 320 && box.height <= 120;
}

async function locatorTarget(locator, label) {
  const box = await locator.boundingBox().catch(() => null);
  if (!box) {
    return { label };
  }
  return {
    label,
    x: Math.round(box.x + box.width / 2),
    y: Math.round(box.y + box.height / 2),
    width: Math.round(box.width),
    height: Math.round(box.height),
  };
}

async function saveScreenshot(page, config, prefix) {
  const file = path.join(config.screenshotDir, `${prefix}-${Date.now()}.png`);
  await page.screenshot({ path: file, fullPage: true });
  return file;
}

async function pageStatus(page) {
  return {
    title: await page.title().catch(() => ''),
    pageUrl: page.url(),
    bodyText: await page.locator('body').innerText({ timeout: 3000 }).catch(() => ''),
  };
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

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
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
