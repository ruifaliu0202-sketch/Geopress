#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const args = parseArgs(process.argv.slice(2));
const profileDir = path.resolve(required(args, 'profile-dir'));
const externalContentId = required(args, 'external-content-id').trim();
const externalUrl = (args['external-url'] || '').trim();
const title = (args.title || '').trim();
const contentId = (args['content-id'] || '').trim();
const publishJobId = (args['publish-job-id'] || '').trim();
const chromePath = args['chrome-path'] || process.env.GEOPRESS_CHROME_PATH || undefined;
const outputFile = args.output ? path.resolve(args.output) : '';
const debugDir = args['debug-dir'] ? path.resolve(args['debug-dir']) : '';
const timeoutMs = Number(args['timeout-ms'] || 60000);
const settleMs = Number(args['settle-ms'] || 2500);
const screenshotDir = path.resolve(args['screenshot-dir'] || debugDir || path.join(scriptDir, '..', 'runtime'));
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const noteManagerUrl = args['note-manager-url'] || 'https://creator.xiaohongshu.com/new/note-manager';
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

let context;
try {
  const playwright = await importPlaywright();
  await mkdir(profileDir, { recursive: true });
  await mkdir(screenshotDir, { recursive: true });
  if (debugDir) {
    await mkdir(debugDir, { recursive: true });
  }

  context = await playwright.chromium.launchPersistentContext(profileDir, {
    executablePath: chromePath,
    headless,
    viewport: { width: 1440, height: 960 },
    locale: 'zh-CN',
    args: chromiumArgs,
  });

  const page = context.pages()[0] ?? (await context.newPage());
  const result = await collectContentMetadata(page);
  await writeResult(result);
  process.stdout.write(`${JSON.stringify(result)}\n`);
  if (!result.ok && result.status !== 'pending_reconcile') {
    process.exitCode = 2;
  }
} catch (error) {
  const result = {
    ok: false,
    platform: 'xiaohongshu',
    status: isProfileInUseError(error) ? 'profile_in_use' : 'content_metadata_failed',
    profileDir,
    externalContentId,
    externalUrl,
    title,
    contentId,
    publishJobId,
    capturedAt: new Date().toISOString(),
    error: error instanceof Error ? error.message : String(error),
  };
  await writeResult(result).catch(() => undefined);
  process.stderr.write(`${JSON.stringify(result)}\n`);
  process.exitCode = 1;
} finally {
  await context?.close().catch(() => undefined);
}

