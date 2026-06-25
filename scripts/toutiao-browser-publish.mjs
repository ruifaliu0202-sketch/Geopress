#!/usr/bin/env node

import { runBrowserPublish } from './lib/geopress-browser-publish.mjs';

await runBrowserPublish({
  platform: 'toutiao',
  platformName: '头条号',
  publishUrl: process.env.GEOPRESS_TOUTIAO_PUBLISH_URL || 'https://mp.toutiao.com/profile_v4/',
  loginTextPattern: /登录|扫码|二维码|验证码|手机验证码|账号密码|抖音|今日头条/,
  creatorShellPattern: /发布|发文|创作|内容管理|文章管理|作品管理|数据|头条号/,
  editorEntryTexts: ['发布', '发文', '写文章', '发布文章', '新建文章', '文章'],
  titleSelectors: [
    'textarea[placeholder*="标题"]',
    'input[placeholder*="标题"]',
    '[contenteditable="true"][data-placeholder*="标题"]',
    '[contenteditable="true"][placeholder*="标题"]',
  ],
  bodySelectors: [
    '.ProseMirror',
    '.ql-editor',
    '[contenteditable="true"][data-placeholder*="正文"]',
    '[contenteditable="true"][placeholder*="正文"]',
    '[contenteditable="true"]',
    'textarea[placeholder*="正文"]',
  ],
  publishTexts: ['发布', '提交', '发表'],
  successTexts: ['发布成功', '提交成功', '发表成功', '已提交审核', '审核中', '发布成功，请等待审核'],
  blockingTexts: ['发布失败', '提交失败', '请完成验证', '验证码', '账号异常', '标题不能为空', '正文不能为空', '内容不能为空'],
  editorTexts: ['标题', '正文', '发布', '存草稿'],
});
