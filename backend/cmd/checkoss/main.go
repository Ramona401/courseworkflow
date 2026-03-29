package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/services"
)

func main() {
	cfg := config.Load()
	database.Init(cfg)

	var moduleID int
	rows, _ := database.DB.Query(context.Background(), "SELECT external_module_id FROM courses WHERE course_code='G1-01'")
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&moduleID)
	}

	ossService := services.NewOSSService(cfg)

	// 1. FetchModuleIndex（indexes JSON）— 索引来源
	fmt.Println("=== indexes JSON（FetchModuleIndex）===")
	indexFile, err := ossService.FetchModuleIndex(moduleID)
	if err != nil {
		fmt.Println("FetchModuleIndex error:", err)
	} else {
		sort.Slice(indexFile.Indexes, func(i, j int) bool {
			return indexFile.Indexes[i].SortOrder < indexFile.Indexes[j].SortOrder
		})
		fmt.Printf("条目数: %d\n", len(indexFile.Indexes))
		for _, entry := range indexFile.Indexes {
			fmt.Printf("  sort=%d name=%s\n", entry.SortOrder, entry.Name)
		}
	}

	// 2. FetchModuleDetail（modules JSON）— HTML来源
	fmt.Println("\n=== modules JSON（FetchModuleDetail）===")
	detail, err := ossService.FetchModuleDetail(moduleID)
	if err != nil {
		fmt.Println("FetchModuleDetail error:", err)
	} else {
		re := regexp.MustCompile(`^P(\d+)`)
		sort.Slice(detail.Lessons, func(i, j int) bool {
			return detail.Lessons[i].Order < detail.Lessons[j].Order
		})
		fmt.Printf("条目数: %d\n", len(detail.Lessons))
		for _, l := range detail.Lessons {
			pageNum := 0
			if m := re.FindStringSubmatch(l.Title); m != nil {
				pageNum, _ = strconv.Atoi(m[1])
			}
			fmt.Printf("  order=%d id=%d title=%s → 解析页码P%02d\n", l.Order, l.ID, l.Title, pageNum)
		}
	}
}