async function collectContentMetadata(page) {
  const networkCapture = createNetworkCapture(page);
  await page.goto(noteManagerUrl, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(settleMs);

  const loginState = await getLoginState(page);
  if (!loginState.loggedIn) {
    return finalizeResult(page, networkCapture, {
      ok: false,
      status: 'not_logged_in',
      dataSource: 'browser_creator_note_manager',
      loginState,
      error: 'xiaohongshu creator session is not logged in',
    });
  }

  const listMatch = await findFromCapturedAndDirectRequests(page, networkCapture);
  if (listMatch) {
    return finalizeResult(page, networkCapture, {
      ok: true,
      status: 'collected',
      dataSource: listMatch.dataSource || 'browser_creator_note_manager',
      loginState,
      metadata: listMatch.metadata,
    });
  }

  const baseDetailMatch = await requestNoteBaseDetail(page);
  if (baseDetailMatch) {
    return finalizeResult(page, networkCapture, {
      ok: true,
      status: 'collected',
      dataSource: baseDetailMatch.dataSource || 'browser_creator_note_base_detail',
      loginState,
      metadata: baseDetailMatch.metadata,
    });
  }

  const detailMatch = await requestNoteDetail(page);
  if (detailMatch) {
    return finalizeResult(page, networkCapture, {
      ok: true,
      status: 'collected_low_confidence',
      dataSource: detailMatch.dataSource || 'browser_creator_note_detail',
      loginState,
      metadata: detailMatch.metadata,
    });
  }

  const searchMatch = await searchByTitle(page, networkCapture);
  if (searchMatch) {
    return finalizeResult(page, networkCapture, {
      ok: true,
      status: 'collected',
      dataSource: searchMatch.dataSource || 'browser_creator_note_search',
      loginState,
      metadata: searchMatch.metadata,
    });
  }

  return finalizeResult(page, networkCapture, {
    ok: false,
    status: 'pending_reconcile',
    dataSource: 'browser_creator_note_manager',
    loginState,
    error: 'xiaohongshu note is not visible in creator note manager yet',
  });
}

async function findFromCapturedAndDirectRequests(page, networkCapture) {
  const capturedEvents = await networkCapture.flush(1000);
  const capturedMatch = noteFromEvents(capturedEvents);
  if (capturedMatch) {
    return capturedMatch;
  }

  // 小红书后台接口对直接 fetch 的风控较敏感；这里的请求只作为补充诊断，主路径仍是页面自然触发。
  for (const tab of [0, 1, 2, 3]) {
    const url = `https://creator.xiaohongshu.com/api/galaxy/v2/creator/note/user/posted?tab=${tab}&page=0`;
    const response = await fetchInPage(page, url);
    if (response.parsed?.success) {
      const match = noteFromListResponse(response.parsed, {
        dataSource: 'browser_context_request_note_list',
        sourceUrl: url,
      });
      if (match) {
        return match;
      }
    }
  }
  return null;
}

async function requestNoteBaseDetail(page) {
  const url = `https://creator.xiaohongshu.com/api/galaxy/creator/datacenter/note/base?note_id=${encodeURIComponent(externalContentId)}`;
  const response = await fetchInPage(page, url);
  if (!response.parsed?.success || !response.parsed?.data) {
    return null;
  }
  const metadata = metadataFromBaseDetailData(response.parsed.data, {
    sourceUrl: url,
    confidence: 'high',
    matchStrategy: 'external_content_id_note_base',
  });
  if (!metadata) {
    return null;
  }
  return {
    dataSource: 'browser_context_request_note_base',
    metadata,
  };
}

async function requestNoteDetail(page) {
  const url = `https://creator.xiaohongshu.com/api/galaxy/creator/data/note_detail_new?note_id=${encodeURIComponent(externalContentId)}`;
  const response = await fetchInPage(page, url);
  if (!response.parsed?.success || !response.parsed?.data) {
    return null;
  }
  const metadata = metadataFromDetailData(response.parsed.data, {
    sourceUrl: url,
    confidence: 'low',
    matchStrategy: 'external_content_id_note_detail',
  });
  if (!metadata) {
    return null;
  }
  return {
    dataSource: 'browser_context_request_note_detail',
    metadata,
  };
}

async function searchByTitle(page, networkCapture) {
  const keywords = searchKeywords(title);
  if (keywords.length === 0) {
    return null;
  }
  const input = page.getByPlaceholder(/搜索已发布的笔记|搜索/).first();
  const hasInput = await input.isVisible({ timeout: 3000 }).catch(() => false);
  if (!hasInput) {
    return null;
  }

  for (const keyword of keywords) {
    await input.fill('').catch(() => undefined);
    await input.fill(keyword);
    await page.keyboard.press('Enter');
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => undefined);
    await page.waitForTimeout(settleMs);
    const events = await networkCapture.flush(1500);
    const match = noteFromEvents(events);
    if (match) {
      match.metadata.matchKeyword = keyword;
      return match;
    }
  }
  return null;
}

async function finalizeResult(page, networkCapture, partial) {
  const events = await networkCapture.flush();
  const pageInfo = await pageStatus(page);
  const screenshotPath = await saveScreenshot(page, statusToScreenshotPrefix(partial.status)).catch(() => '');
  const result = {
    ok: Boolean(partial.ok),
    platform: 'xiaohongshu',
    status: partial.status || 'unknown',
    profileDir,
    externalContentId,
    externalUrl,
    title,
    contentId,
    publishJobId,
    noteManagerUrl,
    pageUrl: pageInfo.pageUrl,
    pageTitle: pageInfo.title,
    capturedAt: new Date().toISOString(),
    dataSource: partial.dataSource || '',
    loginState: sanitizeForOutput(partial.loginState || {}),
    metadata: partial.metadata || null,
    diagnostics: {
      error: partial.error || '',
      screenshotPath,
      eventCount: events.length,
      relevantEvents: events.slice(-30).map(summarizeEvent),
    },
  };
  if (debugDir) {
  await writeFile(path.join(debugDir, 'content-metadata-result.json'), `${JSON.stringify(sanitizeForOutput(result), null, 2)}\n`, 'utf8');
  }
  return sanitizeForOutput(result);
}

