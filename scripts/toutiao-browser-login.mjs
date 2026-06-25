#!/usr/bin/env node

import { runBrowserLogin } from './lib/geopress-browser-login.mjs';

await runBrowserLogin({
  platform: 'toutiao',
  platformName: '头条号',
  loginUrl: process.env.GEOPRESS_TOUTIAO_LOGIN_URL || 'https://mp.toutiao.com/auth/page/login/',
  qrSelector: 'canvas,img,svg,[class*="qrcode"],[class*="qr-code"],[class*="scan"]',
  loggedInCookieNames: ['sessionid', 'sessionid_ss', 'sid_tt', 'uid_tt', 'uid_tt_ss'],
  loggedInTextPattern: /发布|发文|创作|内容管理|文章管理|作品管理|数据|账号|头条号/,
  loginTextPattern: /登录|扫码|二维码|验证码|手机验证码|账号密码|抖音|今日头条/,
  qrReadyPattern: /扫码|二维码|扫一扫|抖音|今日头条/,
  qrSwitchTexts: ['扫码登录', '二维码登录', '扫一扫登录', '抖音扫码登录', '今日头条扫码登录'],
});
