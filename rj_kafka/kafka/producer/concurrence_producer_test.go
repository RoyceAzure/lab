package producer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	mock_producer "github.com/RoyceAzure/lab/rj_kafka/kafka/producer/mock"
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

func handleResult(successMsgs, failedMsgs, writerSuccess, writerFailed []kafka.Message) []map[string]struct{} {
	allRes := make([]map[string]struct{}, 4)
	successM := make(map[string]struct{}, len(successMsgs))
	for _, v := range successMsgs {
		successM[string(v.Key)] = struct{}{}
	}
	allRes[0] = successM

	failedM := make(map[string]struct{}, len(failedMsgs))
	for _, v := range failedMsgs {
		failedM[string(v.Key)] = struct{}{}
	}
	allRes[1] = failedM

	writerSuccessM := make(map[string]struct{}, len(writerSuccess))
	for _, v := range writerSuccess {
		writerSuccessM[string(v.Key)] = struct{}{}
	}
	allRes[2] = writerSuccessM

	writerFailedM := make(map[string]struct{}, len(writerFailed))
	for _, v := range writerFailed {
		writerFailedM[string(v.Key)] = struct{}{}
	}
	allRes[3] = writerFailedM
	return allRes
}

func handleAllRes(allTest, writerSuccess, writerFailed []kafka.Message, sendFailed []kafka.Message) []map[string]struct{} {
	allRes := make([]map[string]struct{}, 2)
	allTestM := make(map[string]struct{}, len(allTest))
	for _, v := range allTest {
		allTestM[string(v.Key)] = struct{}{}
	}
	allRes[0] = allTestM

	writerM := make(map[string]struct{}, len(writerSuccess)+len(writerFailed)+len(sendFailed))
	for _, v := range writerSuccess {
		writerM[string(v.Key)] = struct{}{}
	}

	for _, v := range writerFailed {
		writerM[string(v.Key)] = struct{}{}
	}
	for _, v := range sendFailed {
		writerM[string(v.Key)] = struct{}{}
	}

	allRes[1] = writerM

	return allRes
}

