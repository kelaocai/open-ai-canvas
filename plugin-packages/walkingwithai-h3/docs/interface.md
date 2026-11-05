# 接口

POST /api/v1/assets 上传 file 和 asset_type，读取 asset_id。POST /api/v1/jobs 提交 feature=minimax-h3、inputs 和 parameters。带 first_frame 角色的图片时使用 mode=i2v，上传后的 asset_id 写入 inputs.first_frame；该模式只接受一张首帧，不混用参考图片、音频或视频。其他请求使用 mode=r2v，素材写入 reference_images/reference_videos/reference_audios。GET /api/v1/jobs/{job_id} 查询 queued/running/completed/failed/cancelled；完成后 GET /api/v1/jobs/{job_id}/outputs 获取 JSON 清单，下载 outputs[].url。相对路径按渠道地址解析；outputs/{index}/download 是训练产物接口，不用于视频。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v2",
  "id": "walkingwithai-h3",
  "name": "与AI同行 H3 工作台",
  "version": "1.1.0",
  "author": "影策",
  "description": "支持全能参考开场与真实首帧顺序续接。",
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>",
  "permissions": [
    "generation.run",
    "media.read"
  ],
  "configuration": {
    "fields": []
  },
  "contributes": {
    "providers": [
      {
        "id": "walkingwithai-h3",
        "label": "与AI同行 H3 工作台",
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "baseUrl": "",
        "detachedPolling": true,
        "idempotentCreate": false,
        "auth": {
          "type": "none"
        },
        "parameters": [
          {
            "name": "prompt",
            "type": "string",
            "required": false,
            "mapping": "prompt",
            "description": "统一请求 prompt 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "media[] kind=image",
            "description": "统一请求 images 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "videos",
            "type": "media[]",
            "required": false,
            "mapping": "media[] kind=video",
            "description": "统一请求 videos 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "audios",
            "type": "media[]",
            "required": false,
            "mapping": "media[] kind=audio",
            "description": "统一请求 audios 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "统一请求 duration 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "ratio",
            "description": "统一请求 aspectRatio 映射至工作台任务参数；支持全能参考模式。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "统一请求 resolution 映射至工作台任务参数；支持全能参考模式。"
          }
        ],
        "validations": [
          {
            "assert": {
              "$lte": [
                {
                  "$len": {
                    "$ref": "request.images"
                  }
                },
                9
              ]
            },
            "message": "images 数量超限"
          },
          {
            "assert": {
              "$lte": [
                {
                  "$len": {
                    "$ref": "request.videos"
                  }
                },
                3
              ]
            },
            "message": "videos 数量超限"
          },
          {
            "assert": {
              "$lte": [
                {
                  "$len": {
                    "$ref": "request.audios"
                  }
                },
                3
              ]
            },
            "message": "audios 数量超限"
          },
          {
            "assert": {
              "$lte": [
                {
                  "$len": {
                    "$filter": {
                      "from": {
                        "$ref": "request.images"
                      },
                      "as": "image",
                      "where": {
                        "$eq": [
                          {
                            "$ref": "image.role"
                          },
                          "first_frame"
                        ]
                      }
                    }
                  }
                },
                1
              ]
            },
            "message": "H3 首帧模式只能传入一张首帧图片"
          },
          {
            "assert": {
              "$if": {
                "condition": {
                  "$gt": [
                    {
                      "$len": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "image",
                          "where": {
                            "$eq": [
                              {
                                "$ref": "image.role"
                              },
                              "first_frame"
                            ]
                          }
                        }
                      }
                    },
                    0
                  ]
                },
                "then": {
                  "$eq": [
                    {
                      "$len": {
                        "$ref": "request.images"
                      }
                    },
                    1
                  ]
                },
                "else": true
              }
            },
            "message": "H3 首帧模式不能混用角色或场景参考图"
          },
          {
            "assert": {
              "$if": {
                "condition": {
                  "$gt": [
                    {
                      "$len": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "image",
                          "where": {
                            "$eq": [
                              {
                                "$ref": "image.role"
                              },
                              "first_frame"
                            ]
                          }
                        }
                      }
                    },
                    0
                  ]
                },
                "then": {
                  "$eq": [
                    {
                      "$add": [
                        {
                          "$len": {
                            "$ref": "request.audios"
                          }
                        },
                        {
                          "$len": {
                            "$ref": "request.videos"
                          }
                        }
                      ]
                    },
                    0
                  ]
                },
                "else": true
              }
            },
            "message": "H3 首帧续接不能混用参考音频或参考视频"
          }
        ],
        "upload": {
          "kinds": [
            "image",
            "video",
            "audio"
          ],
          "method": "POST",
          "path": "/api/v1/assets",
          "originPath": true,
          "contentType": "multipart/form-data",
          "body": {
            "asset_type": {
              "$ref": "media.kind"
            }
          },
          "files": [
            {
              "name": "file",
              "source": {
                "$ref": "media"
              },
              "filename": {
                "$concat": [
                  {
                    "$ref": "media.kind"
                  },
                  "-",
                  {
                    "$add": [
                      {
                        "$ref": "media.order"
                      },
                      1
                    ]
                  },
                  ".",
                  {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$eq": [
                              {
                                "$ref": "media.mimeType"
                              },
                              "image/jpeg"
                            ]
                          },
                          "then": "jpg"
                        },
                        {
                          "when": {
                            "$eq": [
                              {
                                "$ref": "media.mimeType"
                              },
                              "video/quicktime"
                            ]
                          },
                          "then": "mov"
                        },
                        {
                          "when": {
                            "$eq": [
                              {
                                "$ref": "media.mimeType"
                              },
                              "audio/mpeg"
                            ]
                          },
                          "then": "mp3"
                        },
                        {
                          "when": {
                            "$in": [
                              {
                                "$ref": "media.mimeType"
                              },
                              [
                                "audio/wav",
                                "audio/x-wav",
                                "audio/wave"
                              ]
                            ]
                          },
                          "then": "wav"
                        },
                        {
                          "when": {
                            "$eq": [
                              {
                                "$ref": "media.kind"
                              },
                              "video"
                            ]
                          },
                          "then": "mp4"
                        },
                        {
                          "when": {
                            "$eq": [
                              {
                                "$ref": "media.kind"
                              },
                              "audio"
                            ]
                          },
                          "then": "mp3"
                        }
                      ],
                      "default": {
                        "$coalesce": [
                          {
                            "$at": [
                              {
                                "$split": [
                                  {
                                    "$ref": "media.mimeType"
                                  },
                                  "/"
                                ]
                              },
                              1
                            ]
                          },
                          "png"
                        ]
                      }
                    }
                  }
                ]
              }
            }
          ],
          "id": {
            "$ref": "response.asset_id"
          }
        },
        "create": {
          "method": "POST",
          "path": "/api/v1/jobs",
          "originPath": true,
          "contentType": "application/json",
          "body": {
            "feature": "minimax-h3",
            "mode": {
              "$if": {
                "condition": {
                  "$gt": [
                    {
                      "$len": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "image",
                          "where": {
                            "$eq": [
                              {
                                "$ref": "image.role"
                              },
                              "first_frame"
                            ]
                          }
                        }
                      }
                    },
                    0
                  ]
                },
                "then": "i2v",
                "else": "r2v"
              }
            },
            "inputs": {
              "$if": {
                "condition": {
                  "$gt": [
                    {
                      "$len": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "image",
                          "where": {
                            "$eq": [
                              {
                                "$ref": "image.role"
                              },
                              "first_frame"
                            ]
                          }
                        }
                      }
                    },
                    0
                  ]
                },
                "then": {
                  "prompt": {
                    "$ref": "request.prompt"
                  },
                  "first_frame": {
                    "$at": [
                      {
                        "$map": {
                          "from": {
                            "$filter": {
                              "from": {
                                "$ref": "request.images"
                              },
                              "as": "image",
                              "where": {
                                "$eq": [
                                  {
                                    "$ref": "image.role"
                                  },
                                  "first_frame"
                                ]
                              }
                            }
                          },
                          "as": "image",
                          "in": {
                            "$ref": "image.uploadId"
                          }
                        }
                      },
                      0
                    ]
                  }
                },
                "else": {
                  "prompt": {
                    "$ref": "request.prompt"
                  },
                  "reference_images": {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.images"
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.uploadId"
                      }
                    }
                  },
                  "reference_videos": {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.videos"
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.uploadId"
                      }
                    }
                  },
                  "reference_audios": {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.audios"
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.uploadId"
                      }
                    }
                  }
                }
              }
            },
            "parameters": {
              "duration": {
                "$ref": "request.duration"
              },
              "resolution_preset": {
                "$coalesce": [
                  {
                    "$ref": "request.resolution"
                  },
                  "480P"
                ]
              },
              "aspect_ratio": {
                "$coalesce": [
                  {
                    "$ref": "request.aspectRatio"
                  },
                  "9:16"
                ]
              },
              "quality_mode": "balanced"
            }
          }
        },
        "poll": {
          "method": "GET",
          "path": "/api/v1/jobs/{{taskId}}",
          "originPath": true
        },
        "cancel": {
          "method": "POST",
          "path": "/api/v1/jobs/{{taskId}}/cancel",
          "originPath": true
        },
        "result": {
          "method": "GET",
          "path": "/api/v1/jobs/{{taskId}}/outputs",
          "originPath": true
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.job_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.error"
              },
              {
                "$ref": "response.detail"
              }
            ]
          },
          "resultEphemeral": true,
          "videos": {
            "$map": {
              "from": {
                "$ref": "response.outputs"
              },
              "as": "output",
              "in": {
                "url": {
                  "$ref": "output.url"
                }
              }
            }
          }
        }
      }
    ]
  }
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
