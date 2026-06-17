#!/usr/bin/env node

import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDir, '..');
const args = parseArgs(process.argv.slice(2));
const chromePath = args['chrome-path'] || process.env.GEOPRESS_CHROME_PATH || undefined;
const profileDir = path.resolve(args['profile-dir'] || path.join(projectRoot, 'runtime', 'playwright-chrome-smoke-profile'));
const screenshotDir = path.resolve(args['screenshot-dir'] || path.join(projectRoot, 'runtime'));
const url = args.url || smokeTestDataURL();
const keepOpenMs = Number(args['keep-open-ms'] || 0);
const headless = resolveHeadless(args);
const chromiumArgs = ['--disable-blink-features=AutomationControlled'];

if (['1', 'true', 'yes'].includes(String(process.env.GEOPRESS_CHROMIUM_NO_SANDBOX || '').toLowerCase())) {
  chromiumArgs.push('--no-sandbox', '--disable-setuid-sandbox');
}

const playwright = await importPlaywright();
await mkdir(profileDir, { recursive: true });
await mkdir(screenshotDir, { recursive: true });

const startedAt = Date.now();
const context = await playwright.chromium.launchPersistentContext(profileDir, {
  executablePath: chromePath,
  headless,
  viewport: { width: 1280, height: 800 },
  locale: 'zh-CN',
  args: chromiumArgs,
});

try {
  const page = context.pages()[0] ?? (await context.newPage());
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);

  const screenshotPath = path.join(screenshotDir, `playwright-chrome-smoke-${Date.now()}.png`);
  await page.screenshot({ path: screenshotPath, fullPage: true });

  if (keepOpenMs > 0) {
    await page.waitForTimeout(keepOpenMs);
  }

  const result = {
    ok: true,
    browser: chromePath ? 'chrome_or_chromium_from_executable_path' : 'playwright_bundled_chromium',
    chromePath: chromePath || '',
    headless,
    pageUrl: page.url(),
    title: await page.title().catch(() => ''),
    heading: await page.locator('h1').first().innerText({ timeout: 3000 }).catch(() => ''),
    userAgent: await page.evaluate(() => navigator.userAgent).catch(() => ''),
    profileDir,
    screenshotPath,
    elapsedMs: Date.now() - startedAt,
  };

  console.log(JSON.stringify(result, null, 2));
} finally {
  await context.close();
}

async function importPlaywright() {
  try {
    const mod = await import('playwright');
    return mod.default ?? mod;
  } catch (firstError) {
    const frontendModule = path.resolve(projectRoot, 'frontend', 'node_modules', 'playwright', 'index.js');
    try {
      const mod = await import(pathToFileURL(frontendModule).href);
      return mod.default ?? mod;
    } catch {
      throw firstError;
    }
  }
}

function resolveHeadless(values) {
  if (truthy(values.headed)) {
    return false;
  }
  if (values.headless !== undefined) {
    return truthy(values.headless);
  }
  return !['0', 'false', 'no'].includes(String(process.env.GEOPRESS_BROWSER_HEADLESS || 'true').toLowerCase());
}

function truthy(value) {
  return ['1', 'true', 'yes'].includes(String(value).toLowerCase());
}

function smokeTestDataURL() {
  const html = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <title>Geopress Playwright Chrome Smoke Test</title>
    <style>
      body {
        margin: 0;
        font-family: Arial, sans-serif;
        background: #f7f8fa;
        color: #202124;
      }
      main {
        padding: 48px;
      }
      h1 {
        margin: 0 0 12px;
        font-size: 28px;
      }
      p {
        margin: 0;
        font-size: 16px;
      }
    </style>
  </head>
  <body>
    <main>
      <h1>Playwright Chrome OK</h1>
      <p>Generated at ${new Date().toISOString()}</p>
    </main>
  </body>
</html>`;
  return `data:text/html;charset=utf-8,${encodeURIComponent(html)}`;
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
