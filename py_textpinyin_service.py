
# pycantonese服务，由于golang没有pycantonese库，所以这里用python实现一个简单的服务
# 同时提供普通话拼音服务

import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel
import pycantonese
from contextlib import asynccontextmanager
import os
from fastapi import Request
from pypinyin import pinyin, Style
from pypinyin_dict.pinyin_data import kxhc1983

# pip install uvicorn fastapi pydantic pycantonese pypinyin pypinyin-dict




# # 定义请求体结构，Pydantic v2 处理长文本效率极高
# class LongTextRequest(BaseModel):
#     content: str


# ---------- lifespan：官方推荐的启动/关闭钩子 ----------
@asynccontextmanager
async def lifespan(app: FastAPI):
    # ===== 启动阶段 =====
    # 偷偷触发一次词典加载，当前 worker 后续就再也不碰磁盘
    pycantonese.characters_to_jyutping("不")
    print("[worker] PyCantonese dict warmed-up")
    # 预加载普通话拼音词典
    kxhc1983.load()
    pinyin("不", style=Style.TONE3)
    print("[worker]普通话资源预加载 Pypinyin dict warmed-up")
    yield
    # ===== 关闭阶段 =====
    # 这里可以放资源清理代码，当前用不到就留空
    print("[worker] shutdown complete")


# 初始化 FastAPI 实例
app = FastAPI(lifespan=lifespan)

# 核心接口：使用 async def 确保非阻塞
@app.post("/cantonese_split")
async def cantonese_split(request: Request):
    # 逻辑处理：此处仅演示极简返回以保持低延迟
    # 如果有复杂计算，建议配合 process_pool 进行
    # 请求体已是 JSON 字典，无需再转换
    #sentence = request.content
    reqParams = await request.json() 

    print("请求request:", reqParams)

    ret = {}

    for key, value in reqParams.items():
        print("key:", key, "value:", value)
        jyutping_result = []
        jps_list = pycantonese.characters_to_jyutping(value)
        for i, jp_word in enumerate(jps_list):
            hanzi = jp_word[0]
            jp = jp_word[1]
            parsed_syllable = pycantonese.parse_jyutping(jp)

            initial_list = []
            for syllable in parsed_syllable:
                initial = syllable.onset
                nucleus= syllable.nucleus
                coda = syllable.coda
                tone = syllable.tone
                initial_list.append({
                    "initial": initial,
                    "nucleus": nucleus,
                    "coda": coda,
                    "tone": tone,
                })
            jyutping_result.append({
                "char": hanzi,
                "pinyin": jp,
                "initial_list": initial_list
            })
            #print(hanzi, jp, parsed_syllable)
        
            ret[key] = jyutping_result

    # jps_list = pycantonese.characters_to_jyutping(sentence)
    
    print("ret:", ret)
    return ret


# 普通话拼音接口
@app.post("/mandaren_pinyin")
async def mandaren_pinyin(request: Request):
    reqParams = await request.json() 

    print("mandaren_pinyin 请求:", reqParams)

    ret = {}

    for key, value in reqParams.items():
        print("key:", key, "value:", value)
        pinyin_result = []
        # 使用 TONE3 风格，声调在拼音末尾（如 "ni3 hao3"）
        pinyin_list = pinyin(value, style=Style.TONE3)
        
        for item in pinyin_list:
            py = item[0]  # pinyin 返回的是列表的列表，如 [['ni3'], ['hao3']]
            # 解析声母、韵母、声调
            initial = get_initial(py)
            final, tone = get_final_tone(py)
            
            pinyin_result.append({
                "initial": initial,
                "final": final,
                "tone": tone,
            })
        
        ret[key] = pinyin_result

    print("mandaren_pinyin ret:", ret)
    return ret


# 声母表
INITIALS = ['b', 'p', 'm', 'f', 'd', 't', 'n', 'l', 'g', 'k', 'h', 'j', 'q', 'x', 'r', 'zh', 'ch', 'sh', 'z', 'c', 's', 'y', 'w']

def get_initial(pinyin_str):
    """获取拼音的声母"""
    # 按长度降序排列，优先匹配较长的声母（如 zh, ch, sh）
    for initial in sorted(INITIALS, key=len, reverse=True):
        if pinyin_str.startswith(initial):
            return initial
    return ""

def get_final_tone(pinyin_str):
    """获取拼音的韵母和声调"""
    initial = get_initial(pinyin_str)
    if initial:
        final_part = pinyin_str[len(initial):]
    else:
        final_part = pinyin_str
    
    # 处理 y, w 开头的特殊情况
    if not initial:
        if final_part.startswith("yu"):
            final_part = "v" + final_part[2:]
        elif final_part.startswith("yi"):
            final_part = final_part[1:]
        elif final_part.startswith("y"):
            final_part = "i" + final_part[1:]
        elif final_part.startswith("wu"):
            final_part = final_part[1:]
        elif final_part.startswith("w"):
            final_part = "u" + final_part[1:]
    
    # 处理 j/q/x + u 的特殊情况 -> v
    if initial in ['j', 'q', 'x']:
        if final_part.startswith("u"):
            final_part = "v" + final_part[1:]
    
    # 提取声调（最后一个字符如果是数字）
    tone = 0
    if final_part and final_part[-1].isdigit():
        tone = int(final_part[-1])
        final_part = final_part[:-1]
    
    return final_part, tone


if __name__ == "__main__":
    # 高并发部署配置
    
    uvicorn.run(
        "py_textpinyin_service:app", 
        host="127.0.0.1", 
        port=48001, 
        reload=True,
        workers=1,            # 生产环境建议设为 CPU 核心数
        #loop="uvloop",        # 强制使用极速事件循环
        #http="httptools",     # 使用高性能 HTTP 解析器
        access_log=True      # 禁用日志以进一步降低延迟
    )
    
