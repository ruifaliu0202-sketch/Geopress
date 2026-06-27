#!/usr/bin/env node

import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));

const args = parseArgs(process.argv.slice(2));
const action = required(args, 'action');
const sessionId = required(args, 'session-id');
const profileDir = path.resolve(required(args, 'profile-dir'));
const loginUrl = args['login-url'] || process.env.GEOPRESS_SOHU_LOGIN_URL || 'https://mp.sohu.com/mpfe/v4/login';
const chromePath = args['chrome-path'] || undefined;
const stateFile = args['state-file'] ? path.resolve(args['state-file']) : path.join(profileDir, 'geopress-login-state.json');
const commandFile = args['command-file'] ? path.resolve(args['command-file']) : path.join(profileDir, 'geopress-login-command.json');
const debugDir = args['debug-dir'] ? path.resolve(args['debug-dir']) : path.join(profileDir, 'debug-screenshots');
const watchTimeoutMs = Number(args['watch-timeout-ms'] || 10 * 60 * 1000);
const pollMs = Number(args['poll-ms'] || 800);
const headless = !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
const sohuUserAgent =
  args['user-agent'] ||
  process.env.GEOPRESS_SOHU_USER_AGENT ||
  'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.7827.102 Safari/537.36';

if (action !== 'watch' && action !== 'status') {
  throw new Error(`Unsupported action: ${action}`);
}

if (action === 'status') {
  const state = await readState();
  console.log(JSON.stringify(state || baseState('starting', '读取登录会话状态')));
  process.exit(0);
}

const playwright = await importPlaywright();
await mkdir(profileDir, { recursive: true });

const chromiumArgs = ['--disable-blink-features=AutomationControlled'];
if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

let context;
const pageDiagnostics = createPageDiagnostics();
try {
  context = await playwright.chromium.launchPersistentContext(profileDir, {
    executablePath: chromePath,
    headless,
    viewport: { width: 1280, height: 900 },
    locale: 'zh-CN',
    userAgent: sohuUserAgent,
    args: chromiumArgs,
  });
} catch (error) {
  if (!isProfileInUseError(error)) {
    throw error;
  }

  const existingState = await readState();
  if (isReusableExistingState(existingState)) {
    await writeAndPrintRawState({
      ...existingState,
      lastCheckedAt: new Date().toISOString(),
      warnings: mergeWarnings(existingState.warnings, '已有登录浏览器会话正在运行，已复用当前会话状态。'),
    });
    process.exit(0);
  }

  await writeAndPrintRawState({
    ...baseState('profile_in_use', '该账号的浏览器 profile 正在被其他 Chromium 会话使用，请回到已有登录窗口继续操作，或关闭占用后稍后重试。'),
    allowedActions: ['continue_check'],
    warnings: ['不要直接删除 SingletonLock；如果确认没有浏览器进程占用，再清理该账号 profile。'],
    rawStatus: {
      reason: 'chromium_profile_in_use',
    },
  });
  process.exit(0);
}

