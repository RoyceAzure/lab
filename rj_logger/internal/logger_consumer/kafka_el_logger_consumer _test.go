package logger_consumer

import (
	"encoding/binary"
	"fmt"
	"testing"

	mock_el "github.com/RoyceAzure/lab/rj_logger/pkg/elsearch/mock"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func generateTestMessage(n int) []kafka.Message {
	t := make([]kafka.Message, 0, n)
	for i := 0; i < n; i++ {
		var buf []byte
		binary.BigEndian.PutUint32(buf, uint32(i))
		v := fmt.Sprintf("this is % th msg", i)
		m := kafka.Message{
			Key:   buf,
			Value: []byte(v),
		}
		t = append(t, m)
	}
	return t

}

func TestBasic(t *testing.T) {
	ctrl := gomock.NewController(t) // t 是 *testing.T
	defer ctrl.Finish()

	mockDao := mock_el.NewMockIElSearchDao(ctrl)

	TestDatas := [][]kafka.Message{
		generateTestMessage(10),
	}

	cases := []struct {
		name        string
		setUPMock   func(m *mock_el.MockIElSearchDao)
		bufferSize  int
		nRoutine    int
		setupIn     func(bufferSize int, input []kafka.Message) chan kafka.Message
		setupOut    func(bufferSize int) chan kafka.Message
		exceptDatas []kafka.Message
	}{
		{
			name: "10 case, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil)
			},
			bufferSize: 10,
			nRoutine:   1,
			setupIn: func(bufferSize int, input []kafka.Message) chan kafka.Message {
				in := make(chan kafka.Message, bufferSize)
				for _, m := range input {
					in <- m
				}
				return in
			},
			setupOut: func(bufferSize int) chan kafka.Message {
				return make(chan kafka.Message, bufferSize)
			},
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := TestDatas[i]
			in := tc.setupIn(tc.bufferSize, data)
			out := tc.setupOut(tc.bufferSize)
			tc.setUPMock(mockDao)

			KafkaElProcesser, err := NewKafkaElProcesser(mockDao, tc.bufferSize)
			require.Nil(t, err)
		})
	}
}
