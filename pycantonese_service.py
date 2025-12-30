
# pycantonese服务，由于golang没有pycantonese库，所以这里用python实现一个简单的服务


import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel
import pycantonese
from contextlib import asynccontextmanager
import os
from fastapi import Request

# pip install uvicorn fastapi pydantic pycantonese




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


if __name__ == "__main__":
    # 高并发部署配置
    
    uvicorn.run(
        "pycantonese_service:app", 
        host="127.0.0.1", 
        port=48000, 
        reload=True,
        workers=1,            # 生产环境建议设为 CPU 核心数
        #loop="uvloop",        # 强制使用极速事件循环
        #http="httptools",     # 使用高性能 HTTP 解析器
        access_log=True      # 禁用日志以进一步降低延迟
    )
    
