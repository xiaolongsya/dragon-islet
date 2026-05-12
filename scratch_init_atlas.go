package main

import (
	"dragon-islet/internal/initialize"
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"fmt"
)

func main() {
	initialize.InitConfig()
	initialize.InitDB()

	title := "【创世之鳞】龙屿图谱 v1.0.0：从神话走向代码"
	content := `### 🌊 创世背景
龙屿（Dragon Islet）原本只存在于想象的边界，现在我们通过代码将其显化。v1.0.0 版本的发布，标志着龙屿底层架构与核心幻化能力的全面落成。

### 🛠️ 技术进阶实录
在 V1.0.0 的铸造过程中，我们攻克了以下核心命题：

1. **龙息显像 (AI Image Generation)**
   - **模型升级**：全面接入 GPT-Image-2 专用接口，支持精准的 Prompt 捕捉。
   - **画幅重塑**：突破单一比例限制，实现了 16:9、9:16、3:2 等多场景适配。
   - **性能优化**：固定 1K 分辨率显像，将平均生成耗时缩短至 50 秒内。

2. **神启契约 (AI Fortune)**
   - **动态解签**：基于 LLM 的文本生成能力，为游侠每日的签位注入诗意灵魂。
   - **持久化方案**：采用数据库账本模式，确保每位游侠每日仅能窥见一次天机。

3. **因果限流机制 (Stricter Quotas)**
   - **占位防刷**：引入了“先生成占位记录、后异步填充”的方案，物理级封杀并发刷图请求。
   - **状态轮询**：完善了 10 分钟超长稳健轮询机制，确保即使上游队列拥堵，显像也能最终落成。

4. **全域交互适配 (Mobile First)**
   - **底栏重构**：针对手持终端，将左侧导航重塑为底部交互 Bar，提升单手操作体验。
   - **响应式气泡**：动态计算消息宽度，确保在各种刘海屏下均能完美呈现。

### 📜 祭司寄语
铸龙非一日之功，V1.0.0 只是第一枚鳞片的落位。后续我们将开启更广阔的“灵力系统”与“社交纽带”功能。

—— 龙屿开发团队 敬上`

	archive := &model.Archive{
		Title:   title,
		Content: content,
		Type:    1, // 1 代表铸龙图谱
	}

	if err := global.DB.Create(archive).Error; err != nil {
		fmt.Printf("写入失败: %v\n", err)
	} else {
		fmt.Println("V1.0.0 创世文章已成功刻入铸龙图谱！")
	}
}
