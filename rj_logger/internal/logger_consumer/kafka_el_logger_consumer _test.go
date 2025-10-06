package logger_consumer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	mock_el "github.com/RoyceAzure/lab/rj_logger/pkg/elsearch/mock"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type TestMsg struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

func generateTestMessage(n int) []kafka.Message {
	t := make([]kafka.Message, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 4, 4)
		binary.BigEndian.PutUint32(buf, uint32(i))
		testMsg := TestMsg{
			Topic:   "test",
			Message: fmt.Sprintf("this is test message %d", i),
		}

		b, err := json.Marshal(testMsg)
		if err != nil {
			panic(err)
		}
		m := kafka.Message{
			Key:   buf,
			Value: b,
		}
		t = append(t, m)
	}
	return t

}

func handleResult(exp, act []kafka.Message) (excepted map[string]any, actual map[string]any) {
	excepted, actual = make(map[string]any), make(map[string]any)
	for _, v := range exp {
		excepted[string(v.Key)] = v.Value
	}

	for _, v := range act {
		actual[string(v.Key)] = v.Value
	}
	return
}

func TestBasic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDao := mock_el.NewMockIElSearchDao(ctrl)

	cases := []struct {
		name        string
		setUPMock   func(m *mock_el.MockIElSearchDao)
		caseN       int
		bufferSize  int
		nRoutine    int
		setupIn     func(input []kafka.Message, in chan kafka.Message)
		exceptDatas []kafka.Message
	}{
		{
			name: "10 case, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      10,
			bufferSize: 10,
			nRoutine:   1,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name: "100 case,10 routine, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      100,
			bufferSize: 100,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name: "100 case,10 routine, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      100,
			bufferSize: 10,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name: "0 case,10 routine, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      0,
			bufferSize: 10,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name: "1 case,10 routine, all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      0,
			bufferSize: 10,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name: "100 case,10 routine,1 buffer size all pass",
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      100,
			bufferSize: 1,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := generateTestMessage(tc.caseN)

			in, out := make(chan kafka.Message, tc.bufferSize), make(chan kafka.Message, tc.caseN)
			go func() {
				defer close(in)
				tc.setupIn(data, in)
			}()

			tc.setUPMock(mockDao)

			KafkaElProcesser, err := NewKafkaElProcesser(mockDao, tc.bufferSize)
			require.Nil(t, err)

			var wg sync.WaitGroup
			for i := 0; i < tc.nRoutine; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					KafkaElProcesser.Process(context.Background(), in, out)
				}()
			}
			wg.Wait()
			close(out)

			res := make([]kafka.Message, 0, tc.bufferSize)
			for o := range out {
				res = append(res, o)
			}

			excepted, actual := handleResult(data, res)

			require.Equal(t, excepted, actual)
		})
	}
}