try {
  const page = context.pages()[0] ?? (await context.newPage());
  attachPageDiagnostics(page, pageDiagnostics);
  await page.goto(loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(1200);
  const initialBlankState = await detectBlankPage(page, pageDiagnostics);
  if (initialBlankState.blank) {
    await writeAndPrintState(page, 'blank_page', '搜狐号登录页为空白，请刷新页面或重新发起登录。', {
      allowedActions: ['reload_page', 'continue_check', 'debug_screenshot'],
      rawStatus: initialBlankState,
    });
  } else {
  await switchToPhoneLogin(page);
  await writeAndPrintState(page, 'phone_required', '请输入搜狐号绑定手机号', { allowedActions: ['submit_phone'] });
  }

  const deadline = Date.now() + watchTimeoutMs;
  let finished = false;
  while (Date.now() < deadline) {
    await page.waitForTimeout(pollMs);

    const loggedIn = await isLoggedIn(page);
    if (loggedIn) {
      await writeAndPrintState(page, 'connected', '搜狐号登录已确认', { loggedIn: true, completedAt: new Date().toISOString() }, false);
      finished = true;
      break;
    }

    if (await hasRiskChallenge(page)) {
      await writeAndPrintState(page, 'manual_intervention_required', '检测到滑块或风控挑战，请在浏览器窗口中手动完成后继续检测', {
        allowedActions: ['continue_check'],
      }, false);
    }

    const blankState = await detectBlankPage(page, pageDiagnostics);
    if (blankState.blank) {
      await writeAndPrintState(page, 'blank_page', '搜狐号登录页为空白，请刷新页面或重新发起登录。', {
        allowedActions: ['reload_page', 'continue_check', 'debug_screenshot'],
        rawStatus: blankState,
      }, false);
    }

    const command = await readCommand();
    if (!command) {
      continue;
    }
    await handleCommandSafely(page, command);
  }

  if (!finished) {
    await writeAndPrintState(page, 'expired', '搜狐号登录会话已超时，请重新发起登录', { timedOut: true }, false);
  }
} finally {
  await context.close();
}

async function handleCommandSafely(page, command) {
  try {
    await handleCommand(page, command);
  } catch (error) {
    await writeAndPrintState(page, 'action_failed', '登录页面操作失败，请刷新状态后重试。', {
      warnings: [String(error?.message || error || 'unknown error')],
      allowedActions: ['continue_check'],
      lastCommandId: String(command.commandId || ''),
      rawStatus: {
        reason: 'interactive_login_action_failed',
      },
    });
  }
}

async function handleCommand(page, command) {
  const type = String(command.type || '');
  const commandId = String(command.commandId || '');
  if (type === 'reload_page') {
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 30000 }).catch(async () => {
      await page.goto(loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
    });
    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
    await page.waitForTimeout(1200);
    const blankState = await detectBlankPage(page, pageDiagnostics);
    if (blankState.blank) {
      await writeAndPrintState(page, 'blank_page', '搜狐号登录页仍为空白，请关闭当前窗口后重新发起登录。', {
        allowedActions: ['reload_page', 'continue_check', 'debug_screenshot'],
        lastCommandId: commandId,
        rawStatus: blankState,
      });
      return;
    }
    await switchToPhoneLogin(page);
    await writeAndPrintState(page, 'phone_required', '搜狐号登录页已刷新，请输入绑定手机号', {
      allowedActions: ['submit_phone'],
      lastCommandId: commandId,
    });
    return;
  }

  if (type === 'submit_phone') {
    const phoneNumber = String(command.phoneNumber || '').trim();
    if (!phoneNumber) {
      await writeAndPrintState(page, 'phone_required', '手机号不能为空', {
        warnings: ['请填写手机号后继续。'],
        allowedActions: ['submit_phone'],
        lastCommandId: commandId,
      }, false);
      return;
    }
    await fillPhoneNumber(page, phoneNumber);
    await waitForCaptchaImage(page);
    await writeAndPrintState(page, 'captcha_required', '请输入图形验证码', {
      phoneNumber,
      captchaScreenshotData: await screenshotCaptcha(page),
      allowedActions: ['submit_captcha', 'refresh_captcha'],
      lastCommandId: commandId,
    });
    return;
  }

  if (type === 'refresh_captcha') {
    await clickCaptchaImage(page);
    await page.waitForTimeout(800);
    await waitForCaptchaImage(page);
    await writeAndPrintState(page, 'captcha_required', '图形验证码已刷新', {
      captchaScreenshotData: await screenshotCaptcha(page),
      allowedActions: ['submit_captcha', 'refresh_captcha'],
      lastCommandId: commandId,
    });
    return;
  }

  if (type === 'submit_captcha') {
    const captchaCode = String(command.captchaCode || '').trim();
    if (!captchaCode) {
      await writeAndPrintState(page, 'captcha_required', '图形验证码不能为空', {
        captchaScreenshotData: await screenshotCaptcha(page),
        warnings: ['请填写图形验证码。'],
        allowedActions: ['submit_captcha', 'refresh_captcha'],
        lastCommandId: commandId,
      }, false);
      return;
    }
    await fillCaptchaCode(page, captchaCode);
    await clickByText(page, '获取验证码');
    await page.waitForTimeout(1200);
    if (await hasRiskChallenge(page)) {
      await writeAndPrintState(page, 'manual_intervention_required', '发送短信前触发了滑块或风控，请在浏览器窗口完成后继续检测', {
        allowedActions: ['continue_check'],
        lastCommandId: commandId,
      });
      return;
    }
    await writeAndPrintState(page, 'sms_code_required', '短信验证码已请求，请输入手机收到的验证码', {
      allowedActions: ['submit_sms_code', 'refresh_captcha'],
      lastCommandId: commandId,
    });
    return;
  }

  if (type === 'submit_sms_code') {
    const smsCode = String(command.smsCode || '').trim();
    if (!smsCode) {
      await writeAndPrintState(page, 'sms_code_required', '短信验证码不能为空', {
        allowedActions: ['submit_sms_code'],
        lastCommandId: commandId,
      }, false);
      return;
    }
    await fillSMSCode(page, smsCode);
    await acceptAgreements(page);
    await clickLoginButton(page);
    await page.waitForTimeout(2500);
    if (await isLoggedIn(page)) {
      await writeAndPrintState(page, 'connected', '搜狐号登录已确认', { loggedIn: true, completedAt: new Date().toISOString(), lastCommandId: commandId });
      return;
    }
    if (await hasRiskChallenge(page)) {
      await writeAndPrintState(page, 'manual_intervention_required', '提交登录时触发了滑块或风控，请在浏览器窗口完成后继续检测', {
        allowedActions: ['continue_check'],
        lastCommandId: commandId,
      });
      return;
    }
    await writeAndPrintState(page, 'sms_code_required', '尚未确认登录，请检查验证码后重试', {
      warnings: ['如果验证码已过期，请刷新图形验证码后重新获取短信验证码。'],
      allowedActions: ['submit_sms_code', 'refresh_captcha'],
      lastCommandId: commandId,
    });
    return;
  }

  if (type === 'continue_check') {
    if (await isLoggedIn(page)) {
      await writeAndPrintState(page, 'connected', '搜狐号登录已确认', { loggedIn: true, completedAt: new Date().toISOString(), lastCommandId: commandId });
      return;
    }
    const blankState = await detectBlankPage(page, pageDiagnostics);
    if (blankState.blank) {
      await writeAndPrintState(page, 'blank_page', '搜狐号登录页为空白，请刷新页面或重新发起登录。', {
        allowedActions: ['reload_page', 'continue_check', 'debug_screenshot'],
        lastCommandId: commandId,
        rawStatus: blankState,
      });
      return;
    }
    await writeAndPrintState(page, 'manual_intervention_required', '仍未检测到登录态，请继续在浏览器窗口完成验证', {
      allowedActions: ['continue_check'],
      lastCommandId: commandId,
    });
  }

  if (type === 'debug_screenshot') {
    const screenshotPath = await captureDebugScreenshot(page);
    const pageText = await page.locator('body').innerText({ timeout: 2000 }).catch(() => '');
    await writeAndPrintState(page, 'debug_screenshot_ready', '已截取当前搜狐号登录页面', {
      allowedActions: ['reload_page', 'continue_check', 'debug_screenshot'],
      debugScreenshotPath: screenshotPath,
      lastCommandId: commandId,
      rawStatus: {
        ...(await detectBlankPage(page, pageDiagnostics)),
        pageText: pageText.slice(0, 2000),
      },
    });
  }
}

