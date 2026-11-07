# Modelverse Images

星图 Modelverse 图片生成与编辑协议。Provider ID 为 `modelverse-image`。

无参考图调用 `/v1/images/generations`；有参考图调用 `/v1/images/edits`，使用 multipart/form-data 的 image 文件字段。宿主解析 URL 和内联 data URL 后上传文件，排除 mask。当前协议不发送蒙版，不自动重试付费请求。

本协议独立于 OpenAI Images，渠道需要显式选择此协议。切换后核对参考图上限和图生图价格档。
