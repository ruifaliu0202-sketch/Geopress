#!/usr/bin/env node

import { runBrowserLogin } from './lib/geopress-browser-login.mjs';

await runBrowserLogin({
  platform: 'netease',
  platformName: '网易号',
  loginUrl: process.env.GEOPRESS_NETEASE_LOGIN_URL || 'https://mp.163.com/',
  qrSelector: 'canvas,img,svg,[class*="qrcode"],[class*="qr-code"],[class*="scan"]',
  loggedInCookieNames: ['NTES_SESS', 'S_INFO', 'P_INFO', 'mp_info', 'urs_id'],
  loggedInTextPattern: /发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|网易号/,
  loginTextPattern: /登录|扫码|二维码|验证码|手机验证码|账号密码|网易邮箱/,
  qrReadyPattern: /扫码|二维码|扫一扫|网易新闻|网易号/,
  qrSwitchTexts: ['扫码登录', '二维码登录', '扫一扫登录', '网易新闻扫码登录'],
});
