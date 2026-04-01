// backend/cmd/email-worker/main.go
/*
	邮件消费者
*/
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/trace"
	"strconv"
	"strings"
	"syscall"
	"time"

	cf "github.com/chenzanhong/formallanglab-email/configs"
	"github.com/chenzanhong/formallanglab-email/middleware/breaker"
	"github.com/chenzanhong/formallanglab-email/middleware/metrics"
	"github.com/chenzanhong/formallanglab-email/model"
	"github.com/chenzanhong/zlog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"gopkg.in/gomail.v2"
)

// 全局熔断器管理器
var breakerManager *breaker.CircuitBreakerManager

func main() {
	config, err := cf.LoadEmailWorkerConfig()
	if err != nil {
		log.Fatalf("加载配置失败：%v", err.Error())
	}
	zlog.InitLogger(config.Zlog)
	cf.SetEmailEnvVariables(config)
	metrics.PrometheusRegister()

	// 初始化熔断器管理器
	breakerManager = breaker.NewCircuitBreakerManager()

	fmt.Println(config.Kafka.Brokers)
	brokers := strings.Split(strings.TrimSpace(config.Kafka.Brokers), ",") // 从配置读取
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   config.Kafka.Topic,
		GroupID: "email-service-group",
	})
	fmt.Println(config.Kafka.Topic)

	go func() {
		if v, ok := os.LookupEnv("SERVER_PORT"); ok {
			fmt.Println("----- SERVER_PORT: ", v, " -----")
			mux := http.NewServeMux()
			mux.HandleFunc("/gdesign/master/health", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			})
			http.ListenAndServe(fmt.Sprintf(":%s", v), mux)
		} else {
			zlog.Panic("SERVER_PORT is required")
		}
	}()

	// 启动 Prometheus metrics server
	go func() {
		if v, ok := os.LookupEnv("METRICS_PORT"); ok {
			fmt.Println("----- METRICS_PORT: ", v, " -----")
			mux := http.NewServeMux()
			mux.Handle("/gdesign/email/metrics", promhttp.Handler())
			zlog.Info("Prometheus metrics server starting on :" + v)
			if err := http.ListenAndServe(fmt.Sprintf(":%s", v), mux); err != nil && err != http.ErrServerClosed {
				zlog.Fatalf("Metrics server failed: %v", err)
			}
		}
	}()

	// 应用 trace
	go func() {
		if v, ok := os.LookupEnv("ENABLE_TRACE"); ok && v == "true" {
			f, _ := os.Create("server.trace")
			defer f.Close()
			trace.Start(f)
			// go tool trace server.trace
			defer trace.Stop()
		}
	}()

	// 启动pprof http服务
	go func() {
		if v, ok := os.LookupEnv("PPROF_PORT"); ok && v != "" && v != "0" {
			fmt.Println("----- PPROF_PORT: ", v, " -----")
			zlog.Infof("Starting pprof on localhost:%s", v)
			http.ListenAndServe(fmt.Sprintf("localhost:%s", v), nil)
		}
	}()

	// 优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	zlog.Info("Email worker started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			zlog.Info("Shutting down email worker...")
			reader.Close()

			return
		default:
			// fmt.Println(1)
			msg, err := reader.ReadMessage(ctx) // 阻塞读取
			// fmt.Println(2)
			if err != nil {
				if strings.Contains(err.Error(), "failed to open connection") {
					zlog.Errorw("Kafka connection failed", "topic", config.Kafka.Topic, "error", err)
				} else {
					zlog.Warnw("Failed to read message", "error", err)
				}
				time.Sleep(2 * time.Second)

				continue
			}

			var event model.KafkaEmailEvent
			if err = json.Unmarshal(msg.Value, &event); err != nil {
				zlog.Infof("Failed to unmarshal email event: %v", err)
				continue
			}

			// 发送邮件
			err = sendEmailWithMetrics(event, config.MaxRetries)
			if err != nil {
				zlog.Errorw("Email sending failed after retries", "to", event.To, "error", err)
			} else {
				zlog.Infow("Email sent successfully", "to", event.To, "subject", event.Subject)
			}
		}
	}
}

// sendEmailWithMetrics 发送邮件并记录 Prometheus 指标
func sendEmailWithMetrics(event model.KafkaEmailEvent, maxRetries int) error {
	start := time.Now()
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, execErr := breakerManager.ExecuteWithBreaker(breakerManager.GetSMTPBreaker(), func() (interface{}, error) {
			err = sendEmailSync(event.To, event.Subject, event.ContentType, event.Body)
			return nil, err
		})

		if execErr != nil {
			zlog.Warnw("Circuit breaker tripped for SMTP service", "attempt", attempt, "error", execErr)
			err = execErr
			break
		}

		if err == nil {
			break
		}

		zlog.Warnw("Failed to send email, retrying...", "attempt", attempt, "error", err)
		if attempt < maxRetries {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		} else {
			// 可选：发送到死信队列（DLQ），不过验证码一分钟后过期，所以这里不放死信队列，前端用户手动重试
		}
	}

	duration := time.Since(start).Seconds()
	metrics.ObserveOperationDuration("email", "send", duration)
	if err != nil {
		metrics.IncOperation("email", "send", "failure")
	} else {
		metrics.IncOperation("email", "send", "success")
	}

	return err
}

// sendEmailSync 是实际的同步发送逻辑
func sendEmailSync(email, subject, contentType, body string) error {
	myEmail := os.Getenv("EMAIL_NAME")
	myPassword := os.Getenv("EMAIL_PASSWORD")
	smtpServerHost := os.Getenv("SMTP_SERVER_HOST")
	smtpServerPortStr := os.Getenv("SMTP_SERVER_PORT")

	if myEmail == "" || myPassword == "" || smtpServerHost == "" || smtpServerPortStr == "" {
		return errors.New("环境变量未正确设置")
	}

	smtpServerPort, err := strconv.Atoi(smtpServerPortStr)
	if err != nil {
		return fmt.Errorf("SMTP 端口解析失败：%w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", myEmail)
	m.SetHeader("To", email)
	m.SetHeader("Subject", subject)
	m.SetBody(contentType, body)

	d := gomail.NewDialer(smtpServerHost, smtpServerPort, myEmail, myPassword)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // 生产环境建议使用有效证书

	if err := d.DialAndSend(m); err != nil {
		if strings.Contains(err.Error(), "535") {
			return errors.New("SMTP 身份验证失败，请检查邮箱账号或授权码")
		} else if strings.Contains(err.Error(), "connection refused") {
			return errors.New("无法连接 SMTP 服务器，请检查网络或服务器地址")
		}

		return fmt.Errorf("邮件发送失败: %w", err)
	}

	zlog.Infow("邮件发送成功", "to", email, "subject", subject)

	return nil
}
