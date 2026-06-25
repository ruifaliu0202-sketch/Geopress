#!/usr/bin/env node

import { runBrowserLogin } from './lib/geopress-browser-login.mjs';

await runBrowserLogin({
  platform: 'sohu',
  platformName: '搜狐号',
  loginUrl: process.env.GEOPRESS_SOHU_LOGIN_URL || 'https://mp.sohu.com/mpfe/v4/',
  qrSelector: 'canvas,img,svg,[class*="qrcode"],[class*="qr-code"],[class*="scan"]',
  loggedInCookieNames: ['ppinf', 'pprdig', 'SUV', 'IPLOC', 'tgw_l7_route'],
  loggedInTextPattern: /发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|搜狐号/,
  loginTextPattern: /登录|扫码|二维码|验证码|手机验证码|账号密码|搜狐/,
  qrReadyPattern: /扫码|二维码|扫一扫|搜狐/,
  qrSwitchTexts: ['扫码登录', '二维码登录', '扫一扫登录', '搜狐新闻扫码登录'],
});
