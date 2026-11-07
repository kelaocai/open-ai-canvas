# 接口与计费

官方接口：https://astraflow.ucloud.cn/reference/modelverse-api-protocol/video/mini-max/mini-max-h3/submit-mini-max-h3-task

apiKey 使用 Bearer 鉴权。创建 POST /v1/tasks/submit，查询 GET /v1/tasks/status?task_id=...。

model 固定 MiniMax-H3；prompt 与按 role 排序的 images/videos/audios 写入 input.content[]。所有媒体使用公网URL，不支持Base64。首尾帧与参考模式互斥，不能仅传参考音频。

duration 为4–15秒整数，resolution 将768p映射768P、1440p映射2K。首尾帧模式 ratio=adaptive，其余使用 aspectRatio。watermark 写入 aigc_watermark，不发送官方未声明的generate_audio、seed等字段。

响应 output.task_id 为任务ID，output.task_status 为状态，output.urls 为视频结果；output.error_message 为失败原因。临时媒体由宿主持久化。

官方成本：768P输入/输出视频0.5元/秒，2K输入/输出视频0.8元/秒，图片0.2元/张。销售价格、成本和参考素材附加价格由模型价格档配置；不把秒数伪装成Token。报价未知视频时长按单段上限估算，正式下单由服务器读取已上传资源并使用ffprobe核验毫秒时长。失败任务按现有账务流程退回预扣。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v2",
  "id": "modelverse-h3",
  "name": "星图 MiniMax H3",
  "version": "1.0.0",
  "author": "RightMovie",
  "description": "星图 Modelverse MiniMax-H3 异步多模态视频协议。",
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
        "id": "modelverse-h3-video",
        "label": "星图 Modelverse MiniMax H3",
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "baseUrl": "https://api.modelverse.cn",
        "requiresPublicMediaUrls": true,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "星图模型ID，固定MiniMax-H3。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "input.content[type=text].text",
            "description": "唯一非空文本项。"
          },
          {
            "name": "images",
            "type": "media[]",
            "mapping": "input.content[type=image_url]",
            "description": "保留首帧、尾帧和参考图role。"
          },
          {
            "name": "videos",
            "type": "media[]",
            "mapping": "input.content[type=video_url]",
            "description": "参考视频，使用reference_video角色。"
          },
          {
            "name": "audios",
            "type": "media[]",
            "mapping": "input.content[type=audio_url]",
            "description": "参考音频，使用reference_audio角色。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": true,
            "mapping": "parameters.duration",
            "values": [
              "4",
              "5",
              "6",
              "7",
              "8",
              "9",
              "10",
              "11",
              "12",
              "13",
              "14",
              "15"
            ],
            "description": "输出视频4至15秒。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": true,
            "mapping": "parameters.resolution",
            "values": [
              "768p",
              "1440p"
            ],
            "description": "768p与1440p映射为官方768P与2K。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "mapping": "parameters.ratio",
            "values": [
              "21:9",
              "16:9",
              "4:3",
              "1:1",
              "3:4",
              "9:16"
            ],
            "description": "首尾帧使用adaptive，其余传具体比例。"
          },
          {
            "name": "watermark",
            "type": "boolean",
            "mapping": "parameters.aigc_watermark",
            "description": "AIGC水印开关。"
          }
        ],
        "validations": [
          {
            "assert": {
              "$eq": [
                {
                  "$ref": "request.model"
                },
                "MiniMax-H3"
              ]
            },
            "message": "该协议仅支持星图 MiniMax-H3"
          },
          {
            "assert": {
              "$and": [
                {
                  "$gte": [
                    {
                      "$ref": "request.duration"
                    },
                    4
                  ]
                },
                {
                  "$lte": [
                    {
                      "$ref": "request.duration"
                    },
                    15
                  ]
                }
              ]
            },
            "message": "H3 输出时长必须在4至15秒之间"
          },
          {
            "assert": {
              "$in": [
                {
                  "$ref": "request.resolution"
                },
                [
                  "768p",
                  "768P",
                  "1440p",
                  "2k",
                  "2K"
                ]
              ]
            },
            "message": "H3 仅支持768P与2K"
          },
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
            "message": "H3 images数量超过限制"
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
            "message": "H3 videos数量超过限制"
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
            "message": "H3 audios数量超过限制"
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
                        "$in": [
                          {
                            "$ref": "image.role"
                          },
                          [
                            "first_frame"
                          ]
                        ]
                      }
                    }
                  }
                },
                1
              ]
            },
            "message": "H3 首帧、尾帧各最多1张"
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
                        "$in": [
                          {
                            "$ref": "image.role"
                          },
                          [
                            "last_frame"
                          ]
                        ]
                      }
                    }
                  }
                },
                1
              ]
            },
            "message": "H3 首帧、尾帧各最多1张"
          },
          {
            "assert": {
              "$or": [
                {
                  "$eq": [
                    {
                      "$len": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "image",
                          "where": {
                            "$in": [
                              {
                                "$ref": "image.role"
                              },
                              [
                                "first_frame",
                                "last_frame"
                              ]
                            ]
                          }
                        }
                      }
                    },
                    0
                  ]
                },
                {
                  "$and": [
                    {
                      "$eq": [
                        {
                          "$len": {
                            "$filter": {
                              "from": {
                                "$ref": "request.images"
                              },
                              "as": "image",
                              "where": {
                                "$in": [
                                  {
                                    "$ref": "image.role"
                                  },
                                  [
                                    "first_frame",
                                    "last_frame"
                                  ]
                                ]
                              }
                            }
                          }
                        },
                        {
                          "$len": {
                            "$ref": "request.images"
                          }
                        }
                      ]
                    },
                    {
                      "$eq": [
                        {
                          "$len": {
                            "$ref": "request.videos"
                          }
                        },
                        0
                      ]
                    },
                    {
                      "$eq": [
                        {
                          "$len": {
                            "$ref": "request.audios"
                          }
                        },
                        0
                      ]
                    }
                  ]
                }
              ]
            },
            "message": "H3 首尾帧不能与多模态参考素材混用"
          },
          {
            "assert": {
              "$or": [
                {
                  "$eq": [
                    {
                      "$len": {
                        "$ref": "request.audios"
                      }
                    },
                    0
                  ]
                },
                {
                  "$gt": [
                    {
                      "$len": {
                        "$ref": "request.images"
                      }
                    },
                    0
                  ]
                },
                {
                  "$gt": [
                    {
                      "$len": {
                        "$ref": "request.videos"
                      }
                    },
                    0
                  ]
                }
              ]
            },
            "message": "H3 不能仅提供参考音频"
          },
          {
            "assert": {
              "$eq": [
                {
                  "$len": {
                    "$filter": {
                      "from": {
                        "$ref": "request.images"
                      },
                      "as": "media",
                      "where": {
                        "$in": [
                          {
                            "$coalesce": [
                              {
                                "$ref": "media.role"
                              },
                              "reference_image"
                            ]
                          },
                          [
                            "first_frame",
                            "last_frame",
                            "reference_image"
                          ]
                        ]
                      }
                    }
                  }
                },
                {
                  "$len": {
                    "$ref": "request.images"
                  }
                }
              ]
            },
            "message": "H3 素材角色不受支持：images"
          },
          {
            "assert": {
              "$eq": [
                {
                  "$len": {
                    "$filter": {
                      "from": {
                        "$ref": "request.videos"
                      },
                      "as": "media",
                      "where": {
                        "$in": [
                          {
                            "$coalesce": [
                              {
                                "$ref": "media.role"
                              },
                              "reference_video"
                            ]
                          },
                          [
                            "reference_video"
                          ]
                        ]
                      }
                    }
                  }
                },
                {
                  "$len": {
                    "$ref": "request.videos"
                  }
                }
              ]
            },
            "message": "H3 素材角色不受支持：videos"
          },
          {
            "assert": {
              "$eq": [
                {
                  "$len": {
                    "$filter": {
                      "from": {
                        "$ref": "request.audios"
                      },
                      "as": "media",
                      "where": {
                        "$in": [
                          {
                            "$coalesce": [
                              {
                                "$ref": "media.role"
                              },
                              "reference_audio"
                            ]
                          },
                          [
                            "reference_audio"
                          ]
                        ]
                      }
                    }
                  }
                },
                {
                  "$len": {
                    "$ref": "request.audios"
                  }
                }
              ]
            },
            "message": "H3 素材角色不受支持：audios"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/tasks/submit",
          "contentType": "application/json",
          "body": {
            "model": {
              "$ref": "request.model"
            },
            "input": {
              "content": {
                "$concatArrays": [
                  [
                    {
                      "type": "text",
                      "text": {
                        "$ref": "request.prompt"
                      }
                    }
                  ],
                  {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.images"
                        }
                      },
                      "as": "media",
                      "in": {
                        "type": "image_url",
                        "image_url": {
                          "url": {
                            "$ref": "media.value"
                          }
                        },
                        "role": {
                          "$coalesce": [
                            {
                              "$ref": "media.role"
                            },
                            "reference_image"
                          ]
                        }
                      }
                    }
                  },
                  {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.videos"
                        }
                      },
                      "as": "media",
                      "in": {
                        "type": "video_url",
                        "video_url": {
                          "url": {
                            "$ref": "media.value"
                          }
                        },
                        "role": {
                          "$coalesce": [
                            {
                              "$ref": "media.role"
                            },
                            "reference_video"
                          ]
                        }
                      }
                    }
                  },
                  {
                    "$map": {
                      "from": {
                        "$sortByOrder": {
                          "$ref": "request.audios"
                        }
                      },
                      "as": "media",
                      "in": {
                        "type": "audio_url",
                        "audio_url": {
                          "url": {
                            "$ref": "media.value"
                          }
                        },
                        "role": {
                          "$coalesce": [
                            {
                              "$ref": "media.role"
                            },
                            "reference_audio"
                          ]
                        }
                      }
                    }
                  }
                ]
              }
            },
            "parameters": {
              "duration": {
                "$ref": "request.duration"
              },
              "resolution": {
                "$if": {
                  "condition": {
                    "$in": [
                      {
                        "$ref": "request.resolution"
                      },
                      [
                        "768p",
                        "768P"
                      ]
                    ]
                  },
                  "then": "768P",
                  "else": "2K"
                }
              },
              "ratio": {
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
                              "$in": [
                                {
                                  "$ref": "image.role"
                                },
                                [
                                  "first_frame",
                                  "last_frame"
                                ]
                              ]
                            }
                          }
                        }
                      },
                      0
                    ]
                  },
                  "then": "adaptive",
                  "else": {
                    "$coalesce": [
                      {
                        "$ref": "request.aspectRatio"
                      },
                      "16:9"
                    ]
                  }
                }
              },
              "aigc_watermark": {
                "$ref": "request.watermark"
              }
            }
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/status",
          "query": {
            "task_id": {
              "$ref": "taskId"
            }
          }
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.output.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.output.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.output.error_message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.message"
              }
            ]
          },
          "videos": {
            "$ref": "response.output.urls"
          },
          "errorPaths": [
            "error.code"
          ],
          "resultEphemeral": true
        }
      }
    ]
  },
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>"
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
