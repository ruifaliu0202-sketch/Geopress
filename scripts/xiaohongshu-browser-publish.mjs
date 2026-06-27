#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const profileDir = path.resolve(required(args, 'profile-dir'));
const title = required(args, 'title').trim();
const body = required(args, 'body').trim();
const publishMode = args['publish-mode'] || 'long_article';
const caption = (args.caption || shortCaption(body)).trim();
const chromePath = args['chrome-path'] || undefined;
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const publishUrl = args['publish-url'] || 'https://creator.xiaohongshu.com/publish/publish?from=homepage&target=article';
const screenshotDir = path.resolve(args['screenshot-dir'] || path.join(scriptDir, '..', 'runtime'));
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

if (publishMode !== 'long_article') {
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
  const networkCapture = createNetworkCapture(page);
  const publishOutcome = await publishLongArticle(page);
  await page.waitForTimeout(3000);
  const networkEvents = await networkCapture.flush();
  const publishIdentity = inferPublishIdentity(networkEvents, publishOutcome, page.url());
  const networkCapturePath = await saveNetworkCapture(networkEvents, publishIdentity);
  const screenshotPath = await saveScreenshot(page, publishOutcome.status === 'published' ? 'xhs-publish-verified' : 'xhs-publish-pending-verification');
  console.log(JSON.stringify({
    status: publishOutcome.status,
    message: publishOutcome.message,
    externalId: publishIdentity.externalId,
    externalUrl: publishIdentity.externalUrl,
    pageUrl: page.url(),
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
  const screenshotPath = page ? await saveScreenshot(page, 'xhs-publish-error').catch(() => '') : '';
  throw new Error(`${error.message}${screenshotPath ? ` (screenshot: ${screenshotPath})` : ''}`);
} finally {
  await context.close();
}

async function publishLongArticle(page) {
  await page.goto(publishUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);

  await ensureLoggedIn(page);
  await openLongArticleEditor(page);
  await fillLongArticleEditor(page);
  await clickExactText(page, '一键排版');
  await waitForAnyText(page, ['选择模板', '下一步'], 30000);
  await clickExactText(page, '下一步');
  await waitForFinalSettings(page);
  await fillFinalSettings(page);
  return clickPublish(page);
}

async function ensureLoggedIn(page) {
  const currentURL = page.url();
  if (/login/i.test(currentURL)) {
    throw new Error('Xiaohongshu browser profile is not logged in');
  }
  const hasLoginText = await page.getByText('登录', { exact: true }).count().catch(() => 0);
  const hasCreatorShell = await page.getByText('创作服务平台', { exact: true }).count().catch(() => 0);
  if (hasLoginText > 0 && hasCreatorShell === 0) {
    throw new Error('Xiaohongshu browser profile is not logged in');
  }
}

async function openLongArticleEditor(page) {
  if (await visibleCount(page.locator('textarea[placeholder="输入标题"]')) > 0) {
    return;
  }
  if (await visibleCount(page.getByText('新的创作', { exact: true })) > 0) {
    await clickExactText(page, '新的创作');
    await page.waitForTimeout(2500);
  }
  if (await visibleCount(page.locator('textarea[placeholder="输入标题"]')) === 0) {
    throw new Error('Xiaohongshu long article editor did not open');
  }
}

async function fillLongArticleEditor(page) {
  await page.locator('textarea[placeholder="输入标题"]').first().fill(title);
  const editor = page.locator('[contenteditable="true"]').first();
  await editor.waitFor({ state: 'visible', timeout: 15000 });
  await editor.fill(body);
  await page.waitForTimeout(800);
}

async function waitForFinalSettings(page) {
  await waitForAnyText(page, ['图片编辑', '内容设置', '笔记预览'], 30000);
  await page.locator('input[placeholder*="标题"]').first().waitFor({ state: 'visible', timeout: 20000 });
}

async function fillFinalSettings(page) {
  const titleInput = page.locator('input[placeholder*="标题"]').first();
  await titleInput.fill(title);

  const editors = page.locator('[contenteditable="true"]');
  if (await editors.count()) {
    await editors.first().fill(caption);
  }
  await page.waitForTimeout(800);
}

async function clickPublish(page) {
  const beforeURL = page.url();
  await page.keyboard.press('Escape').catch(() => undefined);
  await page.evaluate(() => {
    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur();
    }
  }).catch(() => undefined);

  await page.evaluate(() => {
    const publishPage = document.querySelector('.publish-page');
    if (publishPage) publishPage.scrollTop = publishPage.scrollHeight;
    window.scrollTo(0, document.body.scrollHeight);
  });
  await page.waitForTimeout(1000);

  const publishAction = await findPublishAction(page);
  let clickTarget;
  if (publishAction) {
    clickTarget = publishAction.target;
    await publishAction.locator.click();
  } else {
    clickTarget = await clickBottomPublishAction(page);
  }
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  return waitForPublishOutcome(page, beforeURL, clickTarget);
}

async function findPublishAction(page) {
  const locators = [
    page.getByRole('button', { name: /^发布$/ }).last(),
    page.locator('button').filter({ hasText: /^发布$/ }).last(),
    page.locator('[role="button"]').filter({ hasText: /^发布$/ }).last(),
    page.locator('.publishBtn, .publish-btn').filter({ hasText: /^发布$/ }).last(),
    page.getByText('发布', { exact: true }).last(),
  ];

  for (const locator of locators) {
    if (await waitVisible(locator, 3000) && await isReasonablePublishAction(locator)) {
      return { locator, target: await locatorTarget(locator, 'selector') };
    }
  }

  const handle = await page.evaluateHandle(() => {
    const isVisible = (element) => {
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return style.visibility !== 'hidden'
        && style.display !== 'none'
        && rect.width > 0
        && rect.height > 0;
    };
    const textParents = [];
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const node = walker.currentNode;
      if (node.textContent?.trim() === '发布' && node.parentElement) {
        textParents.push(node.parentElement);
      }
    }
    const candidates = Array.from(new Set([
      ...textParents,
      ...Array.from(document.querySelectorAll('button, [role="button"], a, span, div'))
        .filter((element) => element.textContent?.trim() === '发布'),
    ]))
      .filter((element) => isVisible(element))
      .map((element) => {
        const action = element.closest('button, [role="button"], a') || element;
        const rect = action.getBoundingClientRect();
        return { action, bottom: rect.bottom, area: rect.width * rect.height, rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } };
      })
      .filter((item) => item.area > 0 && item.rect.y > window.innerHeight * 0.55 && item.rect.width <= 260 && item.rect.height <= 90)
      .sort((left, right) => right.bottom - left.bottom || right.area - left.area);
    return candidates[0]?.action || null;
  });
  const element = handle.asElement();
  if (element) {
    return { locator: element, target: await locatorTarget(element, 'dom-text') };
  }

  return null;
}

