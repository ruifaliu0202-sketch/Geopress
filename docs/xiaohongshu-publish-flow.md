# Xiaohongshu Publish Flow Notes

Date: 2026-06-12

This note records the Xiaohongshu Creator Platform publish flow confirmed with the saved browser session at:

```text
runtime/browser-profiles/wks_personal/acc_xhs_personal
```

The exploration used Playwright with the system Chrome executable:

```text
/usr/bin/google-chrome
```

No final publish action was clicked during exploration.

## Current Project State

- Frontend publish dialog prepares Xiaohongshu long-article content for review, lets the operator edit title/body, and then calls backend browser publishing after explicit confirmation.
- Backend routes Xiaohongshu publishing through `internal/integration/xiaohongshu.BrowserLongArticlePublisher`.
- `MockHumanPublisher` remains in the codebase for tests and non-browser fallback work, but it is no longer the default workspace publish path.
- Browser login already persists a reusable Chrome profile under `runtime/browser-profiles/<workspaceId>/<accountId>`.

Relevant files:

- `frontend/src/App.tsx`
- `frontend/src/api.ts`
- `frontend/src/types.ts`
- `backend/internal/http/handler/workspace.go`
- `backend/internal/integration/xiaohongshu/browser_login.go`
- `backend/internal/integration/xiaohongshu/publisher.go`
- `scripts/xiaohongshu-browser-login.mjs`

## Confirmed Creator Pages

Logged-in home:

```text
https://creator.xiaohongshu.com/new/home
```

Home page visible entry points:

- `发布图文笔记`
- `发布视频笔记`

Direct publish URLs confirmed:

```text
https://creator.xiaohongshu.com/publish/publish?from=homepage&target=image
https://creator.xiaohongshu.com/publish/publish?from=homepage&target=article
```

## Upload Image/Text Note Flow

Opening `target=image` lands on the image/text upload page.

Top tabs:

- `上传视频`
- `上传图文`
- `写长文`
- `发播客`
- `草稿箱`

Within `上传图文`, the initial choices are:

- `上传图片`
- `文字配图`

Image upload constraints shown by the page:

- Max image size: 32 MB.
- Supported formats: png, jpg, jpeg, webp.
- gif/live images are not supported.
- Recommended ratio: 3:4 to 2:1.
- Recommended minimum resolution: 720x960.

After uploading an image, the publish edit page shows:

- Image editor area.
- Uploaded image count such as `1/18`.
- `获取封面建议`.
- Title input with placeholder `填写标题会有更多赞哦`.
- `智能标题`.
- Body editor as a TipTap/ProseMirror `contenteditable` textbox.
- Hashtag suggestions.
- Buttons: `话题`, `用户`, `表情`.
- Body counter, observed as `0/1000` before input.
- Activity topics, for example `世界杯聊个球`, `热AI训练营`.
- Content settings:
  - `加入合集`
  - `原创声明`
  - `添加内容类型声明`
  - `添加组件`
  - `添加地点`
  - `选择群聊`
  - `标记地点或标记朋友`
  - `添加路线`
  - `选择文件`
- More settings:
  - `允许合拍`
  - `允许正文复制`
  - Visibility selector, default `公开可见`
  - `定时发布`
- Preview:
  - `笔记预览`
  - `封面预览`
- Bottom action buttons:
  - `暂存离开`
  - `发布`

Visibility dropdown options observed in hidden DOM:

- `公开可见`
- `仅自己可见`
- `仅互关好友可见`
- `只给谁看`
- `不给谁看`

Content declaration options observed in hidden DOM:

- `虚构演绎，仅供娱乐`
- `笔记含AI合成内容`
- `内容包含营销广告`
- `内容来源声明`

Source declaration options observed in hidden DOM:

- `自主拍摄`
- `来源转载`

## Text-To-Image Flow

Clicking `文字配图` enters a text editor page.

Initial text-to-image page fields/actions:

- Header: `写文字`
- Main editor: TipTap/ProseMirror `contenteditable` textbox.
- Button: `表情`
- Button: `再写一张`
- Button: `生成图片`

After entering text and clicking `生成图片`, the page moves to a card/template selection step:

- Header: `预览图片`
- Text: `选择一个喜欢的卡片`
- Action: `换配色`
- Template/style categories:
  - `基础`
  - `插图`
  - `美漫`
  - `边框`
  - `清新`
  - `备忘`
  - `涂鸦`
  - `涂写`
  - `手写`
  - `光影`
- Button: `下一步`
- Auto-save text appears, for example `自动保存于 23:06`

Implementation inference: after `下一步`, the generated card should become the image asset and proceed to the standard image/text publish edit page. This final transition was not clicked during exploration to avoid generating an unintended draft state beyond the confirmed template screen.

## Long Article Flow

Opening `target=article` lands on the long article entry page.

Visible actions:

- `新的创作`
- `导入链接`
- `长文合集`
- `新建长文合集`

Clicking `新的创作` opens a long article editor:

- Back action: `返回`
- Title textarea placeholder: `输入标题`
- Title counter: `0/64`
- Body editor: TipTap/ProseMirror `contenteditable`.
- Word count: `字数：0`
- Bottom buttons:
  - `暂存离开`
  - `一键排版`

Implementation decision: the first browser publishing implementation targets Xiaohongshu long articles. Geopress AI generation now uses an explicit publish format contract, `xiaohongshu_long_article`, so the model is told the target platform, field limits, structure, style, validation rules, and automation channel before it writes.

Current intended long-article automation path:

```text
keywords + knowledge + xiaohongshu_long_article format
-> AI draft
-> operator checks title/body in Geopress
-> backend Playwright opens target=article with saved profile
-> 新的创作
-> fill title/body
-> 一键排版
-> keep default template
-> 下一步
-> fill final title/caption
-> 发布
```

## Automation Notes

The existing browser profile can be reused by `launchPersistentContext(profileDir, ...)`.

Important runtime constraints:

- Only one Chrome/Playwright process can use a profile directory at a time. Concurrent usage triggers a Chromium `ProcessSingleton` error.
- On Ubuntu 26.04, Playwright's bundled Chromium install is unsupported for the current project version. Use system Chrome through `GEOPRESS_CHROME_PATH=/usr/bin/google-chrome`.
- Selectors should prefer user-visible text and role/placeholder queries where possible, but the Creator Platform uses dynamic classes and TipTap editors. Automation must include fallbacks and screenshot/error capture.
- The final `发布` click should only happen after the user explicitly confirms in Geopress.
- Browser publishing script for the first implementation: `scripts/xiaohongshu-browser-publish.mjs`.
- Publish format contract source for the first implementation: `backend/internal/ai/publish_format.go`.

## Screenshots Captured

Exploration screenshots were saved under `runtime/`:

- `runtime/xhs-home.png`
- `runtime/xhs-publish-image-entry.png`
- `runtime/xhs-publish-after-click.png`
- `runtime/xhs-image-after-upload.png`
- `runtime/xhs-image-filled-before-submit.png`
- `runtime/xhs-image-filled-bottom.png`
- `runtime/xhs-text-to-image.png`
- `runtime/xhs-text-to-image-filled.png`
- `runtime/xhs-text-to-image-after-generate.png`
- `runtime/xhs-article-entry.png`
- `runtime/xhs-article-new-create.png`