// writer buffer容量夠，沒有任何訊息被遣返
func TestBasicProducer(t *testing.T) {
	testCases := []struct {
		name               string
		testMsgs           int
		earilyStop         time.Duration
		setUpWriterMock    func(*[]kafka.Message, *[]kafka.Message, *mock_producer.MockWriter)
		generateTestMsg    func(int) []kafka.Message
		handlerSuccessfunc func(*[]kafka.Message) func(m kafka.Message)
		handlerErrorfunc   func(*[]kafka.Message) func(ProducerError)
	}{
		{
			name:     "all pass, writer EOF end",
			testMsgs: 1000,
			setUpWriterMock: func(writerSuccess *[]kafka.Message, writerFailed *[]kafka.Message, writer *mock_producer.MockWriter) {
				writer.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, msgs ...kafka.Message) error {
						if len(msgs) == 0 {
							return nil
						}
						index := binary.BigEndian.Uint32(msgs[0].Key)
						if index%3 == 0 {
							*writerSuccess = append(*writerSuccess, msgs...)
							return nil
						} else {
							*writerFailed = append(*writerFailed, msgs...)
							return fmt.Errorf("temporary failed")
						}
					}).AnyTimes()
				writer.EXPECT().Close().Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			handlerSuccessfunc: func(successMsgs *[]kafka.Message) func(m kafka.Message) {
				return func(m kafka.Message) {
					*successMsgs = append(*successMsgs, m)
				}
			},
			handlerErrorfunc: func(errMsgs *[]kafka.Message) func(ProducerError) {
				return func(err ProducerError) {
					*errMsgs = append(*errMsgs, err.Message)
				}
			},
		},
		{
			name:     "big data, all pass, writer EOF end",
			testMsgs: 20000,
			setUpWriterMock: func(writerSuccess *[]kafka.Message, writerFailed *[]kafka.Message, writer *mock_producer.MockWriter) {
				writer.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, msgs ...kafka.Message) error {
						if len(msgs) == 0 {
							return nil
						}
						index := binary.BigEndian.Uint32(msgs[0].Key)
						if index%3 == 0 {
							*writerSuccess = append(*writerSuccess, msgs...)
							return nil
						} else {
							*writerFailed = append(*writerFailed, msgs...)
							return fmt.Errorf("temporary failed")
						}
					}).AnyTimes()
				writer.EXPECT().Close().Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			handlerSuccessfunc: func(successMsgs *[]kafka.Message) func(m kafka.Message) {
				return func(m kafka.Message) {
					*successMsgs = append(*successMsgs, m)
				}
			},
			handlerErrorfunc: func(errMsgs *[]kafka.Message) func(ProducerError) {
				return func(err ProducerError) {
					*errMsgs = append(*errMsgs, err.Message)
				}
			},
		},
		{
			name:     "with fatal error",
			testMsgs: 10000,
			setUpWriterMock: func(writerSuccess *[]kafka.Message, writerFailed *[]kafka.Message, writer *mock_producer.MockWriter) {
				writer.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, msgs ...kafka.Message) error {
						if len(msgs) == 0 {
							return nil
						}
						index := binary.BigEndian.Uint32(msgs[0].Key)
						if index >= 4000 {
							*writerFailed = append(*writerFailed, msgs...)
							return kafka.TopicAuthorizationFailed
						} else {
							*writerSuccess = append(*writerSuccess, msgs...)
							return nil
						}
					}).AnyTimes()
				writer.EXPECT().Close().Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			handlerSuccessfunc: func(successMsgs *[]kafka.Message) func(m kafka.Message) {
				return func(m kafka.Message) {
					*successMsgs = append(*successMsgs, m)
				}
			},
			handlerErrorfunc: func(errMsgs *[]kafka.Message) func(ProducerError) {
				return func(err ProducerError) {
					*errMsgs = append(*errMsgs, err.Message)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			successMsgs, failedMsgs := make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
			writerSuccess, writerFailed := make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
			sendFailedMsg := make([]kafka.Message, 0, tc.testMsgs)
			mock_writer := mock_producer.NewMockWriter(ctrl)
			tc.setUpWriterMock(&writerSuccess, &writerFailed, mock_writer)

			testMsgs := tc.generateTestMsg(tc.testMsgs)

			cfg := config.DefaultConfig()
			cfg.Brokers = []string{"test_broker"}
			cfg.Topic = "test_topic"
			cfg.BatchSize = 2000
			cfg.CommitInterval = 100 * time.Millisecond
			kafkaWriter, err := NewConcurrencekafkaProducer(mock_writer,
				*cfg,
				SetHandlerSuccessfunc(tc.handlerSuccessfunc(&successMsgs)),
				SetHandlerFailedfunc(tc.handlerErrorfunc(&failedMsgs)),
			)
			require.Nil(t, err)
			kafkaWriter.Start()

			testEnd := make(chan struct{})
			go func() {
				defer func() {
					t.Log("calling writer Close()")
					err = kafkaWriter.Close(time.Second * 20)
					testEnd <- struct{}{}
				}()
				buffer := 1000
				start := 0
				end := start + buffer
				l := len(testMsgs)
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					if end > l {
						end = l
					}
					sendFailed, err := kafkaWriter.Produce(context.Background(), testMsgs[start:end])
					if err != nil {
						sendFailedMsg = append(sendFailedMsg, sendFailed...)
					}
					if end == l {
						return
					}
					start = end
					end += buffer
				}
			}()

			<-testEnd
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			require.Nil(t, err)
			eachRes := handleResult(successMsgs, failedMsgs, writerSuccess, writerFailed)

			require.Equal(t, eachRes[0], eachRes[2], "successMsgs and writerSuccess not equal")
			require.Equal(t, eachRes[1], eachRes[3], "failedMsgs and writerFailed not equal")
			allRes := handleAllRes(testMsgs, successMsgs, failedMsgs, sendFailedMsg)
			require.Equal(t, len(allRes[0]), len(allRes[1]))
			require.Equal(t, allRes[0], allRes[1], "all test res and all writer res are not equal")
		})
	}
}