async function clickBottomPublishAction(page) {
  await assertFinalPublishSettings(page);
  const viewport = page.viewportSize() || { width: 1440, height: 1000 };
  const x = Math.round(viewport.width / 2 + 30);
  const y = Math.round(viewport.height - 44);
  await page.mouse.click(x, y);
  return { method: 'coordinate-fallback', x, y };
}

async function assertFinalPublishSettings(page) {
  const status = await pageStatus(page);
  const bodyText = status.bodyText || '';
  const requiredTexts = ['内容设置', '更多设置'];
  const missing = requiredTexts.filter((value) => !bodyText.includes(value));
  if (missing.length > 0) {
    throw new Error(`Xiaohongshu publish button was not found and final settings page is not confirmed: missing ${missing.join(', ')}`);
  }
  if (!bodyText.includes('发布') && !bodyText.includes('暂存')) {
    throw new Error('Xiaohongshu publish button was not found and final action bar is not confirmed');
  }
}

async function waitVisible(locator, timeout) {
  try {
    await locator.waitFor({ state: 'visible', timeout });
    return true;
  } catch {
    return false;
  }
}

async function isReasonablePublishAction(locator) {
  const box = await locator.boundingBox().catch(() => null);
  if (!box) {
    return false;
  }
  return box.y > 500 && box.width <= 260 && box.height <= 90;
}

