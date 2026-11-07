# 接口合同

- 默认地址：`https://api.modelverse.cn`，Bearer 鉴权。
- 图片编辑：`POST /v1/images/edits`，`Content-Type: multipart/form-data`，参考图上传至 `image` 文件字段，不发送 JSON images。排除 mask 引用。
- 官方接口依据：https://github.com/UCloudDoc-Team/modelverse/blob/master/api_doc/image_api/gpt-image-1.5.md 。GPT Image 2 使用相同编辑入口，需以真实任务验收。
- 文生图：`POST /v1/images/generations`，不发送空的 `images`。
- 保留 model、prompt、n、size、quality 等参数映射，返回 data 中的 URL 或 Base64 图片。
- 不支持蒙版编辑；不得为本协议打开 mask 能力。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v2",
  "id": "modelverse-images",
  "name": "Modelverse Images",
  "version": "1.0.1",
  "author": "RightMovie",
  "description": "Modelverse 图片协议，编辑按官方接口使用 multipart image 文件。",
  "permissions": [
    "generation.run",
    "media.read"
  ],
  "configuration": {
    "fields": [
      {
        "name": "apiKey",
        "type": "secret",
        "label": "API Key",
        "required": true
      }
    ]
  },
  "contributes": {
    "providers": [
      {
        "id": "modelverse-image",
        "label": "Modelverse Images",
        "capabilities": [
          "image"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "baseUrl": "https://api.modelverse.cn",
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "imageParameters": [
          {
            "name": "size",
            "label": "画幅 / 尺寸",
            "type": "string",
            "binding": "aspectRatio",
            "encoding": "pixels",
            "pricing": true
          },
          {
            "name": "imageResolution",
            "label": "分辨率",
            "type": "string",
            "binding": "resolution",
            "enabled": false,
            "pricing": true
          },
          {
            "name": "quality",
            "label": "图片质量",
            "type": "string",
            "binding": "quality",
            "pricing": true
          },
          {
            "name": "count",
            "label": "生成数量",
            "type": "integer",
            "binding": "imageCount",
            "min": 1,
            "max": 100,
            "step": 1,
            "default": 1
          },
          {
            "name": "transparentBackground",
            "label": "透明背景",
            "type": "boolean",
            "binding": "transparentBackground",
            "default": false,
            "enabled": false
          },
          {
            "name": "outputFormat",
            "label": "输出格式",
            "type": "string",
            "binding": "providerOptions.modelverse-image.output_format",
            "values": [
              "png",
              "jpeg",
              "webp"
            ],
            "default": "png",
            "enabled": false
          },
          {
            "name": "outputCompression",
            "label": "压缩质量",
            "type": "integer",
            "binding": "providerOptions.modelverse-image.output_compression",
            "min": 0,
            "max": 100,
            "step": 1,
            "enabled": false
          }
        ],
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "图片模型 ID。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "图片提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "provider image/reference fields",
            "description": "参考图或编辑源图，role 由业务层确定。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "n/sample_count",
            "description": "输出数量。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "size/aspect_ratio",
            "description": "比例或尺寸，语义按协议说明。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution/imageSize",
            "description": "分辨率档位。"
          },
          {
            "name": "quality",
            "type": "string",
            "required": false,
            "mapping": "quality",
            "description": "质量档位。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的厂商扩展字段。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/images/generations",
          "pathTemplate": {
            "$if": {
              "condition": {
                "$gt": [
                  {
                    "$len": {
                      "$ref": "request.images"
                    }
                  },
                  0
                ]
              },
              "then": "/v1/images/edits",
              "else": "/v1/images/generations"
            }
          },
          "contentType": "application/json",
          "body": {
            "model": {
              "$ref": "request.model"
            },
            "prompt": {
              "$ref": "request.prompt"
            },
            "n": {
              "$omitEmpty": {
                "$if": {
                  "condition": {
                    "$gt": [
                      {
                        "$ref": "request.imageCount"
                      },
                      0
                    ]
                  },
                  "then": {
                    "$ref": "request.imageCount"
                  },
                  "else": 1
                }
              }
            },
            "size": {
              "$omitEmpty": {
                "$if": {
                  "condition": {
                    "$in": [
                      {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.aspectRatio"
                          }
                        }
                      },
                      [
                        "",
                        "auto"
                      ]
                    ]
                  },
                  "then": null,
                  "else": {
                    "$ref": "request.aspectRatio"
                  }
                }
              }
            },
            "resolution": {
              "$omitEmpty": {
                "$ref": "request.resolution"
              }
            },
            "quality": {
              "$omitEmpty": {
                "$if": {
                  "condition": {
                    "$in": [
                      {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.quality"
                          }
                        }
                      },
                      [
                        "",
                        "auto"
                      ]
                    ]
                  },
                  "then": null,
                  "else": {
                    "$ref": "request.quality"
                  }
                }
              }
            },
            "background": {
              "$omitEmpty": {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.modelverse-image.background"
                  },
                  {
                    "$if": {
                      "condition": {
                        "$eq": [
                          {
                            "$ref": "request.extra.transparentBackground"
                          },
                          "true"
                        ]
                      },
                      "then": "transparent",
                      "else": null
                    }
                  }
                ]
              }
            },
            "output_format": {
              "$omitEmpty": {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.modelverse-image.output_format"
                  },
                  "png"
                ]
              }
            },
            "output_compression": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.modelverse-image.output_compression"
              }
            },
            "moderation": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.modelverse-image.moderation"
              }
            },
            "response_format": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.modelverse-image.response_format"
              }
            },
            "style": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.modelverse-image.style"
              }
            },
            "user": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.modelverse-image.user"
              }
            }
          },
          "contentTypeTemplate": {
            "$if": {
              "condition": {
                "$gt": [
                  {
                    "$len": {
                      "$ref": "request.images"
                    }
                  },
                  0
                ]
              },
              "then": "multipart/form-data",
              "else": "application/json"
            }
          },
          "files": [
            {
              "name": "image",
              "source": {
                "$filter": {
                  "from": {
                    "$sortByOrder": {
                      "$ref": "request.images"
                    }
                  },
                  "as": "media",
                  "where": {
                    "$ne": [
                      {
                        "$ref": "media.role"
                      },
                      "mask"
                    ]
                  }
                }
              },
              "filename": "source.png"
            }
          ]
        },
        "response": {
          "status": "succeeded",
          "images": {
            "$map": {
              "from": {
                "$ref": "response.data"
              },
              "as": "item",
              "in": {
                "url": {
                  "$omitEmpty": {
                    "$ref": "item.url"
                  }
                },
                "dataUrl": {
                  "$if": {
                    "condition": {
                      "$ref": "item.b64_json"
                    },
                    "then": {
                      "$concat": [
                        "data:image/png;base64,",
                        {
                          "$ref": "item.b64_json"
                        }
                      ]
                    },
                    "else": null
                  }
                }
              }
            }
          },
          "usage": {
            "$ref": "response.usage"
          },
          "errorPaths": [
            "error.code"
          ],
          "messagePaths": [
            "error.message"
          ]
        }
      }
    ]
  },
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>"
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
