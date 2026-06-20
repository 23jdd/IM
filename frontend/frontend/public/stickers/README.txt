流萤 / 自定义表情包 — 使用说明
================================

1. 把表情图片（建议 png，正方形、≤200KB）放到本目录：
     frontend/frontend/public/stickers/
   例如：firefly_01.png, firefly_02.png ...

2. 在同目录的 index.json 里列出文件名（JSON 数组），例如：
     [
       "firefly_01.png",
       "firefly_02.png",
       "firefly_03.png"
     ]

3. 重新构建前端（npm run build）后，聊天输入框的「表情」按钮即会显示这些表情，
   点击即可发送；收发双方用同一套本地资源显示。

说明：表情不经过服务器上传，消息里只携带文件名，因此所有客户端需放置同一套表情图片。
