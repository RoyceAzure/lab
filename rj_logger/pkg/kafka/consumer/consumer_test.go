package consumer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	mock_consumer "github.com/RoyceAzure/lab/rj_logger/pkg/kafka/consumer/mock"
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
		buf := make([]byte, 4)
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

func handleResultConsumer(success, failed []kafka.Message, readIn chan kafka.Message) (excepted map[string]any, actual map[string]any) {
	readInMsgs := make([]kafka.Message, 0)
	for msg := range readIn {
		readInMsgs = append(readInMsgs, msg)
	}

	excepted, actual = make(map[string]any), make(map[string]any)
	for _, v := range readInMsgs {
		excepted[string(v.Key)] = v.Value
	}

	for _, v := range success {
		actual[string(v.Key)] = v.Value
	}

	for _, v := range failed {
		actual[string(v.Key)] = v.Value
	}
	return
}

func TestBasicConsumer(t *testing.T) {
	var successMsgs, failedMsgs []kafka.Message
	successMu, errorMu := sync.Mutex{}, sync.Mutex{}
	testCases := []struct {
		name               string
		testMsgs           int
		setUpReaderMock    func(chan kafka.Message, chan kafka.Message, *mock_consumer.MockKafkaReader)
		generateTestMsg    func(int) []kafka.Message
		setUpProcesserMock func(m *mock_consumer.MockProcesser)
		handlerSuccessfunc func(kafka.Message)
		handlerErrorfunc   func(ConsuemError)
	}{
		{
			name:     "all pass, reader EOF end",
			testMsgs: 1000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							close(expected)
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			setUpProcesserMock: func(m *mock_consumer.MockProcesser) {
				m.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			handlerSuccessfunc: func(msg kafka.Message) {
				successMu.Lock()
				defer successMu.Unlock()
				successMsgs = append(successMsgs, msg)
			},
			handlerErrorfunc: func(err ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			successMsgs, failedMsgs = make([]kafka.Message, tc.testMsgs), make([]kafka.Message, tc.testMsgs)
			Inch, expected := make(chan kafka.Message, tc.testMsgs), make(chan kafka.Message, tc.testMsgs)

			mockProcesser := mock_consumer.NewMockProcesser(ctrl)
			tc.setUpProcesserMock(mockProcesser)

			mock_reader := mock_consumer.NewMockKafkaReader(ctrl)
			tc.setUpReaderMock(Inch, expected, mock_reader)

			testMsgs := tc.generateTestMsg(tc.testMsgs)

			go func() {
				defer close(Inch)
				for _, msg := range testMsgs {
					Inch <- msg
				}
			}()

			concumser := NewConsumer(mock_reader, mockProcesser, tc.handlerSuccessfunc, tc.handlerErrorfunc)
			concumser.Start()

			select {
			case <-concumser.C():
			case <-time.After(time.Second * 10):
				require.Fail(t, "timeout: test did not complete within 10 seconds")
			}

			exp, actual := handleResultConsumer(successMsgs, failedMsgs, expected)
			require.Equal(t, exp, actual)
		})
	}
}
