非实时语音合成通过 HTTP API 将文本转换为语音，适用于有声读物、课件配音、内容生产等对延迟不敏感的场景。支持 Qwen-TTS、CosyVoice 和 MiniMax 多种模型系列，提供丰富音色、多语言支持、声音复刻与声音设计等能力。

## **概述**

通过 HTTP API 将文本转换为语音文件，适用于有声读物、课件配音、内容批量生产等对延迟不敏感的场景。

-   通过 HTTP API 调用，发送完整文本获取音频，支持流式输出（边合成边播放）
    
-   覆盖多种语言，支持中文方言
    
-   支持[声音复刻](https://help.aliyun.com/zh/model-studio/voice-cloning-user-guide)与[声音设计](https://help.aliyun.com/zh/model-studio/voice-design-user-guide)音色定制
    
-   支持[指令控制](#nrt-instruct-h3)，可通过自然语言指令控制语音表现力
    

如果您需要实时语音合成（低延迟流式输出），请参见[实时语音合成-千问](https://help.aliyun.com/zh/model-studio/realtime-tts-user-guide)（WebSocket API）。如需了解各模型的选型建议，请参见[语音合成](https://help.aliyun.com/zh/model-studio/tts-model/)。

## **前提条件**

-   已[配置 API Key](https://help.aliyun.com/zh/model-studio/get-api-key)并将其[设置到环境变量](https://help.aliyun.com/zh/model-studio/configure-api-key-through-environment-variables)。
    
-   如果通过 DashScope SDK 调用，需要[安装最新版SDK](https://help.aliyun.com/zh/model-studio/install-sdk)。
    

## **快速开始**

以下是各模型系列的语音合成示例代码。更多语言的示例代码和详细参数说明，请参见各模型的[API 参考](#bb4dbbdb74em4)。

## Qwen-TTS

以下示例演示如何使用[系统音色](#bac280ddf5a1u)进行语音合成。

## 非流式输出

非流式模式下，通过返回的`url`获取合成的语音文件。URL 有效期为 24 小时。

## Python

```
import os
import dashscope

# 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1
dashscope.base_http_api_url = 'https://dashscope.aliyuncs.com/api/v1'

text = "那我来给大家推荐一款T恤，这款呢真的是超级好看，这个颜色呢很显气质，而且呢也是搭配的绝佳单品，大家可以闭眼入，真的是非常好看，对身材的包容性也很好，不管啥身材的宝宝呢，穿上去都是很好看的。推荐宝宝们下单哦。"
# SpeechSynthesizer接口使用方法：dashscope.audio.qwen_tts.SpeechSynthesizer.call(...)
response = dashscope.MultiModalConversation.call(
    # 如需使用指令控制功能，请将model替换为qwen3-tts-instruct-flash
    model="qwen3-tts-flash",
    # 新加坡和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
    # 若没有配置环境变量，请用百炼API Key将下行替换为：api_key = "sk-xxx"
    api_key=os.getenv("DASHSCOPE_API_KEY"),
    text=text,
    voice="Cherry",
    language_type="Chinese", # 建议与文本语种一致，以获得正确的发音和自然的语调。
    # 如需使用指令控制功能，请取消下方注释，并将model替换为qwen3-tts-instruct-flash
    # instructions='语速较快，带有明显的上扬语调，适合介绍时尚产品。',
    # optimize_instructions=True,
    stream=False
)
print(response)
```

## Java

需要导入Gson依赖，若是使用Maven或者Gradle，添加依赖方式如下：

## Maven

在`pom.xml`中添加如下内容：

```
<!-- https://mvnrepository.com/artifact/com.google.code.gson/gson -->
<dependency>
    <groupId>com.google.code.gson</groupId>
    <artifactId>gson</artifactId>
    <version>2.13.1</version>
</dependency>
```

## Gradle

在`build.gradle`中添加如下内容：

```
// https://mvnrepository.com/artifact/com.google.code.gson/gson
implementation("com.google.code.gson:gson:2.13.1")
```

```
import com.alibaba.dashscope.aigc.multimodalconversation.AudioParameters;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversation;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversationParam;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversationResult;
import com.alibaba.dashscope.exception.ApiException;
import com.alibaba.dashscope.exception.NoApiKeyException;
import com.alibaba.dashscope.exception.UploadFileException;
import com.alibaba.dashscope.protocol.Protocol;
import com.alibaba.dashscope.utils.Constants;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.URL;

public class Main {
    // 如需使用指令控制功能，请将MODEL替换为qwen3-tts-instruct-flash
    private static final String MODEL = "qwen3-tts-flash";
    public static void call() throws ApiException, NoApiKeyException, UploadFileException {
        MultiModalConversation conv = new MultiModalConversation();
        MultiModalConversationParam param = MultiModalConversationParam.builder()
                // 新加坡和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
                // 若没有配置环境变量，请用百炼API Key将下行替换为：.apiKey("sk-xxx")
                .apiKey(System.getenv("DASHSCOPE_API_KEY"))
                .model(MODEL)
                .text("Today is a wonderful day to build something people love!")
                .voice(AudioParameters.Voice.CHERRY)
                .languageType("English") // 建议与文本语种一致，以获得正确的发音和自然的语调。
                // 如需使用指令控制功能，请取消下方注释，并将model替换为qwen3-tts-instruct-flash
                // .parameter("instructions","语速较快，带有明显的上扬语调，适合介绍时尚产品。")
                // .parameter("optimize_instructions",true)
                .build();
        MultiModalConversationResult result = conv.call(param);
        String audioUrl = result.getOutput().getAudio().getUrl();
        System.out.print(audioUrl);

        // 下载音频文件到本地
        try (InputStream in = new URL(audioUrl).openStream();
             FileOutputStream out = new FileOutputStream("downloaded_audio.wav")) {
            byte[] buffer = new byte[1024];
            int bytesRead;
            while ((bytesRead = in.read(buffer)) != -1) {
                out.write(buffer, 0, bytesRead);
            }
            System.out.println("\n音频文件已下载到本地: downloaded_audio.wav");
        } catch (Exception e) {
            System.out.println("\n下载音频文件时出错: " + e.getMessage());
        }
    }
    public static void main(String[] args) {
        try {
            // 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1
            Constants.baseHttpApiUrl = "https://dashscope.aliyuncs.com/api/v1";
            call();
        } catch (ApiException | NoApiKeyException | UploadFileException e) {
            System.out.println(e.getMessage());
        }
        System.exit(0);
    }
}
```

## cURL

```
# ======= 重要提示 =======
# 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation
# 新加坡地域和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
# === 执行时请删除该注释 ===

curl -X POST 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation' \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H 'Content-Type: application/json' \
-d '{
    "model": "qwen3-tts-flash",
    "input": {
        "text": "那我来给大家推荐一款T恤，这款呢真的是超级好看，这个颜色呢很显气质，而且呢也是搭配的绝佳单品，大家可以闭眼入，真的是非常好看，对身材的包容性也很好，不管啥身材的宝宝呢，穿上去都是很好看的。推荐宝宝们下单哦。",
        "voice": "Cherry",
        "language_type": "Chinese"
    }
}'
```

## 流式输出

流式模式下，音频数据以 Base64 编码的 PCM 格式逐段返回，最后一个数据包中包含完整音频的 URL。

## Python

```
# coding=utf-8
#
# Installation instructions for pyaudio:
# APPLE Mac OS X
#   brew install portaudio
#   pip install pyaudio
# Debian/Ubuntu
#   sudo apt-get install python-pyaudio python3-pyaudio
#   or
#   pip install pyaudio
# CentOS
#   sudo yum install -y portaudio portaudio-devel && pip install pyaudio
# Microsoft Windows
#   python -m pip install pyaudio

import os
import dashscope
import pyaudio
import time
import base64
import numpy as np

# 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1
dashscope.base_http_api_url = 'https://dashscope.aliyuncs.com/api/v1'

p = pyaudio.PyAudio()
# 创建音频流
stream = p.open(format=pyaudio.paInt16,
                channels=1,
                rate=24000,
                output=True)

text = "你好啊，我是千问"
response = dashscope.MultiModalConversation.call(
    # 新加坡和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
    # 若没有配置环境变量，请用百炼API Key将下行替换为：api_key = "sk-xxx"
    api_key=os.getenv("DASHSCOPE_API_KEY"),
    # 如需使用指令控制功能，请将model替换为qwen3-tts-instruct-flash
    model="qwen3-tts-flash",
    text=text,
    voice="Cherry",
    language_type="Chinese",  # 建议与文本语种一致，以获得正确的发音和自然的语调。
    # 如需使用指令控制功能，请取消下方注释，并将model替换为qwen3-tts-instruct-flash
    # instructions='语速较快，带有明显的上扬语调，适合介绍时尚产品。',
    # optimize_instructions=True,
    stream=True
)

for chunk in response:
    if chunk.output is not None:
      audio = chunk.output.audio
      if audio.data is not None:
          wav_bytes = base64.b64decode(audio.data)
          audio_np = np.frombuffer(wav_bytes, dtype=np.int16)
          # 直接播放音频数据
          stream.write(audio_np.tobytes())
      if chunk.output.finish_reason == "stop":
          print("finish at: {} ", chunk.output.audio.expires_at)
time.sleep(0.8)
# 清理资源
stream.stop_stream()
stream.close()
p.terminate()
```

## Java

需要导入Gson依赖，若是使用Maven或者Gradle，添加依赖方式如下：

### Maven

在`pom.xml`中添加如下内容：

```
<!-- https://mvnrepository.com/artifact/com.google.code.gson/gson -->
<dependency>
    <groupId>com.google.code.gson</groupId>
    <artifactId>gson</artifactId>
    <version>2.13.1</version>
</dependency>
```

### Gradle

在`build.gradle`中添加如下内容：

```
// https://mvnrepository.com/artifact/com.google.code.gson/gson
implementation("com.google.code.gson:gson:2.13.1")
```

```
import com.alibaba.dashscope.aigc.multimodalconversation.AudioParameters;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversation;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversationParam;
import com.alibaba.dashscope.aigc.multimodalconversation.MultiModalConversationResult;
import com.alibaba.dashscope.exception.ApiException;
import com.alibaba.dashscope.exception.NoApiKeyException;
import com.alibaba.dashscope.exception.UploadFileException;
import com.alibaba.dashscope.protocol.Protocol;
import com.alibaba.dashscope.utils.Constants;
import io.reactivex.Flowable;
import javax.sound.sampled.*;
import java.util.Base64;

public class Main {
    // 如需使用指令控制功能，请将MODEL替换为qwen3-tts-instruct-flash
    private static final String MODEL = "qwen3-tts-flash";
    public static void streamCall() throws ApiException, NoApiKeyException, UploadFileException {
        MultiModalConversation conv = new MultiModalConversation();
        MultiModalConversationParam param = MultiModalConversationParam.builder()
                // 新加坡和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
                // 若没有配置环境变量，请用百炼API Key将下行替换为：.apiKey("sk-xxx")
                .apiKey(System.getenv("DASHSCOPE_API_KEY"))
                .model(MODEL)
                .text("Today is a wonderful day to build something people love!")
                .voice(AudioParameters.Voice.CHERRY)
                .languageType("English") // 建议与文本语种一致，以获得正确的发音和自然的语调。
                // 如需使用指令控制功能，请取消下方注释，并将model替换为qwen3-tts-instruct-flash
                // .parameter("instructions","语速较快，带有明显的上扬语调，适合介绍时尚产品。")
                // .parameter("optimize_instructions",true)
                .build();
        Flowable<MultiModalConversationResult> result = conv.streamCall(param);
        result.blockingForEach(r -> {
            try {
                // 1. 获取Base64编码的音频数据
                String base64Data = r.getOutput().getAudio().getData();
                byte[] audioBytes = Base64.getDecoder().decode(base64Data);

                // 2. 配置音频格式（根据API返回的音频格式调整）
                AudioFormat format = new AudioFormat(
                        AudioFormat.Encoding.PCM_SIGNED,
                        24000, // 采样率（需与API返回格式一致）
                        16,    // 采样位数
                        1,     // 声道数
                        2,     // 帧大小（位数/字节数）
                        24000, // 数据传输率（需与采样率一致）
                        false  // 是否压缩
                );

                // 3. 实时播放音频数据
                DataLine.Info info = new DataLine.Info(SourceDataLine.class, format);
                try (SourceDataLine line = (SourceDataLine) AudioSystem.getLine(info)) {
                    if (line != null) {
                        line.open(format);
                        line.start();
                        line.write(audioBytes, 0, audioBytes.length);
                        line.drain();
                    }
                }
            } catch (LineUnavailableException e) {
                e.printStackTrace();
            }
        });
    }
    public static void main(String[] args) {
        // 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1
        Constants.baseHttpApiUrl = "https://dashscope.aliyuncs.com/api/v1";
        try {
            streamCall();
        } catch (ApiException | NoApiKeyException | UploadFileException e) {
            System.out.println(e.getMessage());
        }
        System.exit(0);
    }
}
```

## cURL

```
# ======= 重要提示 =======
# 以下为北京地域url，若使用新加坡地域的模型，需将url替换为：https://dashscope-intl.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation
# 新加坡地域和北京地域的API Key不同。获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key
# === 执行时请删除该注释 ===

curl -X POST 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation' \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H 'Content-Type: application/json' \
-H 'X-DashScope-SSE: enable' \
-d '{
    "model": "qwen3-tts-flash",
    "input": {
        "text": "那我来给大家推荐一款T恤，这款呢真的是超级好看，这个颜色呢很显气质，而且呢也是搭配的绝佳单品，大家可以闭眼入，真的是非常好看，对身材的包容性也很好，不管啥身材的宝宝呢，穿上去都是很好看的。推荐宝宝们下单哦。",
        "voice": "Cherry",
        "language_type": "Chinese"
    }
}'
```

## CosyVoice

以下示例演示如何通过 HTTP API 调用 CosyVoice 模型进行非实时语音合成。

**重要**

CosyVoice 非实时语音合成仅在北京地域可用。

## 非流式输出

非流式模式下，返回体中包含合成音频的 URL，有效期为 24 小时。

```
curl -X POST https://dashscope.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H "Content-Type: application/json" \
-d '{
    "model": "cosyvoice-v3-flash",
    "input": {
      "text": "我家的后面有一个很大的园。",
      "voice": "longanyang",
      "format": "wav",
      "sample_rate": 24000
    }
}'
```

## 流式输出

添加 `X-DashScope-SSE: enable` Header 开启流式输出，服务端会以 SSE（Server-Sent Events）分段返回音频数据。

```
curl -X POST https://dashscope.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H "Content-Type: application/json" \
-H "X-DashScope-SSE: enable" \
-d '{
    "model": "cosyvoice-v3-flash",
    "input": {
      "text": "我家的后面有一个很大的园。",
      "voice": "longanyang",
      "format": "wav",
      "sample_rate": 24000
    }
}'
```

## MiniMax

以下示例演示如何调用 MiniMax 模型进行非实时语音合成。MiniMax 支持情感控制、语速语调调节等特性。

**重要**

MiniMax 非实时语音合成仅在北京地域可用。

## 非流式输出

非流式模式下，返回完整的合成音频。

```
curl -X POST "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation" \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H "Content-Type: application/json" \
-d '{
  "model": "MiniMax/speech-2.8-hd",
  "input": {
    "text": "今天天气真不错，适合出去走走。",
    "voice_setting": {
      "voice_id": "male-qn-qingse",
      "speed": 1,
      "vol": 1,
      "pitch": 0,
      "emotion": "happy"
    },
    "audio_setting": {
      "sample_rate": 32000,
      "bitrate": 128000,
      "format": "mp3",
      "channel": 1
    }
  }
}'
```

## 流式输出

添加 `X-DashScope-SSE: enable` Header 开启流式输出。

```
# 获取API Key：https://help.aliyun.com/zh/model-studio/get-api-key

curl -X POST "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation" \
-H "Authorization: Bearer $DASHSCOPE_API_KEY" \
-H "Content-Type: application/json" \
-H "X-DashScope-SSE: enable" \
-d '{
  "model": "MiniMax/speech-2.8-hd",
  "input": {
    "text": "今天天气真不错，适合出去走走。",
    "voice_setting": {
      "voice_id": "male-qn-qingse",
      "speed": 1,
      "vol": 1,
      "pitch": 0,
      "emotion": "happy"
    },
    "audio_setting": {
      "sample_rate": 32000,
      "bitrate": 128000,
      "format": "mp3",
      "channel": 1
    }
  }
}'
```

## **进阶功能**

### **指令控制**

指令控制允许您通过自然语言描述精确控制语音的表达效果，无需调整复杂的音频参数。只需用简单的文字描述，即可让合成语音呈现特定的音调、语速、情感或音色特点，无需调整复杂的音频参数。

**支持的模型**：

-   CosyVoice：`cosyvoice-v3.5-plus`、`cosyvoice-v3.5-flash`、`cosyvoice-v3-flash`
    
    不同模型对指令的格式要求不同：
    
    -   `cosyvoice-v3.5-plus`、`cosyvoice-v3.5-flash`：可输入任意指令控制合成效果（如情感、语速等）。
        
    -   `cosyvoice-v3-flash` 的声音设计或声音复刻音色：可输入任意指令控制合成效果。
        
    -   `cosyvoice-v3-flash` 的系统音色：指令必须使用固定格式和内容，详情请参见[CosyVoice音色列表](https://help.aliyun.com/zh/model-studio/cosyvoice-voice-list)。
        
-   Qwen-TTS：仅支持千问3-TTS-Instruct-Flash-Realtime系列模型。
    

**使用方式**：

-   CosyVoice：通过 `instructions` 参数指定指令内容，例如“语速较快，带有明显的上扬语调，适合介绍时尚产品”。
    
-   Qwen-TTS：通过 `instruction` 参数指定指令内容，例如“语速较快，带有明显的上扬语调，适合介绍时尚产品”。
    

**指令文本支持的语言**：

-   CosyVoice：
    
    -   `cosyvoice-v3.5-plus`、`cosyvoice-v3.5-flash`：中文、英文、法语、德语、日语、韩语、俄语、葡萄牙语、泰语、印尼语、越南语。
        
    -   `cosyvoice-v3-flash`：中文、英文、法语、德语、日语、韩语、俄语。
        
-   Qwen-TTS：仅支持中文和英文。
    

**指令文本长度限制**：

-   CosyVoice：不超过 100 字符。汉字（包括简体/繁体汉字、日文汉字和韩文汉字）按 2 个字符计算，其他字符（如标点符号、字母、数字、日韩文假名/谚文等）按 1 个字符计算。
    
-   Qwen-TTS：不超过 1600 Token。
    

**适用场景**：

-   有声书和广播剧配音
    
-   广告和宣传片配音
    
-   游戏角色和动画配音
    
-   情感化的智能语音助手
    
-   纪录片和新闻播报
    

**如何编写高质量的声音描述：**

-   核心原则：
    
    1.  具体而非模糊：使用能描绘具体声音特质的词语，如“低沉”、“清脆”、“语速偏快”。避免使用“好听”、“普通”等主观且缺乏信息量的词汇。
        
    2.  多维而非单一：好的描述通常结合多个维度（如音调、语速、情感等）。仅描述单一维度（如“高音”）过于宽泛，难以生成特色鲜明的效果。
        
    3.  客观而非主观：聚焦声音本身的物理和感知特征，而非个人喜好。例如，用“音调偏高，带有活力”代替“我最喜欢的声音”。
        
    4.  原创而非模仿：请描述声音的特质，而非要求模仿特定人物（如名人、演员）。模仿请求涉及版权风险，且模型不支持直接模仿。
        
    5.  简洁而非冗余：确保每个词都有意义。避免重复同义词或堆砌无意义的强调词（如”非常非常棒的声音”）。
        
-   描述维度参考：组合多个维度可以创造更丰富的表达效果。
    
    | **维度** | **描述示例** |
    | --- | --- |
    | 音调  | 高音、中音、低音、偏高、偏低 |
    | 语速  | 快速、中速、缓慢、偏快、偏慢 |
    | 情感  | 开朗、沉稳、温柔、严肃、活泼、冷静、治愈 |
    | 特点  | 有磁性、清脆、沙哑、圆润、甜美、浑厚、有力 |
    | 用途  | 新闻播报、广告配音、有声书、动画角色、语音助手、纪录片解说 |
    
-   示例：
    
    -   标准播音风格：吐字清晰精准，字正腔圆
        
    -   情绪递进效果：音量由正常对话迅速增强至高喊，性格直率，情绪易激动且外露
        
    -   特殊情感状态：哭腔导致发音略微含糊，略显沙哑，带有明显哭腔的紧张感
        
    -   广告配音风格：音调偏高，语速中等，充满活力和感染力，适合广告配音
        
    -   温柔治愈风格：语速偏慢，音调温柔甜美，语气治愈温暖，像贴心朋友般关怀
        

## **适用范围**

**不同服务部署范围支持的模型不同：**

## 中国内地

服务部署范围为[中国内地](https://help.aliyun.com/zh/model-studio/regions/#080da663a75xh)时，模型推理计算资源仅限于中国内地；静态数据存储于您所选的地域。该部署范围支持的地域：华北2（北京）。

调用以下模型时，请选择北京地域的[API Key](https://bailian.console.aliyun.com/?tab=model#/api-key)：

-   **CosyVoice**：cosyvoice-v3.5-plus、cosyvoice-v3.5-flash、cosyvoice-v3-plus、cosyvoice-v3-flash、cosyvoice-v2
    
-   **MiniMax**：MiniMax/speech-2.8-hd、MiniMax/speech-02-hd、MiniMax/speech-2.8-turbo、MiniMax/speech-02-turbo
    
-   **Qwen-TTS**：
    
    -   **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash（稳定版，当前等同qwen3-tts-instruct-flash-2026-01-26）、qwen3-tts-instruct-flash-2026-01-26（最新快照版）
        
    -   **千问3-TTS-VD****：**qwen3-tts-vd-2026-01-26（最新快照版）
        
    -   **千问3-TTS-VC****：**qwen3-tts-vc-2026-01-22（最新快照版）
        
    -   **千问3-TTS-Flash**：qwen3-tts-flash（稳定版，当前等同qwen3-tts-flash-2025-11-27）、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18
        
    -   **千问-TTS**：qwen-tts（稳定版，当前等同qwen-tts-2025-04-10）、qwen-tts-latest（最新版，当前等同qwen-tts-2025-05-22）、qwen-tts-2025-05-22（快照版）、qwen-tts-2025-04-10（快照版）
        

## 国际

服务部署范围为[国际](https://help.aliyun.com/zh/model-studio/regions/#080da663a75xh)时，模型推理计算资源在全球范围内动态调度（不含中国内地）；静态数据存储于您所选的地域。该部署范围支持的地域：新加坡。

调用以下模型时，请选择新加坡地域的[API Key](https://modelstudio.console.aliyun.com/?tab=dashboard#/api-key)：

-   **Qwen-TTS**：
    
    -   **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash（稳定版，当前等同qwen3-tts-instruct-flash-2026-01-26）、qwen3-tts-instruct-flash-2026-01-26（最新快照版）
        
    -   **千问3-TTS-VD****：**qwen3-tts-vd-2026-01-26（最新快照版）
        
    -   **千问3-TTS-VC****：**qwen3-tts-vc-2026-01-22（最新快照版）
        
    -   **千问3-TTS-Flash**：qwen3-tts-flash（稳定版，当前等同qwen3-tts-flash-2025-11-27）、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18
        

## **支持的系统音色**

不同模型支持的音色有所差异。使用时，将请求参数`voice`设置为下表中**voice 参数**列对应的值即可。

-   [CosyVoice音色列表](https://help.aliyun.com/zh/model-studio/cosyvoice-voice-list)
    
-   Qwen-TTS音色列表：
    
    | `**voice**`**参数** | **详情** | **支持语种** | **支持模型** |
    | `Cherry` | **音色名**：芊悦 **描述**：阳光积极、亲切自然小姐姐（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 - **千问-TTS**：qwen-tts、qwen-tts-2025-04-10、qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Serena` | **音色名**：苏瑶 **描述**：温柔小姐姐（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 - **千问-TTS**：qwen-tts、qwen-tts-2025-04-10、qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Ethan` | **音色名**：晨煦 **描述**：标准普通话，带部分北方口音。阳光、温暖、活力、朝气（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 - **千问-TTS**：qwen-tts、qwen-tts-2025-04-10、qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Chelsie` | **音色名**：千雪 **描述**：二次元虚拟女友（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 - **千问-TTS**：qwen-tts、qwen-tts-2025-04-10、qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Momo` | **音色名**：茉兔 **描述**：撒娇搞怪，逗你开心（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Vivian` | **音色名**：十三 **描述**：拽拽的、可爱的小暴躁（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Moon` | **音色名**：月白 **描述**：率性帅气的月白（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Maia` | **音色名**：四月 **描述**：知性与温柔的碰撞（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Kai` | **音色名**：凯 **描述**：耳朵的一场SPA（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Nofish` | **音色名**：不吃鱼 **描述**：不会翘舌音的设计师（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Bella` | **音色名**：萌宝 **描述**：喝酒不打醉拳的小萝莉（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Jennifer` | **音色名**：詹妮弗 **描述**：品牌级、电影质感般美语女声（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Ryan` | **音色名**：甜茶 **描述**：节奏拉满，戏感炸裂，真实与张力共舞（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Katerina` | **音色名**：卡捷琳娜 **描述**：御姐音色，韵律回味十足（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Aiden` | **音色名**：艾登 **描述**：精通厨艺的美语大男孩（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Eldric Sage` | **音色名**：沧明子 **描述**：沉稳睿智的老者，沧桑如松却心明如镜（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Mia` | **音色名**：乖小妹 **描述**：温顺如春水，乖巧如初雪（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Mochi` | **音色名**：沙小弥 **描述**：聪明伶俐的小大人，童真未泯却早慧如禅（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Bellona` | **音色名**：燕铮莺 **描述**：声音洪亮，吐字清晰，人物鲜活，听得人热血沸腾；金戈铁马入梦来，字正腔圆间尽显千面人声的江湖（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Vincent` | **音色名**：田叔 **描述**：一口独特的沙哑烟嗓，一开口便道尽了千军万马与江湖豪情（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Bunny` | **音色名**：萌小姬 **描述**：“萌属性”爆棚的小萝莉（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Neil` | **音色名**：阿闻 **描述**：平直的基线语调，字正腔圆的咬字发音，这就是最专业的新闻主持人（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Elias` | **音色名**：墨讲师 **描述**：既保持学科严谨性，又通过叙事技巧将复杂知识转化为可消化的认知模块（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Arthur` | **音色名**：徐大爷 **描述**：被岁月和旱烟浸泡过的质朴嗓音，不疾不徐地摇开了满村的奇闻异事（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Nini` | **音色名**：邻家妹妹 **描述**：糯米糍一样又软又黏的嗓音，那一声声拉长了的“哥哥”，甜得能把人的骨头都叫酥了（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Seren` | **音色名**：小婉 **描述**：温和舒缓的声线，助你更快地进入睡眠，晚安，好梦（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Pip` | **音色名**：顽屁小孩 **描述**：调皮捣蛋却充满童真的他来了，这是你记忆中的小新吗（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Stella` | **音色名**：少女阿月 **描述**：平时是甜到发腻的迷糊少女音，但在喊出“代表月亮消灭你”时，瞬间充满不容置疑的爱与正义（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Instruct-Flash**：qwen3-tts-instruct-flash、qwen3-tts-instruct-flash-2026-01-26 - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Bodega` | **音色名**：博德加 **描述**：热情的西班牙大叔（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Sonrisa` | **音色名**：索尼莎 **描述**：热情开朗的拉美大姐（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Alek` | **音色名**：阿列克 **描述**：一开口，是战斗民族的冷，也是毛呢大衣下的暖（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Dolce` | **音色名**：多尔切 **描述**：慵懒的意大利大叔（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Sohee` | **音色名**：素熙 **描述**：温柔开朗，情绪丰富的韩国欧尼（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Ono Anna` | **音色名**：小野杏 **描述**：鬼灵精怪的青梅竹马（女性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Lenn` | **音色名**：莱恩 **描述**：理性是底色，叛逆藏在细节里——穿西装也听后朋克的德国青年（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Emilien` | **音色名**：埃米尔安 **描述**：浪漫的法国大哥哥（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Andre` | **音色名**：安德雷 **描述**：声音磁性，自然舒服、沉稳男生（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Radio Gol` | **音色名**：拉迪奥·戈尔 **描述**：足球诗人Rádio Gol！今天我要用名字为你们解说足球（男性） | 中文（普通话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27 |
    | `Jada` | **音色名**：上海-阿珍 **描述**：风风火火的沪上阿姐（女性） | 中文（上海话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 - **千问-TTS**：qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Dylan` | **音色名**：北京-晓东 **描述**：北京胡同里长大的少年（男性） | 中文（北京话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 - **千问-TTS**：qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Li` | **音色名**：南京-老李 **描述**：耐心的瑜伽老师（男性） | 中文（南京话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Marcus` | **音色名**：陕西-秦川 **描述**：面宽话短，心实声沉——老陕的味道（男性） | 中文（陕西话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Roy` | **音色名**：闽南-阿杰 **描述**：诙谐直爽、市井活泼的台湾哥仔形象（男性） | 中文（闽南语）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Peter` | **音色名**：天津-李彼得 **描述**：天津相声，专业捧哏（男性） | 中文（天津话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Sunny` | **音色名**：四川-晴儿 **描述**：甜到你心里的川妹子（女性） | 中文（四川话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 - **千问-TTS**：qwen-tts-latest、qwen-tts-2025-05-22 |
    | `Eric` | **音色名**：四川-程川 **描述**：一个跳脱市井的四川成都男子（男性） | 中文（四川话）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Rocky` | **音色名**：粤语-阿强 **描述**：幽默风趣的阿强，在线陪聊（男性） | 中文（粤语）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    | `Kiki` | **音色名**：粤语-阿清 **描述**：甜美的港妹闺蜜（女性） | 中文（粤语）、英语、法语、德语、俄语、意大利语、西班牙语、葡萄牙语、日语、韩语 | - **千问3-TTS-Flash**：qwen3-tts-flash、qwen3-tts-flash-2025-11-27、qwen3-tts-flash-2025-09-18 |
    

## **API 参考**

-   [非实时语音合成-CosyVoice API参考](https://help.aliyun.com/zh/model-studio/non-realtime-cosyvoice-api/)
    
-   [非实时语音合成-千问API参考](https://help.aliyun.com/zh/model-studio/qwen-tts-api)
    
-   [非实时语音合成-MiniMax API 参考](https://help.aliyun.com/zh/model-studio/minimax-speech-synthesis/)
    

## **常见问题**

### **Q：音频文件链接的有效期是多久？**

A：音频文件链接在生成后 24 小时内有效，过期后需重新调用接口生成。

 span.aliyun-docs-icon { color: transparent !important; font-size: 0 !important; } span.aliyun-docs-icon:before { color: black; font-size: 16px; } span.aliyun-docs-icon.icon-size-20:before { font-size: 20px; } span.aliyun-docs-icon.icon-size-22:before { font-size: 22px; } span.aliyun-docs-icon.icon-size-24:before { font-size: 24px; } span.aliyun-docs-icon.icon-size-26:before { font-size: 26px; } span.aliyun-docs-icon.icon-size-28:before { font-size: 28px; }