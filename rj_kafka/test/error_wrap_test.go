package test

import (
	"errors"
	"fmt"
	"testing"

	kerrors "github.com/RoyceAzure/lab/rj_kafka/kafka/errors"
)

func wrapError(err error, layer int) error {
	return fmt.Errorf("layer %d: %w", layer, err)
}

func TestErrorWrapping(t *testing.T) {
	// 創建多層包裝的錯誤
	err := kerrors.ErrReadTimeout
	for i := 1; i <= 3; i++ {
		err = wrapError(err, i)
	}

	// 測試錯誤識別
	if !errors.Is(err, kerrors.ErrReadTimeout) {
		t.Error("應該能夠識別出 ErrReadTimeout，但是失敗了")
	}

	// 印出錯誤鏈
	t.Logf("完整錯誤訊息: %v", err)

	// 使用自定義的 KafkaError 包裝
	kafkaErr := kerrors.NewKafkaError("Read", "test-topic", err)

	// 測試通過 KafkaError 識別原始錯誤
	if !errors.Is(kafkaErr, kerrors.ErrReadTimeout) {
		t.Error("應該能夠通過 KafkaError 識別出 ErrReadTimeout，但是失敗了")
	}

	// 測試 KafkaError 的比較
	targetErr := kerrors.NewKafkaError("Read", "test-topic", nil)
	if !errors.Is(kafkaErr, targetErr) {
		t.Error("應該能夠匹配相同操作和主題的 KafkaError")
	}

	// 測試部分匹配（只匹配操作）
	partialErr := kerrors.NewKafkaError("Read", "", nil)
	if !errors.Is(kafkaErr, partialErr) {
		t.Error("應該能夠匹配只有操作名稱的 KafkaError")
	}

	// 測試不匹配的情況
	differentOpErr := kerrors.NewKafkaError("Write", "test-topic", nil)
	if errors.Is(kafkaErr, differentOpErr) {
		t.Error("不應該匹配不同操作的 KafkaError")
	}
}

func TestErrorComparison(t *testing.T) {
	// 創建兩個相同文字但不同身份的錯誤
	err1 := errors.New("read timeout")
	err2 := errors.New("read timeout")

	// 雖然錯誤訊息相同，但是它們是不同的錯誤
	if errors.Is(err1, err2) {
		t.Error("不同身份的錯誤不應該被視為相同")
	}

	// 直接比較字串
	if err1.Error() == err2.Error() {
		t.Log("錯誤訊息相同，但errors.Is認為是不同的錯誤")
	}

	// 使用同一個預定義的錯誤
	err3 := kerrors.ErrReadTimeout
	err4 := kerrors.ErrReadTimeout

	// 這裡會是true，因為是同一個錯誤實例
	if !errors.Is(err3, err4) {
		t.Error("相同的預定義錯誤應該被視為相同")
	}

	// 包裝後的錯誤比較
	wrappedErr := fmt.Errorf("wrapped: %w", err3)
	if !errors.Is(wrappedErr, kerrors.ErrReadTimeout) {
		t.Error("包裝後應該能識別出原始錯誤")
	}
}