function noteFromEvents(events) {
  for (const event of [...events].reverse()) {
    if (!event.parsed) {
      continue;
    }
    if (/datacenter\/note\/base/i.test(event.url)) {
      const metadata = metadataFromBaseDetailData(event.parsed.data, {
        sourceUrl: event.url,
        confidence: 'high',
        matchStrategy: 'external_content_id_note_base',
      });
      if (metadata) {
        return {
          dataSource: 'browser_creator_note_base_detail',
          metadata,
        };
      }
    }
    const match = noteFromListResponse(event.parsed, {
      dataSource: event.url.includes('/search') ? 'browser_creator_note_search' : 'browser_creator_note_manager',
      sourceUrl: event.url,
    });
    if (match) {
      return match;
    }
  }
  return null;
}

function noteFromListResponse(parsed, options) {
  const notes = parsed?.data?.notes;
  if (!Array.isArray(notes)) {
    return null;
  }
  const normalizedTitle = normalizeText(title);
  const candidates = notes
    .map((note) => ({ note, score: scoreNoteMatch(note, normalizedTitle) }))
    .filter((item) => item.score > 0)
    .sort((left, right) => right.score - left.score);
  if (candidates.length === 0) {
    return null;
  }
  const best = candidates[0];
  return {
    dataSource: options.dataSource,
    metadata: metadataFromListNote(best.note, {
      sourceUrl: options.sourceUrl,
      confidence: best.score >= 100 ? 'high' : 'medium',
      matchStrategy: best.note.id === externalContentId ? 'external_content_id' : 'title_keyword',
      matchScore: best.score,
    }),
  };
}

function metadataFromListNote(note, options) {
  const noteId = stringValue(note.id || note.note_id || note.noteId || externalContentId);
  const displayTitle = stringValue(note.display_title || note.title || note.desc || title);
  const capturedAt = new Date().toISOString();
  const viewCount = numberValue(note.view_count);
  const likeCount = numberValue(note.likes ?? note.like_count);
  const commentCount = numberValue(note.comments_count ?? note.comment_count);
  const favoriteCount = numberValue(note.collected_count ?? note.collect_count);
  const shareCount = numberValue(note.shared_count ?? note.share_count);
  return {
    externalContentId: noteId,
    externalUrl: firstNonEmpty(externalUrl, noteId ? `https://www.xiaohongshu.com/explore/${noteId}` : ''),
    title: displayTitle,
    status: note.permission_msg ? 'warning' : 'published_visible',
    statusText: stringValue(note.permission_msg || ''),
    publishedAt: timeFromSecondsOrMs(note.visible_time || note.post_time || note.user_update_time),
    capturedAt,
    confidence: options.confidence,
    matchStrategy: options.matchStrategy,
    matchScore: options.matchScore || 0,
    sourceUrl: options.sourceUrl,
    metrics: {
      impressionCount: numberValue(note.imp_count ?? note.impl_count),
      viewCount,
      likeCount,
      commentCount,
      shareCount,
      favoriteCount,
      clickCount: numberValue(note.click_count),
      engagementRate: engagementRate(viewCount, likeCount, commentCount, shareCount, favoriteCount),
    },
    rawMetrics: {
      note: sanitizeForOutput(note),
      sourceUrl: options.sourceUrl,
      sourceType: options.matchStrategy,
    },
  };
}

function metadataFromBaseDetailData(data, options) {
  const noteInfo = data.note_info || data.noteInfo || {};
  const noteId = stringValue(noteInfo.id || externalContentId);
  if (noteId && noteId !== externalContentId) {
    return null;
  }
  const viewCount = numberValue(data.view_count ?? noteInfo.view_count);
  const likeCount = numberValue(data.like_count ?? noteInfo.like_count);
  const commentCount = numberValue(data.comment_count ?? noteInfo.comment_count);
  const shareCount = numberValue(data.share_count ?? noteInfo.share_count);
  const favoriteCount = numberValue(data.collect_count ?? noteInfo.collect_count);
  return {
    externalContentId: noteId,
    externalUrl: firstNonEmpty(externalUrl, noteId ? `https://www.xiaohongshu.com/explore/${noteId}` : ''),
    title: stringValue(noteInfo.desc || noteInfo.title || title),
    status: 'base_detail_visible',
    statusText: '',
    publishedAt: timeFromSecondsOrMs(noteInfo.post_time || noteInfo.user_update_time),
    capturedAt: new Date().toISOString(),
    confidence: options.confidence,
    matchStrategy: options.matchStrategy,
    matchScore: 100,
    sourceUrl: options.sourceUrl,
    metrics: {
      impressionCount: numberValue(data.imp_count ?? data.impl_count),
      viewCount,
      likeCount,
      commentCount,
      shareCount,
      favoriteCount,
      clickCount: numberValue(data.click_count),
      engagementRate: engagementRate(viewCount, likeCount, commentCount, shareCount, favoriteCount),
    },
    rawMetrics: {
      detail: sanitizeForOutput(data),
      sourceUrl: options.sourceUrl,
      sourceType: options.matchStrategy,
    },
  };
}