async function locatorTarget(locator, method) {
  const box = await locator.boundingBox().catch(() => null);
  if (!box) {
    return { method };
  }
  return {
    method,
    x: Math.round(box.x + box.width / 2),
    y: Math.round(box.y + box.height / 2),
    width: Math.round(box.width),
    height: Math.round(box.height),
  };
}

async function waitForPublishOutcome(page, beforeURL, clickTarget) {
  const deadline = Date.now() + 30000;
  let lastStatus = await pageStatus(page);
  while (Date.now() < deadline) {
    lastStatus = await pageStatus(page);
    const bodyText = lastStatus.bodyText || '';
    const successText = firstIncluded(bodyText, [
      '发布成功',
      '发布完成',
      '提交成功',
      '笔记已发布',
      '发布成功，请等待审核',
      '发布成功，待审核',
      '审核中',
      '已提交审核',
      '提交审核成功',
    ]);
    if (successText) {
      return {
        status: 'published',
        message: `小红书已确认提交：${successText}`,
        beforeURL,
        afterURL: lastStatus.pageUrl,
        clickTarget,
        matchedText: successText,
      };
    }

    const blockingText = firstIncluded(bodyText, [
      '发布失败',
      '提交失败',
      '请完成验证',
      '验证码',
      '账号异常',
      '内容不能为空',
      '标题不能为空',
      '请添加正文',
      '请填写标题',
      '请稍后重试',
    ]);
    if (blockingText) {
      throw new Error(`Xiaohongshu publish was blocked: ${blockingText}`);
    }

    const leftEditor = lastStatus.pageUrl !== beforeURL && !bodyText.includes('内容设置') && !bodyText.includes('更多设置');
    if (leftEditor) {
      return {
        status: 'published',
        message: '已点击小红书发布按钮并离开编辑页，平台已接收提交。',
        beforeURL,
        afterURL: lastStatus.pageUrl,
        clickTarget,
        leftEditor: true,
      };
    }

    await page.waitForTimeout(1000);
  }

  const stillOnFinalSettings = (lastStatus.bodyText || '').includes('内容设置')
    && (lastStatus.bodyText || '').includes('更多设置')
    && (lastStatus.bodyText || '').includes('暂存离开');
  return {
    status: 'submitted_pending_verification',
    message: stillOnFinalSettings
      ? '已尝试点击小红书发布按钮，但页面仍停留在发布设置页，未检测到平台确认提交，请人工核对。'
      : '已尝试点击小红书发布按钮，但未检测到明确成功提示，请人工核对。',
    beforeURL,
    afterURL: lastStatus.pageUrl,
    clickTarget,
    stillOnFinalSettings,
  };
}

function firstIncluded(value, candidates) {
  return candidates.find((item) => value.includes(item)) || '';
}

async function clickExactText(page, text) {
  const locator = page.getByText(text, { exact: true }).first();
  await locator.waitFor({ state: 'visible', timeout: 15000 });
  await locator.click();
}

async function waitForAnyText(page, values, timeout) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    for (const value of values) {
      if (await visibleCount(page.getByText(value, { exact: true })) > 0) {
        return;
      }
    }
    await page.waitForTimeout(500);
  }
  throw new Error(`Timed out waiting for one of: ${values.join(', ')}`);
}

async function visibleCount(locator) {
  const count = await locator.count().catch(() => 0);
  let visible = 0;
  for (let index = 0; index < count; index += 1) {
    if (await locator.nth(index).isVisible().catch(() => false)) {
      visible += 1;
    }
  }
  return visible;
}

async function saveScreenshot(page, prefix) {
  const file = path.join(screenshotDir, `${prefix}-${Date.now()}.png`);
  await page.screenshot({ path: file, fullPage: true });
  return file;
}

async function saveNetworkCapture(events, identity) {
  const file = path.join(screenshotDir, `xhs-publish-network-${Date.now()}.json`);
  await writeFile(file, JSON.stringify({ identity, events }, null, 2), 'utf8');
  return file;
}

async function pageStatus(page) {
  return {
    title: await page.title().catch(() => ''),
    pageUrl: page.url(),
    bodyText: await page.locator('body').innerText({ timeout: 3000 }).catch(() => ''),
  };
}

