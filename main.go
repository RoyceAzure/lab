package main

import (
	"crypto/rand"
	"math/big"
)

const (
	SYMBOL_NUM int = 4

	SYMBOL_S1 int = 1
	SYMBOL_S2 int = 2
	SYMBOL_S3 int = 3
	SYMBOL_S4 int = 4

	// 連續配對數量
	MIN_AWARD_RANK_THRESHOLD int = 3
	RANK_3                   int = 3
	RANK_4                   int = 4
	RANK_5                   int = 5

	COL int = 5 //共5輪
	ROW int = 3 //共3列
)

// 中獎金額
var PAY_TABLE = map[int]map[int]float64{
	SYMBOL_S1: {
		RANK_3: 3.0,
		RANK_4: 6.0,
		RANK_5: 9.0,
	},
	SYMBOL_S2: {
		RANK_3: 2.0,
		RANK_4: 4.0,
		RANK_5: 6.0,
	},
	SYMBOL_S3: {
		RANK_3: 1.5,
		RANK_4: 2.5,
		RANK_5: 3.5,
	},
	SYMBOL_S4: {
		RANK_3: 1.0,
		RANK_4: 2.0,
		RANK_5: 3.0,
	},
}

type SlotResult struct {
	Symbols [][]int
	Awards  []Award
	Win     float64
}

type Award struct {
	Symbol    int   // 中獎圖標
	Rank      int   // 最大連線數
	Pay       int   // 得分
	Positions []Pos // 圖標位置
}

type Pos struct {
	Col int
	Row int
}

// 隨機
func GetRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	bigInt := new(big.Int).SetInt64(int64(max))
	i, _ := rand.Int(rand.Reader, bigInt)
	return int(i.Int64())
}

// 生成結果
func SpinSlotGame() SlotResult {
	result := SlotResult{}

	result.Symbols = getRandomSymbol()

	result.Awards, result.Win = getAndCalculateAllAwards(result.Symbols)

	return result
}

// 隨機生成盤面
func getRandomSymbol() [][]int {
	symbols := [][]int{}
	// TODO: Do something
	for i := 0; i < ROW; i++ {
		temp := []int{}
		for j := 0; j < COL; j++ {
			temp = append(temp, GetRandomInt(SYMBOL_NUM))
		}
		symbols = append(symbols, temp)
	}

	return symbols
}

// 取得和計算當前盤面所有獎項
func getAndCalculateAllAwards(symbols [][]int) (awards []Award, totalWin float64) {
	totalWin = 0

	// 對每個起始位置的每個符號進行搜尋
	for startRow := 0; startRow < ROW; startRow++ {
		symbol := symbols[startRow][0]
		findAllPaths(symbol, 0, startRow, symbols, []Pos{{Col: 0, Row: startRow}}, &awards, &totalWin)
	}

	return awards, totalWin
}

func findAllPaths(symbol int, col int, row int, symbols [][]int, currentPath []Pos, awards *[]Award, totalWin *float64) {
	// 如果已經到達最後一列，計算獎項
	if col == COL-1 {
		if len(currentPath) >= MIN_AWARD_RANK_THRESHOLD {
			pay := int(PAY_TABLE[symbol][len(currentPath)])
			*totalWin += float64(pay)
			*awards = append(*awards, Award{
				Symbol:    symbol,
				Rank:      len(currentPath),
				Pay:       pay,
				Positions: make([]Pos, len(currentPath)),
			})
			copy((*awards)[len(*awards)-1].Positions, currentPath)
		}
		return
	}

	// 尋找下一列中所有匹配的符號
	nextCol := col + 1
	found := false

	for nextRow := 0; nextRow < ROW; nextRow++ {
		if symbols[nextRow][nextCol] == symbol {
			found = true

			// 遞歸搜尋下一列
			findAllPaths(symbol, nextCol, nextRow, symbols, append(currentPath, Pos{Col: nextCol, Row: nextRow}), awards, totalWin)
		}
	}

	// 如果下一列沒找到匹配符號，檢查當前路徑是否達到最小獲獎長度
	if !found && len(currentPath) >= MIN_AWARD_RANK_THRESHOLD {
		pay := int(PAY_TABLE[symbol][len(currentPath)])
		*totalWin += float64(pay)
		*awards = append(*awards, Award{
			Symbol:    symbol,
			Rank:      len(currentPath),
			Pay:       pay,
			Positions: make([]Pos, len(currentPath)),
		})
		copy((*awards)[len(*awards)-1].Positions, currentPath)
	}
}