function metadataFromDetailData(data, options) {
  const seven = data.seven || {};
  const thirty = data.thirty || {};
  const noteInfo = data.note_info || data.noteInfo || {};
  const source = hasAnyMetric(seven) ? seven : hasAnyMetric(thirty) ? thirty : data;
  const viewCount = numberValue(source.view_count ?? noteInfo.view_count);
  const likeCount = numberValue(source.like_count ?? noteInfo.like_count);
  const commentCount = numberValue(source.comment_count ?? noteInfo.comment_count);
  const shareCount = numberValue(source.share_count ?? noteInfo.share_count);
  const favoriteCount = numberValue(source.collect_count ?? data.collect_count);
  return {
    externalContentId,
    externalUrl: firstNonEmpty(externalUrl, externalContentId ? `https://www.xiaohongshu.com/explore/${externalContentId}` : ''),
    title: stringValue(noteInfo.desc || noteInfo.title || title),
    status: 'detail_visible',
    statusText: '',
    publishedAt: timeFromSecondsOrMs(noteInfo.post_time || noteInfo.user_update_time),
    capturedAt: new Date().toISOString(),
    confidence: options.confidence,
    matchStrategy: options.matchStrategy,
    matchScore: 60,
    sourceUrl: options.sourceUrl,
    metrics: {
      impressionCount: numberValue(source.imp_count ?? source.impl_count),
      viewCount,
      likeCount,
      commentCount,
      shareCount,
      favoriteCount,
      clickCount: numberValue(source.click_count),
      engagementRate: engagementRate(viewCount, likeCount, commentCount, shareCount, favoriteCount),
    },
    rawMetrics: {
      detail: sanitizeForOutput(data),
      sourceUrl: options.sourceUrl,
      sourceType: options.matchStrategy,
    },
  };
}

function scoreNoteMatch(note, normalizedTitle) {
  const noteId = stringValue(note.id || note.note_id || note.noteId);
  if (noteId && noteId === externalContentId) {
    return 120;
  }
  const displayTitle = normalizeText(note.display_title || note.title || note.desc || '');
  if (!displayTitle || !normalizedTitle) {
    return 0;
  }
  if (displayTitle === normalizedTitle) {
    return 90;
  }
  if (displayTitle.includes(normalizedTitle) || normalizedTitle.includes(displayTitle)) {
    return 70;
  }
  for (const keyword of searchKeywords(title)) {
    if (keyword.length >= 4 && displayTitle.includes(normalizeText(keyword))) {
      return 45;
    }
  }
  return 0;
}