async function switchToPhoneLogin(page) {
  const phoneTab = page.getByText('手机登录', { exact: false }).first();
  if (await phoneTab.isVisible({ timeout: 5000 }).catch(() => false)) {
    await phoneTab.click().catch(() => undefined);
    await page.waitForTimeout(1000);
  }
}

async function fillInput(page, placeholderPattern, value) {
  const inputs = page.locator('input');
  const count = await inputs.count();
  for (let index = 0; index < count; index += 1) {
    const input = inputs.nth(index);
    const placeholder = await input.getAttribute('placeholder').catch(() => '') || '';
    if (placeholderPattern.test(placeholder) && await isActionableInput(input)) {
      await input.fill(value);
      return;
    }
  }
  throw new Error(`No input found for ${placeholderPattern}`);
}

async function fillPhoneNumber(page, value) {
  await fillFirstInput(page, [
    '.phone-input-field input',
    '.phone-input-row input[placeholder*="手机号"]',
    'input[placeholder*="手机号"]',
  ], value, 'phone number');
}

async function fillCaptchaCode(page, value) {
  await fillFirstInput(page, [
    '.check-code input',
    'input[placeholder*="图形验证码"]',
  ], value, 'graphic captcha');
}

async function fillSMSCode(page, value) {
  await fillFirstInput(page, [
    '.login-code input',
    'input[placeholder*="手机验证码"]',
  ], value, 'sms code');
}

async function fillFirstInput(page, selectors, value, label) {
  for (const selector of selectors) {
    const locator = page.locator(selector);
    const count = await locator.count();
    for (let index = 0; index < count; index += 1) {
      const input = locator.nth(index);
      if (await isActionableInput(input)) {
        await input.fill(value);
        return;
      }
    }
  }
  throw new Error(`No input found for ${label}`);
}

async function isActionableInput(input) {
  if (!(await input.isVisible().catch(() => false))) {
    return false;
  }
  const box = await input.boundingBox().catch(() => null);
  return Boolean(box && box.x >= 0 && box.y >= 0 && box.width >= 20 && box.height >= 10);
}