function createNetworkCapture(page) {
  const events = [];
  const pending = new Set();
  page.on('response', (response) => {
    const task = captureNetworkResponse(response, events).catch(() => undefined);
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

async function captureNetworkResponse(response, events) {
  const url = response.url();
  if (!isXiaohongshuResponse(url)) {
    return;
  }
  const request = response.request();
  const method = request.method();
  const status = response.status();
  const contentType = response.headers()['content-type'] || '';
  const urlInteresting = /api|note|publish|post|item|creator|edith|sns|submit|create|web\/v/i.test(url);
  if (!urlInteresting && !/json/i.test(contentType)) {
    return;
  }

  const bodyText = await response.text().catch(() => '');
  if (!bodyText || bodyText.length > 250000) {
    return;
  }

  let parsed = null;
  if (/^\s*[\[{]/.test(bodyText)) {
    parsed = JSON.parse(bodyText);
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
    requestPostDataSample: requestPostData ? requestPostData.slice(0, 2000) : '',
    bodySample: bodyText.slice(0, 3000),
  });
  if (events.length > 120) {
    events.splice(0, events.length - 120);
  }
}

function isXiaohongshuResponse(value) {
  try {
    const host = new URL(value).hostname;
    return /(^|\.)xiaohongshu\.com$|(^|\.)xhscdn\.com$/.test(host);
  } catch {
    return false;
  }
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
  if (/^(noteid|itemid|publishid|postid|opusid|objectid|contentid|taskid|noteids|itemids)$/.test(normalized)) {
    return true;
  }
  if (normalized === 'id') {
    return pathParts.some((item) => /note|item|publish|post|opus|object|content|task|data/i.test(item));
  }
  return false;
}

function isCandidateIDValue(value) {
  if (value.length < 6 || value.length > 80) {
    return false;
  }
  return /^[A-Za-z0-9_-]+$/.test(value);
}

function inferPublishIdentity(events, publishOutcome, pageUrl) {
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
          score: scoreIDCandidate(candidate, event.url),
        });
      }
    }
  }
  const candidates = [...deduped.values()].sort((left, right) => right.score - left.score).slice(0, 20);
  const best = candidates[0] || null;
  const note = candidates.find((candidate) => /note/i.test(`${candidate.key}.${candidate.path}.${candidate.url}`)) || null;
  const externalId = best?.value || '';
  const externalUrl = note?.value ? `https://www.xiaohongshu.com/explore/${note.value}` : '';
  return {
    externalId,
    externalUrl,
    inferredFrom: best ? { key: best.key, path: best.path, url: best.url, score: best.score } : null,
    noteIdCandidate: note ? { key: note.key, path: note.path, value: note.value, url: note.url, score: note.score } : null,
    pageUrl,
    publishOutcomeStatus: publishOutcome.status,
    candidates,
  };
}

function scoreIDCandidate(candidate, url) {
  const target = `${candidate.key}.${candidate.path}.${url}`.toLowerCase();
  let score = 0;
  if (/note[_-]?id|noteid/.test(target)) score += 100;
  if (/item[_-]?id|itemid/.test(target)) score += 75;
  if (/publish[_-]?id|publishid/.test(target)) score += 60;
  if (/post[_-]?id|postid|opus[_-]?id|opusid/.test(target)) score += 55;
  if (/task[_-]?id|taskid/.test(target)) score += 35;
  if (/\/web_api\/sns\/v2\/note/i.test(url) && candidate.path === 'data.id') score += 120;
  if (/publish|note|post|item|submit|create/.test(url)) score += 20;
  if (candidate.key.toLowerCase() === 'id') score -= 15;
  if (/user|avatar|image|file|upload|material|template/.test(target)) score -= 40;
  return score;
}

function shortCaption(value) {
  const first = value
    .split(/\n+/)
    .map((item) => item.trim())
    .find(Boolean) || value.trim();
  const hashtags = Array.from(value.matchAll(/#[\p{L}\p{N}_-]+/gu)).map((item) => item[0]);
  const base = first.length > 220 ? `${first.slice(0, 220)}...` : first;
  return `${base}${hashtags.length > 0 ? `\n\n${hashtags.slice(0, 5).join(' ')}` : ''}`;
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