function searchKeywords(value) {
  const normalized = normalizeText(value);
  if (!normalized) {
    return [];
  }
  const parts = normalized
    .split(/[\s:：,，.。!！?？#]+/)
    .map((item) => item.trim())
    .filter((item) => item.length >= 4);
  const result = [];
  if (parts.length > 0) {
    result.push(parts[0]);
  }
  for (const part of parts) {
    if (!result.includes(part)) {
      result.push(part);
    }
    if (result.length >= 4) {
      break;
    }
  }
  if (!result.includes(normalized) && normalized.length <= 18) {
    result.push(normalized);
  }
  return result.slice(0, 5);
}

async function fetchInPage(page, url) {
  return page.evaluate(async (target) => {
    try {
      const response = await fetch(target, { credentials: 'include' });
      const bodyText = await response.text();
      let parsed = null;
      try {
        parsed = JSON.parse(bodyText);
      } catch {
        parsed = null;
      }
      return {
        ok: response.ok,
        status: response.status,
        url: response.url,
        bodyText: bodyText.slice(0, 2000),
        parsed,
      };
    } catch (error) {
      return {
        ok: false,
        status: 0,
        url: target,
        bodyText: error instanceof Error ? error.message : String(error),
        parsed: null,
      };
    }
  }, url);
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
    async flush(waitMs = 3000) {
      const deadline = Date.now() + waitMs;
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
  const contentType = response.headers()['content-type'] || '';
  const interesting = /creator\/note\/user\/posted|note\/managemaent\/search|note_detail_new|datacenter\/note\/base/i.test(url);
  if (!interesting) {
    return;
  }
  const bodyText = await response.text().catch(() => '');
  if (!bodyText || bodyText.length > 500000) {
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
  events.push({
    at: new Date().toISOString(),
    url,
    method,
    status: response.status(),
    contentType: contentType.split(';')[0],
    parsed,
    bodySample: bodyText.slice(0, 2000),
  });
  if (events.length > 120) {
    events.splice(0, events.length - 120);
  }
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

async function pageStatus(page) {
  return {
    title: await page.title().catch(() => ''),
    pageUrl: page.url(),
  };
}

async function saveScreenshot(page, prefix) {
  const file = path.join(screenshotDir, `${prefix}-${Date.now()}.png`);
  await page.screenshot({ path: file, fullPage: true });
  return file;
}

async function writeResult(result) {
  if (!outputFile) {
    return;
  }
  await mkdir(path.dirname(outputFile), { recursive: true });
  await writeFile(outputFile, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
}

function summarizeEvent(event) {
  return {
    at: event.at,
    url: event.url,
    method: event.method,
    status: event.status,
    bodySample: redactSensitiveText(event.bodySample),
  };
}

function statusToScreenshotPrefix(status) {
  if (status === 'collected' || status === 'collected_low_confidence') {
    return 'xhs-content-metadata-collected';
  }
  if (status === 'pending_reconcile') {
    return 'xhs-content-metadata-pending';
  }
  return 'xhs-content-metadata-error';
}

function hasAnyMetric(value) {
  return ['view_count', 'like_count', 'comment_count', 'share_count', 'collect_count'].some((key) => value[key] != null);
}

function numberValue(value) {
  if (value == null || value === '') {
    return 0;
  }
  const number = Number(value);
  return Number.isFinite(number) ? number : 0;
}

function stringValue(value) {
  return String(value || '').trim();
}

function normalizeText(value) {
  return stringValue(value).replace(/\s+/g, '').toLowerCase();
}

function timeFromSecondsOrMs(value) {
  const number = Number(value);
  if (!Number.isFinite(number) || number <= 0) {
    return '';
  }
  const ms = number > 1000000000000 ? number : number * 1000;
  return new Date(ms).toISOString();
}

function engagementRate(viewCount, likeCount, commentCount, shareCount, favoriteCount) {
  if (!viewCount) {
    return 0;
  }
  return Number(((likeCount + commentCount + shareCount + favoriteCount) / viewCount).toFixed(4));
}

function firstNonEmpty(...values) {
  for (const value of values) {
    const text = stringValue(value);
    if (text) {
      return text;
    }
  }
  return '';
}

function sanitizeForOutput(value, pathParts = []) {
  if (Array.isArray(value)) {
    return value.map((item, index) => sanitizeForOutput(item, [...pathParts, String(index)]));
  }
  if (!value || typeof value !== 'object') {
    return typeof value === 'string' ? redactSensitiveText(value) : value;
  }
  const result = {};
  for (const [key, item] of Object.entries(value)) {
    if (isSensitiveKey(key)) {
      continue;
    }
    result[key] = sanitizeForOutput(item, [...pathParts, key]);
  }
  return result;
}

function isSensitiveKey(key) {
  return /token|session|cookie|authorization|credential/i.test(key);
}

function redactSensitiveText(value) {
  return stringValue(value)
    .replace(/"(?:[^"]*token|[^"]*session|[^"]*cookie|authorization|credential)[^"]*"\s*:\s*"[^"]*"/gi, '"redacted":"[redacted]"')
    .replace(/((?:token|session|cookie|authorization|credential)[A-Za-z0-9_.-]*=)[^;&\s"]+/gi, '$1[redacted]');
}

function isXiaohongshuResponse(value) {
  try {
    const host = new URL(value).hostname;
    return /(^|\.)xiaohongshu\.com$|(^|\.)xhscdn\.com$/.test(host);
  } catch {
    return false;
  }
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
