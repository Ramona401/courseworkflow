package main

// generate-style-thumbnails 是系统预设画风缩略图的一次性运维命令。
//
// 默认行为：
//   - 初始化正式配置和数据库连接；
//   - 读取AI管理中心中的图片网关配置；
//   - 已存在的稳定缩略图直接复用；
//   - 只生成缺失的预设缩略图；
//   - 写入manifest.json；
//   - 校验生成数量和文件状态；
//   - 将图片和manifest权限调整为Nginx可读取的0644。
//
// 强制重生全部图片：
//
//	go run ./cmd/generate-style-thumbnails --force
//
// 命令应从backend目录执行，使config.Load能够读取backend/.env。

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/services"
)

func main() {
	os.Exit(
		run(),
	)
}

func run() int {
	force :=
		flag.Bool(
			"force",
			false,
			"重新生成全部预设画风缩略图",
		)

	timeout :=
		flag.Duration(
			"timeout",
			25*time.Minute,
			"整批生成的最长执行时间",
		)

	flag.Parse()

	cfg :=
		config.Load()

	database.Init(
		cfg,
	)
	defer database.Close()

	imageConfig, err :=
		ai.GetImageConfig(
			cfg.GetAESKey(),
		)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"读取图片网关配置失败:",
			err,
		)
		return 1
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			*timeout,
		)
	defer cancel()

	manifest, err :=
		services.GenerateCoursewarePresetStyleThumbnails(
			ctx,
			imageConfig,
			*force,
		)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"系统预设画风缩略图生成失败:",
			err,
		)
		return 1
	}

	expectedCount :=
		len(
			services.ListCoursewarePresetStyleDescriptors(),
		)

	if manifest == nil {
		fmt.Fprintln(
			os.Stderr,
			"系统预设画风缩略图生成失败：清单为空",
		)
		return 1
	}

	if len(manifest.Styles) !=
		expectedCount {
		fmt.Fprintf(
			os.Stderr,
			"系统预设画风缩略图数量不完整：期望%d张，实际%d张\n",
			expectedCount,
			len(manifest.Styles),
		)
		return 1
	}

	if err :=
		ensurePresetThumbnailPublicFiles(
			manifest,
		); err != nil {
		fmt.Fprintln(
			os.Stderr,
			"系统预设画风缩略图公开文件校验失败:",
			err,
		)
		return 1
	}

	fmt.Println(
		"系统预设画风缩略图生成完成",
	)
	fmt.Println(
		"运行模式:",
		func() string {
			if *force {
				return "强制全部重生"
			}

			return "断点复用，仅生成缺失项"
		}(),
	)
	fmt.Println(
		"生成或复用数量:",
		len(manifest.Styles),
	)
	fmt.Println(
		"图片尺寸:",
		manifest.ImageSize,
	)
	fmt.Println(
		"浏览器清单地址:",
		"/uploads/courseware-assets/style-presets/manifest.json",
	)

	for _, item :=
		range manifest.Styles {
		fmt.Printf(
			"- %s [%s] %s (%s, %d bytes)\n",
			item.Label,
			item.Key,
			item.URL,
			item.MimeType,
			item.FileSize,
		)
	}

	return 0
}
