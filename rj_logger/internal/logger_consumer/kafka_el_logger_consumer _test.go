package logger_consumer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	mock_el "github.com/RoyceAzure/lab/rj_logger/pkg/elsearch/mock"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type TestMsg struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

type TestCase struct {
	name             string
	generateCasefunc func(n int) []kafka.Message
	setUPMock        func(m *mock_el.MockIElSearchDao)
	caseN            int
	bufferSize       int
	nRoutine         int
	setupIn          func(input []kafka.Message, in chan kafka.Message)
	exceptDatas      []kafka.Message
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

func generateBadTestMessage(n int) []kafka.Message {
	t := make([]kafka.Message, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 4, 4)
		binary.BigEndian.PutUint32(buf, uint32(i))
		testMsg := fmt.Sprintf("this is test message %d", i)

		m := kafka.Message{
			Key:   buf,
			Value: []byte(testMsg),
		}
		t = append(t, m)
	}
	return t
}

func handleResult(exp, act, dlq []kafka.Message) (excepted map[string]any, actual map[string]any) {
	excepted, actual = make(map[string]any), make(map[string]any)
	for _, v := range exp {
		excepted[string(v.Key)] = v.Value
	}

	for _, v := range act {
		actual[string(v.Key)] = v.Value
	}

	for _, v := range dlq {
		actual[string(v.Key)] = v.Value
	}
	return
}

func TestBasic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDao := mock_el.NewMockIElSearchDao(ctrl)

	cases := []TestCase{
		{
			name:             "10 case, all pass",
			generateCasefunc: generateTestMessage,
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
			name:             "10 case, all pass",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      10,
			bufferSize: 100,
			nRoutine:   100,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name:             "100 case,10 routine, all pass",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      1000,
			bufferSize: 100,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name:             "100 case,10 routine, all pass",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      1000,
			bufferSize: 10,
			nRoutine:   10,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name:             "0 case,10 routine, all pass",
			generateCasefunc: generateTestMessage,
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
			name:             "1 case,10 routine, all pass",
			generateCasefunc: generateTestMessage,
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
			name:             "100 case,10 routine,1 buffer size all pass",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			caseN:      1000,
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
			data := tc.generateCasefunc(tc.caseN)

			in, out := make(chan kafka.Message, tc.bufferSize), make(chan kafka.Message, tc.caseN)
			dlq := make(chan kafka.Message, tc.bufferSize)
			go func() {
				defer close(in)
				tc.setupIn(data, in)
			}()

			tc.setUPMock(mockDao)

			KafkaElProcesser, err := NewKafkaElProcesser(mockDao, tc.bufferSize, time.Millisecond)
			require.Nil(t, err)

			var wg sync.WaitGroup
			for i := 0; i < tc.nRoutine; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					KafkaElProcesser.Process(context.Background(), in, out, dlq)
				}()
			}

			go func() {
				for {
					select {
					case _, ok := <-dlq:
						if !ok {
							return
						}
					}
				}
			}()

			wg.Wait()
			close(out)
			close(dlq)

			res := make([]kafka.Message, 0, tc.caseN)
			for o := range out {
				res = append(res, o)
			}

			failedMsg := make([]kafka.Message, 0, tc.caseN)
			for m := range dlq {
				failedMsg = append(failedMsg, m)
			}

			excepted, actual := handleResult(data, res, failedMsg)

			require.Equal(t, excepted, actual)
		})
	}
}

func TestFailedCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDao := mock_el.NewMockIElSearchDao(ctrl)

	cases := []TestCase{
		{
			name:             "前三次失敗，後續成功",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				callCount := 0
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(key string, docs []map[string]interface{}) error {
						callCount++
						if callCount <= 3 {
							return errors.New("fail")
						}
						return nil
					}).AnyTimes()
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
			name:             "前三次失敗，後續成功",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				callCount := 0
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(key string, docs []map[string]interface{}) error {
						callCount++
						if callCount <= 3 {
							return errors.New("fail")
						}
						return nil
					}).AnyTimes()
			},
			caseN:      10,
			bufferSize: 100,
			nRoutine:   100,
			setupIn: func(input []kafka.Message, in chan kafka.Message) {
				for _, m := range input {
					in <- m
				}
			},
		},
		{
			name:             "基數失敗，偶數成功",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				callCount := 0
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(key string, docs []map[string]interface{}) error {
						callCount++
						if callCount%2 != 0 {
							return errors.New("fail")
						}
						return nil
					}).AnyTimes()
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
			name:             "全部失敗",
			generateCasefunc: generateTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(errors.New("failed")).AnyTimes()
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.generateCasefunc(tc.caseN)

			in, out := make(chan kafka.Message, tc.bufferSize), make(chan kafka.Message, tc.caseN)
			dlq := make(chan kafka.Message, tc.bufferSize)
			go func() {
				defer close(in)
				tc.setupIn(data, in)
			}()

			tc.setUPMock(mockDao)

			KafkaElProcesser, err := NewKafkaElProcesser(mockDao, tc.bufferSize, time.Millisecond)
			require.Nil(t, err)

			var wg sync.WaitGroup
			for i := 0; i < tc.nRoutine; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					KafkaElProcesser.Process(context.Background(), in, out, dlq)
				}()
			}

			go func() {
				for {
					select {
					case _, ok := <-dlq:
						if !ok {
							return
						}
					}
				}
			}()

			wg.Wait()
			close(out)
			close(dlq)

			res := make([]kafka.Message, 0, tc.caseN)
			for o := range out {
				res = append(res, o)
			}

			failedMsg := make([]kafka.Message, 0, tc.caseN)
			for m := range dlq {
				failedMsg = append(failedMsg, m)
			}

			excepted, actual := handleResult(data, res, failedMsg)

			require.Equal(t, excepted, actual)
		})
	}
}

func TestFailedMessageCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDao := mock_el.NewMockIElSearchDao(ctrl)

	cases := []TestCase{
		{
			name:             "全部失敗",
			generateCasefunc: generateBadTestMessage,
			setUPMock: func(m *mock_el.MockIElSearchDao) {
				m.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Times(0)
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.generateCasefunc(tc.caseN)

			in, out := make(chan kafka.Message, tc.bufferSize), make(chan kafka.Message, tc.caseN)
			dlq := make(chan kafka.Message, tc.bufferSize)
			go func() {
				defer close(in)
				tc.setupIn(data, in)
			}()

			tc.setUPMock(mockDao)

			KafkaElProcesser, err := NewKafkaElProcesser(mockDao, tc.bufferSize, time.Millisecond)
			require.Nil(t, err)

			var wg sync.WaitGroup
			for i := 0; i < tc.nRoutine; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					KafkaElProcesser.Process(context.Background(), in, out, dlq)
				}()
			}

			failedMsg := make([]kafka.Message, 0, tc.caseN)

			go func() {
				for m := range dlq {
					failedMsg = append(failedMsg, m)
				}
			}()

			res := make([]kafka.Message, 0, tc.caseN)
			go func() {
				for o := range out {
					res = append(res, o)
				}

			}()

			wg.Wait()
			close(out)
			close(dlq)

			excepted, actual := handleResult(data, res, failedMsg)

			require.Len(t, actual, len(excepted))
			require.Equal(t, excepted, actual)
		})
	}
}
