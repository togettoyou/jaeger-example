package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func init() {
	cfg := jaegercfg.Configuration{
		ServiceName: "jaeger-example", // 服务名
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, _, err := cfg.NewTracer(
		jaegercfg.Logger(jaegerlog.StdLogger),
		jaegercfg.Metrics(metrics.NullFactory),
	)
	if err != nil {
		panic(err)
	}
	opentracing.SetGlobalTracer(tracer)
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
	span := opentracing.StartSpan("请求 /home")
	defer span.Finish()

	// 发起异步请求
	asyncReq, _ := http.NewRequest("GET", "http://localhost:8080/async", nil)
	// 传递span的上下文信息
	// 将关于本地追踪调用的span context，设置到http header上，并传递出去
	err := span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(asyncReq.Header))
	if err != nil {
		log.Fatalf("[asyncReq]无法添加span context到http header: %v", err)
	}
	go func() {
		if _, err := http.DefaultClient.Do(asyncReq); err != nil {
			// 请求失败，为span设置tags和logs
			span.SetTag("error", true)
			span.LogKV(fmt.Sprintf("请求 /async error: %v", err))
		}
	}()

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

	// 发起同步请求
	syncReq, _ := http.NewRequest("GET", "http://localhost:8888/service", nil)
	err = span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(syncReq.Header))
	if err != nil {
		log.Fatalf("[syncReq]无法添加span context到http header: %v", err)
	}
	if _, err = http.DefaultClient.Do(syncReq); err != nil {
		span.SetTag("error", true)
		span.LogKV(fmt.Sprintf("请求 /service error: %v", err))
	}
	w.Write([]byte("请求结束！"))
}

// 模拟业务请求
func serviceHandler(w http.ResponseWriter, r *http.Request) {
	// 通过http header，提取span元数据信息
	var sp opentracing.Span
	opName := r.URL.Path
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// 获取失败，则直接新建一个根节点 span
		sp = opentracing.StartSpan(opName)
	} else {
		sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
	}
	defer sp.Finish()

	dbReq, _ := http.NewRequest("GET", "http://localhost:8888/db", nil)
	err = sp.Tracer().Inject(sp.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(dbReq.Header))
	if err != nil {
		log.Fatalf("[dbReq]无法添加span context到http header: %v", err)
	}
	if _, err = http.DefaultClient.Do(dbReq); err != nil {
		sp.SetTag("error", true)
		sp.LogKV("请求 /da error", err)
	}

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
}

// 模拟DB调用
func dbHandler(w http.ResponseWriter, r *http.Request) {
	// 通过http header，提取span元数据信息
	var sp opentracing.Span
	opName := r.URL.Path
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// 获取失败，则直接新建一个根节点 span
		sp = opentracing.StartSpan(opName)
	} else {
		sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
	}
	defer sp.Finish()

	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
}
