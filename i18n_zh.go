package main

// Chinese (Simplified) interface texts.
// Not reviewed by a native speaker.
var messagesZH = map[string]string{
	// Page
	"dryRunBanner":  "dry-run：不会发送任何数据",
	"sessionOn":     "会话进行中",
	"sessionOff":    "没有会话",
	"weather":       "天气",
	"tier":          "等级",
	"hull":          "舰船级别",
	"fit":           "装配",
	"allCans":       "我会搜刮所有容器",
	"allCansHelp":   "勾选后，Abyss Tracker 把发送的记录归为“all cans”，否则归为“Bioadaptive and some cans”。",
	"fitNeedsToken": "填写 Abyss Tracker 令牌后可列出你的装配",
	"none":          "无",
	"startSession":  "开始会话",
	"updateSession": "更新会话",
	"endSession":    "结束会话",
	"settings":      "设置",
	"tokenLabel":    "Abyss Tracker 令牌",
	"tokenSaved":    "已保存；留空则保持不变",
	"tokenHowTo":    "在 abysstracker.com → My account → Personal access tokens 创建",
	"logDirLabel":   "EVE 聊天日志文件夹",
	"logDirHelp":    "EVE 在此写入 Local_….txt 文件",
	"language":      "语言",
	"save":          "保存",

	// Session
	"settingsSaved":  "设置已保存。",
	"sessionStarted": "会话进行中：%s T%d。第一次深渊前请复制货柜舱。",
	"sessionEnded":   "会话已结束。",
	"errNoLogDir":    "找不到 EVE 聊天日志文件夹：%s（设置）",
	"errNoToken":     "请先填写 Abyss Tracker 令牌（设置）",

	// Local log
	"logFollowed":   "正在跟踪本地频道日志：%s。",
	"noLocalLog":    "%s 中没有本地频道日志：请启动 EVE。",
	"logUnreadable": "无法读取本地频道日志：%v",

	// Runs
	"baseSaved":           "已保存初始货柜舱。",
	"noBaseAtStart":       "检测到深渊开始时没有货柜舱副本：本次不会发送。",
	"midRunIgnored":       "会话开始时已在深渊中：已忽略。请为下一次复制货柜舱。",
	"runDoneCopy":         "深渊已结束。请复制货柜舱（Ctrl+A，Ctrl+C）。",
	"runLost":             "深渊失败（在深渊中阵亡）。",
	"movedRecopy":         "星系已变更：下一次深渊前请重新复制货柜舱。",
	"notSentNoBase":       "本次未发送：开始时没有货柜舱副本。请现在复制货柜舱，供下一次使用。",
	"notSentNewRun":       "本次未发送：未复制货柜舱就开始了新的深渊。",
	"notSentMoved":        "本次未发送：复制货柜舱前星系已变更。",
	"notSentStopped":      "本次未发送：复制货柜舱前会话已结束。",
	"notSentUnknownItems": "本次未发送：EVE 不认识这些物品（%s）。复制的内容可能不是货柜舱。",
	"notSentError":        "本次未发送：%v",

	// Upload
	"dryRunStatus": "[dry-run] 用时 %v 的深渊未发送：\n%s",
	"dryRunNotify": "[dry-run] 检测到用时 %v 的深渊。",
	"sentStatus":   "已发送：%s",
	"sentNotify":   "已发送到 Abyss Tracker。",
	"rejected":     "Abyss Tracker 拒绝了本次记录：%s",
	"sendFailed":   "无法发送，本次记录保留在队列中：%v",
}
