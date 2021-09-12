package main

import (
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func init() {
	exp, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		panic(err)
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		// 应用程序信息
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("opentelemetry-example"), // 服务名
			semconv.ServiceVersionKey.String("0.0.1"),
		)),
	)
	otel.SetTracerProvider(tp)
}

func main() {
	port := 8080
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/home", homeHandler)
	mux.HandleFunc("/async", serviceHandler)
	mux.HandleFunc("/service", serviceHandler)
	mux.HandleFunc("/db", dbHandler)
	fmt.Printf("http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// 主页 Html
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<a href="/home"> 点击开始发起请求 </a>`))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("开始请求...\n"))

	// 在入口处设置一个根节点 span
	tr := otel.Tracer("请求 /home")
	ctx, span := tr.Start(r.Context(), "bar")
	defer span.End()

	// 发起异步请求
	asyncReq, _ := http.NewRequest("GET", "http://localhost:8080/async", nil)
	// 传递span的上下文信息
	// 将关于本地追踪调用的span context，设置到http header上，并传递出去
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(asyncReq.Header),
	)
	go func() {
		if _, err := http.DefaultClient.Do(asyncReq); err != nil {
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("请求 /async error", err.Error()),
			)
		}
	}()

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

	// 发起同步请求
	syncReq, _ := http.NewRequest("GET", "http://localhost:8080/service", nil)
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(asyncReq.Header),
	)
	if _, err := http.DefaultClient.Do(syncReq); err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("请求 /service error", err.Error()),
		)
	}
	w.Write([]byte("请求结束！"))
}

// 模拟业务请求
func serviceHandler(w http.ResponseWriter, r *http.Request) {
	// 通过http header，提取span元数据信息
	opName := r.URL.Path
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tr := otel.Tracer(opName)
	ctx, span := tr.Start(ctx, opName)
	defer span.End()

	dbReq, _ := http.NewRequest("GET", "http://localhost:8080/db", nil)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(dbReq.Header))
	if _, err := http.DefaultClient.Do(dbReq); err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("请求 /db error", err.Error()),
		)
	}

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
}

// 模拟DB调用
func dbHandler(w http.ResponseWriter, r *http.Request) {
	// 通过http header，提取span元数据信息
	opName := r.URL.Path
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tr := otel.Tracer(opName)
	ctx, span := tr.Start(ctx, opName)
	defer span.End()

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
}