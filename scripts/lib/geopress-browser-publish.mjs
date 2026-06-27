import { mkdir, writeFile } from 'node:fs/promises';
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
  const userAgent = args['user-agent'] || config.userAgent || undefined;
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
    userAgent,
    args: chromiumArgs,
  });

  let page;
  try {
    page = context.pages()[0] ?? (await context.newPage());
    const runtime = { ...config, profileDir, title, body, publishMode, publishUrl, screenshotDir };
    const networkCapture = createNetworkCapture(page, runtime);
    const publishOutcome = await publishArticle(page, runtime);
    await page.waitForTimeout(Number(config.postPublishNetworkSettleMs || 3000));
    const networkEvents = await networkCapture.flush();
    const publishIdentity = inferPublishIdentity(networkEvents, publishOutcome, page.url(), runtime);
    const networkCapturePath = await saveNetworkCapture(networkEvents, publishIdentity, runtime);
    const screenshotPath = await saveScreenshot(page, runtime, publishOutcome.status === 'published' ? `${config.platform}-publish-verified` : `${config.platform}-publish-pending`);
    console.log(JSON.stringify({
      status: publishOutcome.status,
      message: publishOutcome.message,
      pageUrl: page.url(),
      externalUrl: firstNonEmpty(publishOutcome.externalUrl, publishIdentity.externalUrl),
      externalId: firstNonEmpty(publishOutcome.externalId, publishIdentity.externalId),
      screenshotPath,
      submittedAt: new Date().toISOString(),
      rawStatus: {
        ...(await pageStatus(page)),
        publishOutcome,
        publishIdentity,
        networkCapturePath,
        networkCandidates: publishIdentity.candidates,
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

async function saveNetworkCapture(events, identity, config) {
  const file = path.join(config.screenshotDir, `${config.platform}-publish-network-${Date.now()}.json`);
  await writeFile(file, JSON.stringify(sanitizeForOutput({ identity, events }), null, 2), 'utf8');
  return file;
}

async function pageStatus(page) {
  return {
    title: await page.title().catch(() => ''),
    pageUrl: page.url(),
    bodyText: await page.locator('body').innerText({ timeout: 3000 }).catch(() => ''),
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
  const urlInteresting = isInterestingPublishURL(url, config);
  if (!urlInteresting && !/json/i.test(contentType)) {
    return;
  }

  const bodyText = await response.text().catch(() => '');
  if (!bodyText || bodyText.length > Number(config.maxCaptureBodyBytes || 250000)) {
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
  const idCandidates = parsed ? extractIDCandidates(parsed) : [];
  if (!urlInteresting && idCandidates.length === 0) {
    return;
  }

  const requestPostData = method !== 'GET' && urlInteresting ? request.postData() || '' : '';
  events.push({
    at: new Date().toISOString(),
    url,
    method,
    status,
    resourceType: request.resourceType(),
    contentType: contentType.split(';')[0],
    idCandidates: idCandidates.slice(0, 30),
    requestPostDataSample: requestPostData ? sanitizeText(requestPostData.slice(0, 2000)) : '',
    bodySample: parsed
      ? JSON.stringify(sanitizeForOutput(parsed)).slice(0, 3000)
      : sanitizeText(bodyText.slice(0, 3000)),
  });
  if (events.length > Number(config.maxNetworkEvents || 120)) {
    events.splice(0, events.length - Number(config.maxNetworkEvents || 120));
  }
}

function isPlatformResponse(value, config) {
  try {
    const host = new URL(value).hostname;
    if (config.responseHostPattern && config.responseHostPattern.test(host)) {
      return true;
    }
    const publishHost = new URL(config.publishUrl).hostname;
    return host === publishHost || host.endsWith(`.${publishHost}`);
  } catch {
    return false;
  }
}

function isInterestingPublishURL(value, config) {
  if (config.publishResponseURLPattern && config.publishResponseURLPattern.test(value)) {
    return true;
  }
  return /api|article|content|item|news|publish|post|submit|create|save|draft|opus|doc/i.test(value);
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
    if (!isCandidateIDKey(key, path) || !isCandidateIDValue(text)) {
      return;
    }
    const signature = `${path.join('.')}:${text}`;
    if (seen.has(signature)) {
      return;
    }
    seen.add(signature);
    result.push({
      path: path.join('.'),
      key,
      value: text,
    });
  };

  visit(value, pathParts);
  return result;
}

function isCandidateIDKey(key, pathParts) {
  const normalized = key.replace(/[-_]/g, '').toLowerCase();
  if (/^(articleid|itemid|newsid|docid|docidstr|postid|opusid|contentid|groupid|objectid|publishid|taskid)$/.test(normalized)) {
    return true;
  }
  if (normalized === 'id') {
    return pathParts.some((item) => /article|item|news|doc|publish|post|opus|object|content|task|data|result/i.test(item));
  }
  return false;
}

function isCandidateIDValue(value) {
  if (value.length < 6 || value.length > 96) {
    return false;
  }
  return /^[A-Za-z0-9_-]+$/.test(value);
}

function inferPublishIdentity(events, publishOutcome, pageUrl, config) {
  const deduped = new Map();
  for (const event of events) {
    for (const candidate of event.idCandidates || []) {
      const key = `${candidate.path}:${candidate.value}`;
      if (!deduped.has(key)) {
        deduped.set(key, {
          ...candidate,
          url: event.url,
          method: event.method,
          status: event.status,
          score: scoreIDCandidate(candidate, event.url, config),
        });
      }
    }
  }
  const candidates = [...deduped.values()].sort((left, right) => right.score - left.score).slice(0, 20);
  const best = candidates[0] || null;
  const minScore = Number(config.identityMinScore || 50);
  const strongBest = best && best.score >= minScore && isStrongContentIDCandidate(best);
  const externalId = strongBest ? best.value : '';
  return {
    externalId,
    externalUrl: firstNonEmpty(extractLikelyExternalURL(pageUrl), publishOutcome.externalUrl),
    status: externalId ? 'matched' : 'pending_reconcile',
    inferredFrom: strongBest ? { key: best.key, path: best.path, url: best.url, score: best.score } : null,
    pageUrl,
    publishOutcomeStatus: publishOutcome.status,
    candidates,
  };
}

function scoreIDCandidate(candidate, url, config) {
  const target = `${candidate.key}.${candidate.path}.${url}`.toLowerCase();
  let score = 0;
  if (/article[_-]?id|articleid|news[_-]?id|newsid|doc[_-]?id|docid/.test(target)) score += 110;
  if (/item[_-]?id|itemid|group[_-]?id|groupid/.test(target)) score += 85;
  if (/post[_-]?id|postid|opus[_-]?id|opusid|content[_-]?id|contentid/.test(target)) score += 80;
  if (/publish[_-]?id|publishid/.test(target)) score += 45;
  if (/task[_-]?id|taskid/.test(target)) score += 20;
  if (/\/(api|apiv?|mp|creator|profile|publish|article|content|item|news|submit|create|save)/.test(url.toLowerCase())) score += 25;
  if (/publish|submit|create|save|article|content|item|news|post|doc/.test(url.toLowerCase())) score += 25;
  if (candidate.key.toLowerCase() === 'id') score -= 15;
  if (/user|account|author|avatar|image|file|upload|material|template|category|comment|tag|topic/.test(target)) score -= 55;
  if (config.identityScoreAdjust) {
    score += Number(config.identityScoreAdjust(candidate, url) || 0);
  }
  return score;
}

function isStrongContentIDCandidate(candidate) {
  const target = `${candidate.key}.${candidate.path}.${candidate.url}`.toLowerCase();
  if (/article[_-]?id|articleid|news[_-]?id|newsid|doc[_-]?id|docid|item[_-]?id|itemid|post[_-]?id|postid|opus[_-]?id|opusid|content[_-]?id|contentid|group[_-]?id|groupid/.test(target)) {
    return true;
  }
  return candidate.key.toLowerCase() === 'id'
    && /article|content|item|news|post|doc|publish|submit|create|save/.test(target)
    && !/user|account|author|avatar|image|file|upload|material|template|category|comment|tag|topic/.test(target);
}

function sanitizeForOutput(value, pathParts = []) {
  if (Array.isArray(value)) {
    return value.slice(0, 50).map((item, index) => sanitizeForOutput(item, [...pathParts, String(index)]));
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

function firstNonEmpty(...values) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }
  return '';
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