async function clickByText(page, text) {
  const target = page.getByText(text, { exact: false }).first();
  if (!(await target.isVisible({ timeout: 5000 }).catch(() => false))) {
    throw new Error(`No visible text button found: ${text}`);
  }
  await target.click();
}

async function clickLoginButton(page) {
  const buttons = page.getByText('登录', { exact: true });
  const count = await buttons.count();
  for (let index = count - 1; index >= 0; index -= 1) {
    const button = buttons.nth(index);
    if (await button.isVisible().catch(() => false)) {
      await button.click().catch(() => undefined);
      return;
    }
  }
}

async function acceptAgreements(page) {
  const visibleBoxes = page.locator('.el-checkbox__inner');
  const visibleCount = await visibleBoxes.count();
  for (let index = 0; index < visibleCount; index += 1) {
    const box = visibleBoxes.nth(index);
    if (await box.isVisible().catch(() => false)) {
      await box.click({ force: true }).catch(() => undefined);
      break;
    }
  }

  const checkboxes = page.locator('input[type="checkbox"]');
  const count = await checkboxes.count();
  for (let index = 0; index < count; index += 1) {
    const checkbox = checkboxes.nth(index);
    if (!(await checkbox.isChecked().catch(() => false))) {
      await checkbox.check({ force: true }).catch(() => undefined);
    }
  }
}

async function screenshotCaptcha(page) {
  const candidate = await findCaptchaImage(page);
  if (candidate) {
    const image = await candidate.screenshot({ type: 'png' });
    return `data:image/png;base64,${image.toString('base64')}`;
  }
  return '';
}

async function clickCaptchaImage(page) {
  const candidate = await findCaptchaImage(page);
  if (candidate) {
    await candidate.click().catch(() => undefined);
  }
}

async function waitForCaptchaImage(page) {
  await page.locator('img.check-phone-img').first().waitFor({ state: 'visible', timeout: 5000 }).catch(() => undefined);
  await page.waitForFunction(() => {
    const img = document.querySelector('img.check-phone-img');
    return Boolean(img && img.complete && img.naturalWidth > 0 && img.naturalHeight > 0);
  }, undefined, { timeout: 5000 }).catch(() => undefined);
}

async function findCaptchaImage(page) {
  const preferred = page.locator('img.check-phone-img').first();
  if (await isUsableCaptchaCandidate(preferred)) {
    return preferred;
  }

  const candidates = page.locator('img,canvas,svg,[class*="captcha"],[class*="verify"],[class*="code"]');
  const count = await candidates.count();
  for (let index = 0; index < count; index += 1) {
    const candidate = candidates.nth(index);
    if (await isUsableCaptchaCandidate(candidate)) {
      return candidate;
    }
  }
  return null;
}

async function isUsableCaptchaCandidate(candidate) {
  if (!(await candidate.isVisible().catch(() => false))) {
    return false;
  }
  const box = await candidate.boundingBox().catch(() => null);
  if (!box || box.x < 0 || box.y < 0 || box.width < 70 || box.height < 25 || box.width > 180 || box.height > 80) {
    return false;
  }
  const className = await candidate.getAttribute('class').catch(() => '') || '';
  const src = await candidate.getAttribute('src').catch(() => '') || '';
  return /check-phone-img|captcha|picture|verify|code/i.test(`${className} ${src}`);
}

async function isLoggedIn(page) {
  const cookies = await page.context().cookies();
  const cookieNames = new Set(cookies.map((cookie) => cookie.name));
  if (cookieNames.has('ppinf') || cookieNames.has('pprdig')) {
    return true;
  }
  const text = await page.locator('body').innerText({ timeout: 2000 }).catch(() => '');
  return /发布|发文|创作|内容管理|文章管理|作品管理|数据|收益|搜狐号/.test(text) && !/登录|手机验证码|图形验证码/.test(text);
}

function createPageDiagnostics() {
  return {
    consoleMessages: [],
    requestFailures: [],
    pageErrors: [],
  };
}

function attachPageDiagnostics(page, diagnostics) {
  page.on('console', (message) => {
    diagnostics.consoleMessages.push({
      type: message.type(),
      text: sanitizeDiagnosticText(message.text()).slice(0, 1000),
    });
    trimDiagnostics(diagnostics.consoleMessages);
  });
  page.on('requestfailed', (request) => {
    diagnostics.requestFailures.push({
      url: sanitizeDiagnosticText(request.url()).slice(0, 1000),
      method: request.method(),
      failureText: sanitizeDiagnosticText(request.failure()?.errorText || ''),
    });
    trimDiagnostics(diagnostics.requestFailures);
  });
  page.on('pageerror', (error) => {
    diagnostics.pageErrors.push({
      message: sanitizeDiagnosticText(error.message).slice(0, 1000),
    });
    trimDiagnostics(diagnostics.pageErrors);
  });
}

