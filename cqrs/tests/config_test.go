package tests

import (
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/config"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	cf, err := config.GetPGConfig()
	require.NoError(t, err)
	require.NotNil(t, cf)
	require.NotEmpty(t, cf.DbName)
	require.NotEmpty(t, cf.DbHost)
	require.NotEmpty(t, cf.DbPort)
	require.NotEmpty(t, cf.DbUser)
	require.NotEmpty(t, cf.DbPas)
}

func TestConfigConcurrency(t *testing.T) {
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			cf, err := config.GetPGConfig()
			require.NoError(t, err)
			require.NotNil(t, cf)
			// 確保所有執行緒拿到的是同一個實例
			cf2, err := config.GetPGConfig()
			require.NoError(t, err)
			require.NotNil(t, cf2)
			require.Same(t, cf, cf2)
		}()
	}
	wg.Wait()
}

func TestConfigEnvOverride(t *testing.T) {
	// 設置環境變數
	os.Setenv("POSTGRES_DB", "test_db_override")
	defer os.Unsetenv("POSTGRES_DB")

	cf, err := config.GetPGConfig()
	require.NoError(t, err)
	require.Equal(t, "test_db_override", cf.DbName)
}

func TestConfigMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	baseline := m.Alloc

	// 重複獲取配置
	for i := 0; i < 1000; i++ {
		_, err := config.GetPGConfig()
		require.NoError(t, err)
	}

	runtime.ReadMemStats(&m)
	// 確保記憶體使用沒有顯著增加
	require.Less(t, m.Alloc-baseline, uint64(1024*1024), "memory usage increased significantly")
}
