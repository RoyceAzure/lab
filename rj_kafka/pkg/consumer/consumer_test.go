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

	"github.com/RoyceAzure/lab/rj_kafka/pkg/config"
	mock_consumer "github.com/RoyceAzure/lab/rj_kafka/pkg/consumer/mock"
	"github.com/RoyceAzure/lab/rj_kafka/pkg/model"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type TestMsg struct {
	Id      int    `json:"id"`
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

func generateTestMessage(n int) []kafka.Message {
	t := make([]kafka.Message, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(i))
		testMsg := TestMsg{
			Id:      i,
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
		earilyStop         time.Duration
		setUpReaderMock    func(chan kafka.Message, chan kafka.Message, *mock_consumer.MockKafkaReader)
		generateTestMsg    func(int) []kafka.Message
		setUpProcesserMock func(m *mock_consumer.MockProcesser)
		handlerSuccessfunc func(kafka.Message)
		handlerErrorfunc   func(model.ConsuemError)
	}{
		{
			name:     "all pass, reader EOF end",
			testMsgs: 1000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:       "all pass, reader EOF end, early stop",
			testMsgs:   10000,
			earilyStop: 2 * time.Nanosecond,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:     "0 msgs , eof",
			testMsgs: 0,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:     "all pass, big data",
			testMsgs: 100000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:     "partial failed",
			testMsgs: 1000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}

						if int(binary.BigEndian.Uint32(m.Key))%3 == 0 {
							// 每三個失敗
							return kafka.Message{}, fmt.Errorf("test failed")
						}

						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:     "partial read auth failed",
			testMsgs: 1000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}

						if int(binary.BigEndian.Uint32(m.Key)) == 500 {
							// 每三個失敗
							return kafka.Message{}, fmt.Errorf("SASL Authentication Failed")
						}

						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:       "partial read failed and early stop",
			earilyStop: 10 * time.Nanosecond,
			testMsgs:   10000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}

						if int(binary.BigEndian.Uint32(m.Key))%3 == 0 {
							// 每三個失敗
							return kafka.Message{}, fmt.Errorf("SASL Authentication Failed")
						}

						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
				errorMu.Lock()
				defer errorMu.Unlock()
				failedMsgs = append(failedMsgs, err.Message)
			},
		},
		{
			name:     "partial process failed",
			testMsgs: 1000,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			setUpProcesserMock: func(m *mock_consumer.MockProcesser) {
				m.EXPECT().Process(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, msgs []kafka.Message) error {
						if int(binary.BigEndian.Uint32(msgs[0].Key))%2 == 0 {
							// 每三個失敗
							return fmt.Errorf("process batch failed")
						}
						return nil
					}).AnyTimes()
			},
			handlerSuccessfunc: func(msg kafka.Message) {
				successMu.Lock()
				defer successMu.Unlock()
				successMsgs = append(successMsgs, msg)
			},
			handlerErrorfunc: func(err model.ConsuemError) {
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
			successMsgs, failedMsgs = make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
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

			cfg := config.DefaultConfig()
			cfg.BatchSize = 1000
			cfg.WorkerNum = 10
			concumser := NewConsumer(mock_reader,
				mockProcesser,
				*cfg,
				SetHandlerSuccessfunc(tc.handlerSuccessfunc),
				SetHandlerFailedfunc(tc.handlerErrorfunc))

			concumser.Start()

			if tc.earilyStop > 0 {
				select {
				case <-time.After(tc.earilyStop):
					concumser.Close(time.Second * 5)
				case <-concumser.C():
				case <-time.After(time.Second * 10):
					require.Fail(t, "timeout: test did not complete within 10 seconds")
				}
			} else {
				select {
				case <-concumser.C():
				case <-time.After(time.Second * 10):
					require.Fail(t, "timeout: test did not complete within 10 seconds")
				}
			}
			close(expected)
			exp, actual := handleResultConsumer(successMsgs, failedMsgs, expected)
			require.Equal(t, exp, actual)
		})
	}
}

func TestConsumerMutiStop(t *testing.T) {
	var successMsgs, failedMsgs []kafka.Message
	successMu, errorMu := sync.Mutex{}, sync.Mutex{}
	testCases := []struct {
		name               string
		testMsgs           int
		earilyStop         time.Duration
		setUpReaderMock    func(chan kafka.Message, chan kafka.Message, *mock_consumer.MockKafkaReader)
		generateTestMsg    func(int) []kafka.Message
		setUpProcesserMock func(m *mock_consumer.MockProcesser)
		handlerSuccessfunc func(kafka.Message)
		handlerErrorfunc   func(model.ConsuemError)
	}{
		{
			name:       "all pass, reader EOF end, early stop",
			testMsgs:   10000,
			earilyStop: 2 * time.Nanosecond,
			setUpReaderMock: func(in, expected chan kafka.Message, reader *mock_consumer.MockKafkaReader) {
				reader.EXPECT().FetchMessage(gomock.Any()).
					DoAndReturn(func(ctx context.Context) (kafka.Message, error) {
						m, ok := <-in
						if !ok {
							return kafka.Message{}, io.EOF
						}
						expected <- m
						return m, nil
					}).AnyTimes()
				reader.EXPECT().CommitMessages(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				reader.EXPECT().Close().Return(nil).AnyTimes()
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
			handlerErrorfunc: func(err model.ConsuemError) {
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
			successMsgs, failedMsgs = make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
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

			cfg := config.DefaultConfig()
			cfg.BatchSize = 1000
			cfg.WorkerNum = 10
			concumser := NewConsumer(mock_reader,
				mockProcesser,
				*cfg,
				SetHandlerSuccessfunc(tc.handlerSuccessfunc),
				SetHandlerFailedfunc(tc.handlerErrorfunc))

			concumser.Start()

			if tc.earilyStop > 0 {
				select {
				case <-time.After(tc.earilyStop):
					concumser.Close(time.Second * 5)
					concumser.Close(time.Second * 5)
					concumser.Close(time.Second * 5)
					concumser.Close(time.Second * 5)
				case <-concumser.C():
				case <-time.After(time.Second * 10):
					require.Fail(t, "timeout: test did not complete within 10 seconds")
				}
			} else {
				select {
				case <-concumser.C():
				case <-time.After(time.Second * 10):
					require.Fail(t, "timeout: test did not complete within 10 seconds")
				}
			}
			close(expected)
			exp, actual := handleResultConsumer(successMsgs, failedMsgs, expected)
			require.Equal(t, exp, actual)
		})
	}
}