async function detectBlankPage(page, diagnostics) {
  const detail = await page.evaluate(() => {
    const bodyText = (document.body?.innerText || document.body?.textContent || '').replace(/\s+/g, ' ').trim();
    const app = document.querySelector('#app');
    const appText = (app?.innerText || app?.textContent || '').replace(/\s+/g, ' ').trim();
    return {
      bodyTextLength: bodyText.length,
      appTextLength: appText.length,
      bodyChildCount: document.body?.children?.length || 0,
      appChildCount: app?.children?.length || 0,
      readyState: document.readyState,
      title: document.title,
      htmlLength: document.documentElement?.outerHTML?.length || 0,
    };
  }).catch((error) => ({
    bodyTextLength: 0,
    appTextLength: 0,
    bodyChildCount: 0,
    appChildCount: 0,
    readyState: '',
    title: '',
    htmlLength: 0,
    evaluateError: String(error?.message || error || ''),
  }));
  return {
    blank: detail.bodyTextLength === 0 && detail.appChildCount === 0,
    pageUrl: page.url(),
    ...detail,
    consoleMessages: diagnostics.consoleMessages.slice(-10),
    requestFailures: diagnostics.requestFailures.slice(-10),
    pageErrors: diagnostics.pageErrors.slice(-10),
  };
}

function trimDiagnostics(values) {
  if (values.length > 30) {
    values.splice(0, values.length - 30);
  }
}

function sanitizeDiagnosticText(value) {
  return String(value || '')
    .replace(/(token|session|cookie|authorization|csrf|signature|password)=?[^&\s]+/gi, '$1=[redacted]')
    .replace(/\s+/g, ' ')
    .trim();
}

async function hasRiskChallenge(page) {
  const text = await page.locator('body').innerText({ timeout: 2000 }).catch(() => '');
  return /滑块|拖动|安全验证|风险|验证通过|人机验证|请完成验证/.test(text);
}

async function captureDebugScreenshot(page) {
  await mkdir(debugDir, { recursive: true });
  const filename = `sohu-login-${new Date().toISOString().replace(/[:.]/g, '-')}.png`;
  const screenshotPath = path.join(debugDir, filename);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  return screenshotPath;
}

async function writeAndPrintState(page, status, message, extra = {}, print = true) {
  const state = {
    ...baseState(status, message),
    pageUrl: page.url(),
    lastCheckedAt: new Date().toISOString(),
    ...extra,
  };
  await writeState(state);
  if (print) {
    process.stdout.write(`${JSON.stringify(state)}\n`);
  }
}

async function writeAndPrintRawState(state) {
  await writeState(state);
  process.stdout.write(`${JSON.stringify(state)}\n`);
}

function baseState(status, message) {
  return {
    sessionId,
    platform: 'sohu',
    loginUrl,
    pageUrl: loginUrl,
    profileDir,
    stateFile,
    commandFile,
    status,
    message,
    loggedIn: status === 'connected',
    startedAt: new Date().toISOString(),
    allowedActions: [],
    warnings: [],
  };
}

function isReusableExistingState(state) {
  if (!state || typeof state !== 'object') {
    return false;
  }
  return Boolean(state.sessionId && state.status && !['expired', 'failed'].includes(state.status));
}

function isProfileInUseError(error) {
  const message = String(error?.message || error || '');
  return /ProcessSingleton|SingletonLock|profile directory.*in use|profile is already in use/i.test(message);
}

function mergeWarnings(existing, warning) {
  const values = Array.isArray(existing) ? existing : [];
  return values.includes(warning) ? values : [...values, warning];
}

async function writeState(state) {
  await mkdir(path.dirname(stateFile), { recursive: true });
  await writeFile(stateFile, JSON.stringify({ ...state, stateFile }, null, 2), 'utf8');
}

async function readState() {
  try {
    return JSON.parse(await readFile(stateFile, 'utf8'));
  } catch {
    return null;
  }
}

async function readCommand() {
  try {
    const raw = await readFile(commandFile, 'utf8');
    await rm(commandFile, { force: true });
    const command = JSON.parse(raw);
    if (command.sessionId && command.sessionId !== sessionId) {
      return null;
    }
    return command;
  } catch {
    return null;
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
