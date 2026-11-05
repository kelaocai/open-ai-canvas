"""RightMovie administrator tools. All writes go through the running backend."""
import os
from decimal import Decimal, ROUND_CEILING
from pathlib import Path
from typing import Literal

import httpx
from mcp.server.fastmcp import FastMCP
from pydantic import BaseModel, Field

mcp = FastMCP("RightMovie Admin")
SCALE = 1_000_000
FIELDS = {
    "unit": "unitPriceMicrocredits",
    "input": "inputTokenPriceMicrocredits",
    "output": "outputTokenPriceMicrocredits",
    "cached": "cachedTokenPriceMicrocredits",
    "cacheWrite": "cacheWriteTokenPriceMicrocredits",
}


def call(action, **params):
    url = os.environ.get("RIGHTMOVIE_API", "http://localhost/rightmovie/api").rstrip("/")
    address = httpx.URL(url)
    if address.scheme != "https" and not (address.scheme == "http" and address.host in {"localhost", "127.0.0.1", "::1"}):
        raise ValueError("远程管理地址必须使用 HTTPS")
    token = Path(os.environ["RIGHTMOVIE_TOKEN_FILE"]).read_text().strip()
    response = httpx.post(url + "/admin/automation", json={"action": action, **params},
                          headers={"Authorization": "Bearer " + token}, timeout=20, trust_env=False)
    body = response.json()
    if response.is_error or body.get("code") != 0:
        raise ValueError(body.get("msg") or f"管理请求失败 ({response.status_code})")
    return body["data"]


@mcp.tool()
def list_channels() -> list:
    """列出凭证授权的系统渠道，不返回上游密钥。"""
    return call("channels")


@mcp.tool()
def list_price_tiers(channel_id: str) -> list:
    """读取渠道所有模型、规格、当前成本、售价、计价货币和价格版本。金额字段以百万分之一为单位。"""
    return call("tiers", channelID=channel_id)


class TierChange(BaseModel):
    model_id: str
    tier_id: str
    model_version: int = Field(ge=1)
    tier_version: int = Field(ge=1)
    # Values are decimal strings to avoid binary floating point in pricing calculations.
    cost_credits: dict[str, str]


def micro(value):
    number = Decimal(value) * SCALE
    if not number.is_finite() or number < 0 or number != number.to_integral_value() or number > 10**12:
        raise ValueError("成本应为非负积分，最多六位小数，最高 100 万积分")
    return int(number)


@mcp.tool()
def set_channel_margin(channel_id: str, changes: list[TierChange], margin_percent: str,
                       reason: str, mode: Literal["preview", "apply"] = "preview") -> dict:
    """同一渠道内原子调价。成本以积分表示：按次或按秒传 unit；Token 模型传每百万 Token 的 input/output/cached/cacheWrite，视频 Token 仅传 output。按目标毛利向上取整到整积分，再换算到模型售价币种。计算使用管理员分组汇率，报价可另查折扣。先预览，再用相同版本应用。"""
    margin = Decimal(margin_percent)
    if not margin.is_finite() or not 0 <= margin < 100:
        raise ValueError("毛利率必须在 0（含）到 100（不含）之间")
    tiers = {(r["modelId"], r["tier"]["id"]): r for r in list_price_tiers(channel_id)}
    updates = []
    for c in changes:
        r = tiers.get((c.model_id, c.tier_id))
        if r is None or (r["modelVersion"], r["tier"]["priceVersion"]) != (c.model_version, c.tier_version):
            raise ValueError("价格版本已变化或规格不存在，请重新读取")
        rate = r["rateMicrocredits"]
        if rate <= 0:
            raise ValueError("请先配置该币种的兑换比例")
        keys = {"unit"} if r["tier"]["billingMode"] != "token" else (
            {"output"} if r["capability"] == "video" else {"input", "output", "cached", "cacheWrite"})
        if set(c.cost_credits) != keys:
            raise ValueError("当前规格需要成本字段 " + ", ".join(sorted(keys)))
        costs, prices = {"configured": True}, {}
        for k in keys:
            cost = micro(c.cost_credits[k])
            costs[FIELDS[k]] = cost
            credits = (Decimal(cost) * 100 / (SCALE * (100 - margin))).to_integral_value(rounding=ROUND_CEILING)
            prices[FIELDS[k]] = int((credits * SCALE**2 / rate).to_integral_value(rounding=ROUND_CEILING))
        updates.append(dict(ModelID=c.model_id, TierID=c.tier_id, ModelVersion=c.model_version,
                            TierVersion=c.tier_version, Cost=costs, Prices=prices))
    return {"mode": mode, "changes": call(mode, channelID=channel_id, reason=reason, changes=updates)}


@mcp.tool()
def quote_model(channel_id: str, model_key: str, capability: str, operation: str,
                options: dict, inputs: dict[str, int] | None = None) -> dict:
    """按授权管理员的用户分组验证实际报价。不创建生成任务或预扣积分。"""
    return call("quote", channelID=channel_id, modelKey=model_key,
                intent=dict(capability=capability, operation=operation, options=options, inputs=inputs or {}))


@mcp.tool()
def list_price_audit(channel_id: str) -> list:
    """查看渠道最近 50 条 MCP 调价审计，包括管理员、原因和前后值。"""
    return call("audit", channelID=channel_id)


if __name__ == "__main__":
    mcp.run(transport="stdio")