func TestProducerAdbvance(t *testing.T) {
	testCases := []struct {
		name               string
		testMsgs           int
		earilyStop         time.Duration
		setUpWriterMock    func(*[]kafka.Message, *[]kafka.Message, *mock_producer.MockWriter)
		generateTestMsg    func(int) []kafka.Message
		handlerSuccessfunc func(*[]kafka.Message) func(m kafka.Message)
		handlerErrorfunc   func(*[]kafka.Message) func(ProducerError)
	}{
		{
			name:     "early stop, producer is already closed",
			testMsgs: 10000,
			setUpWriterMock: func(writerSuccess *[]kafka.Message, writerFailed *[]kafka.Message, writer *mock_producer.MockWriter) {
				writer.EXPECT().WriteMessages(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, msgs ...kafka.Message) error {
						if len(msgs) == 0 {
							return nil
						}
						index := binary.BigEndian.Uint32(msgs[0].Key)
						if index%3 == 0 {
							*writerSuccess = append(*writerSuccess, msgs...)
							return nil
						} else {
							*writerFailed = append(*writerFailed, msgs...)
							return fmt.Errorf("temporary failed")
						}
					}).AnyTimes()
				writer.EXPECT().Close().Return(nil).AnyTimes()
			},
			generateTestMsg: generateTestMessage,
			handlerSuccessfunc: func(successMsgs *[]kafka.Message) func(m kafka.Message) {
				return func(m kafka.Message) {
					*successMsgs = append(*successMsgs, m)
				}
			},
			handlerErrorfunc: func(errMsgs *[]kafka.Message) func(ProducerError) {
				return func(err ProducerError) {
					*errMsgs = append(*errMsgs, err.Message)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			successMsgs, failedMsgs := make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
			writerSuccess, writerFailed := make([]kafka.Message, 0, tc.testMsgs), make([]kafka.Message, 0, tc.testMsgs)
			sendFailedMsg := make([]kafka.Message, 0, tc.testMsgs)
			mock_writer := mock_producer.NewMockWriter(ctrl)
			tc.setUpWriterMock(&writerSuccess, &writerFailed, mock_writer)

			testMsgs := tc.generateTestMsg(tc.testMsgs)

			cfg := config.DefaultConfig()
			cfg.Brokers = []string{"test_broker"}
			cfg.Topic = "test_topic"
			cfg.BatchSize = 2000
			cfg.CommitInterval = 100 * time.Millisecond
			kafkaWriter, err := NewConcurrencekafkaProducer(mock_writer,
				*cfg,
				SetHandlerSuccessfunc(tc.handlerSuccessfunc(&successMsgs)),
				SetHandlerFailedfunc(tc.handlerErrorfunc(&failedMsgs)),
			)
			require.Nil(t, err)
			kafkaWriter.Start()

			testEnd := make(chan struct{})
			go func() {
				defer func() {
					t.Log("calling writer Close()")
					testEnd <- struct{}{}
				}()
				buffer := 1000
				start := 0
				end := start + buffer
				l := len(testMsgs)

				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					if end > l {
						end = l
					}
					sendFailed, err := kafkaWriter.Produce(context.Background(), testMsgs[start:end])
					if err != nil {
						sendFailedMsg = append(sendFailedMsg, sendFailed...)
					}
					if end == l {
						return
					}
					start = end
					end += buffer
				}

			}()

			go func() {
				early_ticker := time.NewTicker(200 * time.Millisecond)
				defer early_ticker.Stop()
				for range early_ticker.C {
					kafkaWriter.Close(time.Second * 20)
					return
				}
			}()

			<-testEnd
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			require.Nil(t, err)
			eachRes := handleResult(successMsgs, failedMsgs, writerSuccess, writerFailed)

			require.Equal(t, eachRes[0], eachRes[2], "successMsgs and writerSuccess not equal")
			require.Equal(t, eachRes[1], eachRes[3], "failedMsgs and writerFailed not equal")
			allRes := handleAllRes(testMsgs, successMsgs, failedMsgs, sendFailedMsg)
			require.Equal(t, len(allRes[0]), len(allRes[1]))
			require.Equal(t, allRes[0], allRes[1], "all test res and all writer res are not equal")
		})
	}
}
