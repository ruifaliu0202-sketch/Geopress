#!/usr/bin/env node

import { runBrowserPublish } from './lib/geopress-browser-publish.mjs';

await runBrowserPublish({
  platform: 'netease',
  platformName: '网易号',
  publishUrl: process.env.GEOPRESS_NETEASE_PUBLISH_URL || 'https://mp.163.com/',
  loginTextPattern: /登录|扫码|二维码|验证码|手机验证码|账号密码|网易邮箱/,
  creatorShellPattern: /发布|发文|创作|内容管理|文章管理|作品管理|数据|网易号/,
  editorEntryTexts: ['发布', '发文', '写文章', '发布文章', '新建文章', '创作'],
  titleSelectors: [
    'input[placeholder*="标题"]',
    'textarea[placeholder*="标题"]',
    '[contenteditable="true"][data-placeholder*="标题"]',
    '[contenteditable="true"][placeholder*="标题"]',
  ],
  bodySelectors: [
    '[contenteditable="true"][data-placeholder*="正文"]',
    '[contenteditable="true"][placeholder*="正文"]',
    '.ql-editor',
    '.ProseMirror',
    '[contenteditable="true"]',
    'textarea[placeholder*="正文"]',
  ],
  publishTexts: ['发布', '提交', '发表'],
  successTexts: ['发布成功', '提交成功', '发表成功', '已提交审核', '审核中', '发布成功，请等待审核'],
  blockingTexts: ['发布失败', '提交失败', '请完成验证', '验证码', '账号异常', '标题不能为空', '正文不能为空', '内容不能为空'],
  editorTexts: ['标题', '正文', '发布', '保存草稿'],
});
